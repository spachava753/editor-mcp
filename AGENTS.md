# Editor MCP - Agent Guidelines

## Project Overview

Editor MCP is a Model Context Protocol (MCP) server written in Go that provides shell execution and text editing capabilities to AI assistants. It serves as a bridge between AI assistants and the local development environment, enabling controlled command execution and file manipulation.

**Architecture**: Single binary Go application implementing the MCP protocol over stdio transport
**Purpose**: Enable AI assistants to execute shell commands and edit files in a controlled manner
**Language**: Go 1.24.1+

## Project Structure and Organization

```
.
├── main.go                 # Entry point, initializes and runs the MCP server
├── internal/
│   ├── server.go          # MCP server implementation and shell tool handler
│   ├── server_test.go     # Tests for server functionality
│   ├── text_edit.go       # Text file editing tool implementation
│   └── text_edit_test.go  # Tests for text editing functionality
├── go.mod                 # Go module dependencies
├── go.sum                 # Dependency checksums
└── .goreleaser.yml        # Release configuration for cross-platform builds
```

## Build, Test, and Development Commands

### Building
```bash
go build -o editor-mcp
```

### Testing
```bash
go test ./...
```

### Installing globally
```bash
go install github.com/spachava753/editor-mcp
```

### Running locally
```bash
./editor-mcp
```

### Releasing (via GitHub Actions)
```bash
git tag v1.0.0
git push origin v1.0.0
```

### Dependency management
```bash
go mod tidy
go mod download
```

## Code Style and Conventions

### Go Conventions
- Follow standard Go formatting using `gofmt`
- Use meaningful variable and function names
- Keep functions focused and single-purpose
- Error handling: always check and handle errors appropriately
- Use context for cancellation and timeouts

### Package Structure
- Main package in root for the executable
- Internal packages in `internal/` directory for implementation details
- Keep public API surface minimal

### Naming Conventions
- Use camelCase for variables and functions
- Use PascalCase for exported types and functions
- Use descriptive names that indicate purpose

## Architecture and Design Patterns

### MCP Server Implementation
- Uses `github.com/modelcontextprotocol/go-sdk` for MCP protocol handling
- Implements stdio transport for communication with MCP clients
- Provides two main tools: `shell` and `text_edit`

### Tool Design
- **Shell Tool**: Executes commands with configurable shell, timeout, and async options
- **Text Edit Tool**: Provides create, str_replace, and insert operations for file manipulation
- Tools return structured JSON responses with error handling

### Error Handling
- All tool operations return detailed error messages
- Non-zero exit codes from shell commands are captured and reported
- File operations validate paths and report specific failures

## Testing Guidelines

### Unit Testing
- All internal packages have corresponding `_test.go` files
- Use table-driven tests where appropriate
- Mock external dependencies when necessary
- Test error conditions and edge cases

### Running Tests
```bash
go test ./... -v        # Verbose output
go test ./... -cover    # With coverage report
go test -race ./...     # Check for race conditions
```

### Test Coverage
- Aim for >80% coverage for critical paths
- Focus on testing tool implementations and error handling
- Integration tests should verify MCP protocol compliance

## Security Considerations

### Command Execution
- Commands run with the same permissions as the MCP server process
- No additional sandboxing beyond OS permissions
- Timeout mechanisms prevent hanging commands
- Environment variables are inherited from parent process

### File Operations
- Path validation to prevent directory traversal
- UTF-8 text files only (no binary file support)
- File operations are atomic per file
- Parent directories must exist (not auto-created)

### Best Practices
- Never expose the MCP server to untrusted clients
- Run with minimal required permissions
- Audit command execution logs regularly
- Consider using read-only mode for sensitive environments

## Configuration

### Environment Setup
- Requires Go 1.24.1 or higher for building
- Compatible with Linux, macOS, and Windows
- No external runtime dependencies

### MCP Client Integration
- Configure MCP clients to use stdio transport
- Specify full path to editor-mcp binary
- Example configuration for Claude Desktop in @README.md

### Development Environment
- Use `.gitignore` to exclude binaries and temporary files
- GitHub Actions configured for automated releases
- GoReleaser handles cross-platform builds

## Related Documentation

- @README.md - Installation and usage instructions
- @.goreleaser.yml - Release configuration details
- @internal/server.go - Shell tool implementation details
- @internal/text_edit.go - Text editing tool implementation