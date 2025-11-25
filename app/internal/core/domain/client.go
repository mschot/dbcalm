package domain

import (
	"time"

	"github.com/google/uuid"
)

type Client struct {
	ID        string    `db:"id"` // UUID
	Secret    string    `db:"secret"` // bcrypt hashed
	Label     string    `db:"label"`
	Scopes    []string  `db:"scopes"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

func NewClient(label string, hashedSecret string, scopes []string) *Client {
	now := time.Now()
	return &Client{
		ID:        uuid.New().String(),
		Secret:    hashedSecret,
		Label:     label,
		Scopes:    scopes,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
