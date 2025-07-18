# Editor MCP

A Model Context Protocol (MCP) server that provides shell execution capabilities to AI assistants like Claude, enabling safe and controlled command execution within a development environment.

## Purpose

Editor MCP serves as a bridge between AI assistants and your local shell environment. It allows AI assistants to:
- Execute shell commands in a controlled manner
- Access system utilities and development tools
- Interact with files and directories through command-line interfaces
- Perform development tasks like building, testing, and deploying applications

This MCP server is particularly useful for AI-powered development environments where the AI needs to perform actual development tasks beyond just file manipulation.

## Installation

### Prerequisites
- Go 1.24.1 or higher
- A compatible MCP client (like CPE, Claude Desktop, or other MCP-enabled tools)

### Install from Source

1. Clone the repository:
   ```bash
   git clone https://github.com/spachava753/editor-mcp.git
   cd editor-mcp
   ```

2. Build the binary:
   ```bash
   go build -o editor-mcp
   ```

3. Install globally (optional):
   ```bash
   go install github.com/spachava753/editor-mcp
   ```

### Binary Installation

Download the latest binary from the [releases page](https://github.com/spachava753/editor-mcp/releases) and place it in your PATH.

## Usage

### As a Standalone MCP Server

The server can be run directly:
```bash
editor-mcp
```

This will start the MCP server in stdio mode, which is the standard mode for MCP communication.

### Integration with MCP Clients

#### Claude Desktop Configuration

Add to your Claude Desktop configuration:

```json
{
  "mcpServers": {
    "shell": {
      "command": "/path/to/editor-mcp",
      "type": "stdio"
    }
  }
}
```

### Available Tools

#### shell

Executes shell commands in a controlled environment.

**Parameters:**
- `command` (string, required): The shell command to execute
- `shell` (string, optional): The shell to use (default: bash)
- `async` (boolean, optional): Execute command asynchronously
- `timeout` (int, optional): Timeout in seconds (not supported with async)

**Example usage:**
```json
{
  "tool": "shell",
  "arguments": {
    "command": "ls -la",
    "timeout": 30
  }
}
```

## Development

### Building

```bash
go build -o editor-mcp
```

### Testing

```bash
go test ./...
```

## Security Considerations

- Commands are executed with the same permissions as the user running the MCP server
- The server includes timeout mechanisms to prevent hanging commands
- Environment variables are inherited from the parent process
- No additional sandboxing is provided beyond standard OS permissions

## License

[MIT License](LICENSE)

## Support

For issues and questions, please open an issue on the [GitHub repository](https://github.com/spachava753/editor-mcp/issues).