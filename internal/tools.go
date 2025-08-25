package internal

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Global registry instance
var globalRegistry *Registry

// InitializeRegistry initializes the global process registry
func InitializeRegistry(config *RegistryConfig) {
	if globalRegistry != nil {
		globalRegistry.Shutdown()
	}
	globalRegistry = NewRegistry(config)
}

// GetRegistry returns the global process registry
func GetRegistry() *Registry {
	if globalRegistry == nil {
		globalRegistry = NewRegistry(nil)
	}
	return globalRegistry
}

// StartProcessTool handles starting new processes
func StartProcessTool(ctx context.Context, req *mcp.CallToolRequest, args StartProcessArgs) (*mcp.CallToolResult, StartProcessOutput, error) {
	if args.Command == "" {
		return nil, StartProcessOutput{}, fmt.Errorf("command cannot be empty")
	}

	registry := GetRegistry()
	proc, err := registry.StartProcess(ctx, args.Command, args.Shell, nil, args.Environment, args.WorkingDir, args.CaptureOutput)
	if err != nil {
		return nil, StartProcessOutput{}, fmt.Errorf("failed to start process: %w", err)
	}

	result := StartProcessOutput{
		ID:      proc.ID,
		PID:     proc.PID,
		Command: proc.Command,
		Shell:   proc.Shell,
	}

	content, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
	}, result, nil
}

// ListProcessesTool handles listing processes
func ListProcessesTool(ctx context.Context, req *mcp.CallToolRequest, args ListProcessesArgs) (*mcp.CallToolResult, ListProcessesOutput, error) {

	// Set defaults
	limit := args.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}

	offset := args.Offset
	if offset < 0 {
		offset = 0
	}

	state := ProcessState(args.State)
	if args.State == "" {
		state = "all"
	}

	registry := GetRegistry()
	processes, total, err := registry.ListProcesses(state, limit, offset, args.SortBy)
	if err != nil {
		return nil, ListProcessesOutput{}, fmt.Errorf("failed to list processes: %w", err)
	}

	// Convert to output format
	processInfos := make([]*ProcessInfo, len(processes))
	for i, proc := range processes {
		processInfos[i] = &ProcessInfo{
			ID:        proc.ID,
			PID:       proc.PID,
			Command:   proc.Command,
			Shell:     proc.Shell,
			StartTime: proc.StartTime.Format(time.RFC3339),
			Runtime:   proc.GetRuntime().String(),
			State:     proc.GetState(),
			ExitCode:  proc.GetExitCode(),
		}
	}

	result := ListProcessesOutput{
		Processes: processInfos,
		Total:     total,
		Limit:     limit,
		Offset:    offset,
	}

	content, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
	}, result, nil
}

// GetProcessStatusTool handles getting detailed process status
func GetProcessStatusTool(ctx context.Context, req *mcp.CallToolRequest, args GetProcessStatusArgs) (*mcp.CallToolResult, ProcessStatus, error) {
	if args.ID == "" {
		return nil, ProcessStatus{}, fmt.Errorf("process ID cannot be empty")
	}

	registry := GetRegistry()
	status, err := registry.GetProcessStatus(args.ID)
	if err != nil {
		return nil, ProcessStatus{}, fmt.Errorf("failed to get process status: %w", err)
	}

	content, _ := json.Marshal(status)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
	}, *status, nil
}

// SendProcessInputTool handles sending input to a process
func SendProcessInputTool(ctx context.Context, req *mcp.CallToolRequest, args SendProcessInputArgs) (*mcp.CallToolResult, SendProcessInputOutput, error) {
	if args.ID == "" {
		return nil, SendProcessInputOutput{}, fmt.Errorf("process ID cannot be empty")
	}
	if args.Input == "" {
		return nil, SendProcessInputOutput{}, fmt.Errorf("input cannot be empty")
	}

	var inputData []byte
	var err error

	if args.Binary {
		// Decode base64 input
		inputData, err = base64.StdEncoding.DecodeString(args.Input)
		if err != nil {
			return nil, SendProcessInputOutput{}, fmt.Errorf("failed to decode base64 input: %w", err)
		}
	} else {
		inputData = []byte(args.Input)
	}

	// Add newline if requested (default true)
	addNewline := true
	if !args.Binary {
		addNewline = args.Newline
	}
	if addNewline && len(inputData) > 0 && inputData[len(inputData)-1] != '\n' {
		inputData = append(inputData, '\n')
	}

	registry := GetRegistry()
	err = registry.SendInputToProcess(args.ID, inputData)
	if err != nil {
		return nil, SendProcessInputOutput{}, fmt.Errorf("failed to send input to process: %w", err)
	}

	result := SendProcessInputOutput{
		BytesSent: len(inputData),
	}

	content, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
	}, result, nil
}

// ReadProcessOutputTool handles reading output from a process
func ReadProcessOutputTool(ctx context.Context, req *mcp.CallToolRequest, args ReadProcessOutputArgs) (*mcp.CallToolResult, ReadProcessOutputOutput, error) {
	if args.ID == "" {
		return nil, ReadProcessOutputOutput{}, fmt.Errorf("process ID cannot be empty")
	}

	// Set defaults
	stream := args.Stream
	if stream == "" {
		stream = "stdout"
	}
	if stream != "stdout" && stream != "stderr" {
		return nil, ReadProcessOutputOutput{}, fmt.Errorf("invalid stream: must be 'stdout' or 'stderr'")
	}

	maxBytes := args.MaxBytes
	if maxBytes <= 0 {
		maxBytes = 1024
	}
	if maxBytes > 1048576 { // 1MB limit
		maxBytes = 1048576
	}

	position := args.Position
	if position < 0 {
		position = 0
	}

	registry := GetRegistry()

	// TODO: Implement blocking reads with timeout if needed
	data, newPos, hasMore, err := registry.ReadProcessOutput(args.ID, stream, position, maxBytes)
	if err != nil {
		return nil, ReadProcessOutputOutput{}, fmt.Errorf("failed to read process output: %w", err)
	}

	result := ReadProcessOutputOutput{
		Data:      string(data),
		Position:  newPos,
		HasMore:   hasMore,
		BytesRead: len(data),
	}

	content, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
	}, result, nil
}

