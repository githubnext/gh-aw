<repo-memory>
## Repo Memory Available

You have access to a persistent repo memory folder at `__GH_AW_MEMORY_DIR__` where you can read and write files that are stored in a git branch.__GH_AW_MEMORY_DESCRIPTION__

- **Read/Write Access**: You can freely read from and write to any files in this folder
- **Git Branch Storage**: Files are stored in the `__GH_AW_MEMORY_BRANCH_NAME__` branch__GH_AW_MEMORY_TARGET_REPO__
- **Automatic Push**: Changes are automatically committed and pushed after the workflow completes
- **Merge Strategy**: In case of conflicts, your changes (current version) win
- **Persistence**: Files persist across workflow runs via git branch storage
- **Allowed File Types**: Only the following file extensions are allowed: `.json`, `.jsonl`, `.txt`, `.md`, `.csv`. Files with other extensions will be rejected during validation.__GH_AW_MEMORY_CONSTRAINTS__
Examples of what you can store:
- `__GH_AW_MEMORY_DIR__notes.md` - general notes and observations
- `__GH_AW_MEMORY_DIR__notes.txt` - plain text notes
- `__GH_AW_MEMORY_DIR__state.json` - structured state data
- `__GH_AW_MEMORY_DIR__history.jsonl` - activity history in JSON Lines format
- `__GH_AW_MEMORY_DIR__data.csv` - tabular data
- `__GH_AW_MEMORY_DIR__history/` - organized history files in subdirectories (with allowed file types)

Feel free to create, read, update, and organize files in this folder as needed for your tasks, using only the allowed file types.
</repo-memory>
