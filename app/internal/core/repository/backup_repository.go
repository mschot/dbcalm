package repository

import (
	"context"

	"github.com/martijn/dbcalm/internal/api/util"
	"github.com/martijn/dbcalm/internal/core/domain"
)

// BackupFilter embeds ListFilter for generic query/order/pagination
type BackupFilter struct {
	util.ListFilter
}

type BackupRepository interface {
	Create(ctx context.Context, backup *domain.Backup) error
	FindByID(ctx context.Context, id string) (*domain.Backup, error)
	Update(ctx context.Context, backup *domain.Backup) error
	Delete(ctx context.Context, id string) error
	DeleteMany(ctx context.Context, ids []string) error
	List(ctx context.Context, filter BackupFilter) ([]*domain.Backup, error)
	Count(ctx context.Context, filter BackupFilter) (int, error)

	// Find the latest backup for a given schedule and type
	FindLatestByScheduleAndType(ctx context.Context, scheduleID *int64, backupType domain.BackupType) (*domain.Backup, error)

	// Find all backups in the chain (for incremental backups)
	FindChain(ctx context.Context, backupID string) ([]*domain.Backup, error)

	// Find backups for retention policy evaluation
	FindBySchedule(ctx context.Context, scheduleID int64) ([]*domain.Backup, error)
}
