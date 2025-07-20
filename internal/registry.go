package internal

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/shirou/gopsutil/v4/process"
)

// RegistryConfig holds configuration for the process registry
type RegistryConfig struct {
	MaxProcesses     int           `json:"max_processes"`
	OutputBufferSize int           `json:"output_buffer_size"`
	CleanupInterval  time.Duration `json:"cleanup_interval"`
	ProcessTimeout   time.Duration `json:"process_timeout"`
	PersistenceFile  string        `json:"persistence_file"`
}

// DefaultRegistryConfig returns the default configuration
func DefaultRegistryConfig() *RegistryConfig {
	return &RegistryConfig{
		MaxProcesses:     1000,
		OutputBufferSize: 100 * 1024 * 1024, // 100MB
		CleanupInterval:  30 * time.Second,
		ProcessTimeout:   0, // No timeout by default
		PersistenceFile:  "",
	}
}

// Registry manages a collection of processes
type Registry struct {
	config    *RegistryConfig
	processes map[string]*Process
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewRegistry creates a new process registry with the given configuration
func NewRegistry(config *RegistryConfig) *Registry {
	if config == nil {
		config = DefaultRegistryConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	registry := &Registry{
		config:    config,
		processes: make(map[string]*Process),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start cleanup goroutine
	go registry.cleanupLoop()

	// Load persisted processes if persistence file is configured
	if config.PersistenceFile != "" {
		registry.loadFromDisk()
	}

	return registry
}

// StartProcess starts a new process with the given parameters
func (r *Registry) StartProcess(ctx context.Context, command, shell string, args []string, env map[string]string, workingDir string, captureOutput bool) (*Process, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Check if we've hit the process limit
	if len(r.processes) >= r.config.MaxProcesses {
		return nil, ErrRegistryFull
	}

	// Generate unique ID
	id := uuid.New().String()

	// Default shell to bash if not specified
	if shell == "" {
		shell = "bash"
	}

	// Create the process structure
	proc := &Process{
		ID:          id,
		Command:     command,
		Args:        args,
		Shell:       shell,
		StartTime:   time.Now(),
		State:       ProcessStateRunning,
		Environment: env,
		WorkingDir:  workingDir,
		outputBuf:   NewOutputBuffer(r.config.OutputBufferSize),
		signalChan:  make(chan os.Signal, 1),
	}

	// Set up the command
	var cmd *exec.Cmd
	if len(args) > 0 {
		// Direct command execution
		cmd = exec.CommandContext(ctx, command, args...)
	} else {
		// Shell command execution
		cmd = exec.CommandContext(ctx, shell, "-c", command)
	}

	// Set environment variables
	if env != nil {
		envList := os.Environ()
		for k, v := range env {
			envList = append(envList, fmt.Sprintf("%s=%s", k, v))
		}
		cmd.Env = envList
	}

	// Set working directory
	if workingDir != "" {
		cmd.Dir = workingDir
	}

	// Set up I/O pipes
	var err error
	proc.stdin, err = cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if captureOutput {
		proc.stdout, err = cmd.StdoutPipe()
		if err != nil {
			proc.stdin.Close()
			return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
		}

		proc.stderr, err = cmd.StderrPipe()
		if err != nil {
			proc.stdin.Close()
			proc.stdout.Close()
			return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
		}
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		proc.Cleanup()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Set process information
	proc.process = cmd.Process
	proc.PID = cmd.Process.Pid

	// Add to registry
	r.processes[id] = proc

	// Start output capture goroutines if capturing output
	if captureOutput && proc.stdout != nil && proc.stderr != nil {
		go r.captureOutput(proc, proc.stdout, true)
		go r.captureOutput(proc, proc.stderr, false)
	}

	// Start process monitoring goroutine
	go r.monitorProcess(proc, cmd)

	return proc, nil
}

// captureOutput reads from a process stream and writes to the output buffer
func (r *Registry) captureOutput(proc *Process, reader io.ReadCloser, isStdout bool) {
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Bytes()
		line = append(line, '\n') // Add newline back

		if isStdout {
			proc.outputBuf.WriteStdout(line)
		} else {
			proc.outputBuf.WriteStderr(line)
		}
	}
}

// monitorProcess monitors a process and updates its state when it completes
func (r *Registry) monitorProcess(proc *Process, cmd *exec.Cmd) {
	err := cmd.Wait()

	// Update process state
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			proc.SetExitCode(exitErr.ExitCode())
		}
	} else {
		proc.SetExitCode(0)
	}

	proc.SetState(ProcessStateTerminated)

	// Save to disk if persistence is enabled
	if r.config.PersistenceFile != "" {
		r.saveToDisk()
	}
}

// GetProcess retrieves a process by ID
func (r *Registry) GetProcess(id string) (*Process, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	proc, exists := r.processes[id]
	if !exists {
		return nil, ErrProcessNotFound
	}

	return proc, nil
}

// ListProcesses returns a list of processes matching the given criteria
func (r *Registry) ListProcesses(state ProcessState, limit, offset int, sortBy string) ([]*Process, int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Filter processes by state
	var filtered []*Process
	for _, proc := range r.processes {
		if state == "" || state == "all" || proc.GetState() == state {
			filtered = append(filtered, proc)
		}
	}

	total := len(filtered)

	// Sort processes (simplified - by start time for now)
	// TODO: Implement proper sorting by different criteria

	// Apply pagination
	start := offset
	if start > len(filtered) {
		start = len(filtered)
	}

	end := start + limit
	if end > len(filtered) {
		end = len(filtered)
	}

	return filtered[start:end], total, nil
}

// TerminateProcess terminates a process gracefully or forcefully
func (r *Registry) TerminateProcess(id string, force bool, gracePeriod time.Duration) error {
	proc, err := r.GetProcess(id)
	if err != nil {
		return err
	}

	if !proc.IsRunning() {
		return ErrProcessAlreadyTerminated
	}

	if force {
		// Force kill the process
		return proc.SendSignal(syscall.SIGKILL)
	}

	// Graceful termination
	if err := proc.SendSignal(syscall.SIGTERM); err != nil {
		return err
	}

	// Wait for graceful termination or force kill after grace period
	if gracePeriod > 0 {
		go func() {
			time.Sleep(gracePeriod)
			if proc.IsRunning() {
				proc.SendSignal(syscall.SIGKILL)
			}
		}()
	}

	return nil
}

// SendSignalToProcess sends a signal to a process
func (r *Registry) SendSignalToProcess(id string, sig os.Signal) error {
	proc, err := r.GetProcess(id)
	if err != nil {
		return err
	}

	return proc.SendSignal(sig)
}

// SendInputToProcess sends input to a process's stdin
func (r *Registry) SendInputToProcess(id string, data []byte) error {
	proc, err := r.GetProcess(id)
	if err != nil {
		return err
	}

	if !proc.IsRunning() {
		return ErrProcessNotRunning
	}

	return proc.WriteToStdin(data)
}

// ReadProcessOutput reads output from a process
func (r *Registry) ReadProcessOutput(id, stream string, position int64, maxBytes int) ([]byte, int64, bool, error) {
	proc, err := r.GetProcess(id)
	if err != nil {
		return nil, 0, false, err
	}

	var data []byte
	var newPos int64
	var hasMore bool

	switch stream {
	case "stdout":
		data, newPos, hasMore = proc.outputBuf.ReadStdout(position, maxBytes)
	case "stderr":
		data, newPos, hasMore = proc.outputBuf.ReadStderr(position, maxBytes)
	default:
		return nil, 0, false, ErrInvalidInput
	}

	return data, newPos, hasMore, nil
}

// GetProcessStatus returns detailed status information for a process
func (r *Registry) GetProcessStatus(id string) (*ProcessStatus, error) {
	proc, err := r.GetProcess(id)
	if err != nil {
		return nil, err
	}

	status := &ProcessStatus{
		ID:        proc.ID,
		PID:       proc.PID,
		Command:   proc.Command,
		Shell:     proc.Shell,
		StartTime: proc.StartTime,
		Runtime:   proc.GetRuntime(),
		State:     proc.GetState(),
		ExitCode:  proc.GetExitCode(),
	}

	// Get system process information if available
	if proc.PID > 0 {
		if sysProc, err := process.NewProcess(int32(proc.PID)); err == nil {
			if cpuPercent, err := sysProc.CPUPercent(); err == nil {
				status.CPUPercent = &cpuPercent
			}
			if memInfo, err := sysProc.MemoryInfo(); err == nil {
				status.MemoryBytes = &memInfo.RSS
			}
		}
	}

	// Get output buffer information
	stdoutSize, stdoutPos := proc.outputBuf.GetStdoutSize()
	stderrSize, stderrPos := proc.outputBuf.GetStderrSize()

	status.OutputInfo = &OutputInfo{
		StdoutSize:     stdoutSize,
		StdoutPosition: stdoutPos,
		StderrSize:     stderrSize,
		StderrPosition: stderrPos,
	}

	return status, nil
}

// cleanupLoop runs periodically to clean up terminated processes
func (r *Registry) cleanupLoop() {
	ticker := time.NewTicker(r.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.cleanupTerminatedProcesses()
		}
	}
}

