package internal

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nalgeon/be"
	"strings"
	"testing"
)

func TestServer(t *testing.T) {
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	server, err := GetServer()
	be.Err(t, err, nil)

	_, err = server.Connect(t.Context(), serverTransport)
	be.Err(t, err, nil)

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "na",
	}, nil)

	clientSession, err := client.Connect(t.Context(), clientTransport)
	be.Err(t, err, nil)

	result, err := clientSession.ListTools(t.Context(), nil)
	be.Err(t, err, nil)
	be.True(t, len(result.Tools) > 0)
}

func TestExecuteShell(t *testing.T) {
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	server, err := GetServer()
	be.Err(t, err, nil)

	serverSession, err := server.Connect(t.Context(), serverTransport)
	be.Err(t, err, nil)
	_ = serverSession

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "na",
	}, nil)

	clientSession, err := client.Connect(t.Context(), clientTransport)
	be.Err(t, err, nil)

	t.Run("success", func(t *testing.T) {
		result, callToolErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Meta: nil,
			Name: ExecuteShellTool.Name,
			Arguments: map[string]any{
				"command": "echo 'hello world'",
			},
		})
		be.Err(t, callToolErr, nil)
		be.True(t, len(result.Content) == 1)
		tc := result.Content[0].(*mcp.TextContent)
		be.True(t, strings.Contains(tc.Text, "hello world"))
	})

	t.Run("error", func(t *testing.T) {
		result, callToolErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Meta: nil,
			Name: ExecuteShellTool.Name,
			Arguments: map[string]any{
				"command": "ech 'hello world'",
			},
		})
		be.Err(t, callToolErr, nil)
		be.True(t, len(result.Content) == 1)
		tc := result.Content[0].(*mcp.TextContent)
		be.True(t, strings.Contains(tc.Text, "Exit Error: exit status 127"))
	})

	t.Run("async", func(t *testing.T) {
		result, callToolErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Meta: nil,
			Name: ExecuteShellTool.Name,
			Arguments: map[string]any{
				"command": "sleep 30 && echo 'hello world'",
				"async":   true,
			},
		})
		be.Err(t, callToolErr, nil)
		be.True(t, len(result.Content) == 1)
		tc := result.Content[0].(*mcp.TextContent)
		be.True(t, strings.Contains(tc.Text, "Command started asynchronously"))
	})

	t.Run("zsh", func(t *testing.T) {
		result, callToolErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Meta: nil,
			Name: ExecuteShellTool.Name,
			Arguments: map[string]any{
				"command": "echo 'hello world'",
				"shell":   "zsh",
			},
		})
		be.Err(t, callToolErr, nil)
		be.True(t, len(result.Content) == 1)
		tc := result.Content[0].(*mcp.TextContent)
		be.True(t, strings.Contains(tc.Text, "hello world"))
	})
}
