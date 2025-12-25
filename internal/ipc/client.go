package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// UnixClient represents an IPC client that connects to the daemon via Unix socket
type UnixClient struct {
	socketPath string
	// deadline for connection
	deadline time.Duration
}

// NewClient creates a new IPC client
func NewClient(socketPath string) Client {
	return &UnixClient{
		socketPath: socketPath,
		deadline:   5 * time.Second,
	}
}

// Call sends a request to the daemon and waits for response
func (c *UnixClient) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	// Create request
	req, err := NewRequest(method, params)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// Connect to daemon
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon: %w (is daemon running?)", err)
	}
	defer conn.Close()

	// Set deadline
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	} else if c.deadline > 0 {
		conn.SetDeadline(time.Now().Add(c.deadline))
	}

	// Send request
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(req); err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	var resp Response
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&resp); err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// Check for error
	if resp.Error != nil {
		return fmt.Errorf("daemon error: %s", resp.Error.Message)
	}

	// Parse result
	if result != nil && resp.Result != nil {
		if err := json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("failed to parse result: %w", err)
		}
	}

	return nil
}
