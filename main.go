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

type HiArgs struct {
	Name string `json:"name" jsonschema:"the name to say hi to"`
}

func SayHi(ctx context.Context, ss *mcp.ServerSession, params *mcp.CallToolParamsFor[HiArgs]) (*mcp.CallToolResultFor[struct{}], error) {
	return &mcp.CallToolResultFor[struct{}]{
		Content: []mcp.Content{
			&mcp.TextContent{Text: "Hi " + params.Arguments.Name},
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
	mcp.AddTool(server, &mcp.Tool{Name: "greet", Description: "say hi"}, SayHi)
	t := mcp.NewLoggingTransport(mcp.NewStdioTransport(), os.Stderr)
	if err := server.Run(context.Background(), t); err != nil {
		log.Printf("Server failed: %v", err)
	}
}
