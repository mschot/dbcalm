package repository

import (
	"context"

	"github.com/martijn/dbcalm/internal/core/domain"
)

type ClientRepository interface {
	Create(ctx context.Context, client *domain.Client) error
	FindByID(ctx context.Context, id string) (*domain.Client, error)
	Update(ctx context.Context, client *domain.Client) error
	Delete(ctx context.Context, id string) error
	List(ctx context.Context) ([]*domain.Client, error)
}
