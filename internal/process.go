package internal

import (
	"io"
	"os"
	"sync"
	"time"
)

// ProcessState represents the current state of a process
type ProcessState string

const (
	ProcessStateRunning    ProcessState = "running"
	ProcessStateStopped    ProcessState = "stopped"
	ProcessStateTerminated ProcessState = "terminated"
	ProcessStateError      ProcessState = "error"
)

// Process represents a managed process with its metadata and I/O streams
type Process struct {
	// Basic process information
	ID          string            `json:"id"`
	PID         int               `json:"pid"`
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Shell       string            `json:"shell"`
	StartTime   time.Time         `json:"start_time"`
	State       ProcessState      `json:"state"`
	ExitCode    *int              `json:"exit_code,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	WorkingDir  string            `json:"working_dir,omitempty"`

	// Process management
	process    *os.Process
	stdin      io.WriteCloser
	stdout     io.ReadCloser
	stderr     io.ReadCloser
	outputBuf  *OutputBuffer
	signalChan chan os.Signal

	// Synchronization
	mu sync.RWMutex
}

// OutputBuffer manages buffered output from a process
type OutputBuffer struct {
	stdout    []byte
	stderr    []byte
	stdoutPos int64
	stderrPos int64
	maxSize   int
	mu        sync.RWMutex
}

// NewOutputBuffer creates a new output buffer with the specified maximum size
func NewOutputBuffer(maxSize int) *OutputBuffer {
	return &OutputBuffer{
		stdout:  make([]byte, 0),
		stderr:  make([]byte, 0),
		maxSize: maxSize,
	}
}

// WriteStdout appends data to stdout buffer
func (ob *OutputBuffer) WriteStdout(data []byte) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.stdout = append(ob.stdout, data...)
	if len(ob.stdout) > ob.maxSize {
		// Trim from beginning to maintain maximum size
		excess := len(ob.stdout) - ob.maxSize
		ob.stdout = ob.stdout[excess:]
		ob.stdoutPos += int64(excess)
	}
}

// WriteStderr appends data to stderr buffer
func (ob *OutputBuffer) WriteStderr(data []byte) {
	ob.mu.Lock()
	defer ob.mu.Unlock()

	ob.stderr = append(ob.stderr, data...)
	if len(ob.stderr) > ob.maxSize {
		// Trim from beginning to maintain maximum size
		excess := len(ob.stderr) - ob.maxSize
		ob.stderr = ob.stderr[excess:]
		ob.stderrPos += int64(excess)
	}
}

// ReadStdout reads stdout data from the specified position
func (ob *OutputBuffer) ReadStdout(position int64, maxBytes int) ([]byte, int64, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if position < ob.stdoutPos {
		// Position is before our buffer start, adjust to buffer start
		position = ob.stdoutPos
	}

	bufferOffset := position - ob.stdoutPos
	if bufferOffset >= int64(len(ob.stdout)) {
		// Position is at or beyond end of buffer
		return nil, position, false
	}

	endOffset := bufferOffset + int64(maxBytes)
	if endOffset > int64(len(ob.stdout)) {
		endOffset = int64(len(ob.stdout))
	}

	data := make([]byte, endOffset-bufferOffset)
	copy(data, ob.stdout[bufferOffset:endOffset])

	newPosition := position + int64(len(data))
	hasMore := newPosition < ob.stdoutPos+int64(len(ob.stdout))

	return data, newPosition, hasMore
}

// ReadStderr reads stderr data from the specified position
func (ob *OutputBuffer) ReadStderr(position int64, maxBytes int) ([]byte, int64, bool) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	if position < ob.stderrPos {
		// Position is before our buffer start, adjust to buffer start
		position = ob.stderrPos
	}

	bufferOffset := position - ob.stderrPos
	if bufferOffset >= int64(len(ob.stderr)) {
		// Position is at or beyond end of buffer
		return nil, position, false
	}

	endOffset := bufferOffset + int64(maxBytes)
	if endOffset > int64(len(ob.stderr)) {
		endOffset = int64(len(ob.stderr))
	}

	data := make([]byte, endOffset-bufferOffset)
	copy(data, ob.stderr[bufferOffset:endOffset])

	newPosition := position + int64(len(data))
	hasMore := newPosition < ob.stderrPos+int64(len(ob.stderr))

	return data, newPosition, hasMore
}

// GetStdoutSize returns the current size and position of stdout buffer
func (ob *OutputBuffer) GetStdoutSize() (size int, position int64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return len(ob.stdout), ob.stdoutPos
}

// GetStderrSize returns the current size and position of stderr buffer
func (ob *OutputBuffer) GetStderrSize() (size int, position int64) {
	ob.mu.RLock()
	defer ob.mu.RUnlock()
	return len(ob.stderr), ob.stderrPos
}

// GetState returns the current state of the process (thread-safe)
func (p *Process) GetState() ProcessState {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.State
}

// SetState updates the state of the process (thread-safe)
func (p *Process) SetState(state ProcessState) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.State = state
}

// GetExitCode returns the exit code if the process has terminated
func (p *Process) GetExitCode() *int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.ExitCode
}

// SetExitCode sets the exit code when the process terminates
func (p *Process) SetExitCode(code int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.ExitCode = &code
}

// GetRuntime returns how long the process has been running
func (p *Process) GetRuntime() time.Duration {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return time.Since(p.StartTime)
}

// IsRunning checks if the process is currently running
func (p *Process) IsRunning() bool {
	state := p.GetState()
	return state == ProcessStateRunning
}

// WriteToStdin writes data to the process's stdin
func (p *Process) WriteToStdin(data []byte) error {
	p.mu.RLock()
	stdin := p.stdin
	p.mu.RUnlock()

	if stdin == nil {
		return ErrProcessNotRunning
	}

	_, err := stdin.Write(data)
	return err
}

// SendSignal sends a signal to the process
func (p *Process) SendSignal(sig os.Signal) error {
	p.mu.RLock()
	process := p.process
	p.mu.RUnlock()

	if process == nil {
		return ErrProcessNotFound
	}

	return process.Signal(sig)
}

// Cleanup closes all resources associated with the process
func (p *Process) Cleanup() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stdin != nil {
		p.stdin.Close()
		p.stdin = nil
	}
	if p.stdout != nil {
		p.stdout.Close()
		p.stdout = nil
	}
	if p.stderr != nil {
		p.stderr.Close()
		p.stderr = nil
	}
	if p.signalChan != nil {
		close(p.signalChan)
		p.signalChan = nil
	}
}
