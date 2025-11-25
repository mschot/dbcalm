package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/martijn/dbcalm/internal/api/dto"
	"github.com/martijn/dbcalm/internal/core/service"
)

type CleanupHandler struct {
	cleanupService *service.CleanupService
}

func NewCleanupHandler(cleanupService *service.CleanupService) *CleanupHandler {
	return &CleanupHandler{
		cleanupService: cleanupService,
	}
}

// Cleanup handles POST /cleanup
func (h *CleanupHandler) Cleanup(c *gin.Context) {
	var req dto.CleanupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body provided, default to cleanup all
		req.ScheduleID = nil
	}

	var process, err = h.cleanupService.CleanupAll(c.Request.Context())

	if req.ScheduleID != nil && *req.ScheduleID > 0 {
		process, err = h.cleanupService.CleanupBySchedule(c.Request.Context(), *req.ScheduleID)
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
		Status: string(process.Status),
		Link:   &link,
		PID:    &process.CommandID,
	}

	// Add schedule_id as resource_id if provided
	if req.ScheduleID != nil && *req.ScheduleID > 0 {
		scheduleID := fmt.Sprintf("%d", *req.ScheduleID)
		response.ResourceID = &scheduleID
	}

	c.JSON(http.StatusAccepted, response)
}