// cleanupTerminatedProcesses removes terminated processes from the registry
func (r *Registry) cleanupTerminatedProcesses() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, proc := range r.processes {
		if proc.GetState() == ProcessStateTerminated {
			// Clean up process resources
			proc.Cleanup()
			delete(r.processes, id)
		}
	}

	// Save to disk if persistence is enabled
	if r.config.PersistenceFile != "" {
		r.saveToDisk()
	}
}

// saveToDisk saves the current process registry to disk
func (r *Registry) saveToDisk() error {
	if r.config.PersistenceFile == "" {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// Create a serializable representation
	data := make(map[string]interface{})
	for id, proc := range r.processes {
		// Only save essential metadata, not runtime resources
		data[id] = map[string]interface{}{
			"id":          proc.ID,
			"pid":         proc.PID,
			"command":     proc.Command,
			"args":        proc.Args,
			"shell":       proc.Shell,
			"start_time":  proc.StartTime,
			"state":       proc.State,
			"exit_code":   proc.ExitCode,
			"environment": proc.Environment,
			"working_dir": proc.WorkingDir,
		}
	}

	file, err := os.Create(r.config.PersistenceFile)
	if err != nil {
		return err
	}
	defer file.Close()

	return json.NewEncoder(file).Encode(data)
}

// loadFromDisk loads the process registry from disk
func (r *Registry) loadFromDisk() error {
	if r.config.PersistenceFile == "" {
		return nil
	}

	file, err := os.Open(r.config.PersistenceFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // File doesn't exist, that's okay
		}
		return err
	}
	defer file.Close()

	var data map[string]interface{}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for id, procData := range data {
		procMap, ok := procData.(map[string]interface{})
		if !ok {
			continue
		}

		// Reconstruct process metadata (but not running processes)
		proc := &Process{
			ID:         id,
			outputBuf:  NewOutputBuffer(r.config.OutputBufferSize),
			signalChan: make(chan os.Signal, 1),
		}

		if pid, ok := procMap["pid"].(float64); ok {
			proc.PID = int(pid)
		}
		if cmd, ok := procMap["command"].(string); ok {
			proc.Command = cmd
		}
		if shell, ok := procMap["shell"].(string); ok {
			proc.Shell = shell
		}
		if startTime, ok := procMap["start_time"].(string); ok {
			if t, err := time.Parse(time.RFC3339, startTime); err == nil {
				proc.StartTime = t
			}
		}
		if state, ok := procMap["state"].(string); ok {
			proc.State = ProcessState(state)
		}
		if exitCode, ok := procMap["exit_code"].(float64); ok {
			code := int(exitCode)
			proc.ExitCode = &code
		}

		// Processes loaded from disk are considered terminated
		// unless we can verify they're still running
		if proc.State == ProcessStateRunning {
			if sysProc, err := process.NewProcess(int32(proc.PID)); err == nil {
				if exists, err := sysProc.IsRunning(); err != nil || !exists {
					proc.State = ProcessStateTerminated
				}
			} else {
				proc.State = ProcessStateTerminated
			}
		}

		r.processes[id] = proc
	}

	return nil
}

