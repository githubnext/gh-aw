# ADR-27479: Comment Memory — File-Based Agent Memory with GitHub Comment Persistence

**Date**: 2026-04-21
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

Agents in this system need persistent memory that survives across workflow runs. Before this change, there was no structured mechanism for cross-run memory: each run started without knowledge of prior context. Requiring agents to explicitly call a `comment_memory` safe-output tool to write memory would add cognitive overhead—agents could forget to call it, call it inconsistently, or produce memory that only captures the tail of a run rather than its full evolution. A simpler model was needed where memory management is invisible to the agent and leverages the file-editing workflow the agent already uses natively.

### Decision

We will introduce a `comment_memory` safe output type that stores agent memory in a managed GitHub issue or PR comment, and make memory available to the agent as ordinary local markdown files under `/tmp/gh-aw/comment-memory/` that are pre-materialized before the agent runs and automatically synced back after the run. Agents edit memory files directly using their existing file-editing tools; the safe-output handler manager detects changes and upserts the managed comment. This decouples the agent from the persistence mechanism and eliminates any explicit memory-tool call requirement.

### Alternatives Considered

#### Alternative 1: Agent-Callable `comment_memory` Safe-Output Tool

The agent would explicitly emit a `comment_memory` safe-output message to persist memory. This was considered because it follows the established safe-output pattern already used by `add_comment`, `push_to_pull_request_branch`, etc. It was not chosen because it requires the agent to reason about *when* and *what* to persist, increases the risk of incomplete or forgotten memory updates, and adds a memory-specific abstraction on top of file editing that the agent already does naturally.

#### Alternative 2: Ephemeral In-Run State Only

Memory could be scoped entirely to a single workflow run, with no persistence across runs. This was considered for its simplicity—no setup/teardown step, no GitHub API calls for memory. It was not chosen because the core requirement is cross-run persistence: agents need to remember context from prior invocations to provide coherent, incrementally improving assistance.

#### Alternative 3: External Store (e.g., Dedicated Memory Issue or Database)

Memory could be stored in a dedicated GitHub issue (a "memory issue") or external database rather than a comment on the triggering item. This was not chosen because it introduces additional infrastructure complexity, requires tracking a separate issue ID per workflow, and diverges from the existing `add_comment` / safe-output infrastructure that already handles per-item comment management with access control and audit trails.

### Consequences

#### Positive
- Agents use their natural file-editing workflow for memory; no new tool API to learn or call
- Memory is automatically persisted without the agent needing to take an explicit action at run end
- Memory survives container and runner boundaries by being stored in GitHub comments
- Comment-memory content is injected into the agent prompt context and threat-detection context, giving the agent its prior knowledge on startup

#### Negative
- Adds a pre-agent setup step to materialize managed comment content into local files, increasing workflow complexity and startup latency
- Memory files under `/tmp/gh-aw/comment-memory/` are always injected into agent context, which can increase prompt size even when the memory is not relevant to the current task
- Managed comment bodies must be parsed for `<comment-memory id="...">` XML markers, creating a fragile coupling between the comment format and the materialization logic
- Auto-sync on file change means any edit to the memory file (including accidental ones) will be persisted to GitHub

#### Neutral
- The `comment_memory` handler is registered in the safe-output compiler, permissions system, and validation config, following the same extension points as other safe-output types
- Existing safe-output handler manager gains automatic file-based sync logic that runs unconditionally after each agent turn
- The W3C-style safe outputs specification is updated to formally document the `comment_memory` type and end-to-end data flow

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Memory Materialization

1. The pre-agent setup step **MUST** scan the managed GitHub comment for `<comment-memory id="...">` blocks and extract each block's user-editable content into a separate markdown file under `/tmp/gh-aw/comment-memory/`.
2. Each memory file **MUST** be named using its `memory_id` value (e.g., `/tmp/gh-aw/comment-memory/{memory_id}.md`).
3. Memory files **MUST** be writable by the agent process at runtime.
4. Memory files **MUST NOT** include the XML marker tags or footer content — only the user-editable body of the managed block **SHALL** be materialized.

### Agent Interaction with Memory

1. Agents **MUST NOT** call a `comment_memory` safe-output tool to persist memory; memory persistence **SHALL** occur exclusively via the automatic file-sync mechanism.
2. Agents **SHOULD** treat files under `/tmp/gh-aw/comment-memory/` as their primary cross-run memory store.
3. Agents **MAY** create new files under `/tmp/gh-aw/comment-memory/` to introduce new named memory slots; the sync mechanism **MUST** detect and persist these as new managed blocks.

### Automatic File Sync

1. The safe-output handler manager **MUST** detect changes to files under `/tmp/gh-aw/comment-memory/` after each agent turn.
2. For each changed file, the handler manager **MUST** upsert the corresponding `<comment-memory id="...">` block in the managed GitHub comment on the triggering issue or PR.
3. The upsert **MUST** preserve the XML marker structure and footer content while replacing only the user-editable body with the updated file content.
4. The handler manager **MUST NOT** persist unchanged memory files to avoid spurious comment updates.

### Comment Format

1. Managed comment bodies **MUST** wrap each memory slot in `<comment-memory id="{memory_id}">...</comment-memory>` XML markers.
2. The `memory_id` **MUST** match the pattern `^[a-zA-Z0-9_-]+$`; any other value **MUST** be rejected with an error.
3. A managed comment **MAY** contain multiple memory slots with distinct `memory_id` values.
4. The managed comment **MUST** include an XML provenance marker (e.g., `<!-- aw: ... -->`) identifying the workflow run that last wrote it.

### Context Injection

1. All materialized memory files **MUST** be included in the unified agent artifact.
2. All materialized memory files **MUST** be included in the threat-detection prompt context.
3. The agent prompt **MUST** include guidance explaining the memory file location (`/tmp/gh-aw/comment-memory/`) and the expected read/write workflow.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24706352108) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
