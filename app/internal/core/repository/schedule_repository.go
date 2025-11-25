package repository

import (
	"context"

	"github.com/martijn/dbcalm/internal/core/domain"
)

type ScheduleFilter struct {
	BackupType *domain.BackupType
	Enabled    *bool
	Limit      int
	Offset     int
}

type ScheduleRepository interface {
	Create(ctx context.Context, schedule *domain.Schedule) error
	FindByID(ctx context.Context, id int64) (*domain.Schedule, error)
	Update(ctx context.Context, schedule *domain.Schedule) error
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, filter ScheduleFilter) ([]*domain.Schedule, error)
	Count(ctx context.Context, filter ScheduleFilter) (int, error)

	// Find all enabled full schedules (for validation)
	FindEnabledFullSchedules(ctx context.Context) ([]*domain.Schedule, error)

	// Find all enabled schedules (for cron generation)
	FindAllEnabled(ctx context.Context) ([]*domain.Schedule, error)
}
