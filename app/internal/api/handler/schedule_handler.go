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

type ScheduleHandler struct {
	scheduleService *service.ScheduleService
}

func NewScheduleHandler(scheduleService *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleService: scheduleService,
	}
}

// CreateSchedule handles POST /schedules
func (h *ScheduleHandler) CreateSchedule(c *gin.Context) {
	var req dto.CreateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	schedule := domain.NewSchedule(
		domain.BackupType(req.BackupType),
		domain.ScheduleFrequency(req.Frequency),
		req.Enabled,
	)

	schedule.DayOfWeek = req.DayOfWeek
	schedule.DayOfMonth = req.DayOfMonth
	schedule.Hour = req.Hour
	schedule.Minute = req.Minute
	schedule.IntervalValue = req.IntervalValue
	schedule.RetentionValue = req.RetentionValue

	if req.IntervalUnit != nil {
		iu := domain.IntervalUnit(*req.IntervalUnit)
		schedule.IntervalUnit = &iu
	}
	if req.RetentionUnit != nil {
		ru := domain.RetentionUnit(*req.RetentionUnit)
		schedule.RetentionUnit = &ru
	}

	if err := h.scheduleService.CreateSchedule(c.Request.Context(), schedule); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	c.JSON(http.StatusCreated, toScheduleResponse(schedule))
}

// GetSchedule handles GET /schedules/:id
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid schedule ID",
			Code:    http.StatusBadRequest,
		})
		return
	}

	schedule, err := h.scheduleService.GetSchedule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: fmt.Sprintf("Schedule not found: %d", id),
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, toScheduleResponse(schedule))
}

// ListSchedules handles GET /schedules
func (h *ScheduleHandler) ListSchedules(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := repository.ScheduleFilter{
		Limit:  limit,
		Offset: offset,
	}

	// Optional filters
	if backupType := c.Query("backup_type"); backupType != "" {
		bt := domain.BackupType(backupType)
		filter.BackupType = &bt
	}

	if enabled := c.Query("enabled"); enabled != "" {
		e := enabled == "true"
		filter.Enabled = &e
	}

	schedules, err := h.scheduleService.ListSchedules(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	count, _ := h.scheduleService.CountSchedules(c.Request.Context(), filter)

	// Calculate pagination info
	page := 1
	if limit > 0 {
		page = (offset / limit) + 1
	}
	totalPages := 0
	if limit > 0 {
		totalPages = (count + limit - 1) / limit
	}

	response := dto.ScheduleListResponse{
		Items: make([]dto.ScheduleResponse, len(schedules)),
		Pagination: dto.PaginationInfo{
			Total:      count,
			Page:       page,
			PerPage:    limit,
			TotalPages: totalPages,
		},
	}

	for i, schedule := range schedules {
		response.Items[i] = toScheduleResponse(schedule)
	}

	c.JSON(http.StatusOK, response)
}

// UpdateSchedule handles PUT /schedules/:id
func (h *ScheduleHandler) UpdateSchedule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid schedule ID",
			Code:    http.StatusBadRequest,
		})
		return
	}

	var req dto.UpdateScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Get existing schedule
	schedule, err := h.scheduleService.GetSchedule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: fmt.Sprintf("Schedule not found: %d", id),
			Code:    http.StatusNotFound,
		})
		return
	}

	// Update fields
	if req.BackupType != nil {
		schedule.BackupType = domain.BackupType(*req.BackupType)
	}
	if req.Frequency != nil {
		schedule.Frequency = domain.ScheduleFrequency(*req.Frequency)
	}
	if req.DayOfWeek != nil {
		schedule.DayOfWeek = req.DayOfWeek
	}
	if req.DayOfMonth != nil {
		schedule.DayOfMonth = req.DayOfMonth
	}
	if req.Hour != nil {
		schedule.Hour = req.Hour
	}
	if req.Minute != nil {
		schedule.Minute = req.Minute
	}
	if req.IntervalValue != nil {
		schedule.IntervalValue = req.IntervalValue
	}
	if req.IntervalUnit != nil {
		iu := domain.IntervalUnit(*req.IntervalUnit)
		schedule.IntervalUnit = &iu
	}
	if req.RetentionValue != nil {
		schedule.RetentionValue = req.RetentionValue
	}
	if req.RetentionUnit != nil {
		ru := domain.RetentionUnit(*req.RetentionUnit)
		schedule.RetentionUnit = &ru
	}
	if req.Enabled != nil {
		schedule.Enabled = *req.Enabled
	}

	if err := h.scheduleService.UpdateSchedule(c.Request.Context(), schedule); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	c.JSON(http.StatusOK, toScheduleResponse(schedule))
}

// DeleteSchedule handles DELETE /schedules/:id
func (h *ScheduleHandler) DeleteSchedule(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid schedule ID",
			Code:    http.StatusBadRequest,
		})
		return
	}

	if err := h.scheduleService.DeleteSchedule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func toScheduleResponse(schedule *domain.Schedule) dto.ScheduleResponse {
	response := dto.ScheduleResponse{
		ID:             schedule.ID,
		BackupType:     string(schedule.BackupType),
		Frequency:      string(schedule.Frequency),
		DayOfWeek:      schedule.DayOfWeek,
		DayOfMonth:     schedule.DayOfMonth,
		Hour:           schedule.Hour,
		Minute:         schedule.Minute,
		IntervalValue:  schedule.IntervalValue,
		RetentionValue: schedule.RetentionValue,
		Enabled:        schedule.Enabled,
		CreatedAt:      schedule.CreatedAt,
		UpdatedAt:      schedule.UpdatedAt,
	}

	if schedule.IntervalUnit != nil {
		iu := string(*schedule.IntervalUnit)
		response.IntervalUnit = &iu
	}
	if schedule.RetentionUnit != nil {
		ru := string(*schedule.RetentionUnit)
		response.RetentionUnit = &ru
	}

	return response
}
