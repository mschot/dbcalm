package handler

import (
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

// Allowed fields for process queries and ordering
var (
	processQueryFields = []string{"id", "command", "command_id", "pid", "status", "return_code", "start_time", "end_time", "type"}
	processOrderFields = []string{"id", "start_time", "end_time", "status"}
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
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "25"))

	filter := repository.ProcessFilter{
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
		if err := util.ValidateFilterFields(filters, processQueryFields); err != nil {
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
		if err := util.ValidateOrderFields(orders, processOrderFields); err != nil {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Bad Request",
				Message: err.Error(),
				Code:    http.StatusBadRequest,
			})
			return
		}

		filter.Order = orders
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
	totalPages := 0
	if perPage > 0 {
		totalPages = (count + perPage - 1) / perPage
	}

	response := dto.ProcessListResponse{
		Items: make([]dto.ProcessResponse, len(processes)),
		Pagination: dto.PaginationInfo{
			Total:      count,
			Page:       page,
			PerPage:    perPage,
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

	// Extract resource_id from args if available (for backup/restore operations)
	if process.Args != nil {
		if id, ok := process.Args["id"].(string); ok {
			response.ResourceID = &id
		}
	}

	return response
}
