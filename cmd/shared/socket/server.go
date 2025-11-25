package socket

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

const (
	readChunkSize  = 16
	readTimeout    = 200 * time.Millisecond
	maxMessageSize = 1024 * 1024 // 1MB
)

type Server struct {
	socketPath string
	processor  RequestProcessor
}

func NewServer(socketPath string, processor RequestProcessor) *Server {
	return &Server{
		socketPath: socketPath,
		processor:  processor,
	}
}

func (s *Server) Start() error {
	// Extract socket directory from path
	sockDir := socketDir(s.socketPath)

	// Remove existing socket if it exists
	if err := os.RemoveAll(s.socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create socket directory if it doesn't exist
	if err := os.MkdirAll(sockDir, 0755); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Create Unix domain socket
	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to create socket: %w", err)
	}
	defer listener.Close()

	// Set socket permissions to match parent directory
	if err := os.Chmod(s.socketPath, 0666); err != nil {
		log.Printf("Warning: failed to set socket permissions: %v", err)
	}

	log.Printf("Socket server listening on %s", s.socketPath)

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		// Handle connection in goroutine
		go s.handleConnection(conn)
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Read data in chunks with timeout
	var data []byte
	reader := bufio.NewReader(conn)

	for {
		// Set read deadline
		conn.SetReadDeadline(time.Now().Add(readTimeout))

		chunk := make([]byte, readChunkSize)
		n, err := reader.Read(chunk)
		if n > 0 {
			data = append(data, chunk[:n]...)
		}

		// Check for size limit
		if len(data) > maxMessageSize {
			s.sendResponse(conn, CommandResponse{
				Code:    400,
				Status:  "Bad Request",
				Message: "Message too large",
			})
			return
		}

		if err != nil {
			// Timeout or EOF means we've received all data
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				break
			}
			if err.Error() == "EOF" {
				break
			}
			log.Printf("Error reading from connection: %v", err)
			return
		}
	}

	if len(data) == 0 {
		return
	}

	// Process the request using the injected processor
	response := s.processor.ProcessRequest(data)

	// Send response
	s.sendResponse(conn, response)
}

func (s *Server) sendResponse(conn net.Conn, response CommandResponse) {
	data, err := json.Marshal(response)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		log.Printf("Failed to send response: %v", err)
	}
}

func GetStatusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 202:
		return "Accepted"
	case 400:
		return "Bad Request"
	case 404:
		return "Not Found"
	case 409:
		return "Conflict"
	case 500:
		return "Internal Server Error"
	case 503:
		return "Service Unavailable"
	default:
		return "Unknown"
	}
}

// socketDir extracts the directory from a socket path
func socketDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' {
			return path[:i]
		}
	}
	return "."
}
