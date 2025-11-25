package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/martijn/dbcalm/internal/api/dto"
	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
	"github.com/martijn/dbcalm/internal/core/service"
)

type ClientHandler struct {
	clientRepo  repository.ClientRepository
	authService *service.AuthService
}

func NewClientHandler(clientRepo repository.ClientRepository, authService *service.AuthService) *ClientHandler {
	return &ClientHandler{
		clientRepo:  clientRepo,
		authService: authService,
	}
}

// CreateClient handles POST /clients
func (h *ClientHandler) CreateClient(c *gin.Context) {
	var req dto.CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Generate secret
	secret := uuid.New().String()

	// Hash secret
	hashedSecret, err := h.authService.HashPassword(secret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: "Failed to create client",
			Code:    http.StatusInternalServerError,
		})
		return
	}

	// Create client
	client := domain.NewClient(req.Label, hashedSecret, []string{"all"})
	if err := h.clientRepo.Create(c.Request.Context(), client); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusCreated, dto.ClientCreateResponse{
		ID:        client.ID,
		Label:     client.Label,
		Secret:    secret, // Only shown on creation!
		Scopes:    client.Scopes,
		CreatedAt: client.CreatedAt,
		UpdatedAt: client.UpdatedAt,
	})
}

// GetClient handles GET /clients/:id
func (h *ClientHandler) GetClient(c *gin.Context) {
	id := c.Param("id")

	client, err := h.clientRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: fmt.Sprintf("Client not found: %s", id),
			Code:    http.StatusNotFound,
		})
		return
	}

	c.JSON(http.StatusOK, toClientResponse(client))
}

// ListClients handles GET /clients
func (h *ClientHandler) ListClients(c *gin.Context) {
	clients, err := h.clientRepo.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	total := len(clients)
	response := dto.ClientListResponse{
		Items: make([]dto.ClientResponse, total),
		Pagination: dto.PaginationInfo{
			Total:      total,
			Page:       1,
			PerPage:    total,
			TotalPages: 1,
		},
	}

	for i, client := range clients {
		response.Items[i] = toClientResponse(client)
	}

	c.JSON(http.StatusOK, response)
}

// UpdateClient handles PUT /clients/:id
func (h *ClientHandler) UpdateClient(c *gin.Context) {
	id := c.Param("id")

	var req dto.UpdateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	// Get existing client
	client, err := h.clientRepo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, dto.ErrorResponse{
			Error:   "Not Found",
			Message: fmt.Sprintf("Client not found: %s", id),
			Code:    http.StatusNotFound,
		})
		return
	}

	// Update label
	client.Label = req.Label
	if err := h.clientRepo.Update(c.Request.Context(), client); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusOK, toClientResponse(client))
}

// DeleteClient handles DELETE /clients/:id
func (h *ClientHandler) DeleteClient(c *gin.Context) {
	id := c.Param("id")

	if err := h.clientRepo.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, dto.ErrorResponse{
			Error:   "Internal Server Error",
			Message: err.Error(),
			Code:    http.StatusInternalServerError,
		})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

func toClientResponse(client *domain.Client) dto.ClientResponse {
	return dto.ClientResponse{
		ID:        client.ID,
		Label:     client.Label,
		Scopes:    client.Scopes,
		CreatedAt: client.CreatedAt,
		UpdatedAt: client.UpdatedAt,
	}
}
