package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/nalgeon/be"
)

func TestTextEdit(t *testing.T) {
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	server := GetServer("test")
	serverSession, err := server.Connect(t.Context(), serverTransport, nil)
	be.Err(t, err, nil)
	_ = serverSession

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "na"}, nil)
	clientSession, err := client.Connect(t.Context(), clientTransport, nil)
	be.Err(t, err, nil)

	t.Run("create_success", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "file.txt")
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command": "create",
				"paths":   []string{p},
				"text":    "hello",
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, !result.IsError)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read created file: %v", err)
		}
		be.Equal(t, string(b), "hello")
	})

	t.Run("create_invalid_dir", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "nope", "file.txt")
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command": "create",
				"paths":   []string{p},
				"text":    "hello",
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, result.IsError)
		be.True(t, len(result.Content) == 1)
		tc := result.Content[0].(*mcp.TextContent)
		be.True(t, strings.Contains(tc.Text, "file path does not exist"))
	})

	t.Run("insert_success", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "a.txt")
		if err := os.WriteFile(p, []byte("line1\nline2\n"), 0o644); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command":      "insert",
				"paths":        []string{p},
				"text":         "X\n",
				"insert_after": 1,
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, !result.IsError)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read modified file: %v", err)
		}
		be.Equal(t, string(b), "line1\nX\nline2\n")
	})

	t.Run("insert_invalid_combos", func(t *testing.T) {
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command":      "insert",
				"paths":        []string{"foo.txt"},
				"text":         "X",
				"insert_after": 0,
				"replace_all":  true,
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, result.IsError)
		be.True(t, len(result.Content) == 1)
		tc := result.Content[0].(*mcp.TextContent)
		be.True(t, strings.Contains(tc.Text, "replace_all is only valid"))
	})

	t.Run("str_replace_single", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "r.txt")
		if err := os.WriteFile(p, []byte("abc def"), 0o644); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command":     "str_replace",
				"paths":       []string{p},
				"old_text":    "abc",
				"text":        "Z",
				"replace_all": false,
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, !result.IsError)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read modified file: %v", err)
		}
		be.Equal(t, string(b), "Z def")
	})

	t.Run("str_replace_all", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "r2.txt")
		if err := os.WriteFile(p, []byte("abc abc"), 0o644); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command":     "str_replace",
				"paths":       []string{p},
				"old_text":    "abc",
				"text":        "Z",
				"replace_all": true,
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, !result.IsError)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read modified file: %v", err)
		}
		be.Equal(t, string(b), "Z Z")
	})

	t.Run("str_replace_no_matches_error", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "x.txt")
		if err := os.WriteFile(p, []byte("abc"), 0o644); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command":     "str_replace",
				"paths":       []string{p},
				"old_text":    "zzz",
				"text":        "Z",
				"replace_all": true,
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, result.IsError)
	})

	t.Run("str_replace_multiple_without_replace_all_error", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "y.txt")
		if err := os.WriteFile(p, []byte("abc abc"), 0o644); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command":  "str_replace",
				"paths":    []string{p},
				"old_text": "abc",
				"text":     "Z",
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, result.IsError)
	})

	t.Run("batch_paths", func(t *testing.T) {
		dir := t.TempDir()
		p1 := filepath.Join(dir, "a.txt")
		p2 := filepath.Join(dir, "b.txt")
		if err := os.WriteFile(p1, []byte("foo"), 0o644); err != nil {
			t.Fatalf("seed file1: %v", err)
		}
		if err := os.WriteFile(p2, []byte("foo"), 0o644); err != nil {
			t.Fatalf("seed file2: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command":     "str_replace",
				"paths":       []string{p1, p2},
				"old_text":    "foo",
				"text":        "bar",
				"replace_all": true,
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, !result.IsError)
		b1, err := os.ReadFile(p1)
		if err != nil {
			t.Fatalf("read file1: %v", err)
		}
		b2, err := os.ReadFile(p2)
		if err != nil {
			t.Fatalf("read file2: %v", err)
		}
		be.Equal(t, string(b1), "bar")
		be.Equal(t, string(b2), "bar")
	})

	t.Run("invalid_command", func(t *testing.T) {
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command": "nope",
				"paths":   []string{"x"},
				"text":    "y",
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, result.IsError)
	})

	t.Run("view_text_file", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "test.txt")
		content := "Hello, World!"
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command": "view",
				"paths":   []string{p},
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, !result.IsError)
		be.True(t, len(result.Content) == 1)
		tc, ok := result.Content[0].(*mcp.TextContent)
		be.True(t, ok)
		be.Equal(t, tc.Text, content)
	})

	t.Run("view_multiple_files", func(t *testing.T) {
		dir := t.TempDir()
		p1 := filepath.Join(dir, "a.txt")
		p2 := filepath.Join(dir, "b.txt")
		if err := os.WriteFile(p1, []byte("first"), 0o644); err != nil {
			t.Fatalf("seed file1: %v", err)
		}
		if err := os.WriteFile(p2, []byte("second"), 0o644); err != nil {
			t.Fatalf("seed file2: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command": "view",
				"paths":   []string{p1, p2},
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, !result.IsError)
		be.True(t, len(result.Content) == 2)
		tc1, ok := result.Content[0].(*mcp.TextContent)
		be.True(t, ok)
		be.Equal(t, tc1.Text, "first")
		tc2, ok := result.Content[1].(*mcp.TextContent)
		be.True(t, ok)
		be.Equal(t, tc2.Text, "second")
	})

	t.Run("view_nonexistent_file", func(t *testing.T) {
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"command": "view",
				"paths":   []string{"/nonexistent/path"},
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, result.IsError)
	})
}
