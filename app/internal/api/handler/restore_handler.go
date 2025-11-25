package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/martijn/dbcalm/internal/api/dto"
	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
	"github.com/martijn/dbcalm/internal/core/service"
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
		c.JSON(http.StatusServiceUnavailable, dto.ErrorResponse{
			Error:   "Service Unavailable",
			Message: err.Error(),
			Code:    http.StatusServiceUnavailable,
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
	// Parse query parameters
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := repository.RestoreFilter{
		Limit:  limit,
		Offset: offset,
	}

	// Optional filters
	if backupID := c.Query("backup_id"); backupID != "" {
		filter.BackupID = &backupID
	}

	if target := c.Query("target"); target != "" {
		t := domain.RestoreTarget(target)
		filter.Target = &t
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
	page := 1
	if limit > 0 {
		page = (offset / limit) + 1
	}
	totalPages := 0
	if limit > 0 {
		totalPages = (count + limit - 1) / limit
	}

	response := dto.RestoreListResponse{
		Items: make([]dto.RestoreResponse, len(restores)),
		Pagination: dto.PaginationInfo{
			Total:      count,
			Page:       page,
			PerPage:    limit,
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
