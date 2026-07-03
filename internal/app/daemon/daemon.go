package daemon

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/kyson-dev/sing-helm/internal/proxy/config/model"
	"github.com/kyson-dev/sing-helm/internal/proxy/engine"
	"github.com/kyson-dev/sing-helm/internal/sys/ipc"
	"github.com/kyson-dev/sing-helm/internal/sys/lock"
	"github.com/kyson-dev/sing-helm/internal/sys/logger"
	"github.com/kyson-dev/sing-helm/internal/sys/paths"
	"github.com/kyson-dev/sing-helm/internal/sys/sysnet"
)

// ServiceRunner abstracts the sing-box engine lifecycle.
type ServiceRunner interface {
	StartFromFile(context.Context, string) error
	ReloadFromFile(context.Context, string) error
	Stop()
}

// Daemon manages the sing-box service lifecycle and responds to IPC commands.
type Daemon struct {
	mu             sync.Mutex
	cancelFunc     context.CancelFunc
	service        ServiceRunner
	serviceFactory func() ServiceRunner
	lock           *lock.DaemonLock
	running        bool
	reloading      bool
	state          *RuntimeState
	dnsProxy       *sysnet.DNSProxy // non-nil when DNS proxy is active (system/default mode)
}

// NewDaemon builds a daemon controller.
func NewDaemon() *Daemon {
	return &Daemon{
		serviceFactory: func() ServiceRunner {
			return engine.NewInstance()
		},
	}
}

// SetServiceFactory overrides the service factory (useful for tests).
func (d *Daemon) SetServiceFactory(factory func() ServiceRunner) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if factory == nil {
		d.serviceFactory = func() ServiceRunner {
			return engine.NewInstance()
		}
		return
	}
	d.serviceFactory = factory
}

// Serve starts the IPC server. Blocks until ctx is cancelled.
func (d *Daemon) Serve(ctx context.Context) error {

	lock, err := lock.AcquireLock(paths.Get().LockFile)
	if err != nil {
		return fmt.Errorf("another instance is already running: %w", err)
	}
	d.lock = lock
	d.loadState()
	_ = saveRuntimeMeta(paths.Get().RuntimeMetaFile, RuntimeMeta{
		ConfigHome: paths.Get().HomeDir,
	})

	ctx, cancel := context.WithCancel(ctx)
	d.cancelFunc = cancel
	defer func() {
		logger.Info("Daemon shutting down")
		cancel()
		d.cleanup()
	}()

	logger.Info("Daemon started, listening for IPC commands")

	if err := ipc.Serve(ctx, paths.Get().SocketFile, d, &ipc.ServerOptions{}); err != nil {
		return err
	}
	return nil
}

// Handle routes IPC commands to the proper handlers.
func (d *Daemon) Handle(ctx context.Context, cmd ipc.CommandMessage) ipc.CommandResult {
	switch cmd.Name {
	case "run":
		return d.handleRun(ctx, cmd.Payload)
	case "stop":
		return d.handleStop()
	case "status":
		return d.handleStatus()
	case "mode":
		return d.handleMode(ctx, cmd.Payload)
	case "route":
		return d.handleRoute(ctx, cmd.Payload)
	case "node.list":
		return d.handleNodeList(cmd.Payload)
	case "node.use":
		return d.handleNodeUse(cmd.Payload)
	case "log":
		return d.handleLog()
	case "health":
		return d.handleHealth()
	case "reload":
		return d.handleReload(ctx)
	default:
		return ipc.CommandResult{Status: "error", Error: fmt.Sprintf("unknown command: %s", cmd.Name)}
	}
}

// --- internal helpers ---

func (d *Daemon) cleanup() {
	d.mu.Lock()
	state := d.state

	if d.cancelFunc != nil {
		d.cancelFunc()
		d.cancelFunc = nil
	}
	d.running = false

	if d.lock != nil {
		d.lock.Release()
		d.lock = nil
	}
	if d.service != nil {
		d.service.Stop()
		d.service = nil
	}
	if state != nil {
		state.PID = 0
		if err := SaveState(state); err != nil {
			logger.Error("Failed to save runtime state", "error", err)
		}
	}
	d.mu.Unlock()

	// stopDNSProxy 涉及 OS 调用（networksetup），在锁外执行避免长时间持锁
	d.stopDNSProxy()
}

// startDNSProxy 启动 DNS 代理并将 macOS 系统 DNS 指向它。
// opts.DNSPort == 0 时默认使用 53 端口（需要 root 权限）。
// 该方法在 d.mu 锁外调用。
func (d *Daemon) startDNSProxy(opts model.RunOptions) {
	listenAddr := opts.ListenAddr
	if listenAddr == "" {
		listenAddr = "127.0.0.1"
	}
	dnsPort := opts.DNSPort
	if dnsPort == 0 {
		dnsPort = 53
	}
	dnsListenAddr := fmt.Sprintf("%s:%d", listenAddr, dnsPort)
	socksAddr := fmt.Sprintf("%s:%d", listenAddr, opts.MixedPort)

	p := sysnet.NewDNSProxy(dnsListenAddr, socksAddr)
	if err := p.Start(); err != nil {
		logger.Error("Failed to start DNS proxy (port 53 requires root)", "error", err)
		return
	}

	d.mu.Lock()
	d.dnsProxy = p
	d.mu.Unlock()

	if err := sysnet.SetSystemDNS(listenAddr); err != nil {
		logger.Error("Failed to configure macOS system DNS", "error", err)
	} else {
		logger.Info("macOS system DNS → sing-helm DNS proxy", "listen", dnsListenAddr, "socks", socksAddr)
	}
}

// stopDNSProxy 停止 DNS 代理并恢复 macOS 系统 DNS 为 DHCP 自动获取。
// 若代理未运行则为空操作。该方法在 d.mu 锁外调用。
func (d *Daemon) stopDNSProxy() {
	d.mu.Lock()
	p := d.dnsProxy
	d.dnsProxy = nil
	d.mu.Unlock()

	if p == nil {
		return
	}
	p.Stop()
	if err := sysnet.RestoreSystemDNS(); err != nil {
		logger.Error("Failed to restore macOS system DNS", "error", err)
	} else {
		logger.Info("macOS system DNS restored to DHCP")
	}
}

func (d *Daemon) newService() ServiceRunner {
	if d.serviceFactory != nil {
		return d.serviceFactory()
	}
	return engine.NewInstance()
}

func (d *Daemon) currentState() (*RuntimeState, error) {
	d.mu.Lock()
	state := d.state
	d.mu.Unlock()
	if state != nil {
		copyState := *state
		return &copyState, nil
	}
	return nil, nil
}

func (d *Daemon) isRunning() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.running
}

func (d *Daemon) setRunning(running bool) {
	d.mu.Lock()
	d.running = running
	d.mu.Unlock()
}

func (d *Daemon) loadState() {
	state, err := LoadState()
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		logger.Error("Failed to load runtime state", "error", err)
		return
	}
	d.mu.Lock()
	d.state = state
	d.state.PID = os.Getpid()
	d.mu.Unlock()
}
