package repository

import (
	"context"

	"github.com/martijn/dbcalm/internal/api/util"
	"github.com/martijn/dbcalm/internal/core/domain"
)

// ProcessFilter embeds ListFilter for generic query/order/pagination
type ProcessFilter struct {
	util.ListFilter
}

type ProcessRepository interface {
	Create(ctx context.Context, process *domain.Process) error
	FindByID(ctx context.Context, id int64) (*domain.Process, error)
	FindByCommandID(ctx context.Context, commandID string) (*domain.Process, error)
	Update(ctx context.Context, process *domain.Process) error
	List(ctx context.Context, filter ProcessFilter) ([]*domain.Process, error)
	Count(ctx context.Context, filter ProcessFilter) (int, error)

	// Find all running processes (for queue management)
	FindRunning(ctx context.Context) ([]*domain.Process, error)
}
