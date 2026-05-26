# ADR-34835: Add `view` Command for Streaming Timeline Replay of Workflow Runs

**Date**: 2026-05-26
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

`gh aw audit` already produces a unified MCP Gateway + AWF Firewall + Agent timeline for a workflow run, but presents it as a tabular summary with aggregate statistics. That format is great for forensic review of completed runs, but it does not resemble what a developer sees when watching a Copilot CLI session live — a flowing, chronological narrative of agent turns, tool invocations, network decisions, and responses. Users investigating a run after the fact want both perspectives, and there is no existing offline way to reconstruct the "live session" view from cached artifacts.

### Decision

We will add a new top-level CLI command, `gh aw view <run-id-or-url>` (registered in the `analysis` command group alongside `logs` and `audit`), that downloads run artifacts via the existing `downloadRunArtifacts` helper, builds the existing `UnifiedTimeline` from those artifacts, and renders it through a new **stream renderer** (`renderUnifiedTimelineStream`) that emits turn-by-turn output simulating a live agentic session. The command is initially marked `Hidden: true` while the output format stabilizes. The `UnifiedTimeline` data model is extended (not forked) to carry `assistant.message` and `reasoning` events with truncated `MessageContent` snippets so the stream can surface what the agent actually said and thought, not just what tools it called.

### Alternatives Considered

#### Alternative 1: Add `--stream` / `--format=stream` flag to `gh aw audit`

Reuse the audit command and gate the new renderer behind a flag. Considered because it avoids a new top-level command. Rejected because audit's contract centres on tabular forensic output with aggregate statistics, and overloading it with a fundamentally different presentation (no stats, no table, indented per-turn flow) would make the command harder to document and reason about. A dedicated command keeps each tool focused on a single output shape.

#### Alternative 2: Extend `gh aw logs` with a `--render` flag

`logs` already downloads artifacts, so extending it would avoid duplicating the download path. Rejected because `logs` is currently scoped to raw log retrieval and display — adding timeline reconstruction and styled stream rendering would expand its surface area beyond its current responsibility. Keeping retrieval and replay-style rendering separate makes the command hierarchy easier to discover.

#### Alternative 3: Reuse the existing table renderer for stream output

Pipe events through `renderUnifiedTimeline` (the existing table renderer) and post-process the output. Rejected because the table renderer's job is fundamentally different — it aligns events into columns and emits summary statistics — whereas the stream renderer needs per-turn section headers, indented children, and inline message snippets. Attempting to share a single renderer would have required adding mode-switches that pollute both code paths.

### Consequences

#### Positive
- Users gain an offline, cacheable "watch the session" view of any past run without needing live access to a running CLI.
- The stream renderer is additive: existing `audit` users see no change, and the timeline data model and `BuildUnifiedTimeline` builder are reused unchanged.
- Surfacing message content (`assistant.message`, `reasoning`) gives reviewers visibility into agent intent, not just tool side effects.
- `Hidden: true` lets the team iterate on the output format without committing to a stable public contract.

#### Negative
- Two CLI commands (`audit` and `view`) now operate on the same artifact set with different output formats — users must learn when to reach for which.
- The stream renderer is a second consumer of `UnifiedTimelineEvent`, so any future field added to the struct must be considered for both renderers.
- Hidden commands are discoverable only via source or word-of-mouth; users who would benefit may not find it until it graduates.
- Including raw message content in the rendered output (even truncated to 3 lines × 80 chars) increases the risk that sensitive payloads appear on a developer's terminal during replay; this is a behaviour change from the previous audit-only flow.

#### Neutral
- Emoji icons in the timeline renderer were replaced with cross-compatible Unicode glyphs (`⚙`, `⊖`, `⊗`, `○`, `●`, `◐`) so output renders consistently across terminals; this affects both the new stream renderer and the existing table renderer.
- The command currently passes a `nil` artifact filter to `downloadRunArtifacts`, fetching all artifacts — a deliberate choice because the timeline opportunistically consumes whichever JSONL files are present, but it means `view` downloads more bytes than `audit` does for the same run.
- A `safe-output-items.jsonl` summary section and a trailing run URL are rendered after the timeline, making the command useful as a single-shot "what happened in this run?" tool.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Command Surface

1. The CLI **MUST** expose a `view` subcommand that accepts exactly one positional argument: a run ID or run URL parseable by `parser.ParseRunURLExtended`.
2. The `view` command **MUST** be registered in the `analysis` command group alongside `logs` and `audit`.
3. The `view` command **MUST** support the `--output` (run artifact cache directory) and `--repo` (`owner/repo` override) flags with the same semantics as `audit`.
4. The `view` command **SHOULD** remain `Hidden: true` until the stream output format is declared stable by the maintainers.
5. The `view` command **MUST NOT** require network access when the target run directory already contains the necessary JSONL artifacts; cached runs **MUST** render without re-downloading.

### Timeline Reuse

1. The `view` command **MUST** consume the `UnifiedTimelineEvent` slice produced by `BuildUnifiedTimeline`; it **MUST NOT** define a parallel timeline representation.
2. Extensions to the timeline data model required by `view` (e.g. `MessageContent`, `TimelineKindAssistantMessage`, `TimelineKindReasoning`) **MUST** live on `UnifiedTimelineEvent` and **MUST** be usable by other timeline consumers without modification.
3. The stream renderer (`renderUnifiedTimelineStream`) **MUST NOT** be invoked by `audit`; the table renderer (`renderUnifiedTimeline`) remains the renderer for `audit`.

### Stream Renderer Behaviour

1. The stream renderer **MUST** emit agent turns as section headers with the form `> Turn N [timestamp]` and **MUST** indent child events (tool calls, network events, assistant/reasoning messages) beneath their owning turn.
2. The stream renderer **MUST** truncate message content to at most `streamMaxMessageLines` non-empty lines and at most `streamMaxLineLength` runes per line, appending an ellipsis marker when content is truncated.
3. The stream renderer **MUST NOT** emit aggregate statistics, summary tables, or column-aligned event tables.
4. The stream renderer **MUST** suppress ANSI styling when stdout is not a terminal (detected via `tty.IsStdoutTerminal`) so piped output remains free of escape codes.
5. Event icons used by the timeline renderers **MUST** be cross-terminal-compatible Unicode glyphs and **MUST NOT** depend on emoji presentation selectors for correctness.

### Output Composition

1. When `view` completes successfully, the command **MUST** print, in order: (a) the streamed timeline (or a warning when empty), (b) a `Safe Outputs` section if `safe-output-items.jsonl` contains any items, and (c) the GitHub Actions run URL when `owner` and `repo` are known.
2. The `Safe Outputs` section **MUST** be omitted entirely when no items were created, and **MUST NOT** print a section header alone.
3. An absent or empty timeline **MUST** produce a warning on stderr and a non-error exit; `view` **MUST NOT** treat empty artifacts as a failure.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
