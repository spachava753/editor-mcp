package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TextEditArgs represents the arguments for the text_edit tool
type TextEditArgs struct {
	Command     string   `json:"command" jsonschema:"Text file operation to do"`
	Paths       []string `json:"paths" jsonschema:"Paths to files. Required for all commands. Batch edits across files"`
	OldText     string   `json:"old_text,omitempty" jsonschema:"Regex to match in a text file to replace text with 'text' parameter. Only required for command 'str_replace'"`
	ReplaceAll  bool     `json:"replace_all,omitempty" jsonschema:"Whether to replace all matches, or just one. Optional, only valid for the command 'str_replace'. If set to true, then every match of the 'old_text' will be replaced with the supplied 'text'"`
	InsertAfter int      `json:"insert_after,omitempty" jsonschema:"The file line after we should insert the given text. Required for the command 'insert'"`
	Text        string   `json:"text" jsonschema:"Text payload"`
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
	Results []TextEditFileResult `json:"results"`
}

func TextEditTool(ctx context.Context, req *mcp.CallToolRequest, args TextEditArgs) (*mcp.CallToolResult, TextEditOutput, error) {

	// helper to return an MCP error result
	errorRes := func(msg string) (*mcp.CallToolResult, TextEditOutput, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: msg}},
			IsError: true,
		}, TextEditOutput{}, nil
	}

	if args.Command == "" {
		return errorRes("command cannot be empty")
	}
	if len(args.Paths) == 0 {
		return errorRes("paths is required")
	}
	if args.Text == "" {
		return errorRes("text is required")
	}

	cmd := strings.ToLower(args.Command)
	switch cmd {
	case "str_replace":
		if args.OldText == "" {
			return errorRes("old_text is required for command 'str_replace'")
		}
		if args.InsertAfter > 0 {
			return errorRes("insert_after is only valid for command 'insert'")
		}
		if _, err := regexp.Compile(args.OldText); err != nil {
			return errorRes(fmt.Sprintf("invalid regex: %v", err))
		}
	case "insert":
		if args.ReplaceAll {
			return errorRes("replace_all is only valid for command 'str_replace'")
		}
		if args.OldText != "" {
			return errorRes("old_text is only valid for command 'str_replace'")
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
	default:
		return errorRes(fmt.Sprintf("invalid command: %s", args.Command))
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
			idx := args.InsertAfter
			if idx < 0 {
				idx = 0
			}
			if idx > len(lines) {
				idx = len(lines)
			}
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
			re, err := regexp.Compile(args.OldText)
			if err != nil {
				// Should not happen due to earlier validation, but keep for safety
				return errorRes(fmt.Sprintf("invalid regex: %v", err))
			}
			s := string(b)
			matches := re.FindAllStringIndex(s, -1)
			if len(matches) == 0 {
				res.Status = "error"
				res.Error = "no matches found"
				results = append(results, res)
				continue
			}
			replaceAll := args.ReplaceAll
			if !replaceAll && len(matches) > 1 {
				res.Status = "error"
				res.Error = "multiple matches found with replace_all=false"
				results = append(results, res)
				continue
			}
			replacements := 0
			var newS string
			if replaceAll {
				newS = re.ReplaceAllString(s, args.Text)
				replacements = len(matches)
			} else {
				loc := matches[0]
				newS = s[:loc[0]] + args.Text + s[loc[1]:]
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

var TextEditToolDef = mcp.Tool{
	Name:        "text_edit",
	Description: "This tool can be used to edit or create text files",
	Annotations: &mcp.ToolAnnotations{
		DestructiveHint: ptr(true),
		Title:           "Text Edit",
	},
}
