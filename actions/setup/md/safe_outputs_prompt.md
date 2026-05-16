<safe-outputs>
<instructions>
gh CLI is NOT authenticated. Use safeoutputs MCP server tools for GitHub writes and completion signaling — tool calls required.

**CRITICAL: You MUST call at least one safe-output tool before finishing.** Multiple calls are allowed up to each tool's configured limit. If no GitHub action was taken (no issues, comments, PRs, etc. were created or updated), you MUST call `noop` with a message explaining why no action was needed. Failing to call any safe-output tool is the #1 cause of workflow failures. Do NOT end your response without calling at least one safe-output tool.

When no action is needed, call noop like this:
```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why no action was required]"}}
```

temporary_id: optional cross-reference field. Format: aw_ + 3–12 alphanumeric chars (/^aw_[A-Za-z0-9]{3,12}$/). When **referencing** a temporary_id in a body or markdown text, use #aw_abc1 — the # is replaced with the real issue/PR number after creation. When using a temporary_id as a **field value** (item_number, issue_number, parent, etc.), pass the raw id without the #: 'aw_abc1'. Omit entirely when not needed.

**Note**: safeoutputs tools do NOT support `@filename` file name expansion. Always provide content inline — do not use `@<filename>` references in tool arguments.
</instructions>
</safe-outputs>
