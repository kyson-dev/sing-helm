package daemon

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/kyson-dev/sing-helm/internal/engine"
	"github.com/kyson-dev/sing-helm/internal/ipc"
	"github.com/kyson-dev/sing-helm/internal/logger"
	"github.com/kyson-dev/sing-helm/internal/model"
	"github.com/kyson-dev/sing-helm/internal/platform"
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
	lock           *platform.DaemonLock
	running        bool
	reloading      bool
	state          *model.RuntimeState
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
	if err := platform.EnsureRuntimeDirs(platform.Get().RuntimeDir, platform.Get().LogFile); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("runtime directory not writable (try sudo): %w", err)
		}
		return fmt.Errorf("runtime directory not writable: %w", err)
	}

	lock, err := platform.AcquireLock(platform.Get().RuntimeDir)
	if err != nil {
		return fmt.Errorf("another instance is already running: %w", err)
	}
	d.lock = lock
	d.loadState()
	_ = platform.SaveRuntimeMeta(platform.Get().RuntimeDir, platform.RuntimeMeta{
		ConfigHome: platform.Get().HomeDir,
	})

	ctx, cancel := context.WithCancel(ctx)
	d.cancelFunc = cancel
	defer func() {
		logger.Info("Daemon shutting down")
		cancel()
		d.cleanup()
	}()

	logger.Info("Daemon started, listening for IPC commands")

	if err := ipc.Serve(ctx, platform.Get().SocketFile, d, &ipc.ServerOptions{}); err != nil {
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
	defer d.mu.Unlock()

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
		if err := model.SaveState(state); err != nil {
			logger.Error("Failed to save runtime state", "error", err)
		}
	}
}

func (d *Daemon) newService() ServiceRunner {
	if d.serviceFactory != nil {
		return d.serviceFactory()
	}
	return engine.NewInstance()
}

func (d *Daemon) currentState() (*model.RuntimeState, error) {
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
	state, err := model.LoadState()
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

func asInt(val any) (int, bool) {
	switch v := val.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	}
	return 0, false
}
