package dbcmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Client communicates with the dbcalm-db-cmd Unix socket service
type Client struct {
	socketPath string
	timeout    time.Duration
}

// NewClient creates a new socket client
func NewClient(socketPath string, timeout time.Duration) *Client {
	return &Client{
		socketPath: socketPath,
		timeout:    timeout,
	}
}

// CommandRequest represents a command sent to the socket
type CommandRequest struct {
	Cmd  string                 `json:"cmd"`
	Args map[string]interface{} `json:"args"`
}

// CommandResponse represents a response from the socket
type CommandResponse struct {
	Code    int    `json:"code"`
	Status  string `json:"status"`
	ID      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

// SendCommand sends a command to the Unix socket and waits for response
func (c *Client) SendCommand(ctx context.Context, cmd string, args map[string]interface{}) (*CommandResponse, error) {
	// Connect to the Unix socket
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to socket %s: %w", c.socketPath, err)
	}
	defer conn.Close()

	// Set deadline
	if err := conn.SetDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set deadline: %w", err)
	}

	// Prepare request
	request := CommandRequest{
		Cmd:  cmd,
		Args: args,
	}

	// Send request
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	if _, err := conn.Write(requestBytes); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Read response
	buffer := make([]byte, 4096)
	n, err := conn.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	var response CommandResponse
	if err := json.Unmarshal(buffer[:n], &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}
