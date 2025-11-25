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

type ProcessHandler struct {
	processService *service.ProcessService
}

func NewProcessHandler(processService *service.ProcessService) *ProcessHandler {
	return &ProcessHandler{
		processService: processService,
	}
}

// ListProcesses handles GET /processes
func (h *ProcessHandler) ListProcesses(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

	filter := repository.ProcessFilter{
		Limit:  limit,
		Offset: offset,
	}

	// Optional filters
	if processType := c.Query("type"); processType != "" {
		pt := domain.ProcessType(processType)
		filter.Type = &pt
	}

	if status := c.Query("status"); status != "" {
		s := domain.ProcessStatus(status)
		filter.Status = &s
	}

	processes, err := h.processService.ListProcesses(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	count, _ := h.processService.CountProcesses(c.Request.Context(), filter)

	// Calculate pagination info
	page := 1
	if limit > 0 {
		page = (offset / limit) + 1
	}
	totalPages := 0
	if limit > 0 {
		totalPages = (count + limit - 1) / limit
	}

	response := dto.ProcessListResponse{
		Items: make([]dto.ProcessResponse, len(processes)),
		Pagination: dto.PaginationInfo{
			Total:      count,
			Page:       page,
			PerPage:    limit,
			TotalPages: totalPages,
		},
	}

	for i, process := range processes {
		response.Items[i] = toProcessResponse(process)
	}

	c.JSON(http.StatusOK, response)
}

// GetProcessByCommandID handles GET /status/:command_id
func (h *ProcessHandler) GetProcessByCommandID(c *gin.Context) {
	commandID := c.Param("command_id")

	process, err := h.processService.GetProcessByCommandID(c.Request.Context(), commandID)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: fmt.Sprintf("Process not found: %s", commandID),
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, toProcessResponse(process))
}

// GetProcess handles GET /processes/:id
func (h *ProcessHandler) GetProcess(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid process ID",
			Code:    http.StatusBadRequest,
		})
		return
	}

	process, err := h.processService.GetProcess(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: fmt.Sprintf("Process not found: %d", id),
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, toProcessResponse(process))
}

func toProcessResponse(process *domain.Process) dto.ProcessResponse {
	response := dto.ProcessResponse{
		ID:         process.ID,
		CommandID:  process.CommandID,
		Command:    process.Command,
		PID:        process.PID,
		Status:     string(process.Status),
		Output:     process.Output,
		Error:      process.Error,
		ReturnCode: process.ReturnCode,
		StartTime:  process.StartTime,
		EndTime:    process.EndTime,
		Type:       string(process.Type),
		Args:       process.Args,
	}

	// Add status link
	link := fmt.Sprintf("/status/%s", process.CommandID)
	response.Link = &link

	return response
}
