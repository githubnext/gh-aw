# ADR-70856: Surface Logs Collection State (Cursors and Episode Coverage) in the MCP Logs Response

**Date**: 2026-10-21
**Status**: Proposed

---

### Context

The `gh aw` MCP server exposes a `logs` tool. The tool does not return the full logs report inline. A guardrail in `pkg/cli/mcp_logs_guardrail.go` writes the report to a cache file and sends back a small JSON envelope (`MCPLogsGuardrailResponse`) containing `file_path` plus a short message. The CLI already records collection limits inside the report. These include one `continuation` cursor or a per-target `continuations` array for multi-repository or multi-workflow runs, `episode_retrieval` stop reasons and issues, and per-episode `coverage` blocks. The adapter only checked the singular `continuation` field. As a result, multi-target cursors and episode evidence gaps never reached MCP callers. Its message also told callers to read "the full data", which could be wrong. Callers such as agents generally act on the envelope without parsing the whole file, so any missing signal there leads them to treat incomplete data as complete.

### Decision

The MCP logs envelope summarizes the collection state the CLI already recorded. It preserves the singular `continuation` and every per-target `continuations` entry, including the repository and all cursor parameters. It sets `partial: true` whenever a cursor or a known episode evidence gap exists. For episode reports, it adds `coverage_status` (`partial` or `unknown`, never `complete`) and a sorted, deduplicated `coverage_reasons` list. Evidence gaps that cannot be resumed are reported as partial without an invented cursor.

Cursor fields and optional episode coverage sections are decoded independently. Malformed episode data must not discard valid cursors or other known partial evidence. An unreadable or malformed report produces an explicit unknown collection state rather than an empty state that could be mistaken for successful collection. When usable evidence already establishes partial collection, that evidence remains partial even if another section cannot be decoded. Unknown state alone does not establish either completeness or a resumable gap.

This best-effort envelope preserves actionable recovery information without requiring callers to parse the entire report or treating optional metadata failures as loss of all collection evidence. The cached report stays byte-for-byte unchanged and remains the detailed source of evidence. Artifact selection, filtering, cache access controls, and actor validation are unchanged.

### Alternatives Considered

#### Alternative 1: Keep only the singular `continuation` and tell callers to read the file

Leaving the envelope minimal and relying on callers to open `file_path` is the smallest change, but leaves collection limits invisible to callers that act on the envelope. Describing the file as "full data" also risks implying completeness that the report does not establish.

#### Alternative 2: Collapse multi-target cursors into one merged continuation

Combining per-target cursors into one `continuation` would give callers a single shape, but each target has its own repository, `end_date`, and `before_run_id`. Merging them loses information needed to resume each target correctly.

#### Alternative 3: Change the CLI report format or report `complete` coverage

Reshaping the report would change the contract for existing file consumers without being necessary to expose collection state. Declaring episodes `complete` when no gaps are found is also unsupported: coverage based on selected runs cannot prove that episode membership is complete, so the default is `unknown`.

#### Alternative 4: Decode all metadata together and discard state on any error

A single typed decode is simpler, but malformed optional coverage data can then hide valid continuation cursors. Returning an empty state on failure also hides uncertainty. Independent decoding preserves usable evidence while explicitly reporting what cannot be established.

### Consequences

#### Positive
- MCP callers can resume every truncated target using its own repository and cursor parameters.
- Episode evidence gaps, such as budget stops and unavailable recorded runs, are visible as `partial` along with deduplicated reasons, and no cursor is invented for them.
- Malformed optional metadata does not suppress valid cursors or known partial evidence; unparseable reports expose uncertainty.
- Cached report bytes are unchanged, so existing consumers of the file are not affected.

#### Negative
- The envelope schema grows (`continuations`, `coverage_status`, `coverage_reasons`), which makes the contract MCP clients and docs have to follow larger.
- The adapter now depends on internal report field names (`episode_retrieval`, `episodes[].coverage`), which ties the MCP layer more closely to the CLI report schema.
- Even when collection is in fact complete, the envelope cannot certify complete episode membership; callers must distinguish `unknown` from known `partial` evidence.

#### Neutral
- Response messages now say "collected report" instead of "full data", and tool guidance and reference docs (`gh-aw-as-mcp-server.md`) were updated to match.
- Collection-state reporting does not change cached report bytes, artifact selection, filtering, or access controls.
