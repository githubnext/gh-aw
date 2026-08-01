# ADR-49495: Forward timeout-minutes to the agentic_execution Step in Every Engine

**Date**: 2026-08-01
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The gh-aw platform generates GitHub Actions workflow YAML through engine implementations (codex, gemini, pi, antigravity, claude, copilot). Each engine exposes a `GetExecutionSteps` method that emits the step lines for its `agentic_execution` step. Workflow authors may set `timeout-minutes` in the workflow frontmatter to cap how long an agent run can execute.

Before this change, `claude_engine.go` and `copilot_engine_execution.go` already forwarded the frontmatter `timeout-minutes` to the generated step YAML. The four remaining engines (codex, gemini, pi, antigravity) did not. Without an explicit step-level `timeout-minutes`, GitHub Actions inherits the job-level or workflow-level timeout (up to 360 minutes), making the frontmatter field a silent no-op for those engines. Users who set a `timeout-minutes` expecting it to be enforced would see their agents run for up to 6 hours.

### Decision

We decided that every engine's `GetExecutionSteps` must emit an explicit `timeout-minutes` field on the `agentic_execution` step. When the workflow author specifies a value, it is forwarded verbatim (including GitHub Actions expression syntax). When no value is specified, the step falls back to `DefaultAgenticWorkflowTimeout` (20 minutes). This makes timeout behavior consistent and predictable across all engines.

The fix was applied engine-by-engine using the pattern already established in `claude_engine.go`, and a cross-engine regression test (`TestAllEnginesEmitTimeoutMinutes`) was added to the test suite to enforce the invariant for any engines added in the future.

### Alternatives Considered

#### Alternative 1: Fix only the codex engine (the originally reported bug) and leave the others for follow-up PRs

The reported issue was specifically about the codex engine. Scoping the fix narrowly would have been lower risk per PR, but it would have left gemini, pi, and antigravity in the same broken state. Because the fix is a small, mechanical copy of an existing pattern, expanding the scope incurred negligible additional risk while closing the gap for all affected engines at once.

#### Alternative 2: Centralize timeout injection in a shared base struct or helper function

Rather than copying the if/else block into each engine file, a shared helper (e.g., `appendStepTimeout(stepLines []string, workflowData *WorkflowData) []string`) could be called by all engines. This would eliminate duplication and make future engines harder to get wrong.

This alternative was not taken because the PR aimed for a minimal, targeted fix with no refactoring. Introducing a shared abstraction would have enlarged the blast radius of the change during review. The regression test (`TestAllEnginesEmitTimeoutMinutes`) provides a safety net that catches any future engine omitting the field, partially mitigating the risk of ongoing copy-paste divergence.

### Consequences

#### Positive
- The `timeout-minutes` frontmatter field now works correctly for all four previously broken engines, honoring the user's intent.
- The cross-engine regression test `TestAllEnginesEmitTimeoutMinutes` enforces the timeout-emission contract for every currently registered engine and will catch any newly added engine that omits it.
- Compiled lock files for affected workflows (smoke-codex, smoke-gemini, smoke-pi, smoke-antigravity, and others) are regenerated, so production runs immediately benefit.

#### Negative
- The timeout-injection if/else block is now duplicated verbatim across four engine files (`codex_engine.go`, `gemini_engine.go`, `pi_engine.go`, `antigravity_engine.go`). Any future change to timeout logic (e.g., new fallback rules, format changes) must be applied to all four locations.
- The `strings.TrimPrefix(workflowData.TimeoutMinutes, "timeout-minutes: ")` parsing assumes a fixed prefix format. A value that does not start with `"timeout-minutes: "` would be forwarded unchanged, potentially producing valid but unexpected step YAML. No input validation was added to guard against this.

#### Neutral
- The default fallback value (20 minutes, from `DefaultAgenticWorkflowTimeout`) now appears explicitly in generated YAML for workflows that do not set `timeout-minutes`, making previously implicit behavior visible in lock files.
- Tests for the codex engine were duplicated in both `codex_engine_test.go` (engine-specific) and `agentic_engine_test.go` (cross-engine). The intent is different (targeted vs. registry-wide coverage), but the overlap may cause confusion for future test authors.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
