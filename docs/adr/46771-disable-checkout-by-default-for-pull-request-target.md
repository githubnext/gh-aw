# ADR-46771: Disable PR Branch Checkout by Default for pull_request_target Workflows

**Date**: 2026-07-20
**Status**: Draft
**Deciders**: pelikhan (via Copilot SWE agent, PR #46771)

---

### Context

The `pull_request_target` GitHub Actions trigger runs in the context of the base branch, which is a security boundary that enables accessing repository secrets from fork PRs. The workflow compiler previously generated a "Checkout PR branch" step by default whenever the workflow had `contents: read` permission, regardless of the trigger type.

For `pull_request_target` workflows, the "Checkout PR branch" step hard-fails in two common scenarios: (1) when the event type is `closed`, because the head branch is deleted on merge; and (2) on any fork PR, because the fork's head branch is inaccessible from the base repository. A pre-existing bug compounded this: the `checkout: false` frontmatter key was not suppressing the PR checkout step — it only skipped the default `actions/checkout` — so users who set `checkout: false` still saw the step generated and failing.

### Decision

We will auto-disable the "Checkout PR branch" step for `pull_request_target` workflows at compile time when no explicit `checkout:` key is present in the frontmatter and no checkout configurations are set. Users who require PR branch checkout for `pull_request_target` can opt in by providing an explicit checkout mapping (e.g., pinning to `base.sha`). We will also fix the `ShouldGeneratePRCheckoutStep` function to respect the `CheckoutDisabled` flag, so `checkout: false` in frontmatter correctly suppresses the step.

### Alternatives Considered

#### Alternative 1: Make the Checkout Step Resilient with `continue-on-error`

Add `continue-on-error: true` to the generated "Checkout PR branch" step, or add explicit error handling in the step script, so that failures on merged/fork PRs do not fail the whole workflow.

This was not chosen because it silently hides failures — the step would report as successful even when checkout did not happen, making downstream steps that depend on the checked-out state behave unpredictably. It also does not address the security concern of attempting to access fork branch content in a `pull_request_target` context.

#### Alternative 2: Runtime Detection and Dynamic Skip

Detect at runtime (inside the step) whether the head branch is accessible — checking if the PR is merged or from a fork — and emit a skip-with-warning rather than executing checkout.

This was not chosen because it adds runtime complexity and latency (the check requires an API call), and the failure condition can be determined statically from the trigger type. Compile-time disabling is simpler, more predictable, and avoids unnecessary step execution in the generated workflow YAML entirely.

### Consequences

#### Positive
- `pull_request_target` workflows no longer hard-fail on `closed` events or fork PRs due to the checkout step attempting to access a deleted or inaccessible branch.
- The pre-existing bug where `checkout: false` did not suppress the "Checkout PR branch" step is fixed; `CheckoutDisabled` is now respected by `ShouldGeneratePRCheckoutStep`.
- Reduces the attack surface from insecure checkouts in `pull_request_target` contexts, since checking out fork code in this trigger is a known security anti-pattern.

#### Negative
- Existing `pull_request_target` workflows that previously relied on the implicit checkout step (without an explicit `checkout:` key) will have the step removed after recompile. In practice these workflows were already failing on closed/fork events, so the blast radius is low.
- Users who intentionally need checkout in a `pull_request_target` context must now explicitly configure it, adding a small onboarding friction for new workflows.

#### Neutral
- The `ai-moderator.lock.yml` compiled workflow is updated to reflect the removal of the checkout step and its associated `checkout_pr_success` output variable and downstream references.
- Validation test expectations for `pull_request_target` are updated: cases that previously emitted an "insecure checkout" error/warning now only emit the "dangerous-trigger" warning, since checkout is no longer auto-generated.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
