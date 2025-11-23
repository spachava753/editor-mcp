A powerful file manipulation tool for creating, reading, and editing text files. This tool provides four distinct
commands for different file operations:

COMMANDS:

1. "create" - Creates a new file or overwrites an existing file with the specified text content
   - Use when: Starting a new file from scratch or completely replacing file contents
   - Parameters: paths (file paths to create), text (full file content)
   - Note: Will overwrite existing files without warning

2. "str_replace" - Performs exact string matching and replacement within existing files
   - Use when: Making precise edits to specific text portions while preserving surrounding content
   - Parameters: paths (files to edit), old_text (exact text to find), text (replacement text), replace_all (optional,
     default false)
   - Note: Match must be exact including whitespace; use replace_all=true for multiple occurrences

3. "insert" - Inserts new text at a specific line number, preserving existing content
   - Use when: Adding content at a precise location without disturbing existing text
   - Parameters: paths (files to edit), insert_after (0-based line number), text (content to insert)
   - Note: Line numbers are 0-based; insert_after=0 inserts after the first line

4. "view" - Views file contents with automatic format detection for text, images, and audio
   - Use when: Reading file contents with proper content type handling
   - Parameters: paths (file paths to view)
   - Returns: Each file as a separate content block with appropriate MIME type (text/image/audio)
   - Note: Supports batch viewing of multiple files

FEATURES & LIMITATIONS:

- Batch operations: All commands support multiple file paths for bulk operations
- File creation: Parent directories are **NOT** created if they don't exist, you must create them manually
- Encoding: UTF-8 text files for create/edit; binary data supported for view (images/audio)
- Atomicity: Each file operation completes independently; partial batch success is possible
- Error handling: Returns individual success/error status for each file in the batch
- No regex support: Use exact string matching only (str_replace)
- No file deletion: This tool cannot delete files, only create, modify, or view them
- Line endings: Preserves existing line endings; new content uses system default
- Content detection: View command automatically detects content type by file extension

COMMON PATTERNS:

- Code refactoring: Use str_replace with replace_all=true for renaming variables/functions
- Adding imports/headers: Use insert with insert_after=0 for file headers
- Appending content: First read the file to count lines using a command like 'nl', then use insert
- Safe editing: Use str_replace to preserve file structure and only change specific parts
- File templates: Use create to generate new files from templates
- Viewing files: Use view to read text files, images, or audio with proper content type handling
- Batch viewing: Use view with multiple paths to view several files at once

IMPORTANT:

- Regex find and replace: if you want to replace some text matched with a specific text, use the shell tool to execute shell commands e.g. `sed -i 's/foo/bar/g' myfile.txt`
