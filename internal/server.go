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

func ExecuteShell(ctx context.Context, req *mcp.CallToolRequest, args ExecuteShellArgs) (*mcp.CallToolResult, ExecuteShellOutput, error) {
	if args.Command == "" {
		return nil, ExecuteShellOutput{}, fmt.Errorf("command cannot be empty")
	}

	// Default shell to bash if not specified
	shell := args.Shell
	if shell == "" {
		shell = "bash"
	}

	// Handle timeout
	timeout := args.Timeout
	if timeout < 0 {
		return nil, ExecuteShellOutput{}, fmt.Errorf("timeout cannot be negative")
	}

	// Return error if both timeout and async are specified
	if args.Async && timeout > 0 {
		return nil, ExecuteShellOutput{}, fmt.Errorf("timeout is not supported for asynchronous execution")
	}

	if args.Async {
		// For async, we'll start the command and return immediately
		cmd := exec.CommandContext(ctx, shell, "-c", args.Command)

		// Start the command without waiting for completion
		err := cmd.Start()
		if err != nil {
			return nil, ExecuteShellOutput{}, fmt.Errorf("failed to start command: %w", err)
		}

		pid := 0
		if cmd.Process != nil {
			pid = cmd.Process.Pid
		}

		result := ExecuteShellOutput{
			Pid: pid,
		}
		content, err := json.Marshal(result)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: string(content)},
			},
		}, result, nil
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
		return nil, ExecuteShellOutput{}, fmt.Errorf("failed to start command: %w", err)
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

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
		IsError: timedOut || exitCode != 0,
	}, result, nil
}

var ExecuteShellTool = mcp.Tool{
	Name: "shell",
	Description: `Executes shell commands with full system access and captures output. Each invocation runs in a fresh shell instance with inherited environment variables.

EXECUTION MODEL:
- Shell: Uses specified shell (default: bash) to interpret commands
- Environment: Inherits all parent process environment variables
- Working Directory: Executes in the current working directory of the MCP server
- Isolation: Each command runs in a new shell process; no state persists between calls

PARAMETERS:
- command (required): Shell command string to execute, can include pipes, redirects, and shell built-ins
- shell (optional): Shell interpreter to use (e.g., "bash", "sh", "zsh", "fish"); defaults to "bash"
- timeout (optional): Maximum execution time in seconds (0 = no timeout, default)
- async (optional): If true, returns immediately with process ID for background execution

OUTPUT:
- pid: Process ID of the executed command
- stdout: Complete standard output captured as string
- stderr: Complete standard error captured as string  
- exit_code: Process exit code (0 typically indicates success)

CAPABILITIES:
- Full shell syntax: Supports pipes (|), redirects (>, >>), command chaining (&&, ||), subshells
- Environment manipulation: Can set variables inline (VAR=value command)
- Script execution: Can run multi-line scripts using heredocs or semicolons
- Background processes: Use async=true for long-running commands (see process management tools)
- File system access: Full read/write access with user permissions
- Network access: Can make network requests, download files, etc.
- System utilities: Access to all installed CLI tools and system commands

LIMITATIONS & CONSIDERATIONS:
- Security: Commands run with full user privileges; no sandboxing or restrictions
- Output size: Both stdout and stderr are fully captured in memory; very large outputs may cause issues
- Binary output: Binary data in stdout/stderr may not display correctly
- Interactive commands: Tools expecting TTY input (like vim, less) won't work properly
- Timeouts: Only enforced for synchronous execution; async processes continue indefinitely
- Shell differences: Commands may behave differently across shells; test shell-specific features
- Working directory: Cannot change the working directory persistently between calls
- Signal handling: Limited to SIGTERM on timeout; use process management tools for fine control

COMMON PATTERNS:
- File inspection: 'ls -la', 'cat file.txt', 'head -n 20 file.log'
- Search operations: 'grep -r "pattern" .', 'find . -name "*.js"'
- System info: 'uname -a', 'df -h', 'ps aux | grep process'
- Network tasks: 'curl https://api.example.com', 'ping -c 4 google.com'
- Package management: 'npm install', 'pip list', 'go mod download'
- Git operations: 'git status', 'git diff', 'git log --oneline'
- Build tasks: 'make build', 'go test ./...', 'npm run build'
- Complex pipelines: 'cat file | grep pattern | awk "{print $2}" | sort | uniq'

ERROR HANDLING:
- Non-zero exit codes marked as errors but still return output
- Command not found results in exit_code 127 with error in stderr
- Timeout results in process termination and partial output return
- Shell syntax errors captured in stderr with appropriate exit code

IMPORTANT:
- When you want to make modifications to files, prefer to use the 'text_edit' tool. You should not try to use this tool to write to files`,
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

	mcp.AddTool(server, &ExecuteShellTool, ExecuteShell)

	mcp.AddTool(server, &StartProcessToolDef, StartProcessTool)
	mcp.AddTool(server, &ListProcessesToolDef, ListProcessesTool)
	mcp.AddTool(server, &GetProcessStatusToolDef, GetProcessStatusTool)
	mcp.AddTool(server, &SendProcessInputToolDef, SendProcessInputTool)
	mcp.AddTool(server, &ReadProcessOutputToolDef, ReadProcessOutputTool)
	mcp.AddTool(server, &TerminateProcessToolDef, TerminateProcessTool)
	mcp.AddTool(server, &SendSignalToolDef, SendSignalTool)

	mcp.AddTool(server, &TextEditToolDef, TextEditTool)

	return server
}
