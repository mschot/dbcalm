package repository

import (
	"context"

	"github.com/martijn/dbcalm/internal/api/util"
	"github.com/martijn/dbcalm/internal/core/domain"
)

// RestoreFilter embeds ListFilter for generic query/order/pagination
type RestoreFilter struct {
	util.ListFilter
}

type RestoreRepository interface {
	Create(ctx context.Context, restore *domain.Restore) error
	FindByID(ctx context.Context, id int64) (*domain.Restore, error)
	Update(ctx context.Context, restore *domain.Restore) error
	List(ctx context.Context, filter RestoreFilter) ([]*domain.Restore, error)
	Count(ctx context.Context, filter RestoreFilter) (int, error)
}
