package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ExecuteShellArgs struct {
	Command string `json:"command" jsonschema:"the command to execute"`
	Shell   string `json:"shell,omitempty" jsonschema:"the shell to use (default bash)"`
	Async   bool   `json:"async,omitempty" jsonschema:"execute the command asynchronously"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"timeout in seconds (default 0 for no timeout)"`
}

type ExecuteShellOutput struct {
	Pid      int    `json:"pid" jsonschema:"the pid of the command"`
	TimedOut bool   `json:"timed_out,omitempty" jsonschema:"if the boolean timed out"`
	ExitCode int    `json:"exit_code" jsonschema:"the exit code"`
	Stderr   string `json:"stderr" jsonschema:"the stderr output of the command"`
	Stdout   string `json:"stdout" jsonschema:"the stdout output of the command"`
}

func ExecuteShell(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[ExecuteShellArgs]) (*mcp.CallToolResultFor[ExecuteShellOutput], error) {
	args := params.Arguments
	if args.Command == "" {
		return nil, fmt.Errorf("command cannot be empty")
	}

	// Default shell to bash if not specified
	shell := args.Shell
	if shell == "" {
		shell = "bash"
	}

	// Handle timeout
	timeout := args.Timeout
	if timeout < 0 {
		return nil, fmt.Errorf("timeout cannot be negative")
	}

	// Return error if both timeout and async are specified
	if args.Async && timeout > 0 {
		return nil, fmt.Errorf("timeout is not supported for asynchronous execution")
	}

	if args.Async {
		// For async, we'll start the command and return immediately
		cmd := exec.CommandContext(ctx, shell, "-c", args.Command)

		// Start the command without waiting for completion
		err := cmd.Start()
		if err != nil {
			return nil, fmt.Errorf("failed to start command: %w", err)
		}

		pid := 0
		if cmd.Process != nil {
			pid = cmd.Process.Pid
		}

		result := ExecuteShellOutput{
			Pid: pid,
		}
		content, err := json.Marshal(result)

		return &mcp.CallToolResultFor[ExecuteShellOutput]{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(content)},
			},
			StructuredContent: result,
		}, nil
	}

	// Execute the command synchronously with proper timeout handling
	cmdCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, shell, "-c", args.Command)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Channel to signal command completion
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	// Wait for command completion or timeout
	var timedOut bool
	var exitCode int
	var cmdErr error

	if timeout > 0 {
		select {
		case cmdErr = <-done:
			// Command completed normally
		case <-time.After(time.Duration(timeout) * time.Second):
			// Timeout occurred - cancel the context and kill the process
			cancel()
			if cmd.Process != nil {
				_ = cmd.Process.Kill() // Ensure process is terminated
			}
			timedOut = true
		}
	} else {
		// No timeout, just wait for completion
		cmdErr = <-done
	}

	if cmdErr != nil {
		var ee *exec.ExitError
		if errors.As(cmdErr, &ee) {
			exitCode = ee.ExitCode()
		}
	}

	// Get the process ID
	pid := 0
	if cmd.Process != nil {
		pid = cmd.Process.Pid
	}

	// Return the result
	result := ExecuteShellOutput{
		Pid:      pid,
		TimedOut: timedOut,
		ExitCode: exitCode,
		Stderr:   stderr.String(),
		Stdout:   stdout.String(),
	}
	content, _ := json.Marshal(result)

	return &mcp.CallToolResultFor[ExecuteShellOutput]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
		IsError:           timedOut || exitCode != 0,
		StructuredContent: result,
	}, nil
}

var ExecuteShellTool = mcp.Tool{
	Name:        "shell",
	Description: `A tool to execute command in a shell. A new shell is used for each execution, so any environment variables will be inherited, or must be defined inline before execution of the command`,
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: ptr(true),
		OpenWorldHint:   ptr(true),
		Title:           "Shell",
	},
}

func GetServer(version string) *mcp.Server {
	// Initialize the process registry with default configuration if not already initialized
	if GetRegistry() == nil {
		InitializeRegistry(nil)
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "editor-mcp",
		Title:   "Editor MCP",
		Version: version,
	}, nil)

	// Add the original shell tool
	mcp.AddTool(server, &ExecuteShellTool, ExecuteShell)

	// Add new process management tools
	mcp.AddTool(server, &StartProcessToolDef, StartProcessTool)
	mcp.AddTool(server, &ListProcessesToolDef, ListProcessesTool)
	mcp.AddTool(server, &GetProcessStatusToolDef, GetProcessStatusTool)
	mcp.AddTool(server, &SendProcessInputToolDef, SendProcessInputTool)
	mcp.AddTool(server, &ReadProcessOutputToolDef, ReadProcessOutputTool)
	mcp.AddTool(server, &TerminateProcessToolDef, TerminateProcessTool)
	mcp.AddTool(server, &SendSignalToolDef, SendSignalTool)

	return server
}
