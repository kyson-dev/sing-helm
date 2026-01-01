package daemon_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kyson/minibox/internal/core/daemon"
	"github.com/kyson/minibox/internal/env"
	"github.com/kyson/minibox/internal/ipc"
)

type fakeService struct {
	startPath  string
	runPath    string
	reloadPath string
	runStarted chan struct{}
	runStopped chan struct{}
	stopCh     chan struct{}
}

func newFakeService() *fakeService {
	return &fakeService{
		runStarted: make(chan struct{}),
		runStopped: make(chan struct{}),
		stopCh:     make(chan struct{}),
	}
}

// 移除 Run 方法，因为 ServiceRunner 接口不再需要它
// 更新 StartFromFile 以模拟服务运行行为

func (f *fakeService) StartFromFile(ctx context.Context, path string) error {
	f.startPath = path
	// 模拟服务启动
	go func() {
		f.runStarted <- struct{}{}
		select {
		case <-ctx.Done():
		case <-f.stopCh:
		}
		f.runStopped <- struct{}{}
	}()
	return nil
}

func (f *fakeService) ReloadFromFile(ctx context.Context, path string) error {
	f.reloadPath = path
	return nil
}

func (f *fakeService) Stop() {
	select {
	case <-f.stopCh:
	default:
		close(f.stopCh)
	}
}

func TestDaemonHandleCommands(t *testing.T) {
	setupEnv(t)

	// 创建带超时的 context，防止测试挂死
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	d := daemon.NewDaemon()
	fake := newFakeService()
	d.SetServiceFactory(func() daemon.ServiceRunner {
		return fake
	})

	// 在后台启动 Daemon Serve，以便初始化上下文和锁
	go func() {
		// Serve 会阻塞，直到 ctx 取消
		_ = d.Serve(ctx)
	}()

	// 等待 Daemon 启动（Serve 内部会初始化 ctx 和 lock）
	time.Sleep(100 * time.Millisecond)

	resp := d.Handle(ctx, ipc.CommandMessage{Name: "run", Payload: map[string]any{}})
	if resp.Status != "ok" {
		t.Fatalf("expected run ok, got status=%s error=%s", resp.Status, resp.Error)
	}

	waitFor(t, fake.runStarted, "run start")

	statusResp := d.Handle(ctx, ipc.CommandMessage{Name: "status"})
	if statusResp.Status != "ok" {
		t.Fatalf("expected status ok, got status=%s error=%s", statusResp.Status, statusResp.Error)
	}
	if running, _ := statusResp.Data["running"].(bool); !running {
		t.Fatalf("expected running true in status")
	}

	healthResp := d.Handle(ctx, ipc.CommandMessage{Name: "health"})
	if healthResp.Status != "ok" {
		t.Fatalf("expected health ok, got status=%s error=%s", healthResp.Status, healthResp.Error)
	}

	modeResp := d.Handle(ctx, ipc.CommandMessage{Name: "mode", Payload: map[string]any{"mode": "default"}})
	if modeResp.Status != "ok" {
		t.Fatalf("expected mode ok, got status=%s error=%s", modeResp.Status, modeResp.Error)
	}

	routeResp := d.Handle(ctx, ipc.CommandMessage{Name: "route", Payload: map[string]any{"route": "global"}})
	if routeResp.Status != "ok" {
		t.Fatalf("expected route ok, got status=%s error=%s", routeResp.Status, routeResp.Error)
	}

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/proxies":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"proxies":{"Proxy":{"type":"Selector","all":["A","B"],"now":"A"}}}`))
		case r.Method == http.MethodPut && strings.HasPrefix(r.URL.Path, "/proxies/"):
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(apiServer.Close)

	apiHost := strings.TrimPrefix(apiServer.URL, "http://")
	listResp := d.Handle(ctx, ipc.CommandMessage{Name: "node.list", Payload: map[string]any{"api": apiHost}})
	if listResp.Status != "ok" {
		t.Fatalf("expected node.list ok, got status=%s error=%s", listResp.Status, listResp.Error)
	}
	if listResp.Data["proxies"] == nil {
		t.Fatalf("expected proxies in response")
	}

	useResp := d.Handle(ctx, ipc.CommandMessage{Name: "node.use", Payload: map[string]any{"api": apiHost, "group": "Proxy", "node": "A"}})
	if useResp.Status != "ok" {
		t.Fatalf("expected node.use ok, got status=%s error=%s", useResp.Status, useResp.Error)
	}

	logResp := d.Handle(ctx, ipc.CommandMessage{Name: "log"})
	if logResp.Status != "ok" {
		t.Fatalf("expected log ok, got status=%s error=%s", logResp.Status, logResp.Error)
	}
	if path, _ := logResp.Data["path"].(string); path == "" {
		t.Fatalf("expected log path in response")
	}

	reloadResp := d.Handle(ctx, ipc.CommandMessage{Name: "reload"})
	if reloadResp.Status != "ok" {
		t.Fatalf("expected reload ok, got status=%s error=%s", reloadResp.Status, reloadResp.Error)
	}
	if fake.reloadPath == "" {
		t.Fatalf("expected reload path to be set")
	}

	stopResp := d.Handle(ctx, ipc.CommandMessage{Name: "stop"})
	if stopResp.Status != "ok" {
		t.Fatalf("expected stop ok, got status=%s error=%s", stopResp.Status, stopResp.Error)
	}
	waitFor(t, fake.runStopped, "run stop")
}

func setupEnv(t *testing.T) {
	t.Helper()
	env.ResetForTest()
	dir := t.TempDir()
	if err := env.Init(dir); err != nil {
		t.Fatalf("env.Init failed: %v", err)
	}
	if err := os.WriteFile(env.Get().ConfigFile, []byte(`{}`), 0644); err != nil {
		t.Fatalf("write profile.json: %v", err)
	}
}

func waitFor(t *testing.T, ch <-chan struct{}, label string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(2 * time.Second):
		t.Fatalf("timed out waiting for %s", label)
	}
}
