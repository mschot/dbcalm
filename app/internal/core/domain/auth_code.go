package domain

import (
	"time"

	"github.com/google/uuid"
)

type AuthCode struct {
	Code      string    `db:"code"`
	Username  string    `db:"username"`
	Scopes    []string  `db:"scopes"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

func NewAuthCode(username string, scopes []string, expirationMinutes int) *AuthCode {
	now := time.Now()
	return &AuthCode{
		Code:      uuid.New().String(),
		Username:  username,
		Scopes:    scopes,
		ExpiresAt: now.Add(time.Duration(expirationMinutes) * time.Minute),
		CreatedAt: now,
	}
}

func (a *AuthCode) IsExpired() bool {
	return time.Now().After(a.ExpiresAt)
}
