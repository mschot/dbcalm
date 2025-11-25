package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/martijn/dbcalm/internal/api/dto"
	"github.com/martijn/dbcalm/internal/core/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// Authorize handles POST /auth/authorize
func (h *AuthHandler) Authorize(c *gin.Context) {
	var req dto.AuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	authCode, err := h.authService.AuthorizeUser(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
			Error:   "Unauthorized",
			Message: "Invalid credentials",
			Code:    http.StatusUnauthorized,
		})
		return
	}

	c.JSON(http.StatusOK, dto.AuthorizeResponse{
		Code: authCode.Code,
	})
}

// Token handles POST /auth/token
func (h *AuthHandler) Token(c *gin.Context) {
	var req dto.TokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: err.Error(),
			Code:    http.StatusBadRequest,
		})
		return
	}

	var token string
	var err error

	switch req.GrantType {
	case "authorization_code":
		if req.Code == "" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Bad Request",
				Message: "code is required for authorization_code grant type",
				Code:    http.StatusBadRequest,
			})
			return
		}

		token, err = h.authService.ExchangeAuthCode(c.Request.Context(), req.Code)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Unauthorized",
				Message: "Invalid or expired authorization code",
				Code:    http.StatusUnauthorized,
			})
			return
		}

	case "client_credentials":
		if req.ClientID == "" || req.ClientSecret == "" {
			c.JSON(http.StatusBadRequest, dto.ErrorResponse{
				Error:   "Bad Request",
				Message: "client_id and client_secret are required for client_credentials grant type",
				Code:    http.StatusBadRequest,
			})
			return
		}

		token, err = h.authService.AuthenticateClient(c.Request.Context(), req.ClientID, req.ClientSecret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Unauthorized",
				Message: "Invalid client credentials",
				Code:    http.StatusUnauthorized,
			})
			return
		}

	default:
		c.JSON(http.StatusBadRequest, dto.ErrorResponse{
			Error:   "Bad Request",
			Message: "Invalid grant_type. Must be 'authorization_code' or 'client_credentials'",
			Code:    http.StatusBadRequest,
		})
		return
	}

	c.JSON(http.StatusOK, dto.TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   3600, // 1 hour
	})
}
