# Agent Guide for editor-mcp

@README.md

## Project Overview
Editor MCP is a Model Context Protocol (MCP) server that lets agentic tools execute shell commands and manage background processes in a controlled way. It exposes tools for synchronous shell execution and an asynchronous process registry with I/O streaming, status, and signal control.

## Project structure and organization
- Root
  - `main.go` entrypoint; stdio MCP server and graceful shutdown
  - `README.md` usage, install, security overview
  - `.goreleaser.yml` optional release config
  - `go.mod`, `go.sum`
  - `prompt.md` internal note for future tool work
- `internal/`
  - `server.go` MCP server wiring; tool registration
  - `tools.go` tool handlers and definitions
  - `types.go` request/response schemas
  - `process.go` process model and output buffer
  - `registry.go` async process registry, lifecycle, persistence hooks
  - `errors.go` shared error types
  - `*_test.go` unit tests

## Build, test, and development commands
- Build: `go build -o editor-mcp`
- Run: `go run .`
- Version: `editor-mcp version`
- Tests (all): `go test ./...`
- Tests with race: `go test -race ./...`
- Install (current module): `go install`
- Release (optional, requires goreleaser): `goreleaser build --clean` or `goreleaser release --clean`

## Code style and conventions
- Go 1.24+; format with `gofmt` or `go fmt`, lint with `go vet` (add staticcheck/golangci-lint if desired)
- Package layout follows standard Go patterns; `internal` encapsulates server and tools
- Errors: wrap with `fmt.Errorf("…: %w", err)`; return typed errors from `internal/errors.go`
- JSON I/O: structs in `internal/types.go` define stable tool schemas with `json` tags
- Concurrency: guard mutable state with `sync.RWMutex`; avoid data races; prefer context-aware operations

## Architecture and design patterns
- MCP Server: built with `github.com/modelcontextprotocol/go-sdk`; stdio transport with logging wrapper
- Tools exposed:
  - `shell`: run a command synchronously with optional timeout
  - Process management: `start_process`, `list_processes`, `get_process_status`, `send_process_input`, `read_process_output`, `terminate_process`, `send_signal`
- Process Registry: central manager for background processes
  - Starts commands via shell or direct exec; tracks PID/state/exit code
  - OutputBuffer keeps bounded stdout/stderr with position-based reads
  - Cleanup loop prunes terminated processes; graceful shutdown on server exit
- Configuration object `RegistryConfig` controls limits (buffer size, cleanup interval, max processes, optional persistence file)

## Testing guidelines
- Unit tests live beside code (`*_test.go`)
- Run fast tests locally: `go test ./...`; enable race detector for concurrency code: `go test -race ./...`
- Prefer table-driven tests; validate error cases and boundary conditions (timeouts, invalid IDs, stream positions)
- For process tools, test both running and terminated states; ensure buffers trim correctly and positions advance
- Group related tests using a top-level TestX with subtests via t.Run, mirroring internal/server_test.go style
- Avoid tiny helpers; call t.TempDir() directly in tests
- Do not ignore errors. If an unexpected error occurs (e.g., reading a file we just created), use t.Fatalf. If an operation may return an error but we do not expect it in the test, use t.Errorf

## Security considerations
- Commands run with the invoking user’s privileges; no extra sandboxing beyond OS permissions
- Open-world/destructive operations are possible; clients should confirm intent before invoking risky commands
- Environment variables are inherited unless overridden
- Timeouts and termination are available but not enforced globally; configure or implement as needed
- Output buffers retain up to the configured limit (default 100MB); avoid leaking sensitive data
- Do not expose the server over the network; stdio MCP is intended for trusted local clients

## Configuration
- Requirements: Go 1.24.1+, a compatible MCP client (e.g., CPE, Claude Desktop)
- Registry tuning (code-level): `internal.DefaultRegistryConfig()` sets defaults
  - `MaxProcesses`, `OutputBufferSize`, `CleanupInterval`, `ProcessTimeout`, `PersistenceFile`
  - Override by calling `internal.InitializeRegistry(customConfig)` before `GetServer`
- Releases: see @.goreleaser.yml if using Goreleaser
- MCP client setup: see @README.md for Claude Desktop configuration; CPE users can consult https://github.com/spachava753/cpe
