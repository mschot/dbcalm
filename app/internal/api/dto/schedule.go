package dto

import "time"

// CreateScheduleRequest represents the schedule creation request
type CreateScheduleRequest struct {
	BackupType     string  `json:"backup_type" binding:"required,oneof=full incremental"`
	Frequency      string  `json:"frequency" binding:"required,oneof=daily weekly monthly hourly interval"`
	DayOfWeek      *int    `json:"day_of_week,omitempty"`      // 0-6 (Sunday-Saturday)
	DayOfMonth     *int    `json:"day_of_month,omitempty"`     // 1-31
	Hour           *int    `json:"hour,omitempty"`             // 0-23
	Minute         *int    `json:"minute,omitempty"`           // 0-59
	IntervalValue  *int    `json:"interval_value,omitempty"`   // For interval frequency
	IntervalUnit   *string `json:"interval_unit,omitempty"`    // "minutes" or "hours"
	RetentionValue *int    `json:"retention_value,omitempty"`  // Retention period value
	RetentionUnit  *string `json:"retention_unit,omitempty"`   // "days", "weeks", or "months"
	Enabled        bool    `json:"enabled"`
}

// UpdateScheduleRequest represents the schedule update request
type UpdateScheduleRequest struct {
	BackupType     *string `json:"backup_type,omitempty"`
	Frequency      *string `json:"frequency,omitempty"`
	DayOfWeek      *int    `json:"day_of_week,omitempty"`
	DayOfMonth     *int    `json:"day_of_month,omitempty"`
	Hour           *int    `json:"hour,omitempty"`
	Minute         *int    `json:"minute,omitempty"`
	IntervalValue  *int    `json:"interval_value,omitempty"`
	IntervalUnit   *string `json:"interval_unit,omitempty"`
	RetentionValue *int    `json:"retention_value,omitempty"`
	RetentionUnit  *string `json:"retention_unit,omitempty"`
	Enabled        *bool   `json:"enabled,omitempty"`
}

// ScheduleResponse represents a schedule
type ScheduleResponse struct {
	ID             int64      `json:"id"`
	BackupType     string     `json:"backup_type"`
	Frequency      string     `json:"frequency"`
	DayOfWeek      *int       `json:"day_of_week,omitempty"`
	DayOfMonth     *int       `json:"day_of_month,omitempty"`
	Hour           *int       `json:"hour,omitempty"`
	Minute         *int       `json:"minute,omitempty"`
	IntervalValue  *int       `json:"interval_value,omitempty"`
	IntervalUnit   *string    `json:"interval_unit,omitempty"`
	RetentionValue *int       `json:"retention_value,omitempty"`
	RetentionUnit  *string    `json:"retention_unit,omitempty"`
	Enabled        bool       `json:"enabled"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// ScheduleListResponse represents a list of schedules
type ScheduleListResponse struct {
	Items      []ScheduleResponse `json:"items"`
	Pagination PaginationInfo     `json:"pagination"`
}
