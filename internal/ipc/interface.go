package ipc

import "context"

// Client defines the interface for IPC communication
type Client interface {
	Call(ctx context.Context, method string, params interface{}, result interface{}) error
}
