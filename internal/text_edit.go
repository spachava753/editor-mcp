package internal

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TextEditArgs represents the arguments for the text_edit tool
type TextEditArgs struct {
	Command     string   `json:"command" jsonschema:"Text file operation to do, must be one of 'str_replace', 'insert', 'create', or 'view'"`
	Paths       []string `json:"paths" jsonschema:"Paths to files. Required for all commands. Batch edits across files. Must be an array of filepath strings"`
	OldText     string   `json:"old_text,omitempty" jsonschema:"Exact text to match in a text file to replace with 'text' parameter. Only required for command 'str_replace'"`
	ReplaceAll  bool     `json:"replace_all,omitempty" jsonschema:"Whether to replace all matches, or just one. Optional, only valid for the command 'str_replace'. If set to true, then every match of the 'old_text' will be replaced with the supplied 'text'"`
	InsertAfter int      `json:"insert_after,omitempty" jsonschema:"The file line after we should insert the given text. Required for the command 'insert'"`
	Text        string   `json:"text,omitempty" jsonschema:"Text payload. Required for create, str_replace, and insert commands"`
}

// TextEditFileResult captures the outcome per path
type TextEditFileResult struct {
	Path         string `json:"path"`
	Status       string `json:"status"`
	Replacements int    `json:"replacements,omitempty"`
	Inserted     bool   `json:"inserted,omitempty"`
	BytesWritten int    `json:"bytes_written,omitempty"`
	Error        string `json:"error,omitempty"`
}

// TextEditOutput is the aggregated result
type TextEditOutput struct {
	Results []TextEditFileResult `json:"results,omitempty"`
}

// handleViewCommand processes the view command and returns appropriate content types
func handleViewCommand(ctx context.Context, args TextEditArgs) (*mcp.CallToolResult, TextEditOutput, error) {
	errorRes := func(msg string) (*mcp.CallToolResult, TextEditOutput, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
			IsError: true,
		}, TextEditOutput{Results: []TextEditFileResult{}}, nil
	}

	if len(args.Paths) == 0 {
		return errorRes("paths is required")
	}

	var contents []mcp.Content
	var results []TextEditFileResult
	hasError := false

	for _, p := range args.Paths {
		if ctx.Err() != nil {
			return nil, TextEditOutput{}, ctx.Err()
		}

		res := TextEditFileResult{Path: p}

		// Check if file exists and is regular
		info, err := os.Stat(p)
		if err != nil {
			contents = append(contents, &mcp.TextContent{
				Text: fmt.Sprintf("Error reading %s: %s", p, err.Error()),
			})
			res.Status = "error"
			res.Error = err.Error()
			results = append(results, res)
			hasError = true
			continue
		}
		if !info.Mode().IsRegular() {
			contents = append(contents, &mcp.TextContent{
				Text: fmt.Sprintf("Error reading %s: not a regular file", p),
			})
			res.Status = "error"
			res.Error = "not a regular file"
			results = append(results, res)
			hasError = true
			continue
		}

		// Read file content
		data, err := os.ReadFile(p)
		if err != nil {
			contents = append(contents, &mcp.TextContent{
				Text: fmt.Sprintf("Error reading %s: %s", p, err.Error()),
			})
			res.Status = "error"
			res.Error = err.Error()
			results = append(results, res)
			hasError = true
			continue
		}

		// Determine MIME type from file extension
		mimeType := mime.TypeByExtension(filepath.Ext(p))
		if mimeType == "" {
			// Default to text/plain for unknown types
			mimeType = "text/plain"
		}

		// Split MIME type to get the primary type
		primaryType := strings.Split(mimeType, "/")[0]

		switch primaryType {
		case "image":
			contents = append(contents, &mcp.ImageContent{
				Data:     []byte(base64.StdEncoding.EncodeToString(data)),
				MIMEType: mimeType,
			})
		case "audio":
			contents = append(contents, &mcp.AudioContent{
				Data:     []byte(base64.StdEncoding.EncodeToString(data)),
				MIMEType: mimeType,
			})
		default:
			// Treat as text
			contents = append(contents, &mcp.TextContent{
				Text: string(data),
			})
		}

		res.Status = "viewed"
		results = append(results, res)
	}

	output := TextEditOutput{
		Results: results,
	}

	structuredOutputContent, err := json.Marshal(output)
	if err != nil {
		return nil, output, err
	}

	contents = append(contents, &mcp.TextContent{
		Text: string(structuredOutputContent),
	})

	return &mcp.CallToolResult{
			Content: contents,
			IsError: hasError,
		}, TextEditOutput{
			Results: results,
		}, nil
}

