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
		sc := result.StructuredContent.(map[string]interface{})
		be.True(t, strings.Contains(sc["stdout"].(string), "hello world"))
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
		sc := result.StructuredContent.(map[string]interface{})
		be.Equal(t, sc["exit_code"].(float64), 127)
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
		sc := result.StructuredContent.(map[string]interface{})
		be.True(t, sc["pid"].(float64) > 0)
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
		sc := result.StructuredContent.(map[string]interface{})
		be.True(t, strings.Contains(sc["stdout"].(string), "hello world"))
	})

	t.Run("timeout_success", func(t *testing.T) {
		result, callToolErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Meta: nil,
			Name: ExecuteShellTool.Name,
			Arguments: map[string]any{
				"command": "echo 'hello world'",
				"timeout": 5,
			},
		})
		be.Err(t, callToolErr, nil)
		sc := result.StructuredContent.(map[string]interface{})
		be.True(t, strings.Contains(sc["stdout"].(string), "hello world"))
		be.True(t, !sc["timed_out"].(bool))
	})

	t.Run("timeout_expired", func(t *testing.T) {
		result, callToolErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Meta: nil,
			Name: ExecuteShellTool.Name,
			Arguments: map[string]any{
				"command": "sleep 2",
				"timeout": 1,
			},
		})
		be.Err(t, callToolErr, nil)
		sc := result.StructuredContent.(map[string]interface{})
		be.True(t, sc["timed_out"].(bool))
	})

	t.Run("negative_timeout_error", func(t *testing.T) {
		result, callToolErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Meta: nil,
			Name: ExecuteShellTool.Name,
			Arguments: map[string]any{
				"command": "echo 'hello'",
				"timeout": -1,
			},
		})
		be.Err(t, callToolErr, nil)
		be.True(t, result.IsError)
		be.True(t, len(result.Content) == 1)
		tc := result.Content[0].(*mcp.TextContent)
		be.Equal(t, tc.Text, "timeout cannot be negative")
	})

	t.Run("async_timeout_error", func(t *testing.T) {
		result, callToolErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Meta: nil,
			Name: ExecuteShellTool.Name,
			Arguments: map[string]any{
				"command": "sleep 30 && echo 'hello world'",
				"async":   true,
				"timeout": 10,
			},
		})
		be.Err(t, callToolErr, nil)
		be.True(t, result.IsError)
		be.True(t, len(result.Content) == 1)
		tc := result.Content[0].(*mcp.TextContent)
		be.Equal(t, tc.Text, "timeout is not supported for asynchronous execution")
	})
}
