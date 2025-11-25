package dto

// CleanupRequest represents the cleanup request
type CleanupRequest struct {
	ScheduleID *int64 `json:"schedule_id,omitempty"` // Optional: cleanup specific schedule
}
