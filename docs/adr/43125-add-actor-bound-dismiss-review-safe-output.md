# ADR-43125: Add Actor-Bound Dismiss-Review Safe Output

**Date**: 2026-07-03
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The safe-outputs system controls which GitHub API mutations AI agents can perform during workflow execution. Before this change, agents could submit PR reviews but had no way to dismiss them — requiring human intervention to clear stale reviews. Agents need to dismiss their own previously-submitted reviews when subsequent PR changes make those reviews outdated. The core constraint is that agents must only be permitted to dismiss reviews they themselves authored; allowing dismissal of arbitrary reviews would enable one actor to silence another's feedback, which is a material security and trust boundary.

### Decision

We will add `dismiss_pull_request_review` (aliased as `dismiss-review`) as a new safe-output type. The handler enforces three invariants before calling `pulls.dismissReview`: the `justification` must be at least 20 characters, the caller-supplied `author` field (if provided) must match the inferred workflow actor (`GITHUB_ACTOR`), and the review fetched from the API must also have been authored by that same actor. These checks are enforced at both the MCP pre-validation layer and the runtime execution layer.

### Alternatives Considered

#### Alternative 1: Allow dismissal of any review (no actor-bound constraint)

Agents could dismiss any review regardless of author, simplifying the implementation by removing identity checks. This was rejected because it would allow an agent to silently clear human reviewer feedback without authorization, undermining the review process and creating a significant privilege-escalation vector within the safe-outputs trust boundary.

#### Alternative 2: Omit review dismissal from safe-outputs entirely

Review dismissal could be left as a human-only operation. This preserves the narrowest possible safe-outputs surface area. This was rejected because it forces manual intervention in automated PR workflows where agents legitimately need to dismiss their own stale reviews after code updates, reducing the practical utility of the agent review system.

### Consequences

#### Positive
- Agents can automate the full review lifecycle (submit → update → dismiss) without requiring human intervention for stale-review cleanup.
- The actor-bound invariant (checked at both MCP and runtime layers) establishes a clear, auditable security boundary: an agent can only dismiss what it authored.

#### Negative
- The actor identity is resolved from `GITHUB_ACTOR` at runtime; workflows that run under different actors across re-runs may find the invariant unexpectedly restrictive if the actor changes between the review submission and the dismissal attempt.
- Adding another safe-output type increases the attack surface of the safe-outputs system and requires coordinated updates across config parsing, schema, type declarations, handler registry, validation, and prompt construction — a high-touch pattern that is easy to implement inconsistently.

#### Neutral
- The `dismiss-review` alias mirrors the naming convention used by `submit-pull-request-review` / `submit-review`, keeping the YAML surface consistent for workflow authors.
- The 20-character minimum for `justification` is enforced identically in both the MCP validation layer and the runtime handler, which adds redundancy but also defense-in-depth.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
