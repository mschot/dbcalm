package domain

import (
	"time"

	"github.com/google/uuid"
)

type ProcessStatus string

const (
	ProcessStatusRunning ProcessStatus = "running"
	ProcessStatusSuccess ProcessStatus = "success"
	ProcessStatusFailed  ProcessStatus = "failed"
)

type ProcessType string

const (
	ProcessTypeBackup             ProcessType = "backup"
	ProcessTypeRestore            ProcessType = "restore"
	ProcessTypeCleanupBackups     ProcessType = "cleanup_backups"
	ProcessTypeUpdateCronSchedules ProcessType = "update_cron_schedules"
)

type Process struct {
	ID        int64                  `db:"id"`
	CommandID string                 `db:"command_id"` // UUID for API polling
	Command   string                 `db:"command"`
	PID       *int                   `db:"pid"`
	Status    ProcessStatus          `db:"status"`
	Output    *string                `db:"output"`
	Error     *string                `db:"error"`
	ReturnCode *int                  `db:"return_code"`
	StartTime time.Time              `db:"start_time"`
	EndTime   *time.Time             `db:"end_time"`
	Type      ProcessType            `db:"type"`
	Args      map[string]interface{} `db:"args"` // JSON-serializable args
}

func NewProcess(command string, processType ProcessType, args map[string]interface{}) *Process {
	// Initialize PID to 0 (placeholder) since database has NOT NULL constraint
	// PID will be set later when the actual process starts
	pid := 0
	return &Process{
		CommandID: uuid.New().String(),
		Command:   command,
		PID:       &pid,
		Status:    ProcessStatusRunning,
		StartTime: time.Now(),
		Type:      processType,
		Args:      args,
	}
}

func (p *Process) SetPID(pid int) {
	p.PID = &pid
}

func (p *Process) Complete(returnCode int, output, errorOutput string) {
	now := time.Now()
	p.EndTime = &now
	p.ReturnCode = &returnCode

	if output != "" {
		p.Output = &output
	}
	if errorOutput != "" {
		p.Error = &errorOutput
	}

	if returnCode == 0 {
		p.Status = ProcessStatusSuccess
	} else {
		p.Status = ProcessStatusFailed
	}
}

func (p *Process) Fail(errorOutput string) {
	now := time.Now()
	p.EndTime = &now
	p.Status = ProcessStatusFailed
	if errorOutput != "" {
		p.Error = &errorOutput
	}
}

func (p *Process) IsComplete() bool {
	return p.Status == ProcessStatusSuccess || p.Status == ProcessStatusFailed
}
