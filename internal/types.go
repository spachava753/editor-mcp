package internal

// StartProcessArgs represents arguments for starting a new process
type StartProcessArgs struct {
	Command       string            `json:"command" jsonschema:"shell command to execute"`
	Shell         string            `json:"shell,omitempty" jsonschema:"the shell to use (default bash)"`
	CaptureOutput bool              `json:"capture_output,omitempty" jsonschema:"whether to capture stdout/stderr"`
	Environment   map[string]string `json:"environment,omitempty" jsonschema:"environment variables to set"`
	WorkingDir    string            `json:"working_dir,omitempty" jsonschema:"working directory for the process"`
}

// StartProcessOutput represents the result of starting a process
type StartProcessOutput struct {
	ID      string `json:"id" jsonschema:"unique process identifier"`
	PID     int    `json:"pid" jsonschema:"system process ID"`
	Command string `json:"command" jsonschema:"command that was executed"`
	Shell   string `json:"shell" jsonschema:"shell used for execution"`
}

// ListProcessesArgs represents arguments for listing processes
type ListProcessesArgs struct {
	State  string `json:"state,omitempty" jsonschema:"filter processes by state (running, stopped, terminated, all)"`
	Limit  int    `json:"limit,omitempty" jsonschema:"maximum number of processes to return (default 100)"`
	Offset int    `json:"offset,omitempty" jsonschema:"number of processes to skip (default 0)"`
	SortBy string `json:"sort,omitempty" jsonschema:"sort criteria: start_time, pid, command (default start_time)"`
}

// ListProcessesOutput represents the result of listing processes
type ListProcessesOutput struct {
	Processes []*ProcessInfo `json:"processes" jsonschema:"list of processes"`
	Total     int            `json:"total" jsonschema:"total number of matching processes"`
	Limit     int            `json:"limit" jsonschema:"limit applied to results"`
	Offset    int            `json:"offset" jsonschema:"offset applied to results"`
}

// ProcessInfo represents basic information about a process
type ProcessInfo struct {
	ID        string       `json:"id" jsonschema:"unique process identifier"`
	PID       int          `json:"pid" jsonschema:"system process ID"`
	Command   string       `json:"command" jsonschema:"command being executed"`
	Shell     string       `json:"shell" jsonschema:"shell used for execution"`
	StartTime string       `json:"start_time" jsonschema:"when the process started (RFC3339 format)"`
	Runtime   string       `json:"runtime" jsonschema:"how long the process has been running"`
	State     ProcessState `json:"state" jsonschema:"current state of the process"`
	ExitCode  *int         `json:"exit_code,omitempty" jsonschema:"exit code if process has terminated"`
}

// GetProcessStatusArgs represents arguments for getting process status
type GetProcessStatusArgs struct {
	ID string `json:"id" jsonschema:"unique process identifier"`
}

// SendProcessInputArgs represents arguments for sending input to a process
type SendProcessInputArgs struct {
	ID      string `json:"id" jsonschema:"unique process identifier"`
	Input   string `json:"input" jsonschema:"text to send to process stdin"`
	Binary  bool   `json:"binary,omitempty" jsonschema:"whether input is base64-encoded binary data"`
	Newline bool   `json:"newline,omitempty" jsonschema:"whether to append a newline character (default true)"`
}

// SendProcessInputOutput represents the result of sending input
type SendProcessInputOutput struct {
	BytesSent int `json:"bytes_sent" jsonschema:"number of bytes sent to process"`
}

// ReadProcessOutputArgs represents arguments for reading process output
type ReadProcessOutputArgs struct {
	ID       string `json:"id" jsonschema:"unique process identifier"`
	Stream   string `json:"stream,omitempty" jsonschema:"which output stream to read: stdout or stderr (default stdout)"`
	Position int64  `json:"position,omitempty" jsonschema:"position in stream to start reading from"`
	MaxBytes int    `json:"max_bytes,omitempty" jsonschema:"maximum number of bytes to read (default 1024)"`
	Blocking bool   `json:"blocking,omitempty" jsonschema:"whether to block until data is available"`
	Timeout  int    `json:"timeout,omitempty" jsonschema:"timeout in seconds for blocking reads"`
}

// ReadProcessOutputOutput represents the result of reading process output
type ReadProcessOutputOutput struct {
	Data      string `json:"data" jsonschema:"output data from the process"`
	Position  int64  `json:"position" jsonschema:"new position in the stream"`
	HasMore   bool   `json:"has_more" jsonschema:"whether there is more data available"`
	BytesRead int    `json:"bytes_read" jsonschema:"number of bytes actually read"`
}

// TerminateProcessArgs represents arguments for terminating a process
type TerminateProcessArgs struct {
	ID          string `json:"id" jsonschema:"unique process identifier"`
	Force       bool   `json:"force,omitempty" jsonschema:"whether to force kill the process (SIGKILL vs SIGTERM)"`
	GracePeriod int    `json:"grace_period,omitempty" jsonschema:"seconds to wait before force killing (default 5)"`
}

// TerminateProcessOutput represents the result of terminating a process
type TerminateProcessOutput struct {
	ID         string `json:"id" jsonschema:"unique process identifier"`
	Terminated bool   `json:"terminated" jsonschema:"whether the process was successfully terminated"`
	WasRunning bool   `json:"was_running" jsonschema:"whether the process was running before termination"`
	Method     string `json:"method" jsonschema:"termination method used (graceful/forced)"`
}

// SendSignalArgs represents arguments for sending a signal to a process
type SendSignalArgs struct {
	ID     string `json:"id" jsonschema:"unique process identifier"`
	Signal string `json:"signal" jsonschema:"signal to send: SIGTERM, SIGKILL, SIGINT, SIGSTOP, SIGCONT, SIGUSR1, SIGUSR2"`
}

// SendSignalOutput represents the result of sending a signal
type SendSignalOutput struct {
	ID     string `json:"id" jsonschema:"unique process identifier"`
	Signal string `json:"signal" jsonschema:"signal that was sent"`
	Sent   bool   `json:"sent" jsonschema:"whether the signal was successfully sent"`
}
