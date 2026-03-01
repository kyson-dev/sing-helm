package ipc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestUnixSenderAndServer(t *testing.T) {
	dir, err := os.MkdirTemp(".", "ipc-test-")
	if err != nil {
		t.Fatalf("create socket dir: %v", err)
	}
	defer os.RemoveAll(dir)
	socket := filepath.Join(dir, "test.sock")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	handler := HandlerFunc(func(ctx context.Context, cmd CommandMessage) CommandResult {
		return CommandResult{
			Status: "ok",
			Data: map[string]any{
				"received": cmd.Name,
			},
		}
	})

	ready := make(chan struct{}, 1)
	errCh := make(chan error, 1)
	go func() {
		errCh <- Serve(ctx, socket, handler, &ServerOptions{Ready: ready})
	}()

	select {
	case <-ready:
	case err := <-errCh:
		if err != nil {
			if isPermissionDenied(err) {
				t.Skipf("unix sockets unavailable: %v", err)
				return
			}
			t.Fatalf("server failed: %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Fatal("server did not become ready")
	}

	sender := NewUnixSender(socket)

	resp, err := sender.Send(context.Background(), CommandMessage{Name: "run"})
	if err != nil {
		t.Fatalf("send failed: %v", err)
	}
	if resp.Data["received"] != "run" {
		t.Fatalf("unexpected response data: %v", resp.Data)
	}
}

func TestFakeSender(t *testing.T) {
	resp := CommandResult{Status: "ok", Data: map[string]any{"foo": "bar"}}
	s := &FakeSender{Response: resp}
	got, err := s.Send(context.Background(), CommandMessage{Name: "check"})
	if err != nil {
		t.Fatalf("fake send error: %v", err)
	}
	if got.Data["foo"] != "bar" {
		t.Fatalf("unexpected fake response %v", got.Data)
	}
}

func isPermissionDenied(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, os.ErrPermission) {
		return true
	}
	if strings.Contains(err.Error(), "operation not permitted") {
		return true
	}
	return false
}
