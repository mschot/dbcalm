package socket

import (
	"encoding/json"
	"fmt"

	"github.com/martijn/dbcalm-cmd/cmd-internal/adapter"
	"github.com/martijn/dbcalm-cmd/cmd-internal/config"
	"github.com/martijn/dbcalm-cmd/cmd-internal/handler"
	"github.com/martijn/dbcalm-cmd/cmd-internal/model"
	"github.com/martijn/dbcalm-cmd/cmd-internal/validator"
	sharedProcess "github.com/martijn/dbcalm/shared/process"
	sharedSocket "github.com/martijn/dbcalm/shared/socket"
)

type CmdCommandProcessor struct {
	config       *config.Config
	adapter      adapter.Adapter
	validator    *validator.Validator
	queueHandler *handler.QueueHandler
}

func NewCmdCommandProcessor(cfg *config.Config, adptr adapter.Adapter, valid *validator.Validator, qHandler *handler.QueueHandler) *CmdCommandProcessor {
	return &CmdCommandProcessor{
		config:       cfg,
		adapter:      adptr,
		validator:    valid,
		queueHandler: qHandler,
	}
}

func (p *CmdCommandProcessor) ProcessRequest(data []byte) sharedSocket.CommandResponse {
	// Parse JSON request
	var req sharedSocket.CommandRequest
	if err := json.Unmarshal(data, &req); err != nil {
		return sharedSocket.CommandResponse{
			Code:    400,
			Status:  "Bad Request",
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		}
	}

	// Validate request
	validationResult := p.validator.Validate(req.Cmd, req.Args)
	if validationResult.Code != validator.StatusOK {
		return sharedSocket.CommandResponse{
			Code:    validationResult.Code,
			Status:  sharedSocket.GetStatusText(validationResult.Code),
			Message: validationResult.Message,
		}
	}

	// Execute command
	var proc *sharedProcess.Process
	var procChan chan *sharedProcess.Process
	var err error

	switch req.Cmd {
	case "update_cron_schedules":
		// Convert schedules from interface{} to []model.Schedule
		schedulesRaw := req.Args["schedules"].([]interface{})
		var schedules []model.Schedule
		for _, scheduleRaw := range schedulesRaw {
			scheduleMap := scheduleRaw.(map[string]interface{})
			schedule := p.mapToSchedule(scheduleMap)
			schedules = append(schedules, schedule)
		}
		proc, procChan, err = p.adapter.UpdateCronSchedules(schedules)

	case "delete_directory":
		path := req.Args["path"].(string)
		proc, procChan, err = p.adapter.DeleteDirectory(path)

	case "cleanup_backups":
		// Convert backup_ids to []string
		var backupIDs []string
		if idsRaw, ok := req.Args["backup_ids"]; ok {
			switch v := idsRaw.(type) {
			case []interface{}:
				for _, item := range v {
					if str, ok := item.(string); ok {
						backupIDs = append(backupIDs, str)
					}
				}
			case []string:
				backupIDs = v
			}
		}

		// Convert folders to []string
		var folders []string
		if foldersRaw, ok := req.Args["folders"]; ok {
			switch v := foldersRaw.(type) {
			case []interface{}:
				for _, item := range v {
					if str, ok := item.(string); ok {
						folders = append(folders, str)
					}
				}
			case []string:
				folders = v
			}
		}

		proc, procChan, err = p.adapter.CleanupBackups(backupIDs, folders)

	default:
		return sharedSocket.CommandResponse{
			Code:    400,
			Status:  "Bad Request",
			Message: fmt.Sprintf("Unknown command: %s", req.Cmd),
		}
	}

	if err != nil {
		return sharedSocket.CommandResponse{
			Code:    500,
			Status:  "Internal Server Error",
			Message: fmt.Sprintf("Failed to execute command: %v", err),
		}
	}

	// Start queue handler for this process
	p.queueHandler.Handle(procChan)

	return sharedSocket.CommandResponse{
		Code:   202,
		Status: "Accepted",
		ID:     proc.CommandID,
	}
}

func (p *CmdCommandProcessor) mapToSchedule(m map[string]interface{}) model.Schedule {
	schedule := model.Schedule{
		Enabled: m["enabled"].(bool),
	}

	// ID
	if id, ok := m["id"]; ok {
		switch v := id.(type) {
		case int:
			schedule.ID = v
		case float64:
			schedule.ID = int(v)
		}
	}

	// BackupType
	if bt, ok := m["backup_type"].(string); ok {
		schedule.BackupType = bt
	}

	// Frequency
	if freq, ok := m["frequency"].(string); ok {
		schedule.Frequency = freq
	}

	// Hour
	if hour, ok := m["hour"]; ok && hour != nil {
		switch v := hour.(type) {
		case int:
			schedule.Hour = &v
		case float64:
			val := int(v)
			schedule.Hour = &val
		}
	}

	// Minute
	if minute, ok := m["minute"]; ok && minute != nil {
		switch v := minute.(type) {
		case int:
			schedule.Minute = &v
		case float64:
			val := int(v)
			schedule.Minute = &val
		}
	}

	// DayOfWeek
	if dow, ok := m["day_of_week"]; ok && dow != nil {
		switch v := dow.(type) {
		case int:
			schedule.DayOfWeek = &v
		case float64:
			val := int(v)
			schedule.DayOfWeek = &val
		}
	}

	// DayOfMonth
	if dom, ok := m["day_of_month"]; ok && dom != nil {
		switch v := dom.(type) {
		case int:
			schedule.DayOfMonth = &v
		case float64:
			val := int(v)
			schedule.DayOfMonth = &val
		}
	}

	// IntervalValue
	if iv, ok := m["interval_value"]; ok && iv != nil {
		switch v := iv.(type) {
		case int:
			schedule.IntervalValue = &v
		case float64:
			val := int(v)
			schedule.IntervalValue = &val
		}
	}

	// IntervalUnit
	if iu, ok := m["interval_unit"]; ok && iu != nil {
		if unit, ok := iu.(string); ok {
			schedule.IntervalUnit = &unit
		}
	}

	return schedule
}
