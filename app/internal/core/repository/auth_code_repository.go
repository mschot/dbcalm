package repository

import (
	"context"

	"github.com/martijn/dbcalm/internal/core/domain"
)

type AuthCodeRepository interface {
	Create(ctx context.Context, authCode *domain.AuthCode) error
	FindByCode(ctx context.Context, code string) (*domain.AuthCode, error)
	Delete(ctx context.Context, code string) error
	DeleteExpired(ctx context.Context) error
}
