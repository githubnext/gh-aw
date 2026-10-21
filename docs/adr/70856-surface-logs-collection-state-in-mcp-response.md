# ADR-70856: Surface Logs Collection State (Cursors and Episode Coverage) in the MCP Logs Response

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: pelikhan, Unknown

---

### Context

The `gh aw` MCP server exposes a `logs` tool. The tool does not return the full logs report inline. A guardrail in `pkg/cli/mcp_logs_guardrail.go` writes the report to a cache file and sends back a small JSON envelope (`MCPLogsGuardrailResponse`) containing `file_path` plus a short message. The CLI already records collection limits inside the report. These include one `continuation` cursor or a per-target `continuations` array for multi-repository or multi-workflow runs, `episode_retrieval` stop reasons and issues, and per-episode `coverage` blocks. The adapter only checked the singular `continuation` field. As a result, multi-target cursors and episode evidence gaps never reached MCP callers. Its message also told callers to read "the full data", which could be wrong. Callers such as agents generally act on the envelope without parsing the whole file, so any missing signal there leads them to treat incomplete data as complete.

### Decision

We will make the MCP logs envelope a faithful summary of the collection state the CLI already recorded. This is done by parsing the report JSON in a single `extractLogsCollectionState` function. The envelope will keep every per-target `continuations` entry exactly as recorded, including the repository and all cursor parameters. It will set `partial: true` whenever any cursor exists or any known episode evidence gap exists. For episode reports, it will add `coverage_status` (`partial` or `unknown`, never `complete`) and a sorted, deduplicated `coverage_reasons` list. Retrieval stop reasons and issues will be namespaced with a `retrieval_` prefix. Evidence gaps that cannot be resumed will be reported as partial with no made-up cursor. The cached report file stays byte-for-byte unchanged, so the envelope adds information and never replaces the report.

### Alternatives Considered

#### Alternative 1: Keep only the singular `continuation` and tell callers to read the file

We could leave the envelope minimal and rely on callers to open `file_path` to find `continuations` and coverage data. This is the smallest change. We rejected it because this behavior is exactly the defect being fixed. The envelope would keep saying results were complete (or "full") when they were not, and callers that trust the envelope would quietly drop targets.

#### Alternative 2: Collapse multi-target cursors into one merged continuation

We could combine per-target cursors into one `continuation` so callers only handle a single shape. We rejected this because each target has its own repository, `end_date`, and `before_run_id`. Merging them loses information and makes it impossible to resume each target correctly.

#### Alternative 3: Change the CLI report format or report `complete` coverage

We could reshape the report itself, or have the adapter declare episodes `complete` when no gaps were found. We rejected both. The PR explicitly avoids changing report contents, artifact selection, and filtering. Coverage based on selected runs also cannot prove that episode membership is complete, so `unknown` is the honest default.

### Consequences

#### Positive
- MCP callers can resume every truncated target using its own repository and cursor parameters.
- Episode evidence gaps, such as budget stops and unavailable recorded runs, are visible as `partial` along with stable, deduplicated reasons, and no cursor is invented for them.
- Cached report bytes are unchanged, so existing consumers of the file are not affected.

#### Negative
- The envelope schema grows (`continuations`, `coverage_status`, `coverage_reasons`), which makes the contract MCP clients and docs have to follow larger.
- The adapter now depends on internal report field names (`episode_retrieval`, `episodes[].coverage`), which ties the MCP layer more closely to the CLI report schema.
- `coverage_status` can never be `complete`, so callers always see `unknown` even for runs that are in fact fully covered.

#### Neutral
- Response messages now say "collected report" instead of "full data", and tool guidance and reference docs (`gh-aw-as-mcp-server.md`) were updated to match.
- The parse failure path returns an empty state and logs at debug level. It does not error, which matches the previous best-effort behavior.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
