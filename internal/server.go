package internal

import (
	_ "embed"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func GetServer(version string) *mcp.Server {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "editor-mcp",
		Title:   "Editor MCP",
		Version: version,
	}, &mcp.ServerOptions{
		HasResources: true,
	})

	mcp.AddTool(server, &ExecuteShellTool, ExecuteShell)
	mcp.AddTool(server, &TextEditToolDef, TextEditTool)

	return server
}

func ptr[T any](t T) *T {
	return &t
}
