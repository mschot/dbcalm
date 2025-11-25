package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/martijn/dbcalm/internal/api/dto"
	"github.com/martijn/dbcalm/internal/core/service"
)

const (
	AuthHeaderKey  = "Authorization"
	AuthContextKey = "auth"
)

// AuthMiddleware creates a JWT authentication middleware
func AuthMiddleware(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader(AuthHeaderKey)
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Unauthorized",
				Message: "Missing authorization header",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Check if it's a Bearer token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Unauthorized",
				Message: "Invalid authorization header format. Expected 'Bearer <token>'",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		token := parts[1]

		// Validate token
		claims, err := authService.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, dto.ErrorResponse{
				Error:   "Unauthorized",
				Message: "Invalid or expired token",
				Code:    http.StatusUnauthorized,
			})
			c.Abort()
			return
		}

		// Store claims in context
		c.Set(AuthContextKey, claims)

		c.Next()
	}
}

// GetAuthClaims retrieves auth claims from context
func GetAuthClaims(c *gin.Context) (*service.TokenClaims, bool) {
	claims, exists := c.Get(AuthContextKey)
	if !exists {
		return nil, false
	}

	tokenClaims, ok := claims.(*service.TokenClaims)
	return tokenClaims, ok
}
