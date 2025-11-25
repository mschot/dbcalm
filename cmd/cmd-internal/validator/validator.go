package validator

import (
	"fmt"
)

const (
	StatusOK      = 200
	StatusInvalid = 400
)

// Validation constants
const (
	MaxHour         = 23
	MaxMinute       = 59
	MaxDayOfWeek    = 6
	MaxDayOfMonth   = 28
	MinDayOfMonth   = 1
	MinIntervalValue = 1
)

type ValidationResult struct {
	Code    int
	Message string
}

type Validator struct {
	commands         map[string]map[string]string
	validFrequencies []string
	validBackupTypes []string
}

func NewValidator() *Validator {
	return &Validator{
		commands: map[string]map[string]string{
			"update_cron_schedules": {
				"schedules": "required",
			},
			"delete_directory": {
				"path": "required",
			},
			"cleanup_backups": {
				"backup_ids": "required",
				"folders":    "required",
			},
		},
		validFrequencies: []string{"daily", "weekly", "monthly", "hourly", "interval"},
		validBackupTypes: []string{"full", "incremental"},
	}
}

func (v *Validator) Validate(cmd string, args map[string]interface{}) ValidationResult {
	// Check if command is valid
	if _, exists := v.commands[cmd]; !exists {
		return ValidationResult{
			Code:    StatusInvalid,
			Message: "Invalid command",
		}
	}

	// Check required arguments
	for arg, requirement := range v.commands[cmd] {
		if requirement == "required" {
			if _, exists := args[arg]; !exists {
				return ValidationResult{
					Code:    StatusInvalid,
					Message: fmt.Sprintf("Missing required argument %s", arg),
				}
			}
		}
	}

	// Special validation for update_cron_schedules
	if cmd == "update_cron_schedules" {
		schedulesRaw, exists := args["schedules"]
		if !exists {
			return ValidationResult{
				Code:    StatusInvalid,
				Message: "schedules argument is required",
			}
		}

		schedules, ok := schedulesRaw.([]interface{})
		if !ok {
			return ValidationResult{
				Code:    StatusInvalid,
				Message: "schedules must be a list",
			}
		}

		// Validate each schedule
		for idx, scheduleRaw := range schedules {
			schedule, ok := scheduleRaw.(map[string]interface{})
			if !ok {
				return ValidationResult{
					Code:    StatusInvalid,
					Message: fmt.Sprintf("Schedule at index %d must be a dict", idx),
				}
			}

			result := v.validateSchedule(schedule)
			if result.Code != StatusOK {
				return ValidationResult{
					Code:    result.Code,
					Message: fmt.Sprintf("Schedule at index %d: %s", idx, result.Message),
				}
			}
		}
	}

	return ValidationResult{
		Code:    StatusOK,
		Message: "",
	}
}

func (v *Validator) validateSchedule(schedule map[string]interface{}) ValidationResult {
	validators := []func(map[string]interface{}) ValidationResult{
		v.validateRequiredFields,
		v.validateBackupType,
		v.validateFrequency,
		v.validateTimeFields,
		v.validateDayFields,
		v.validateIntervalFields,
	}

	for _, validator := range validators {
		result := validator(schedule)
		if result.Code != StatusOK {
			return result
		}
	}

	return ValidationResult{Code: StatusOK, Message: ""}
}

func (v *Validator) validateRequiredFields(schedule map[string]interface{}) ValidationResult {
	requiredFields := []string{"id", "backup_type", "frequency", "enabled"}
	for _, field := range requiredFields {
		if _, exists := schedule[field]; !exists {
			return ValidationResult{
				Code:    StatusInvalid,
				Message: fmt.Sprintf("Schedule missing required field: %s", field),
			}
		}
	}
	return ValidationResult{Code: StatusOK, Message: ""}
}

func (v *Validator) validateBackupType(schedule map[string]interface{}) ValidationResult {
	backupType, ok := schedule["backup_type"].(string)
	if !ok {
		return ValidationResult{
			Code:    StatusInvalid,
			Message: "backup_type must be a string",
		}
	}

	valid := false
	for _, validType := range v.validBackupTypes {
		if backupType == validType {
			valid = true
			break
		}
	}

	if !valid {
		return ValidationResult{
			Code:    StatusInvalid,
			Message: fmt.Sprintf("Invalid backup_type: %s", backupType),
		}
	}

	return ValidationResult{Code: StatusOK, Message: ""}
}

func (v *Validator) validateFrequency(schedule map[string]interface{}) ValidationResult {
	frequency, ok := schedule["frequency"].(string)
	if !ok {
		return ValidationResult{
			Code:    StatusInvalid,
			Message: "frequency must be a string",
		}
	}

	valid := false
	for _, validFreq := range v.validFrequencies {
		if frequency == validFreq {
			valid = true
			break
		}
	}

	if !valid {
		return ValidationResult{
			Code:    StatusInvalid,
			Message: fmt.Sprintf("Invalid frequency: %s", frequency),
		}
	}

	return ValidationResult{Code: StatusOK, Message: ""}
}

