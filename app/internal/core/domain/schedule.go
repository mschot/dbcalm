package domain

import "time"

type ScheduleFrequency string

const (
	FrequencyDaily    ScheduleFrequency = "daily"
	FrequencyWeekly   ScheduleFrequency = "weekly"
	FrequencyMonthly  ScheduleFrequency = "monthly"
	FrequencyHourly   ScheduleFrequency = "hourly"
	FrequencyInterval ScheduleFrequency = "interval"
)

type IntervalUnit string

const (
	IntervalUnitMinutes IntervalUnit = "minutes"
	IntervalUnitHours   IntervalUnit = "hours"
)

type RetentionUnit string

const (
	RetentionUnitDays   RetentionUnit = "days"
	RetentionUnitWeeks  RetentionUnit = "weeks"
	RetentionUnitMonths RetentionUnit = "months"
)

type Schedule struct {
	ID             int64             `db:"id"`
	BackupType     BackupType        `db:"backup_type"`
	Frequency      ScheduleFrequency `db:"frequency"`
	DayOfWeek      *int              `db:"day_of_week"` // 0-6 (Sunday-Saturday)
	DayOfMonth     *int              `db:"day_of_month"` // 1-31
	Hour           *int              `db:"hour"` // 0-23
	Minute         *int              `db:"minute"` // 0-59
	IntervalValue  *int              `db:"interval_value"`
	IntervalUnit   *IntervalUnit     `db:"interval_unit"`
	RetentionValue *int              `db:"retention_value"`
	RetentionUnit  *RetentionUnit    `db:"retention_unit"`
	Enabled        bool              `db:"enabled"`
	CreatedAt      time.Time         `db:"created_at"`
	UpdatedAt      time.Time         `db:"updated_at"`
}

func NewSchedule(backupType BackupType, frequency ScheduleFrequency, enabled bool) *Schedule {
	now := time.Now()
	return &Schedule{
		BackupType: backupType,
		Frequency:  frequency,
		Enabled:    enabled,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}
