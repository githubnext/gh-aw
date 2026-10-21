
---

## Cache Folders Available

You have access to persistent cache folders where you can read and write files to create memories and store information:

__CACHE_LIST__

- **Read/Write Access**: You can freely read from and write to any files in these folders
- **Persistence**: Files in these folders persist across workflow runs via GitHub Actions cache
- **Last Write Wins**: If multiple processes write to the same file, the last write will be preserved
- **File Share**: Use these as simple file shares - organize files as you see fit
- **Allowed File Types**: Only the following file extensions are allowed: `__ALLOWED_EXTENSIONS__`. Files with other extensions will be rejected during validation.
- **Cache Miss**: Cache files may be absent on first run or after eviction. If the workflow defines initialization, a full analysis, or another fallback for absent state, follow that path and do **not** call `report_missing_data` for the expected absence.
- **Required Cache Data**: If the workflow requires cached input and provides no fallback, call `report_missing_data` with `data_type: "cache_memory"` and `reason: "cache_memory_miss"`. Do not invent required input or silently ignore malformed or unreadable state; report unusable required data with `report_missing_data` unless the workflow explicitly defines recovery.

Examples of what you can store:

__CACHE_EXAMPLES__

Feel free to create, read, update, and organize files in these folders as needed for your tasks, using only the allowed file types.
