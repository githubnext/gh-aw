<safe-outputs>
<instructions>
gh CLI is NOT authenticated. Use safeoutputs MCP server tools for GitHub writes and completion signaling — tool calls required.

**CRITICAL: You MUST call at least one safe-output tool before finishing.** Multiple calls are allowed up to each tool's configured limit. If no GitHub action was taken (no issues, comments, PRs, etc. were created or updated), you MUST call `noop` with a message explaining why no action was needed. Failing to call any safe-output tool is the #1 cause of workflow failures. Do NOT end your response without calling at least one safe-output tool.

When no action is needed, call noop like this:
```json
{"noop": {"message": "No action needed: [brief explanation of what was analyzed and why no action was required]"}}
```

temporary_id: optional cross-reference field. Canonical form: '#aw_' followed by 3–12 alphanumeric or underscore characters — e.g., '#aw_abc1', '#aw_pr_fix'. Pattern: /^#?aw_[A-Za-z0-9_]{3,12}$/i (the '#' prefix is optional; bare 'aw_abc1' is accepted and normalised to '#aw_abc1' automatically). Use this form for all field values (temporary_id, item_number, issue_number, parent, etc.). In body/markdown text, '#aw_abc1' references are replaced with the real issue/PR number after creation. Omit entirely when not needed.

**Note**: safeoutputs tools do NOT support `@filename` file name expansion. Always provide content inline — do not use `@<filename>` references in tool arguments.
</instructions>
</safe-outputs>
