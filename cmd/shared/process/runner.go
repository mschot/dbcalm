package process

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Runner struct {
	writer *Writer
}

func NewRunner(writer *Writer) *Runner {
	return &Runner{writer: writer}
}

func (r *Runner) Execute(command []string, commandType string, commandID *string, args map[string]interface{}) (*Process, chan *Process) {
	// Generate command ID if not provided
	if commandID == nil {
		id := uuid.New().String()
		commandID = &id
	}

	processChan := make(chan *Process, 1)

	// Start command
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = getCleanEnvForSystemBinaries()

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		// Create failed process
		now := time.Now()
		errMsg := err.Error()
		returnCode := -1
		
		process := &Process{
			Command:    strings.Join(command, " "),
			CommandID:  *commandID,
			PID:        0,
			Status:     StatusFailed,
			Error:      &errMsg,
			ReturnCode: &returnCode,
			StartTime:  now,
			EndTime:    &now,
			Type:       commandType,
			Args:       args,
		}

		processChan <- process
		return process, processChan
	}

	pid := cmd.Process.Pid
	startTime := time.Now()

	// Create process record in database
	argsJSON, _ := json.Marshal(args)
	processID, err := r.writer.CreateProcess(
		strings.Join(command, " "),
		*commandID,
		pid,
		StatusRunning,
		commandType,
		args,
		startTime,
	)

	if err != nil {
		log.Printf("Failed to create process record: %v", err)
	}

	// Create initial process model
	process := &Process{
		ID:        &processID,
		Command:   strings.Join(command, " "),
		CommandID: *commandID,
		PID:       pid,
		Status:    StatusRunning,
		StartTime: startTime,
		Type:      commandType,
		Args:      args,
		ArgsJSON:  string(argsJSON),
	}

	// Start goroutine to wait for completion
	go r.waitForCompletion(cmd, process, &stdout, &stderr, processChan)

	return process, processChan
}

func (r *Runner) waitForCompletion(cmd *exec.Cmd, process *Process, stdout, stderr *bytes.Buffer, processChan chan *Process) {
	defer close(processChan)

	// Wait for command to complete
	err := cmd.Wait()
	endTime := time.Now()
	process.EndTime = &endTime

	// Get output
	outputStr := stdout.String()
	errorStr := stderr.String()

	// Get return code
	returnCode := cmd.ProcessState.ExitCode()
	process.ReturnCode = &returnCode

	// Match Python's behavior: combine on success, separate on failure
	if returnCode == 0 {
		// Success: combine stdout and stderr into output field
		var combinedOutput string
		if outputStr != "" {
			combinedOutput = outputStr
		}
		if errorStr != "" {
			if combinedOutput != "" {
				combinedOutput += "\n"
			}
			combinedOutput += errorStr
		}
		if combinedOutput != "" {
			process.Output = &combinedOutput
		}
		// Error field stays nil for successful processes
	} else {
		// Failure: keep stdout and stderr separate
		if outputStr != "" {
			process.Output = &outputStr
		}
		if errorStr != "" {
			process.Error = &errorStr
		}
	}

	// Update status
	if err != nil || returnCode != 0 {
		process.Status = StatusFailed
	} else {
		process.Status = StatusSuccess
	}

	// Update database
	if process.ID != nil {
		err := r.writer.UpdateProcessStatus(
			*process.ID,
			process.Status,
			process.Output,
			process.Error,
			process.ReturnCode,
			process.EndTime,
		)
		if err != nil {
			log.Printf("Failed to update process status: %v", err)
		}
	}

	// Send completed process to channel
	processChan <- process
}

func (r *Runner) ExecuteConsecutive(commands [][]string, commandType string, args map[string]interface{}) (*Process, chan *Process) {
	commandID := uuid.New().String()
	masterChan := make(chan *Process, 1)
	hasOneChan := make(chan *Process, 1)

	// Start goroutine to execute commands sequentially
	go r.runCommandsSequentially(commands, commandType, commandID, args, masterChan, hasOneChan)

	// Wait for first process to start
	firstProcess := <-hasOneChan

	return firstProcess, masterChan
}

