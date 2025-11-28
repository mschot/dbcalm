package dto

import "time"

// ProcessResponse represents a process
type ProcessResponse struct {
	ID         int64                  `json:"id"`
	CommandID  string                 `json:"command_id"`
	Command    string                 `json:"command"`
	PID        *int                   `json:"pid,omitempty"`
	Status     string                 `json:"status"`
	Output     *string                `json:"output,omitempty"`
	Error      *string                `json:"error,omitempty"`
	ReturnCode *int                   `json:"return_code,omitempty"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    *time.Time             `json:"end_time,omitempty"`
	Type       string                 `json:"type"`
	Args       map[string]interface{} `json:"args,omitempty"`
	Link       *string                `json:"link,omitempty"`       // Link to status endpoint
	ResourceID *string                `json:"resource_id,omitempty"` // Extracted from args["id"]
}

// ProcessListResponse represents a list of processes
type ProcessListResponse struct {
	Items      []ProcessResponse `json:"items"`
	Pagination PaginationInfo    `json:"pagination"`
}

// AsyncResponse represents an async operation response (202 Accepted)
// Matches Python StatusResponse format
type AsyncResponse struct {
	Status     string  `json:"status"`
	Link       *string `json:"link,omitempty"`
	PID        *string `json:"pid,omitempty"`
	ResourceID *string `json:"resource_id,omitempty"`
}
