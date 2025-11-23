Executes shell commands with full system access and captures output. Each invocation runs in a fresh shell instance with
inherited environment variables.

EXECUTION MODEL:

- Shell: Uses specified shell (default: bash) to interpret commands
- Environment: Inherits all parent process environment variables
- Working Directory: Executes in the current working directory of the MCP server
- Isolation: Each command runs in a new shell process; no state persists between calls

PARAMETERS:

- command (required): Shell command string to execute, can include pipes, redirects, and shell built-ins
- shell (optional): Shell interpreter to use (e.g., "bash", "sh", "zsh", "fish"); defaults to "bash"
- timeout (optional): Maximum execution time in seconds (0 = no timeout, default)
- async (optional): If true, returns immediately with process ID for background execution

OUTPUT:

- pid: Process ID of the executed command
- stdout: Complete standard output captured as string
- stderr: Complete standard error captured as string
- exit_code: Process exit code (0 typically indicates success)

CAPABILITIES:

- Full shell syntax: Supports pipes (|), redirects (>, >>), command chaining (&&, ||), subshells
- Environment manipulation: Can set variables inline (VAR=value command)
- Script execution: Can run multi-line scripts using heredocs or semicolons
- Background processes: Use async=true for long-running commands (see process management tools)
- File system access: Full read/write access with user permissions
- Network access: Can make network requests, download files, etc.
- System utilities: Access to all installed CLI tools and system commands

LIMITATIONS & CONSIDERATIONS:

- Security: Commands run with full user privileges; no sandboxing or restrictions
- Output size: Both stdout and stderr are fully captured in memory; very large outputs may cause issues
- Binary output: Binary data in stdout/stderr may not display correctly
- Interactive commands: Tools expecting TTY input (like vim, less) won't work properly
- Timeouts: Only enforced for synchronous execution; async processes continue indefinitely
- Shell differences: Commands may behave differently across shells; test shell-specific features
- Working directory: Cannot change the working directory persistently between calls
- Signal handling: Limited to SIGTERM on timeout; use process management tools for fine control

COMMON PATTERNS:

- File inspection: 'ls -la', 'cat file.txt', 'head -n 20 file.log'
- Search operations: 'grep -r "pattern" .', 'find . -name "\*.js"'
- System info: 'uname -a', 'df -h', 'ps aux | grep process'
- Network tasks: 'curl https://api.example.com', 'ping -c 4 google.com'
- Package management: 'npm install', 'pip list', 'go mod download'
- Git operations: 'git status', 'git diff', 'git log --oneline'
- Build tasks: 'make build', 'go test ./...', 'npm run build'
- Complex pipelines: 'cat file | grep pattern | awk "{print $2}" | sort | uniq'

ERROR HANDLING:

- Non-zero exit codes marked as errors but still return output
- Command not found results in exit_code 127 with error in stderr
- Timeout results in process termination and partial output return
- Shell syntax errors captured in stderr with appropriate exit code

IMPORTANT:

- When you want to make modifications to files, prefer to use the 'text_edit' tool. You should not try to use this tool to write to files
