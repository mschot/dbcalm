package service

import (
	"context"
	"fmt"
	"time"

	"github.com/martijn/dbcalm/internal/adapter/cmd"
	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type ScheduleService struct {
	scheduleRepo repository.ScheduleRepository
	backupRepo   repository.BackupRepository
	processServ  *ProcessService
	cmdClient    *cmd.Client
	dbcalmBinary string // Path to dbcalm binary
	logDir       string // Log directory
}

func NewScheduleService(
	scheduleRepo repository.ScheduleRepository,
	backupRepo repository.BackupRepository,
	processServ *ProcessService,
	cmdClient *cmd.Client,
	dbcalmBinary string,
	logDir string,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		backupRepo:   backupRepo,
		processServ:  processServ,
		cmdClient:    cmdClient,
		dbcalmBinary: dbcalmBinary,
		logDir:       logDir,
	}
}

// CreateSchedule creates a new schedule
func (s *ScheduleService) CreateSchedule(ctx context.Context, schedule *domain.Schedule) error {
	// Validate schedule
	if err := s.validateSchedule(ctx, schedule); err != nil {
		return err
	}

	// Create schedule
	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	// Update cron file
	if err := s.updateCronFile(ctx); err != nil {
		// Rollback schedule creation
		_ = s.scheduleRepo.Delete(ctx, schedule.ID)
		return fmt.Errorf("failed to update cron file: %w", err)
	}

	return nil
}

// UpdateSchedule updates an existing schedule
func (s *ScheduleService) UpdateSchedule(ctx context.Context, schedule *domain.Schedule) error {
	// Validate schedule
	if err := s.validateSchedule(ctx, schedule); err != nil {
		return err
	}

	// Update schedule
	schedule.UpdatedAt = time.Now()
	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	// Update cron file
	if err := s.updateCronFile(ctx); err != nil {
		return fmt.Errorf("failed to update cron file: %w", err)
	}

	return nil
}

// DeleteSchedule deletes a schedule
func (s *ScheduleService) DeleteSchedule(ctx context.Context, id int64) error {
	if err := s.scheduleRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	// Update cron file
	if err := s.updateCronFile(ctx); err != nil {
		return fmt.Errorf("failed to update cron file: %w", err)
	}

	return nil
}

// GetSchedule retrieves a schedule by ID
func (s *ScheduleService) GetSchedule(ctx context.Context, id int64) (*domain.Schedule, error) {
	return s.scheduleRepo.FindByID(ctx, id)
}

// ListSchedules lists schedules with filtering
func (s *ScheduleService) ListSchedules(ctx context.Context, filter repository.ScheduleFilter) ([]*domain.Schedule, error) {
	return s.scheduleRepo.List(ctx, filter)
}

// CountSchedules counts schedules with filtering
func (s *ScheduleService) CountSchedules(ctx context.Context, filter repository.ScheduleFilter) (int, error) {
	return s.scheduleRepo.Count(ctx, filter)
}

// validateSchedule validates a schedule
func (s *ScheduleService) validateSchedule(ctx context.Context, schedule *domain.Schedule) error {
	// If incremental backup, ensure there's at least one enabled full schedule
	if schedule.BackupType == domain.BackupTypeIncremental && schedule.Enabled {
		fullSchedules, err := s.scheduleRepo.FindEnabledFullSchedules(ctx)
		if err != nil {
			return fmt.Errorf("failed to check full schedules: %w", err)
		}
		if len(fullSchedules) == 0 {
			return fmt.Errorf("incremental backups require at least one enabled full backup schedule")
		}
	}

	// Validate frequency-specific fields
	switch schedule.Frequency {
	case domain.FrequencyDaily:
		if schedule.Hour == nil || schedule.Minute == nil {
			return fmt.Errorf("daily schedules require hour and minute")
		}
	case domain.FrequencyWeekly:
		if schedule.DayOfWeek == nil || schedule.Hour == nil || schedule.Minute == nil {
			return fmt.Errorf("weekly schedules require day_of_week, hour, and minute")
		}
	case domain.FrequencyMonthly:
		if schedule.DayOfMonth == nil || schedule.Hour == nil || schedule.Minute == nil {
			return fmt.Errorf("monthly schedules require day_of_month, hour, and minute")
		}
	case domain.FrequencyHourly:
		if schedule.Minute == nil {
			return fmt.Errorf("hourly schedules require minute")
		}
	case domain.FrequencyInterval:
		if schedule.IntervalValue == nil || schedule.IntervalUnit == nil {
			return fmt.Errorf("interval schedules require interval_value and interval_unit")
		}
	}

	return nil
}

// updateCronFile updates the system cron file with all enabled schedules via socket service
func (s *ScheduleService) updateCronFile(ctx context.Context) error {
	// Get all enabled schedules
	schedules, err := s.scheduleRepo.FindAllEnabled(ctx)
	if err != nil {
		return fmt.Errorf("failed to get enabled schedules: %w", err)
	}

	// Convert schedules to format expected by socket service
	scheduleData := make([]map[string]interface{}, len(schedules))
	for i, schedule := range schedules {
		scheduleData[i] = map[string]interface{}{
			"id":             schedule.ID,
			"backup_type":    string(schedule.BackupType),
			"frequency":      string(schedule.Frequency),
			"day_of_week":    schedule.DayOfWeek,
			"day_of_month":   schedule.DayOfMonth,
			"hour":           schedule.Hour,
			"minute":         schedule.Minute,
			"interval_value": schedule.IntervalValue,
			"interval_unit":  schedule.IntervalUnit,
			"enabled":        schedule.Enabled,
		}
	}

	// Update cron schedules via socket service (matches Python architecture)
	cronArgs := map[string]interface{}{
		"schedules": scheduleData,
	}
	response, err := s.cmdClient.SendCommand(ctx, "update_cron_schedules", cronArgs)
	if err != nil {
		return fmt.Errorf("failed to update cron schedules via socket service: %w", err)
	}
	if response.Code != 200 && response.Code != 202 {
		return fmt.Errorf("update cron schedules failed: %s", response.Status)
	}

	return nil
}

// GetBackupsForSchedule gets all backups for a schedule
func (s *ScheduleService) GetBackupsForSchedule(ctx context.Context, scheduleID int64) ([]*domain.Backup, error) {
	return s.backupRepo.FindBySchedule(ctx, scheduleID)
}
