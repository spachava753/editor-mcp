package internal

import (
	"os"
	"path/filepath"
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

	t.Run("str_replace_success", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "r.txt")
		if err := os.WriteFile(p, []byte("abc def"), 0o644); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"path":     p,
				"old_text": "abc",
				"text":     "Z",
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

	t.Run("str_replace_no_matches_error", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "x.txt")
		if err := os.WriteFile(p, []byte("abc"), 0o644); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"path":     p,
				"old_text": "zzz",
				"text":     "Z",
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, result.IsError)
	})

	t.Run("str_replace_multiple_matches_error", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "y.txt")
		if err := os.WriteFile(p, []byte("abc abc"), 0o644); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"path":     p,
				"old_text": "abc",
				"text":     "Z",
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, result.IsError)
	})

	t.Run("file_not_exist", func(t *testing.T) {
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"path":     "/nonexistent/path",
				"old_text": "abc",
				"text":     "Z",
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, result.IsError)
	})

	t.Run("create_new_file_success", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "newfile.txt")
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"path": p,
				"text": "hello world",
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, !result.IsError)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read created file: %v", err)
		}
		be.Equal(t, string(b), "hello world")
	})

	t.Run("create_file_already_exists_error", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "existing.txt")
		if err := os.WriteFile(p, []byte("existing content"), 0o644); err != nil {
			t.Fatalf("seed file: %v", err)
		}
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"path": p,
				"text": "new content",
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, result.IsError)
		// Verify file wasn't modified
		b, _ := os.ReadFile(p)
		be.Equal(t, string(b), "existing content")
	})

	t.Run("create_file_with_parent_dirs", func(t *testing.T) {
		dir := t.TempDir()
		p := filepath.Join(dir, "subdir", "nested", "newfile.txt")
		result, callErr := clientSession.CallTool(t.Context(), &mcp.CallToolParams{
			Name: TextEditToolDef.Name,
			Arguments: map[string]any{
				"path": p,
				"text": "nested content",
			},
		})
		be.Err(t, callErr, nil)
		be.True(t, !result.IsError)
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read created file: %v", err)
		}
		be.Equal(t, string(b), "nested content")
	})
}
