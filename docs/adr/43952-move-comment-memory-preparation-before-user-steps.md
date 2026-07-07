# ADR-43952: Move Comment-Memory Preparation Before User Steps in Agent Job

**Date**: 2026-07-07
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The gh-aw workflow compiler generates GitHub Actions YAML for agentic workflows in multiple ordered phases. The comment-memory feature populates `/tmp/gh-aw/comment-memory/*.md` files from prior PR comment history so agents can read that state. The script responsible (`setup_comment_memory_files.cjs`) reads its handler configuration from `${RUNNER_TEMP}/gh-aw/safeoutputs/config.json`. That config file was written by the "Generate Safe Outputs Config" step in Phase 3 (MCP setup), which ran *after* the comment-memory preparation step — also in Phase 3. The result was an empty config on every execution, causing `resolveCommentMemoryConfig({})` to return `null` and silently skip all comment-memory file writes. Additionally, user-defined `steps:` blocks (deterministic steps that run before the LLM agent) had no access to comment-memory state.

### Decision

We will move the activation artifact download and comment-memory file preparation steps from Phase 3 (engine install / pre-agent) to Phase 2 (runtime and workspace setup), placing them *before* user-defined `steps:` blocks. We will introduce a new compiler function `generateActivationArtifactAndCommentMemorySteps` that emits: (1) the activation artifact download, (2) a new "Write comment-memory configuration" step that serialises a minimal `config.json` at compile time, and (3) the "Prepare comment memory files" step. The full safeoutputs config written later by MCP setup harmlessly overwrites the minimal config file, since comment-memory has already run.

### Alternatives Considered

#### Alternative 1: Fix Step Ordering Within Phase 3

Move "Generate Safe Outputs Config" to run before "Prepare comment memory files" while keeping both in Phase 3. This would fix the empty-config race without changing the phase structure.

Not chosen because it still leaves comment-memory unavailable to user-defined `steps:` blocks that precede the agent. The user-visible bug (empty comment-memory files in deterministic steps) would persist even after the config ordering fix.

#### Alternative 2: Make the Script Config-Independent

Modify `setup_comment_memory_files.cjs` to discover the comment-memory handler config by a different means (e.g., a dedicated config file separate from the main safeoutputs config, or environment variables injected by the compiler).

Not chosen because it would scatter config sources across the codebase and require the JS runtime to understand compiler-side handler registry details. Writing a minimal config at compile time keeps the config contract in one place and avoids a second config format.

### Consequences

#### Positive
- Comment-memory files are reliably populated before user `steps:` blocks execute, enabling deterministic steps to read prior comment history without an LLM turn.
- The activation artifact (containing `prompt.txt`, `base/` snapshot, and engine-specific directories) is also available earlier, which unblocks future use cases that need prompt injection in early steps.
- The root cause (empty config at script execution time) is eliminated — config is embedded at compile time, not resolved at runtime from a later-phase output.
- The change is validated by a new regression test (`TestCommentMemoryBeforeCustomSteps`) that enforces step ordering in the compiled output.

#### Negative
- The compiler now writes two different versions of `safeoutputs/config.json` at different phases: a minimal early version (compile-time embedded) and the full version (MCP setup). The dual-write is harmless today but introduces a subtle invariant that future maintainers must understand to avoid breaking the ordering.
- The activation artifact download step is now only in Phase 2; any future Phase 3 code that assumed the artifact was downloaded in that phase would need updating.

#### Neutral
- All 258 `.lock.yml` workflow golden files are regenerated as a consequence of the step order change — the diff is large but mechanically produced by the compiler.
- The `TestCheckoutRuntimeOrderInCustomSteps` test was updated for the new 9-step ordering, reflecting the insertion of the activation artifact download step at position 6.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
