package handler

import (
	"log"

	sharedProcess "github.com/martijn/dbcalm/shared/process"
)

type QueueHandler struct {
	// No special dependencies needed for cmd service
	// Just logs completion/failure
}

func NewQueueHandler() *QueueHandler {
	return &QueueHandler{}
}

// Handle processes the completed process from the channel
// For the cmd service, we just log the completion/failure
// No database transformations needed like in db-cmd service
func (h *QueueHandler) Handle(processChan <-chan *sharedProcess.Process) {
	go func() {
		for proc := range processChan {
			if proc.Status == sharedProcess.StatusSuccess {
				log.Printf("Process %s completed successfully (type: %s)", proc.CommandID, proc.Type)
			} else {
				log.Printf("Process %s failed (type: %s)", proc.CommandID, proc.Type)
				if proc.Error != nil {
					log.Printf("Error: %s", *proc.Error)
				}
			}
		}
	}()
}
