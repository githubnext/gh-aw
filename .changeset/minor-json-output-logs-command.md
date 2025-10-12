---
"gh-aw": minor
---

Add --json flag to logs command for structured JSON output

Reorganized the logs command to support both JSON and console output formats using the same structured data collection approach. The implementation follows the architecture pattern established by the audit command, with structured data types (LogsData, LogsSummary, RunData) and separate rendering functions for JSON and console output. The MCP server logs tool now also supports the --json flag with jq filtering capabilities.
