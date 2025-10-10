---
"githubnext/gh-aw": patch
---

Configure Copilot log parsing to use debug logs from /tmp/gh-aw/.copilot/logs/

Adds GetLogFileForParsing() method to CodingAgentEngine interface, allowing each engine to specify which log file or directory should be parsed. The Copilot engine now returns /tmp/gh-aw/.copilot/logs/ to use detailed debug logs containing conversation history and tool calls, while other engines continue using stdout/stderr logs. JavaScript parsers updated to handle directory paths by reading and concatenating all .log and .txt files.
