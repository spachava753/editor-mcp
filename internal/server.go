package internal

import (
	"bytes"
	"context"
	"fmt"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"os/exec"
	"runtime/debug"
)

type ExecuteShellArgs struct {
	Command string `json:"command" jsonschema:"the command to execute"`
	Shell   string `json:"shell" jsonschema:"the shell to use (default bash)"`
	Async   bool   `json:"async" jsonschema:"execute the command asynchronously"`
}

type ExecuteShellOutput struct {
	Pid    string `json:"pid" jsonschema:"the pid of the command"`
	Stderr string `json:"stderr" jsonschema:"the stderr output of the command"`
	Stdout string `json:"stdout" jsonschema:"the stdout output of the command"`
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

	// Handle async execution
	if args.Async {
		// For async, we'll start the command and return immediately
		cmd := exec.Command(shell, "-c", args.Command)

		// Start the command without waiting for completion
		err := cmd.Start()
		if err != nil {
			return nil, fmt.Errorf("failed to start command: %w", err)
		}

		pid := ""
		if cmd.Process != nil {
			pid = fmt.Sprintf("%d", cmd.Process.Pid)
		}

		return &mcp.CallToolResultFor[ExecuteShellOutput]{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: fmt.Sprintf("Command started asynchronously with PID: %s\nNote: Process started but not waited for completion", pid),
				},
			},
		}, nil
	}

	// Execute the command synchronously
	cmd := exec.CommandContext(ctx, shell, "-c", args.Command)

	// Capture stdout and stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()

	// Get the process ID (will be empty if command has already completed)
	pid := ""
	if cmd.Process != nil {
		pid = fmt.Sprintf("%d", cmd.Process.Pid)
	}

	// Return the result
	return &mcp.CallToolResultFor[ExecuteShellOutput]{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: fmt.Sprintf("Command executed with PID: %s\n\nSTDOUT:\n%s\n\nSTDERR:\n%s\n\nExit Error: %v",
					pid, stdout.String(), stderr.String(), err),
			},
		},
	}, nil
}

func GetServer() (*mcp.Server, error) {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, fmt.Errorf("could not read build info")
	}

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "editor-mcp",
		Title:   "Editor MCP",
		Version: bi.Main.Version,
	}, nil)

	mcp.AddTool(server, &mcp.Tool{
		Name:        "shell",
		Description: `A tool to execute command in a shell. A new shell is used for each execution, so any environment variables will be inherited, or must be defined inline before execution of the command`,
		Annotations: &mcp.ToolAnnotations{
			DestructiveHint: ptr(true),
			OpenWorldHint:   ptr(true),
			Title:           "Shell",
		},
	}, ExecuteShell)

	return server, nil
}

func ptr[T any](t T) *T {
	return &t
}
