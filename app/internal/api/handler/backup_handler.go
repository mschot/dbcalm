package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/martijn/dbcalm/internal/api/dto"
	"github.com/martijn/dbcalm/internal/api/util"
	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
	"github.com/martijn/dbcalm/internal/core/service"
)

// Allowed fields for backup queries and ordering
var (
	backupQueryFields = []string{"id", "from_backup_id", "schedule_id", "start_time", "end_time", "process_id"}
	backupOrderFields = []string{"id", "start_time", "end_time"}
)

type BackupHandler struct {
	backupService    *service.BackupService
	scheduleRepo     repository.ScheduleRepository
}

func NewBackupHandler(backupService *service.BackupService, scheduleRepo repository.ScheduleRepository) *BackupHandler {
	return &BackupHandler{
		backupService: backupService,
		scheduleRepo:  scheduleRepo,
	}
}

// CreateBackup handles POST /backups
func (h *BackupHandler) CreateBackup(c *gin.Context) {
	var req dto.CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	var process *domain.Process
	var err error

	if req.Type == "full" {
		process, err = h.backupService.CreateFullBackup(c.Request.Context(), req.BackupID, req.ScheduleID)
	} else {
		process, err = h.backupService.CreateIncrementalBackup(c.Request.Context(), req.BackupID, req.FromBackupID, req.ScheduleID)
	}

	if err != nil {
		var svcErr *service.ServiceError
		if errors.As(err, &svcErr) {
			c.JSON(svcErr.Code, dto.ErrorResponse{
				Error:   http.StatusText(svcErr.Code),
				Message: svcErr.Message,
				Code:    svcErr.Code,
			})
		} else {
			c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
				Error:   "Internal Server Error",
				Message: err.Error(),
				Code:    http.StatusInternalServerError,
			})
		}
		return
	}

	// Build response matching Python StatusResponse format
	link := fmt.Sprintf("/status/%s", process.CommandID)
	response := dto.AsyncResponse{
		Status: string(process.Status),
		Link:   &link,
		PID:    &process.CommandID,
	}

	// Add resource_id (backup_id) if provided in request
	if req.BackupID != nil {
		response.ResourceID = req.BackupID
	}

	c.JSON(http.StatusAccepted, response)
}

// GetBackup handles GET /backups/:id
func (h *BackupHandler) GetBackup(c *gin.Context) {
	id := c.Param("id")

	backup, err := h.backupService.GetBackup(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: fmt.Sprintf("Backup not found: %s", id),
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, toBackupResponse(backup))
}

// ListBackups handles GET /backups
func (h *BackupHandler) ListBackups(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "25"))

	filter := repository.BackupFilter{
		ListFilter: util.ListFilter{
			Page:    page,
			PerPage: perPage,
		},
	}

	// Parse query filters
	if queryStr := c.Query("query"); queryStr != "" {
		filters, err := util.ParseQueryString(queryStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Bad Request",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		// Validate field names
		if err := util.ValidateFilterFields(filters, backupQueryFields); err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Bad Request",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		filter.Filters = filters
	}

	// Parse order
	if orderStr := c.Query("order"); orderStr != "" {
		orders, err := util.ParseOrderString(orderStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Bad Request",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		// Validate field names
		if err := util.ValidateOrderFields(orders, backupOrderFields); err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Bad Request",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		filter.Order = orders
	}

	backups, err := h.backupService.ListBackups(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	count, _ := h.backupService.CountBackups(c.Request.Context(), filter)

	// Calculate pagination info
	totalPages := 0
	if perPage > 0 {
		totalPages = (count + perPage - 1) / perPage
	}

	response := dto.BackupListResponse{
		Items: make([]dto.BackupResponse, len(backups)),
		Pagination: dto.PaginationInfo{
			Total:      count,
			Page:       page,
			PerPage:    perPage,
			TotalPages: totalPages,
		},
	}

	for i, backup := range backups {
		response.Items[i] = h.toBackupResponseWithRetention(c.Request.Context(), backup)
	}

	c.JSON(http.StatusOK, response)
}

func toBackupResponse(backup *domain.Backup) dto.BackupResponse {
	return dto.BackupResponse{
		ID:           backup.ID,
		Type:         string(backup.Type),
		FromBackupID: backup.FromBackupID,
		ScheduleID:   backup.ScheduleID,
		StartTime:    backup.StartTime,
		EndTime:      backup.EndTime,
		ProcessID:    backup.ProcessID,
		Size:         backup.Size,
	}
}

func (h *BackupHandler) toBackupResponseWithRetention(ctx context.Context, backup *domain.Backup) dto.BackupResponse {
	resp := toBackupResponse(backup)

	// Add retention info from schedule if available
	if backup.ScheduleID != nil {
		schedule, err := h.scheduleRepo.FindByID(ctx, *backup.ScheduleID)
		if err == nil && schedule != nil {
			resp.RetentionValue = schedule.RetentionValue
			if schedule.RetentionUnit != nil {
				unit := string(*schedule.RetentionUnit)
				resp.RetentionUnit = &unit
			}
		}
	}

	return resp
}
