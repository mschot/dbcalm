package socket

import (
	"encoding/json"
	"fmt"

	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/adapter"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/config"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/handler"
	"github.com/martijn/dbcalm-db-cmd/db-cmd-internal/validator"
	sharedProcess "github.com/martijn/dbcalm/shared/process"
	sharedSocket "github.com/martijn/dbcalm/shared/socket"
)

type DbCommandProcessor struct {
	config       *config.Config
	adapter      adapter.Adapter
	validator    *validator.Validator
	queueHandler *handler.QueueHandler
}

func NewDbCommandProcessor(cfg *config.Config, adptr adapter.Adapter, valid *validator.Validator, qHandler *handler.QueueHandler) *DbCommandProcessor {
	return &DbCommandProcessor{
		config:       cfg,
		adapter:      adptr,
		validator:    valid,
		queueHandler: qHandler,
	}
}

func (p *DbCommandProcessor) ProcessRequest(data []byte) sharedSocket.CommandResponse {
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
	case "full_backup":
		id := req.Args["id"].(string)
		var scheduleID *int
		if sid, ok := req.Args["schedule_id"].(float64); ok {
			sidInt := int(sid)
			scheduleID = &sidInt
		}
		proc, procChan, err = p.adapter.FullBackup(id, scheduleID)

	case "incremental_backup":
		id := req.Args["id"].(string)
		fromBackupID := req.Args["from_backup_id"].(string)
		var scheduleID *int
		if sid, ok := req.Args["schedule_id"].(float64); ok {
			sidInt := int(sid)
			scheduleID = &sidInt
		}
		proc, procChan, err = p.adapter.IncrementalBackup(id, fromBackupID, scheduleID)

	case "restore_backup":
		// Convert id_list to []string
		var idList []string
		if idListRaw, ok := req.Args["id_list"]; ok {
			switch v := idListRaw.(type) {
			case []interface{}:
				for _, item := range v {
					if str, ok := item.(string); ok {
						idList = append(idList, str)
					}
				}
			case []string:
				idList = v
			}
		}
		target := req.Args["target"].(string)
		proc, procChan, err = p.adapter.RestoreBackup(idList, target)

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
