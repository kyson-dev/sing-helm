package ipc

import "context"

// CommandMessage represents a single CLI request dispatched to the daemon.
type CommandMessage struct {
	Name    string                 `json:"name"`
	Payload map[string]any         `json:"payload,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// CommandResult is returned by the daemon to the CLI that issued the CommandMessage.
type CommandResult struct {
	Status string                 `json:"status"` // e.g. "ok", "error"
	Error  string                 `json:"error,omitempty"`
	Data   map[string]any         `json:"data,omitempty"`
	Extra  map[string]interface{} `json:"extra,omitempty"`
}

// CommandHandler processes CommandMessages in the daemon and returns a result.
type CommandHandler interface {
	Handle(ctx context.Context, cmd CommandMessage) CommandResult
}

// HandlerFunc is a helper wrapper that lets a function satisfy CommandHandler.
type HandlerFunc func(ctx context.Context, cmd CommandMessage) CommandResult

// Handle calls the wrapped function.
func (f HandlerFunc) Handle(ctx context.Context, cmd CommandMessage) CommandResult {
	if f == nil {
		return CommandResult{Status: "error", Error: "handler func nil"}
	}
	return f(ctx, cmd)
}
