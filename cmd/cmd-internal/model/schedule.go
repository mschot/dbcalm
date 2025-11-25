package model

type Schedule struct {
	ID            int     `json:"id"`
	BackupType    string  `json:"backup_type"`
	Frequency     string  `json:"frequency"`
	Hour          *int    `json:"hour"`
	Minute        *int    `json:"minute"`
	DayOfWeek     *int    `json:"day_of_week"`
	DayOfMonth    *int    `json:"day_of_month"`
	IntervalValue *int    `json:"interval_value"`
	IntervalUnit  *string `json:"interval_unit"`
	Enabled       bool    `json:"enabled"`
}
