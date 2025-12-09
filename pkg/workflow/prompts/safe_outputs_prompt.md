<safe-outputs>
<description>GitHub API Access Instructions</description>
<important>
The gh (GitHub CLI) command is NOT authenticated in this environment. Do NOT use gh commands for GitHub API operations.
</important>
<instructions>
To interact with GitHub (create issues, discussions, comments, pull requests, etc.), you MUST use the safe output tools provided by the safeoutputs MCP server.

**CRITICAL**: When the workflow requires creating an issue, discussion, or other GitHub resource:
1. You MUST call the appropriate tool (e.g., `create_issue`, `create_discussion`) from the safeoutputs MCP server
2. Simply writing markdown content or describing what should be created will NOT work
3. The workflow depends on these tool calls being made - without them, follow-up actions will be skipped
4. Each tool call writes structured data that downstream workflow jobs process

**Example**: To create an issue with a portfolio analysis report:
- ✅ CORRECT: Call `create_issue` tool with title and body parameters
- ❌ WRONG: Write markdown text describing the issue or outputting the report content directly

Available tools include: create_issue, create_discussion, create_pull_request_review_comment, and others depending on workflow configuration. Use the MCP server's tool list to see what's available.
</instructions>
</safe-outputs>
