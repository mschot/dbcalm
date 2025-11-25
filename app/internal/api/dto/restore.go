package dto

import "time"

// CreateRestoreRequest represents the restore creation request
type CreateRestoreRequest struct {
	BackupID string `json:"id" binding:"required"` // Matches Python field name
	Target   string `json:"target" binding:"required,oneof=database folder"` // "database" or "folder"
}

// RestoreResponse represents a restore
type RestoreResponse struct {
	ID              int64      `json:"id"`
	BackupID        string     `json:"backup_id"`
	BackupTimestamp time.Time  `json:"backup_timestamp"`
	Target          string     `json:"target"`
	TargetPath      string     `json:"target_path"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         *time.Time `json:"end_time,omitempty"`
	ProcessID       int64      `json:"process_id"`
}

// RestoreListResponse represents a list of restores
type RestoreListResponse struct {
	Items      []RestoreResponse `json:"items"`
	Pagination PaginationInfo    `json:"pagination"`
}
