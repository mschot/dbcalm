package process

import (
	"time"
)

const (
	StatusRunning = "running"
	StatusSuccess = "success"
	StatusFailed  = "failed"
)

type Process struct {
	ID         *int                   `db:"id" json:"id,omitempty"`
	Command    string                 `db:"command" json:"command"`
	CommandID  string                 `db:"command_id" json:"command_id"`
	PID        int                    `db:"pid" json:"pid"`
	Status     string                 `db:"status" json:"status"`
	Output     *string                `db:"output" json:"output,omitempty"`
	Error      *string                `db:"error" json:"error,omitempty"`
	ReturnCode *int                   `db:"return_code" json:"return_code,omitempty"`
	StartTime  time.Time              `db:"start_time" json:"start_time"`
	EndTime    *time.Time             `db:"end_time" json:"end_time,omitempty"`
	Type       string                 `db:"type" json:"type"`
	Args       map[string]interface{} `json:"args"`
	ArgsJSON   string                 `db:"args"` // For database storage
}
