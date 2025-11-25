package dto

import "time"

// CreateClientRequest represents the client creation request
type CreateClientRequest struct {
	Label string `json:"label" binding:"required"`
}

// UpdateClientRequest represents the client update request
type UpdateClientRequest struct {
	Label string `json:"label" binding:"required"`
}

// ClientResponse represents a client
type ClientResponse struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ClientCreateResponse includes the secret (only shown once)
type ClientCreateResponse struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Secret    string    `json:"secret"` // Only included on creation
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ClientListResponse represents a list of clients
type ClientListResponse struct {
	Items      []ClientResponse `json:"items"`
	Pagination PaginationInfo   `json:"pagination"`
}
