package dto

import "time"

// CreateBackupRequest represents the backup creation request
type CreateBackupRequest struct {
	Type         string  `json:"type" binding:"required,oneof=full incremental"` // "full" or "incremental"
	BackupID     *string `json:"backup_id"`                                       // Optional custom ID
	FromBackupID *string `json:"from_backup_id"`                                  // For incremental backups
	ScheduleID   *int64  `json:"schedule_id"`                                     // For scheduled backups
}

// BackupResponse represents a backup
type BackupResponse struct {
	ID             string     `json:"id"`
	Type           string     `json:"-"` // Not sent in JSON, derived field
	FromBackupID   *string    `json:"from_backup_id,omitempty"`
	ScheduleID     *int64     `json:"schedule_id,omitempty"`
	StartTime      time.Time  `json:"start_time"`
	EndTime        *time.Time `json:"end_time,omitempty"`
	ProcessID      int64      `json:"-"` // Not sent in JSON
	Size           *int64     `json:"-"` // Not sent in JSON
	RetentionValue *int       `json:"retention_value,omitempty"`
	RetentionUnit  *string    `json:"retention_unit,omitempty"`
}

// BackupListResponse represents a list of backups
type BackupListResponse struct {
	Items      []BackupResponse `json:"items"`
	Pagination PaginationInfo   `json:"pagination"`
}