func TextEditTool(ctx context.Context, req *mcp.CallToolRequest, args TextEditArgs) (*mcp.CallToolResult, TextEditOutput, error) {

	// helper to return an MCP error result
	errorRes := func(msg string) (*mcp.CallToolResult, TextEditOutput, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
			IsError: true,
		}, TextEditOutput{Results: []TextEditFileResult{}}, nil
	}

	// Handle view command separately since it returns different content types
	cmd := strings.ToLower(args.Command)

	if args.Command == "" {
		return errorRes("command cannot be empty")
	}
	if len(args.Paths) == 0 {
		return errorRes("paths is required")
	}

	switch cmd {
	case "str_replace":
		if args.OldText == "" {
			return errorRes("old_text is required for command 'str_replace'")
		}
		if args.InsertAfter > 0 {
			return errorRes("insert_after is only valid for command 'insert'")
		}
		if args.Text == "" {
			return errorRes("text is required")
		}
	case "insert":
		if args.ReplaceAll {
			return errorRes("replace_all is only valid for command 'str_replace'")
		}
		if args.OldText != "" {
			return errorRes("old_text is only valid for command 'str_replace'")
		}
		if args.Text == "" {
			return errorRes("text is required")
		}
	case "create":
		if args.ReplaceAll {
			return errorRes("replace_all is only valid for command 'str_replace'")
		}
		if args.InsertAfter > 0 {
			return errorRes("insert_after is only valid for command 'insert'")
		}
		if args.OldText != "" {
			return errorRes("old_text is only valid for command 'str_replace'")
		}
		if args.Text == "" {
			return errorRes("text is required")
		}
	case "view":
		if args.ReplaceAll {
			return errorRes("replace_all is only valid for command 'str_replace'")
		}
		if args.InsertAfter > 0 {
			return errorRes("insert_after is only valid for command 'insert'")
		}
		if args.OldText != "" {
			return errorRes("old_text is only valid for command 'str_replace'")
		}
		if args.Text != "" {
			return errorRes("text is not valid for command 'view'")
		}
	default:
		return errorRes(fmt.Sprintf("invalid command: %s", args.Command))
	}

	if cmd == "view" {
		return handleViewCommand(ctx, args)
	}

	results := make([]TextEditFileResult, 0, len(args.Paths))

	for _, p := range args.Paths {
		if ctx.Err() != nil {
			return nil, TextEditOutput{}, ctx.Err()
		}

		res := TextEditFileResult{Path: p}

		switch cmd {
		case "create":
			parent := filepath.Dir(p)
			if info, err := os.Stat(parent); err != nil || !info.IsDir() {
				res.Status = "error"
				res.Error = "file path does not exist"
				results = append(results, res)
				continue
			}
			if err := os.WriteFile(p, []byte(args.Text), 0o644); err != nil {
				res.Status = "error"
				res.Error = err.Error()
			} else {
				res.Status = "created"
				res.BytesWritten = len(args.Text)
			}
			results = append(results, res)

		case "insert":
			info, err := os.Stat(p)
			if err != nil || !info.Mode().IsRegular() {
				res.Status = "error"
				res.Error = "file path does not exist"
				results = append(results, res)
				continue
			}
			b, err := os.ReadFile(p)
			if err != nil {
				res.Status = "error"
				res.Error = err.Error()
				results = append(results, res)
				continue
			}
			lines := strings.SplitAfter(string(b), "\n")
			idx := min(max(args.InsertAfter, 0), len(lines))
			newContent := strings.Builder{}
			for i := 0; i < idx && i < len(lines); i++ {
				newContent.WriteString(lines[i])
			}
			newContent.WriteString(args.Text)
			for i := idx; i < len(lines); i++ {
				newContent.WriteString(lines[i])
			}
			if err := os.WriteFile(p, []byte(newContent.String()), info.Mode().Perm()); err != nil {
				res.Status = "error"
				res.Error = err.Error()
			} else {
				res.Status = "modified"
				res.Inserted = true
				res.BytesWritten = len(args.Text)
			}
			results = append(results, res)

		case "str_replace":
			info, err := os.Stat(p)
			if err != nil || !info.Mode().IsRegular() {
				res.Status = "error"
				res.Error = "file path does not exist"
				results = append(results, res)
				continue
			}
			b, err := os.ReadFile(p)
			if err != nil {
				res.Status = "error"
				res.Error = err.Error()
				results = append(results, res)
				continue
			}
			s := string(b)
			if !strings.Contains(s, args.OldText) {
				res.Status = "error"
				res.Error = "no matches found"
				results = append(results, res)
				continue
			}

			replacements := 0
			var newS string
			replaceAll := args.ReplaceAll

			if replaceAll {
				// Count how many matches there are
				replacements = strings.Count(s, args.OldText)
				newS = strings.ReplaceAll(s, args.OldText, args.Text)
			} else {
				// Check if there are multiple matches when replace_all=false
				if strings.Count(s, args.OldText) > 1 {
					res.Status = "error"
					res.Error = "multiple matches found with replace_all=false"
					results = append(results, res)
					continue
				}
				// Replace only the first occurrence
				idx := strings.Index(s, args.OldText)
				newS = s[:idx] + args.Text + s[idx+len(args.OldText):]
				replacements = 1
			}
			if err := os.WriteFile(p, []byte(newS), info.Mode().Perm()); err != nil {
				res.Status = "error"
				res.Error = err.Error()
			} else {
				res.Status = "modified"
				res.Replacements = replacements
				res.BytesWritten = len(newS) - len(s)
			}
			results = append(results, res)
		}
	}

	out := TextEditOutput{Results: results}
	content, _ := json.Marshal(out)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(content)},
		},
		IsError: anyError(results),
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
