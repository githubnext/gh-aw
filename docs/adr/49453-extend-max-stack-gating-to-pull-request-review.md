# ADR-49453: Extend `max-stack` Gating to `pull_request_review` Events

**Date**: 2026-08-01
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The gh-aw workflow compiler introduced stacked PR support (ADR from PR #49420) that gates workflow runs using a `max-stack` filter on `pull_request` events: by default only the top-of-stack PR triggers runs, with `max-stack: N` to allow the top N layers and `max-stack: -1` to disable filtering entirely.

However, this gating was only applied to `pull_request` event triggers. `pull_request_review` workflows were left ungated, so review-triggered runs executed on every layer of a stacked PR chain — not just the top. This broke the stack filter contract: a review submitted on an intermediate PR would trigger all workflows on that layer even when `max-stack: 1` was set.

The `on.pull_request_review` section also lacked a `max-stack` configuration key in the workflow schema, so the setting could not be configured per-workflow.

### Decision

We will extend the `max-stack` gating logic to cover `pull_request_review` events, using the same semantics already established for `pull_request`:

- Default (`max-stack: 1`): only the top-of-stack PR triggers the workflow.
- `max-stack: N`: the top N layers trigger the workflow.
- `max-stack: -1`: stack gating is disabled and all layers trigger.

The `on.pull_request_review.max-stack` key is added to the workflow schema with the same value domain (`-1` or `>= 1`; `0` and other negative values are rejected). Compiled `.lock.yml` files are regenerated to apply the new `if:` conditions, and trigger-detection and `max-stack`-extraction paths are updated to handle both object and array `on:` forms consistently.

### Alternatives Considered

#### Alternative 1: Keep `pull_request_review` Ungated

Leave review-event workflows running on every stack layer, accepting the inconsistency with `pull_request` event behavior.

This was the status quo before this PR. It was rejected because it violates the stated intent of `max-stack`: users who configure stack gating on a workflow expect it to apply to all event types that workflow listens to, not just `pull_request`. Ungated review runs on lower stack layers create duplicate workflow execution and noisy CI feedback.

#### Alternative 2: Separate `max-stack` Semantics for Review Events

Introduce a distinct configuration key (e.g., `max-stack-review`) or different default behavior for `pull_request_review`, decoupling it from `pull_request` gating.

This was rejected because it adds schema complexity without a clear benefit. The stack position concept is identical for both events (both carry `github.event.pull_request.stack`), and users configuring one event type expect the other to behave consistently. A separate knob would require users to duplicate configuration and reason about two separate gating policies.

### Consequences

#### Positive

- `pull_request_review` workflows now respect `max-stack` gating, closing the gap in the stack filter contract.
- Schema is extended consistently, so `max-stack` is configurable for review events just as it is for PR events.
- Reduces redundant CI runs on lower stack layers when reviews are submitted, lowering resource consumption.
- Test coverage added for review-event stack filtering, default and configured `max-stack`, and schema validation.

#### Negative

- Workflows that previously ran on every stack layer when a review was submitted now run only on the top layer by default. Teams relying on the old (ungated) `pull_request_review` behavior may see unexpected workflow skips after this change.
- All affected compiled `.lock.yml` files must be regenerated, producing a large number of boilerplate changes across the repository that obscure the core logic change.

#### Neutral

- The `if:` condition expansion (`github.event_name != 'pull_request'` → `github.event_name != 'pull_request' && github.event_name != 'pull_request_review'`) is a mechanical change applied uniformly across every compiled workflow, making bulk review the only practical way to audit it.
- Unit and integration tests are updated/added to cover the new behavior; existing `pull_request` test coverage is not affected.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
