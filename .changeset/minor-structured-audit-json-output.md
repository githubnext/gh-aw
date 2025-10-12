---
"gh-aw": minor
---

Reorganize audit command with structured output and JSON support

Added `--json` flag to the audit command for machine-readable output. Enhanced audit reports with comprehensive information including per-job durations, file sizes with descriptions, and improved error/warning categorization. Updated MCP server integration to use JSON output for programmatic access.

Key improvements:
- New `--json` flag for structured JSON output
- Per-job duration tracking from GitHub API
- Enhanced file information with sizes and intelligent descriptions
- Better error and warning categorization
- Dual rendering: human-readable console tables or machine-readable JSON
- MCP server now returns structured JSON instead of console-formatted text
