package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/modelcontextprotocol/go-sdk/mcp"
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
	// TODO: implement shell execution of a command
	return &mcp.CallToolResultFor[ExecuteShellOutput]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Output: "},
		},
	}, nil
}

func main() {
	flag.Parse()

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		fmt.Println("could not read build info")
		os.Exit(1)
	}

	// create the mcp server
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
	t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), os.Stderr)
	if err := server.Run(context.Background(), t); err != nil {
		log.Printf("Server failed: %v", err)
	}
}

func ptr[T any](t T) *T {
	return &t
}
