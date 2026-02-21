<repo-memory>
## Repo Memory Locations Available

You have access to persistent repo memory folders where you can read and write files that are stored in git branches:

__GH_AW_MEMORY_LIST__
- **Read/Write Access**: You can freely read from and write to any files in these folders
- **Git Branch Storage**: Each memory is stored in its own git branch
- **Automatic Push**: Changes are automatically committed and pushed after the workflow completes
- **Merge Strategy**: In case of conflicts, your changes (current version) win
- **Persistence**: Files persist across workflow runs via git branch storage
- **Allowed File Types**: Only the following file extensions are allowed: `__GH_AW_MEMORY_ALLOWED_EXTENSIONS__`. Files with other extensions will be rejected during validation.

Examples of what you can store:
- `/tmp/gh-aw/repo-memory/notes.md` - general notes and observations
- `/tmp/gh-aw/repo-memory/notes.txt` - plain text notes
- `/tmp/gh-aw/repo-memory/state.json` - structured state data
- `/tmp/gh-aw/repo-memory/history.jsonl` - activity history in JSON Lines format
- `/tmp/gh-aw/repo-memory/data.csv` - tabular data
- `/tmp/gh-aw/repo-memory/history/` - organized history files (with allowed file types)

Feel free to create, read, update, and organize files in these folders as needed for your tasks, using only the allowed file types.
</repo-memory>
