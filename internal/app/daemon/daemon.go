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

// tunDNSPlaceholder is the system DNS server address set while running in TUN
// mode. macOS's auto_route never captures traffic to the physically-connected
// local subnet (e.g. a LAN router doubling as DNS server), so if system DNS
// stays pointed at a local address, queries bypass the TUN device entirely.
// Pointing DNS at a known non-local address forces those queries onto the TUN
// default route, where sing-box's own hijack-dns rule and dns.go policy
// (ipv4_only, geosite routing, etc.) handle them - this does not run any
// forwarder of our own, it only nudges which resolver the OS asks.
const tunDNSPlaceholder = "8.8.8.8"

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
	dnsMode        model.ProxyMode // 当前已生效的系统 DNS 覆盖所对应的代理模式，空值表示未设置
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
	dnsMode := d.dnsMode
	d.dnsMode = ""
	d.mu.Unlock()
	if dnsMode == model.ProxyModeTUN {
		if err := sysnet.RestoreSystemDNS(); err != nil {
			logger.Error("Failed to restore system DNS", "error", err)
		}
	}
	d.mu.Lock()
	if state != nil {
		state.PID = 0
		if err := SaveState(state); err != nil {
			logger.Error("Failed to save runtime state", "error", err)
		}
	}
	d.mu.Unlock()
}

// syncSystemDNS applies or restores the macOS system DNS override to match mode.
// Only TUN mode needs the override (see tunDNSPlaceholder); other modes leave
// system DNS untouched.
func (d *Daemon) syncSystemDNS(mode model.ProxyMode) {
	d.mu.Lock()
	prev := d.dnsMode
	d.mu.Unlock()
	if mode == prev {
		return
	}

	if mode == model.ProxyModeTUN {
		if err := sysnet.SetSystemDNS(tunDNSPlaceholder); err != nil {
			logger.Error("Failed to set system DNS for TUN mode", "error", err)
		}
	} else if prev == model.ProxyModeTUN {
		if err := sysnet.RestoreSystemDNS(); err != nil {
			logger.Error("Failed to restore system DNS", "error", err)
		}
	}

	d.mu.Lock()
	d.dnsMode = mode
	d.mu.Unlock()
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
