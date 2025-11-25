package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
)

type ProcessService struct {
	processRepo      repository.ProcessRepository
	mu               sync.RWMutex
	runningProcesses map[int64]*exec.Cmd
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
}

func NewProcessService(processRepo repository.ProcessRepository) *ProcessService {
	ctx, cancel := context.WithCancel(context.Background())
	return &ProcessService{
		processRepo:      processRepo,
		runningProcesses: make(map[int64]*exec.Cmd),
		ctx:              ctx,
		cancel:           cancel,
	}
}

// Start starts the process queue monitor
func (s *ProcessService) Start() {
	s.wg.Add(1)
	go s.monitorProcesses()
}

// Stop stops the process queue monitor
func (s *ProcessService) Stop() {
	s.cancel()
	s.wg.Wait()
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

// KillProcess kills a running process
func (s *ProcessService) KillProcess(ctx context.Context, id int64) error {
	s.mu.RLock()
	cmd, exists := s.runningProcesses[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("process not running: %d", id)
	}

	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	return nil
}

// IsProcessRunning checks if a process is running
func (s *ProcessService) IsProcessRunning(id int64) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.runningProcesses[id]
	return exists
}

// monitorProcesses monitors running processes and handles orphaned processes
func (s *ProcessService) monitorProcesses() {
	defer s.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkOrphanedProcesses()
		}
	}
}

// checkOrphanedProcesses checks for processes marked as running but not in memory
func (s *ProcessService) checkOrphanedProcesses() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	runningProcesses, err := s.processRepo.FindRunning(ctx)
	if err != nil {
		fmt.Printf("Failed to find running processes: %v\n", err)
		return
	}

	s.mu.RLock()
	trackedIDs := make(map[int64]bool)
	for id := range s.runningProcesses {
		trackedIDs[id] = true
	}
	s.mu.RUnlock()

	for _, process := range runningProcesses {
		if !trackedIDs[process.ID] {
			// Process is marked as running but not tracked - it might be orphaned
			if process.PID != nil {
				// Check if PID still exists
				if !s.pidExists(*process.PID) {
					// Process is dead, mark as failed
					process.Fail("Process orphaned and no longer running")
					if err := s.processRepo.Update(ctx, process); err != nil {
						fmt.Printf("Failed to update orphaned process %d: %v\n", process.ID, err)
					}
				}
			}
		}
	}
}

// pidExists checks if a PID exists in the system
func (s *ProcessService) pidExists(pid int) bool {
	// On Unix systems, sending signal 0 checks if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}