// Shutdown gracefully shuts down the registry
func (r *Registry) Shutdown() error {
	// Cancel the cleanup loop
	r.cancel()

	// Terminate all running processes
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, proc := range r.processes {
		if proc.IsRunning() {
			proc.SendSignal(syscall.SIGTERM)
		}
		proc.Cleanup()
	}

	// Save to disk one last time
	if r.config.PersistenceFile != "" {
		r.saveToDisk()
	}

	return nil
}

// ProcessStatus represents the detailed status of a process
type ProcessStatus struct {
	ID          string        `json:"id"`
	PID         int           `json:"pid"`
	Command     string        `json:"command"`
	Shell       string        `json:"shell"`
	StartTime   time.Time     `json:"start_time"`
	Runtime     time.Duration `json:"runtime"`
	State       ProcessState  `json:"state"`
	ExitCode    *int          `json:"exit_code,omitempty"`
	CPUPercent  *float64      `json:"cpu_percent,omitempty"`
	MemoryBytes *uint64       `json:"memory_bytes,omitempty"`
	OutputInfo  *OutputInfo   `json:"output_info,omitempty"`
}

// OutputInfo represents information about process output buffers
type OutputInfo struct {
	StdoutSize     int   `json:"stdout_size"`
	StdoutPosition int64 `json:"stdout_position"`
	StderrSize     int   `json:"stderr_size"`
	StderrPosition int64 `json:"stderr_position"`
}
