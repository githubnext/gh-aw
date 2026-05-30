# ADR-26968: MCP Compile Tool Uses Subprocess Stdout Exclusively for Success Responses

**Date**: 2026-04-18
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh-aw` binary exposes a Model Context Protocol (MCP) server over stdio, where the JSON-RPC transport channel (stdout) must carry only well-formed JSON-RPC 2.0 messages. The `compile` tool invokes an external subprocess (the `gh-aw` binary itself in compile mode) to validate workflow files and return structured JSON results. Prior to this decision, there was ambiguity about whether subprocess stderr output — used for diagnostics and progress logging — should also be included in the MCP tool response text returned over the JSON-RPC channel. Mixing stderr into the response text caused JSON parsing failures when MCP clients attempted to parse the response as JSON, because diagnostic log lines are plain text, not JSON.

### Decision

We will configure the `compile` MCP tool to source its success response content exclusively from subprocess stdout, discarding stderr output from the tool response body. When the subprocess exits successfully, only the data written to its stdout (valid JSON) will be returned as the tool result text. Subprocess stderr is reserved for diagnostics visible at the process level (e.g., captured by a test harness or container logs) and must not appear in the JSON-RPC response. This invariant is enforced by both a unit test (mocking the subprocess command) and a binary-level integration test (launching the full MCP server and verifying raw stdio traffic).

### Alternatives Considered

#### Alternative 1: Include Both Stdout and Stderr in the Tool Response

Concatenating stdout and stderr into the tool response was considered because it preserves all diagnostic information visible to an MCP client. This was rejected because MCP clients that expect the compile result text to be parseable JSON would encounter parse failures whenever the subprocess emitted any log line to stderr. The MCP protocol treats tool result text as opaque to the transport but clients — including the test harness — depend on the structured JSON content.

#### Alternative 2: Route All Output Through Structured JSON on Stdout

An alternative is to require the subprocess to emit all output (including diagnostics) as structured JSON on stdout, eliminating stderr entirely. This would allow richer error context in the tool response. It was not chosen because it would require invasive changes to the `compile` subcommand's logging infrastructure across all callers, not just the MCP server. The subprocess is also invoked directly by users and other tooling where plain-text stderr diagnostics are desirable.

#### Alternative 3: Capture Stderr and Append It as a Separate Content Item

The MCP tool result `content` array can hold multiple items. Stderr text could be appended as a second `TextContent` item tagged as diagnostic. This was considered but rejected because it complicates client logic (clients must filter by content item index or type to obtain the JSON result), and it leaks internal diagnostic format changes into the public tool response contract.

### Consequences

#### Positive
- MCP clients receive a clean, parseable JSON response; no interleaved log noise can break JSON parsing.
- The stdout/stderr split is a well-understood Unix convention; future subprocess implementers can follow it without reading MCP-specific documentation.
- Binary-level integration tests provide strong regression protection at the transport boundary.

#### Negative
- Subprocess diagnostic output is invisible to MCP clients; clients that need to surface compiler warnings embedded only in stderr must rely on the subprocess writing those to stdout instead.
- The unit test uses shell scripts (`sh -c`) to mock subprocess output; this creates a dependency on a POSIX shell being present in the test environment.

#### Neutral
- The integration test must be run against a pre-built binary (`make build`); it will skip automatically when the binary is absent. Teams must remember to build before running integration tests.
- Stderr captured during integration tests is collected into a buffer but not asserted on, leaving stderr content unverified at the integration level.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### MCP Tool Response Content

1. On subprocess success (exit code 0), the `compile` MCP tool **MUST** populate its `TextContent` response exclusively from the subprocess's stdout stream.
2. The `compile` MCP tool **MUST NOT** include any bytes read from the subprocess's stderr stream in the tool result `content` field.
3. The tool response text **MUST** be valid JSON as produced by the subprocess; implementations **MUST NOT** append additional text before or after the subprocess stdout output.
4. Implementations **MAY** log subprocess stderr to a diagnostic sink (e.g., a container log or test buffer) that is separate from the JSON-RPC stdio channel.

### MCP Server Stdio Channel Purity

1. All output written to the MCP server process's stdout **MUST** be valid JSON-RPC 2.0 messages.
2. The MCP server process **MUST NOT** write any non-JSON-RPC content (log lines, progress messages, debug output) to its stdout.
3. Diagnostic output from the MCP server process itself **SHOULD** be directed to stderr or a dedicated log file, never to stdout.

### Test Coverage

1. Any change to `compile` tool output stream handling **MUST** be accompanied by a unit test that verifies stdout-only semantics using a mocked subprocess command.
2. Any change to MCP server stdio stream handling **SHOULD** be covered by a binary-level integration test that validates raw stdout traffic is valid JSON-RPC 2.0 throughout the full initialize–call–shutdown lifecycle.
3. Integration tests that depend on a compiled binary **MAY** skip (using `t.Skip`) when the binary is not present, but **MUST NOT** silently pass.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: the `compile` tool result text contains only subprocess stdout bytes on success, the MCP server stdout channel carries only valid JSON-RPC 2.0 messages, and both a unit test and an integration test exist to enforce these invariants. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24594338493) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
