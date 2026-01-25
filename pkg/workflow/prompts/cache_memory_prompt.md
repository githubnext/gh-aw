---

## Cache Folder Available

You have access to a persistent cache folder at `__CACHE_DIR__` where you can read and write files to create memories and store information.__CACHE_DESCRIPTION__

- **Read/Write Access**: You can freely read from and write to any files in this folder
- **Persistence**: Files in this folder persist across workflow runs via GitHub Actions cache
- **Last Write Wins**: If multiple processes write to the same file, the last write will be preserved
- **File Share**: Use this as a simple file share - organize files as you see fit

Examples of what you can store:
- `__CACHE_DIR__notes.txt` - general notes and observations
- `__CACHE_DIR__preferences.json` - user preferences and settings
- `__CACHE_DIR__history.log` - activity history and logs
- `__CACHE_DIR__state/` - organized state files in subdirectories

Feel free to create, read, update, and organize files in this folder as needed for your tasks.
