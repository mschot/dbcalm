package handler

import (
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

// Allowed fields for restore queries and ordering
var (
	restoreQueryFields = []string{"id", "start_time", "end_time", "target", "target_path", "backup_id", "backup_timestamp", "process_id"}
	restoreOrderFields = []string{"id", "start_time", "end_time", "backup_id"}
)

type RestoreHandler struct {
	restoreService *service.RestoreService
	backupRepo     repository.BackupRepository
}

func NewRestoreHandler(restoreService *service.RestoreService, backupRepo repository.BackupRepository) *RestoreHandler {
	return &RestoreHandler{
		restoreService: restoreService,
		backupRepo:     backupRepo,
	}
}

// CreateRestore handles POST /restore
func (h *RestoreHandler) CreateRestore(c *gin.Context) {
	var req dto.CreateRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Validate backup exists before starting async restore (matches Python behavior)
	backup, err := h.backupRepo.FindByID(c.Request.Context(), req.BackupID)
	if err != nil || backup == nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: fmt.Sprintf("Backup with id %s not found", req.BackupID),
			Code:    http.StatusNotFound,
		})
		return
	}

	var process *domain.Process

	if req.Target == "database" {
		process, err = h.restoreService.RestoreToDatabase(c.Request.Context(), req.BackupID)
	} else {
		process, err = h.restoreService.RestoreToFolder(c.Request.Context(), req.BackupID)
	}

	if err != nil {
		// Return error in AsyncResponse format to match Python behavior
		// Frontend expects {"status": "error message"} format
		var svcErr *service.ServiceError
		var statusCode int
		var message string
		if errors.As(err, &svcErr) {
			statusCode = svcErr.Code
			message = svcErr.Message
		} else {
			statusCode = http.StatusInternalServerError
			message = err.Error()
		}
		c.JSON(statusCode, dto.AsyncResponse{
			Status: message,
		})
		return
	}

	// Build response matching Python StatusResponse format
	link := fmt.Sprintf("/status/%s", process.CommandID)
	response := dto.AsyncResponse{
		Status:     string(process.Status),
		Link:       &link,
		PID:        &process.CommandID,
		ResourceID: &req.BackupID, // Resource is the backup being restored
	}

	c.JSON(http.StatusAccepted, response)
}

// ListRestores handles GET /restores
func (h *RestoreHandler) ListRestores(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "25"))

	filter := repository.RestoreFilter{
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
		if err := util.ValidateFilterFields(filters, restoreQueryFields); err != nil {
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
		if err := util.ValidateOrderFields(orders, restoreOrderFields); err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Bad Request",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		filter.Order = orders
	}

	restores, err := h.restoreService.ListRestores(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	count, _ := h.restoreService.CountRestores(c.Request.Context(), filter)

	// Calculate pagination info
	totalPages := 0
	if perPage > 0 {
		totalPages = (count + perPage - 1) / perPage
	}

	response := dto.RestoreListResponse{
		Items: make([]dto.RestoreResponse, len(restores)),
		Pagination: dto.PaginationInfo{
			Total:      count,
			Page:       page,
			PerPage:    perPage,
			TotalPages: totalPages,
		},
	}

	for i, restore := range restores {
		response.Items[i] = toRestoreResponse(restore)
	}

	c.JSON(http.StatusOK, response)
}

// GetRestore handles GET /restores/:id
func (h *RestoreHandler) GetRestore(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid restore ID",
			Code:    http.StatusBadRequest,
		})
		return
	}

	restore, err := h.restoreService.GetRestore(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: fmt.Sprintf("Restore not found: %d", id),
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, toRestoreResponse(restore))
}

func toRestoreResponse(restore *domain.Restore) dto.RestoreResponse {
	return dto.RestoreResponse{
		ID:              restore.ID,
		BackupID:        restore.BackupID,
		BackupTimestamp: restore.BackupTimestamp,
		Target:          string(restore.Target),
		TargetPath:      restore.TargetPath,
		StartTime:       restore.StartTime,
		EndTime:         restore.EndTime,
		ProcessID:       restore.ProcessID,
	}
}
