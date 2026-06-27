# ADR-41767: Extend Copilot Log Parser for wireApi=responses Format

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: Unknown (Copilot SWE agent, pelikhan)

---

### Context

The `gh aw audit` command reports `tool_usage` metrics extracted by parsing Copilot CLI debug logs. Copilot CLI operates in two wire formats: the original `data:` block format and the newer OpenAI Responses API format (`wireApi=responses`). In the `wireApi=responses` format, the LLM's tool call *requests* still appear in `[DEBUG] data:` blocks, but the tool call *outputs* (function results) are embedded in `[DEBUG] Wire request:` blocks that the parser previously ignored entirely. Additionally, `[INFO] --- End of group ---` lines injected after the closing `}` of data blocks were being appended to JSON buffers, causing silent `json.Unmarshal` failures that dropped all tool call data for those turns.

### Decision

We will extend `ParseLogMetrics` with a second state machine that tracks `[DEBUG] Wire request:` blocks in parallel with the existing `[DEBUG] data:` block state machine. A new `extractWireRequestOutputs` method will parse the `input` array of each wire request block to build a `call_id → tool name` map from `function_call` items, then extract output sizes and truncated previews from matching `function_call_output` items. A `sanitizeJSONBlock` helper will be applied before any `json.Unmarshal` call to strip trailing non-JSON content (including `[INFO]` lines), eliminating the silent parse failures. Tool response previews (`OutputSample`) will be propagated through `ToolCallInfo`, `ToolUsageInfo`, and surfaced in audit reports as a "Response Preview" column.

### Alternatives Considered

#### Alternative 1: Preprocess the entire log to strip [INFO] lines before parsing

A preprocessing pass that filters out all `[INFO]` log lines before the main parse loop would have fixed the JSON contamination bug without adding complexity to the per-line state machine. However, it would have required buffering the entire log content a second time and offered no path to extracting tool outputs from Wire request blocks, leaving `MaxOutputSize` still always zero in `wireApi=responses` mode. It also would have silently discarded [INFO] lines that might carry useful diagnostic information in future formats.

#### Alternative 2: Add a unified format-detection pass and parse one format exclusively

Detecting whether a log file uses `data:` blocks or `Wire request:` blocks upfront (e.g., checking which block type appears first) and then dispatching to a single-format parser would have kept each code path simpler. This was rejected because in practice a session log can theoretically transition between formats mid-file if the CLI upgrades or degrades mid-run, and because Wire request blocks carry *historical* conversation context — meaning output data for a tool call first seen in a `data:` block may only appear in the next Wire request block. Both block types must be processed concurrently in a single pass to correctly associate them.

#### Alternative 3: Extract tool outputs by parsing the data block response choices

The `data:` block JSON already contains the LLM response. In non-responses-API format, tool outputs are available in subsequent messages. We considered whether tool output content could be found in `data:` block structures in `wireApi=responses` mode. Inspection showed that `data:` blocks in this format contain `choices[].message.tool_calls` (the *request*) but not the `function_call_output` (the *response*), which only appears in the `input` array of the next Wire request block. This alternative was infeasible by design of the wire format.

### Consequences

#### Positive
- `tool_usage` in audit reports is now correctly populated for Copilot CLI sessions using `wireApi=responses`, fixing a regression that silently produced empty tool call metrics.
- The new `OutputSample` / "Response Preview" column gives engineers a quick sanity-check of what each tool returned without opening raw logs.
- `sanitizeJSONBlock` hardens all JSON parsing against future cases where non-JSON content appears at the boundaries of a debug block.

#### Negative
- `ParseLogMetrics` now maintains two concurrent state machines (`inDataBlock` / `inWireBlock`) with mutual-flush coordination, increasing the cyclomatic complexity of an already non-trivial parser function.
- The Wire request block parsing is tightly coupled to the `wireApi=responses` conversation history layout (`input[].type == "function_call"` / `"function_call_output"`); if the Copilot CLI changes the wire format or field names, the extractor will silently produce no output rather than returning an error.

#### Neutral
- The `OutputSample` field is added to `ToolCallInfo` (internal metrics) and `ToolUsageInfo` (report struct); JSON output gains the `output_sample` key under `omitempty`, so existing consumers that do not read it are unaffected.
- Unit tests for the new helpers (`sanitizeJSONBlock`, `truncateOutputSample`, `extractWireRequestOutputs`) are added in `copilot_wire_request_test.go`, establishing a baseline for future parser format coverage.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
