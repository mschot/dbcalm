package repository

import (
	"context"

	"github.com/martijn/dbcalm/internal/core/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByUsername(ctx context.Context, username string) (*domain.User, error)
	Update(ctx context.Context, user *domain.User) error
	Delete(ctx context.Context, username string) error
	List(ctx context.Context) ([]*domain.User, error)
}
