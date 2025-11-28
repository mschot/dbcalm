package service

import (
	"context"
	"fmt"

	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type ProcessService struct {
	processRepo repository.ProcessRepository
}

func NewProcessService(processRepo repository.ProcessRepository) *ProcessService {
	return &ProcessService{
		processRepo: processRepo,
	}
}

// Start is a no-op - process monitoring is handled by the cmd services that spawn them
func (s *ProcessService) Start() {
}

// Stop is a no-op
func (s *ProcessService) Stop() {
}

// CreateProcess creates a new process record
func (s *ProcessService) CreateProcess(ctx context.Context, command string, processType domain.ProcessType, args map[string]interface{}) (*domain.Process, error) {
	process := domain.NewProcess(command, processType, args)

	if err := s.processRepo.Create(ctx, process); err != nil {
		return nil, fmt.Errorf("failed to create process: %w", err)
	}

	return process, nil
}

// GetProcess retrieves a process by ID
func (s *ProcessService) GetProcess(ctx context.Context, id int64) (*domain.Process, error) {
	return s.processRepo.FindByID(ctx, id)
}

// GetProcessByCommandID retrieves a process by command ID
func (s *ProcessService) GetProcessByCommandID(ctx context.Context, commandID string) (*domain.Process, error) {
	return s.processRepo.FindByCommandID(ctx, commandID)
}

// ListProcesses lists processes with filtering
func (s *ProcessService) ListProcesses(ctx context.Context, filter repository.ProcessFilter) ([]*domain.Process, error) {
	return s.processRepo.List(ctx, filter)
}

// CountProcesses counts processes with filtering
func (s *ProcessService) CountProcesses(ctx context.Context, filter repository.ProcessFilter) (int, error) {
	return s.processRepo.Count(ctx, filter)
}

