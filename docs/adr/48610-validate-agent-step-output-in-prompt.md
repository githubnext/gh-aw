# ADR-48610: Compile-Time Validation of Agent-Job Step Outputs in Prompt Body

**Date**: 2026-07-28
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

gh-aw workflows have a two-job architecture: the **activation job** renders the prompt (markdown body) and starts the agent; the **agent job** runs the steps defined in `steps:`, `pre-steps:`, `pre-agent-steps:`, and `post-steps:` sections. Because the prompt is rendered before the agent job executes, any `${{ steps.STEP_ID.outputs.* }}` expression referencing an agent-job step silently resolves to an empty string at prompt-creation time. This is a confusing silent failure: the workflow compiles successfully, the agent receives an incomplete prompt, and the root cause is non-obvious. Authors familiar with standard GitHub Actions expect step outputs to be available in surrounding content, and this architectural constraint is not surfaced until runtime (if at all).

### Decision

We will add a compile-time validator (`validateStepsOutputsNotInPrompt`) that scans the prompt body for `${{ steps.STEP_ID.* }}` expressions and cross-references the step ID against the set of IDs declared in agent-job step sections. If a match is found, compilation is rejected with an actionable error that names the offending step and directs authors to the file-based alternative (`/tmp/gh-aw/agent/result.txt`). Built-in activation-job steps (e.g., `steps.sanitized`) are not flagged because they do not appear in user-visible step lists. The validator is wired into the existing `validateExpressions` gate in `compiler_validators.go`.

### Alternatives Considered

#### Alternative 1: Runtime Warning Without Compilation Failure

Detect the pattern at compile time but emit only a log warning rather than a hard error, allowing the workflow to proceed. Authors would see a warning in CI logs but the workflow would still run with an empty prompt value.

This was rejected because a warning is too easy to overlook, and the resulting runtime behavior (silently empty prompt variable) is indistinguishable from an intentional empty value. The goal is to prevent a class of hard-to-debug bugs, which requires failing loudly at the earliest opportunity.

#### Alternative 2: Documentation Only, No Enforcement

Document the two-job execution order in the workflow authoring guide and rely on authors to internalize the constraint without automated enforcement.

This was rejected because it scales poorly: each new author must discover the limitation individually (typically after a confusing empty-prompt incident), and there is no mechanism to prevent the pattern from being introduced by automated tooling or copy-paste from standard GitHub Actions workflows.

### Consequences

#### Positive
- Prevents the silent empty-prompt class of bugs at the earliest point in the authoring cycle (compile time), before the workflow is ever run.
- Provides an actionable error message that names the offending step ID and links directly to the file-based alternative, reducing debugging time.
- Makes the two-job execution boundary explicit in the toolchain, reinforcing the architectural constraint for all authors.

#### Negative
- Any workflow that currently relies on the silent-empty behavior (however unintentionally) will fail compilation after this change is deployed, requiring a migration to file-based data passing.
- The YAML parsing step in `addStepIDsFromYAML` adds a small compile-time cost for every workflow that contains step sections, even when the prompt contains no `${{ steps.* }}` expressions (mitigated by an early-exit fast path).

#### Neutral
- The validator only flags agent-job step IDs that appear in the user-visible step YAML sections; built-in activation-job step references (e.g., `${{ steps.sanitized.outputs.text }}`) pass through unchanged, so existing valid workflows are unaffected.
- The file-based pattern (`/tmp/gh-aw/agent/result.txt`) that replaces the forbidden pattern was already the recommended approach; this change codifies that recommendation as an enforcement rule.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
