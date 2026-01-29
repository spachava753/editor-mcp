package internal

import (
	"context"
	_ "embed"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TextEditArgs represents the arguments for the text_edit tool
type TextEditArgs struct {
	Path    string `json:"path" jsonschema:"required,Path to the file to edit or create"`
	OldText string `json:"old_text,omitempty" jsonschema:"Exact text to find and replace. If empty, creates a new file instead"`
	Text    string `json:"text" jsonschema:"required,Replacement text or content for new file"`
}

// TextEditFileResult captures the outcome per path
type TextEditFileResult struct {
	Path   string `json:"path"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

// TextEditOutput is the aggregated result
type TextEditOutput struct {
	Results []TextEditFileResult `json:"results,omitempty"`
}

func TextEditTool(ctx context.Context, req *mcp.CallToolRequest, args TextEditArgs) (*mcp.CallToolResult, TextEditOutput, error) {
	// helper to return an MCP error result
	errorRes := func(msg string) (*mcp.CallToolResult, TextEditOutput, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
			IsError: true,
		}, TextEditOutput{Results: []TextEditFileResult{}}, nil
	}

	if args.Path == "" {
		return errorRes("path is required")
	}
	if args.Text == "" {
		return errorRes("text is required")
	}

	if ctx.Err() != nil {
		return nil, TextEditOutput{}, ctx.Err()
	}

	res := TextEditFileResult{Path: args.Path}

	// If old_text is empty, create a new file
	if args.OldText == "" {
		// Check that the file doesn't already exist
		if _, err := os.Stat(args.Path); err == nil {
			res.Status = "error"
			res.Error = "file already exists; use old_text to edit existing files"
			out := TextEditOutput{Results: []TextEditFileResult{res}}
			content, _ := json.Marshal(out)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
				IsError: true,
			}, out, nil
		}

		// Create parent directories if they don't exist
		dir := filepath.Dir(args.Path)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			res.Status = "error"
			res.Error = err.Error()
			out := TextEditOutput{Results: []TextEditFileResult{res}}
			content, _ := json.Marshal(out)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
				IsError: true,
			}, out, nil
		}

		// Create the new file
		if err := os.WriteFile(args.Path, []byte(args.Text), 0o644); err != nil {
			res.Status = "error"
			res.Error = err.Error()
			out := TextEditOutput{Results: []TextEditFileResult{res}}
			content, _ := json.Marshal(out)
			return &mcp.CallToolResult{
				Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
				IsError: true,
			}, out, nil
		}

		res.Status = "created"
		out := TextEditOutput{Results: []TextEditFileResult{res}}
		content, _ := json.Marshal(out)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
			IsError: false,
		}, out, nil
	}

	// Editing existing file - check that file exists
	info, err := os.Stat(args.Path)
	if err != nil || !info.Mode().IsRegular() {
		res.Status = "error"
		res.Error = "file path does not exist"
		out := TextEditOutput{Results: []TextEditFileResult{res}}
		content, _ := json.Marshal(out)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
			IsError: true,
		}, out, nil
	}

	b, err := os.ReadFile(args.Path)
	if err != nil {
		res.Status = "error"
		res.Error = err.Error()
		out := TextEditOutput{Results: []TextEditFileResult{res}}
		content, _ := json.Marshal(out)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
			IsError: true,
		}, out, nil
	}

	s := string(b)
	if !strings.Contains(s, args.OldText) {
		res.Status = "error"
		res.Error = "no matches found"
		out := TextEditOutput{Results: []TextEditFileResult{res}}
		content, _ := json.Marshal(out)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
			IsError: true,
		}, out, nil
	}

	// Check if there are multiple matches - throw error if so
	if strings.Count(s, args.OldText) > 1 {
		res.Status = "error"
		res.Error = "multiple matches found"
		out := TextEditOutput{Results: []TextEditFileResult{res}}
		content, _ := json.Marshal(out)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
			IsError: true,
		}, out, nil
	}

	// Replace the single occurrence
	idx := strings.Index(s, args.OldText)
	newS := s[:idx] + args.Text + s[idx+len(args.OldText):]

	if err := os.WriteFile(args.Path, []byte(newS), info.Mode().Perm()); err != nil {
		res.Status = "error"
		res.Error = err.Error()
		out := TextEditOutput{Results: []TextEditFileResult{res}}
		content, _ := json.Marshal(out)
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: string(content)}},
			IsError: true,
		}, out, nil
	}

	res.Status = "modified"
	out := TextEditOutput{Results: []TextEditFileResult{res}}
	content, _ := json.Marshal(out)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
		IsError: false,
	}, out, nil
}

func anyError(results []TextEditFileResult) bool {
	for _, r := range results {
		if r.Status == "error" {
			return true
		}
	}
	return false
}

//go:embed text_edit_desc.md
var textEditToolDesc string

var TextEditToolDef = mcp.Tool{
	Name:        "text_edit",
	Description: textEditToolDesc,
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: ptr(true),
		Title:           "Text Edit",
	},
}
