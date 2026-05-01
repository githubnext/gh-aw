---
# Validation Strategy - Bash-first approach for end-of-session build and test validation.
#
# Import this shared component to ensure agentic workflows use bash commands
# (not MCP tools) for build, test, and validation steps — especially after
# long file-exploration or analysis phases where MCP connections may time out.
#
# Usage:
#   imports:
#     - shared/validation-strategy.md
---

## Validation Strategy

### Use Bash for Build and Test Validation

**Always use direct `bash` commands for build, test, and validation steps** — especially for any step that runs *after* a file-exploration or analysis phase.

MCP connections time out after approximately **5 minutes of inactivity**. Workflows with long analysis phases routinely exceed this threshold. When the agent finally attempts an end-of-session validation call via an MCP tool, the MCP transport may have been torn down, causing the workflow to fail at the last step.

**Correct — use bash for validation:**
```bash
make build
make test-unit
make recompile
make agent-finish
go test ./...
```

**Incorrect — MCP tools fail after long inactivity:**
```
Use the mcpscripts-make tool with args: "build"     ← may fail with context canceled
Use the mcpscripts-go tool with args: "test ./..."  ← may fail with context canceled
```

### Add Intermediate Validation Checkpoints

Add a **build checkpoint** using bash after the first major code edit (not just at the very end of the session). This surfaces compile errors early before spending more context on subsequent edits.

### When MCP Tools Are Safe to Use

- Early in a session, before any long exploration phase
- For short-lived workflows with no significant idle time between tool calls
- For non-validation operations that are inherently early in the session (e.g., fetching data, reading files)
