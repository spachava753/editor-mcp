Replaces old text with new text in an existing file. Performs exact string matching and replacement while preserving surrounding content.

Parameters:
- path: Path to the file to edit (required)
- old_text: Exact text to find and replace (required)
- text: Replacement text (required)

Notes:
- Match must be exact including whitespace
- File must exist; this tool cannot create new files
- Returns error if old_text is not found
- Returns error if multiple occurrences of old_text exist in the file
