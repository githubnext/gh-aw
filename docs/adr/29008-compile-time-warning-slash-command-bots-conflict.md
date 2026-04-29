# ADR-29008: Compile-Time Warning for slash_command + bots Conflict

**Date**: 2026-04-29
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

Agentic workflow frontmatter supports two independent trigger mechanisms: `slash_command` (explicit manual invocation by a user) and `bots` (automatic invocation when a named bot actor performs a matching event). When both are configured in the same workflow, bot activity — such as `copilot[bot]` opening a pull request — can match the `slash_command` event types (e.g., `pull_request`, `pull_request_comment`) and start the workflow automatically. This causes any subsequent manual slash command invocation to be silently ignored while the bot-triggered run is already in progress. The failure mode is invisible to the workflow author at authoring time and difficult to diagnose at runtime.

### Decision

We will emit a non-fatal compile-time warning in `validateToolConfiguration` whenever a workflow's parsed frontmatter contains both a non-empty `slash_command` (via `workflowData.Command`) and a non-empty `bots` list (via `workflowData.Bots`). The warning message names the conflict, provides a concrete example of how it manifests, and directs the author to remove the `bots:` field if exclusively slash-command-driven execution is intended. Using a warning rather than a hard error preserves backward compatibility for any workflow that intentionally wants both triggers.

### Alternatives Considered

#### Alternative 1: Hard Compiler Error

Reject compilation entirely when both `slash_command` and `bots` are present. This is more forceful and guarantees the conflict never reaches production, but it is a breaking change for any existing workflows that may have intentionally paired the triggers for a dual-invocation use case. A hard error would require all such authors to migrate immediately, with no grace period.

#### Alternative 2: Documentation-Only

Document the conflict in the workflow authoring guide without any code change. This is the least disruptive approach, but it provides no signal at the point of workflow authoring and leaves authors who have not read the documentation entirely exposed to the silent failure mode.

#### Alternative 3: Auto-Remove bots: at Compile Time

Silently strip the `bots:` field from the compiled output when `slash_command` is also present. This eliminates the conflict without requiring author action, but it modifies the workflow against the author's explicit configuration without consent or explanation, which is worse than the original problem.

### Consequences

#### Positive
- Authors receive an actionable, human-readable warning at compile time — before the workflow is deployed — rather than discovering the issue through silent runtime failures.
- The warning message includes both the root cause and the recommended fix, minimizing the debugging burden.

#### Negative
- The warning is non-fatal and can be ignored; a deployed workflow with both triggers will still exhibit the conflict if the author does not act on the warning.
- The check must be updated if the semantics of `slash_command` event routing change (e.g., if `slash_command` is decoupled from `pull_request` event types in a future compiler revision).

#### Neutral
- The warning is emitted to `stderr` using the existing `formatCompilerMessage` infrastructure, consistent with all other compiler diagnostics.
- Warning count is incremented via `IncrementWarningCount()`, so tooling that surfaces the aggregate warning count will reflect the new check without further changes.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Conflict Detection

1. The compiler **MUST** emit a warning diagnostic when a workflow's parsed frontmatter contains both a non-empty `slash_command` trigger (i.e., `workflowData.Command` is non-empty) and a non-empty `bots` list (i.e., `workflowData.Bots` has at least one entry).
2. The warning **MUST** be written to `stderr` using the `formatCompilerMessage` helper with severity `"warning"`.
3. The warning **MUST** increment the compiler's warning counter via `IncrementWarningCount()`.
4. Compilation **MUST NOT** fail (i.e., return a non-nil error) solely because this conflict is detected; the check **MUST** be non-fatal.
5. The warning message **MUST** identify the conflict, describe the failure mode (bot activity starting the workflow before a manual slash command), and provide a remediation step (removing the `bots:` field).

### Conflict Scope

1. The check **MUST** fire if and only if both conditions are true simultaneously: `len(workflowData.Command) > 0` AND `len(workflowData.Bots) > 0`.
2. A workflow with `slash_command` and no `bots` entries **MUST NOT** trigger the warning.
3. A workflow with `bots` entries and no `slash_command` **MUST NOT** trigger the warning.
4. The number of entries in `workflowData.Bots` (one vs. multiple) **MUST NOT** affect whether the warning fires; the warning **MUST** fire for any non-zero `bots` list size when `slash_command` is also present.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. In particular, converting the warning to a hard error or suppressing the `IncrementWarningCount()` call both constitute non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25085998674) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
