Edits an existing file or creates a new file. When old_text is provided, performs exact string matching and replacement. When old_text is empty, creates a new file with the given content.

Parameters:
- path: Path to the file to edit or create (required)
- old_text: Exact text to find and replace. If empty, creates a new file instead (optional)
- text: Replacement text or content for new file (required)

Notes:
- For editing: match must be exact including whitespace
- For editing: returns error if old_text is not found
- For editing: returns error if multiple occurrences of old_text exist in the file
- For creating: returns error if file already exists at the path
- For creating: parent directories are created automatically if they don't exist