func (r *Runner) runCommandsSequentially(commands [][]string, commandType string, commandID string, args map[string]interface{}, masterChan, hasOneChan chan *Process) {
	defer close(masterChan)
	defer close(hasOneChan)

	var lastProcess *Process

	for i, command := range commands {
		// Execute command
		process, processChan := r.Execute(command, commandType, &commandID, args)

		// Send first process to hasOneChan
		if i == 0 {
			hasOneChan <- process
		}

		// Wait for completion
		completedProcess := <-processChan
		lastProcess = completedProcess

		// Stop on first failure
		if completedProcess.ReturnCode != nil && *completedProcess.ReturnCode != 0 {
			log.Printf("Command failed with return code %d, stopping execution", *completedProcess.ReturnCode)
			break
		}
	}

	// Send final process to master channel
	if lastProcess != nil {
		masterChan <- lastProcess
	}
}

func getCleanEnvForSystemBinaries() []string {
	env := os.Environ()

	// Check if we're running from a compiled binary (equivalent to PyInstaller frozen check)
	// In Go, we can check if LD_LIBRARY_PATH is set and reset it
	var cleanEnv []string
	for _, e := range env {
		if !strings.HasPrefix(e, "LD_LIBRARY_PATH=") {
			cleanEnv = append(cleanEnv, e)
		}
	}

	// Set clean LD_LIBRARY_PATH for system binaries
	cleanEnv = append(cleanEnv, "LD_LIBRARY_PATH=/usr/lib/x86_64-linux-gnu:/usr/lib:/lib")

	return cleanEnv
}

func (r *Runner) ExecuteWithStreaming(command []string, commandType string, commandID *string, args map[string]interface{}, outputWriter io.Writer) (*Process, chan *Process) {
	// Generate command ID if not provided
	if commandID == nil {
		id := uuid.New().String()
		commandID = &id
	}

	processChan := make(chan *Process, 1)

	// Start command
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Env = getCleanEnvForSystemBinaries()

	// Set output to writer if provided
	if outputWriter != nil {
		cmd.Stdout = outputWriter
		cmd.Stderr = outputWriter
	}

	err := cmd.Start()
	if err != nil {
		now := time.Now()
		errMsg := err.Error()
		returnCode := -1

		process := &Process{
			Command:    strings.Join(command, " "),
			CommandID:  *commandID,
			PID:        0,
			Status:     StatusFailed,
			Error:      &errMsg,
			ReturnCode: &returnCode,
			StartTime:  now,
			EndTime:    &now,
			Type:       commandType,
			Args:       args,
		}

		processChan <- process
		return process, processChan
	}

	pid := cmd.Process.Pid
	startTime := time.Now()

	// Create process record in database
	argsJSON, _ := json.Marshal(args)
	processID, err := r.writer.CreateProcess(
		strings.Join(command, " "),
		*commandID,
		pid,
		StatusRunning,
		commandType,
		args,
		startTime,
	)

	if err != nil {
		log.Printf("Failed to create process record: %v", err)
	}

	// Create initial process model
	process := &Process{
		ID:        &processID,
		Command:   strings.Join(command, " "),
		CommandID: *commandID,
		PID:       pid,
		Status:    StatusRunning,
		StartTime: startTime,
		Type:      commandType,
		Args:      args,
		ArgsJSON:  string(argsJSON),
	}

	// Start goroutine to wait for completion (without capturing output)
	go r.waitForCompletionNoCapture(cmd, process, processChan)

	return process, processChan
}

func (r *Runner) waitForCompletionNoCapture(cmd *exec.Cmd, process *Process, processChan chan *Process) {
	defer close(processChan)

	// Wait for command to complete
	err := cmd.Wait()
	endTime := time.Now()
	process.EndTime = &endTime

	// Get return code
	returnCode := cmd.ProcessState.ExitCode()
	process.ReturnCode = &returnCode

	// Update status
	if err != nil || returnCode != 0 {
		process.Status = StatusFailed
	} else {
		process.Status = StatusSuccess
	}

	// Update database
	if process.ID != nil {
		err := r.writer.UpdateProcessStatus(
			*process.ID,
			process.Status,
			nil, // No output captured when streaming
			nil,
			process.ReturnCode,
			process.EndTime,
		)
		if err != nil {
			log.Printf("Failed to update process status: %v", err)
		}
	}

	// Send completed process to channel
	processChan <- process
}