func (v *Validator) validateTimeFields(schedule map[string]interface{}) ValidationResult {
	frequency, _ := schedule["frequency"].(string)

	// Validate hour (0-23) - required for daily, weekly, monthly
	if frequency == "daily" || frequency == "weekly" || frequency == "monthly" {
		hourRaw, exists := schedule["hour"]
		if !exists || hourRaw == nil {
			return ValidationResult{
				Code:    StatusInvalid,
				Message: "hour is required for daily, weekly, and monthly schedules",
			}
		}

		hour, ok := v.toInt(hourRaw)
		if !ok {
			return ValidationResult{
				Code:    StatusInvalid,
				Message: fmt.Sprintf("Invalid hour: %v", hourRaw),
			}
		}

		if hour < 0 || hour > MaxHour {
			return ValidationResult{
				Code:    StatusInvalid,
				Message: fmt.Sprintf("Invalid hour: %d. Must be 0-%d", hour, MaxHour),
			}
		}
	}

	// Validate minute (0-59) - required for daily, weekly, monthly, and hourly
	if frequency == "daily" || frequency == "weekly" || frequency == "monthly" || frequency == "hourly" {
		minuteRaw, exists := schedule["minute"]
		if !exists || minuteRaw == nil {
			return ValidationResult{
				Code:    StatusInvalid,
				Message: "minute is required for daily, weekly, monthly, and hourly schedules",
			}
		}

		minute, ok := v.toInt(minuteRaw)
		if !ok {
			return ValidationResult{
				Code:    StatusInvalid,
				Message: fmt.Sprintf("Invalid minute: %v", minuteRaw),
			}
		}

		if minute < 0 || minute > MaxMinute {
			return ValidationResult{
				Code:    StatusInvalid,
				Message: fmt.Sprintf("Invalid minute: %d. Must be 0-%d", minute, MaxMinute),
			}
		}
	}

	return ValidationResult{Code: StatusOK, Message: ""}
}

func (v *Validator) validateDayFields(schedule map[string]interface{}) ValidationResult {
	frequency, _ := schedule["frequency"].(string)

	// Validate day_of_week for weekly schedules
	if frequency == "weekly" {
		if dayRaw, exists := schedule["day_of_week"]; exists && dayRaw != nil {
			day, ok := v.toInt(dayRaw)
			if !ok {
				return ValidationResult{
					Code:    StatusInvalid,
					Message: fmt.Sprintf("Invalid day_of_week: %v", dayRaw),
				}
			}

			if day < 0 || day > MaxDayOfWeek {
				return ValidationResult{
					Code:    StatusInvalid,
					Message: fmt.Sprintf("Invalid day_of_week: %d. Must be 0-%d", day, MaxDayOfWeek),
				}
			}
		}
	}

	// Validate day_of_month for monthly schedules
	if frequency == "monthly" {
		if dayRaw, exists := schedule["day_of_month"]; exists && dayRaw != nil {
			day, ok := v.toInt(dayRaw)
			if !ok {
				return ValidationResult{
					Code:    StatusInvalid,
					Message: fmt.Sprintf("Invalid day_of_month: %v", dayRaw),
				}
			}

			if day < MinDayOfMonth || day > MaxDayOfMonth {
				return ValidationResult{
					Code:    StatusInvalid,
					Message: fmt.Sprintf("Invalid day_of_month: %d. Must be %d-%d", day, MinDayOfMonth, MaxDayOfMonth),
				}
			}
		}
	}

	return ValidationResult{Code: StatusOK, Message: ""}
}

func (v *Validator) validateIntervalFields(schedule map[string]interface{}) ValidationResult {
	frequency, _ := schedule["frequency"].(string)

	if frequency != "interval" {
		return ValidationResult{Code: StatusOK, Message: ""}
	}

	// interval_value is required for interval schedules
	intervalValueRaw, exists := schedule["interval_value"]
	if !exists || intervalValueRaw == nil {
		return ValidationResult{
			Code:    StatusInvalid,
			Message: "interval_value is required for interval schedules",
		}
	}

	intervalValue, ok := v.toInt(intervalValueRaw)
	if !ok {
		return ValidationResult{
			Code:    StatusInvalid,
			Message: fmt.Sprintf("Invalid interval_value: %v", intervalValueRaw),
		}
	}

	if intervalValue < MinIntervalValue {
		return ValidationResult{
			Code:    StatusInvalid,
			Message: fmt.Sprintf("Invalid interval_value: %d. Must be >= %d", intervalValue, MinIntervalValue),
		}
	}

	// interval_unit is required for interval schedules
	intervalUnitRaw, exists := schedule["interval_unit"]
	if !exists || intervalUnitRaw == nil {
		return ValidationResult{
			Code:    StatusInvalid,
			Message: "interval_unit is required for interval schedules",
		}
	}

	intervalUnit, ok := intervalUnitRaw.(string)
	if !ok {
		return ValidationResult{
			Code:    StatusInvalid,
			Message: fmt.Sprintf("Invalid interval_unit: %v", intervalUnitRaw),
		}
	}

	if intervalUnit != "minutes" && intervalUnit != "hours" {
		return ValidationResult{
			Code:    StatusInvalid,
			Message: fmt.Sprintf("Invalid interval_unit: %s. Must be 'minutes' or 'hours'", intervalUnit),
		}
	}

	return ValidationResult{Code: StatusOK, Message: ""}
}

// toInt converts an interface{} to int, handling both int and float64 (from JSON)
func (v *Validator) toInt(val interface{}) (int, bool) {
	switch v := val.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case int64:
		return int(v), true
	default:
		return 0, false
	}
}
