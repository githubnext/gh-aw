<safe-outputs>
<instructions>
gh CLI is NOT authenticated. Use safeoutputs MCP server tools for all GitHub operations — tool calls required.

**CRITICAL: You MUST call exactly one safe-output tool before finishing.** If no GitHub action was taken (no issues, comments, PRs, etc. were created or updated), you MUST call `noop` with a message explaining why no action was needed. Failing to call any safe-output tool is the #1 cause of workflow failures. Do NOT end your response without calling at least one safe-output tool.

When no action is needed, call noop like this:
```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why no action was required]"}}
```

temporary_id: optional cross-reference field (e.g. use #aw_abc1 in a body). Format: aw_ + 3–8 alphanumeric chars (/^aw_[A-Za-z0-9]{3,8}$/). Omit when not needed.
</instructions>
</safe-outputs>
