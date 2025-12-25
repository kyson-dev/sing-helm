package ipc

import (
	"encoding/json"
	"fmt"
)

// Request represents an IPC request from client to daemon
type Request struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// Response represents an IPC response from daemon to client
type Response struct {
	ID     string          `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *Error          `json:"error,omitempty"`
}

// Error represents an error in the response
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Error codes
const (
	ErrCodeInternal          = -32603
	ErrCodeInvalidParams     = -32602
	ErrCodeMethodNotFound    = -32601
	ErrCodeDaemonNotRunning  = -32000
	ErrCodeSingBoxNotRunning = -32001
)

const (
	// Daemon lifecycle
	MethodReload = "reload" // 重新加载配置文件
	MethodStop   = "stop"   // 停止 daemon
)

// NewRequest creates a new request with auto-generated ID
func NewRequest(method string, params interface{}) (*Request, error) {
	req := &Request{
		ID:     generateID(),
		Method: method,
	}

	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %w", err)
		}
		req.Params = data
	}

	return req, nil
}

// NewResponse creates a successful response
func NewResponse(id string, result interface{}) (*Response, error) {
	resp := &Response{ID: id}

	if result != nil {
		data, err := json.Marshal(result)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal result: %w", err)
		}
		resp.Result = data
	}

	return resp, nil
}

// NewErrorResponse creates an error response
func NewErrorResponse(id string, code int, message string) *Response {
	return &Response{
		ID: id,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	}
}

// ParseResult parses the result from a response
func (r *Response) ParseResult(v interface{}) error {
	if r.Error != nil {
		return fmt.Errorf("error %d: %s", r.Error.Code, r.Error.Message)
	}
	if r.Result == nil {
		return nil
	}
	return json.Unmarshal(r.Result, v)
}

// generateID generates a simple request ID
var requestCounter int64

func generateID() string {
	requestCounter++
	return fmt.Sprintf("req-%d", requestCounter)
}