// TerminateProcessTool handles terminating processes
func TerminateProcessTool(ctx context.Context, req *mcp.CallToolRequest, args TerminateProcessArgs) (*mcp.CallToolResult, TerminateProcessOutput, error) {
	if args.ID == "" {
		return nil, TerminateProcessOutput{}, fmt.Errorf("process ID cannot be empty")
	}

	registry := GetRegistry()

	// Check if process exists and is running
	proc, err := registry.GetProcess(args.ID)
	if err != nil {
		return nil, TerminateProcessOutput{}, fmt.Errorf("failed to get process: %w", err)
	}

	wasRunning := proc.IsRunning()

	// Set default grace period
	gracePeriod := time.Duration(args.GracePeriod) * time.Second
	if args.GracePeriod == 0 {
		gracePeriod = 5 * time.Second
	}

	// Terminate the process
	err = registry.TerminateProcess(args.ID, args.Force, gracePeriod)
	if err != nil && !errors.Is(err, ErrProcessAlreadyTerminated) {
		return nil, TerminateProcessOutput{}, fmt.Errorf("failed to terminate process: %w", err)
	}

	method := "graceful"
	if args.Force {
		method = "forced"
	}

	result := TerminateProcessOutput{
		ID:         args.ID,
		Terminated: !errors.Is(err, ErrProcessAlreadyTerminated),
		WasRunning: wasRunning,
		Method:     method,
	}

	content, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
	}, result, nil
}

// SendSignalTool handles sending signals to processes
func SendSignalTool(ctx context.Context, req *mcp.CallToolRequest, args SendSignalArgs) (*mcp.CallToolResult, SendSignalOutput, error) {
	if args.ID == "" {
		return nil, SendSignalOutput{}, fmt.Errorf("process ID cannot be empty")
	}
	if args.Signal == "" {
		return nil, SendSignalOutput{}, fmt.Errorf("signal cannot be empty")
	}

	// Parse signal
	var sig os.Signal
	switch strings.ToUpper(args.Signal) {
	case "SIGTERM":
		sig = syscall.SIGTERM
	case "SIGKILL":
		sig = syscall.SIGKILL
	case "SIGINT":
		sig = syscall.SIGINT
	case "SIGSTOP":
		sig = syscall.SIGSTOP
	case "SIGCONT":
		sig = syscall.SIGCONT
	case "SIGUSR1":
		sig = syscall.SIGUSR1
	case "SIGUSR2":
		sig = syscall.SIGUSR2
	default:
		// Try to parse as number
		if num, err := strconv.Atoi(args.Signal); err == nil {
			sig = syscall.Signal(num)
		} else {
			return nil, SendSignalOutput{}, fmt.Errorf("invalid signal: %s", args.Signal)
		}
	}

	registry := GetRegistry()
	err := registry.SendSignalToProcess(args.ID, sig)

	sent := err == nil
	if err != nil && !errors.Is(err, ErrProcessNotFound) && !errors.Is(err, ErrProcessNotRunning) {
		return nil, SendSignalOutput{}, fmt.Errorf("failed to send signal: %w", err)
	}

	result := SendSignalOutput{
		ID:     args.ID,
		Signal: args.Signal,
		Sent:   sent,
	}

	content, _ := json.Marshal(result)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
	}, result, nil
}

// Tool definitions
var StartProcessToolDef = mcp.Tool{
	Name:        "start_process",
	Description: "Start a new process with enhanced async support and process tracking",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: ptr(true),
		OpenWorldHint:   ptr(true),
		Title:           "Start Process",
	},
}

var ListProcessesToolDef = mcp.Tool{
	Name:        "list_processes",
	Description: "List all active processes in the registry with filtering and pagination",
	Annotations: &mcp.ToolAnnotations{
		Title: "List Processes",
	},
}

var GetProcessStatusToolDef = mcp.Tool{
	Name:        "get_process_status",
	Description: "Get detailed status information for a specific process including resource usage",
	Annotations: &mcp.ToolAnnotations{
		Title: "Get Process Status",
	},
}

var SendProcessInputToolDef = mcp.Tool{
	Name:        "send_process_input",
	Description: "Send input data to a running process's stdin stream",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: ptr(true),
		Title:           "Send Process Input",
	},
}

var ReadProcessOutputToolDef = mcp.Tool{
	Name:        "read_process_output",
	Description: "Read output data from a process's stdout or stderr stream with position tracking",
	Annotations: &mcp.ToolAnnotations{
		Title: "Read Process Output",
	},
}

var TerminateProcessToolDef = mcp.Tool{
	Name:        "terminate_process",
	Description: "Terminate a running process gracefully or forcefully with configurable grace period",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: ptr(true),
		Title:           "Terminate Process",
	},
}

var SendSignalToolDef = mcp.Tool{
	Name:        "send_signal",
	Description: "Send a Unix signal to a running process for fine-grained process control",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: ptr(true),
		Title:           "Send Signal",
	},
}

func ptr[T any](t T) *T {
	return &t
}
