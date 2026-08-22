# ADR-54816: Bound Generated Agent and Detection Jobs with a Configurable Timeout Precedence Chain

**Date**: 2026-08-22
**Status**: Draft
**Deciders**: Unknown

---

### Context

Generated GitHub Actions workflows include `agent` and `detection` jobs that run potentially long-lived agentic execution steps. Previously, `timeout-minutes` in the gh-aw workflow configuration only bounded the `agentic_execution` step itself; setup steps and other non-agentic job steps had no explicit upper bound and could run up to GitHub Actions' default 6-hour job timeout. This created a risk where a slow or hung setup step (e.g., slow checkout, tool installation) could hold a runner indefinitely, consuming capacity and incurring uncontrolled costs. Users also needed a way to express different expected durations per workflow without a single global setting.

### Decision

We will propagate `timeout-minutes` to the job-level `timeout-minutes` field on both `agent` and `detection` jobs, in addition to each agentic execution step's step-level timeout. The compiler resolves the effective timeout through a fixed precedence chain: `jobs.agent.timeout-minutes` (or `jobs.detection.timeout-minutes`) → top-level `timeout-minutes` → `vars.GH_AW_DEFAULT_TIMEOUT_MINUTES` (a runtime GitHub Actions variable enabling org/enterprise-wide defaults) → 60-minute hard fallback. The same resolved value is applied to both the job and the execution step to keep them aligned.

### Alternatives Considered

#### Alternative 1: Keep Step-Level Timeout Only (Status Quo)

The `agentic_execution` step would remain the only bounded unit. Setup, cleanup, and other job steps would still be subject to GitHub Actions' 6-hour default. This is simple and requires no new configuration schema, but it means a hung setup step can silently exhaust runner resources without the workflow-author's timeout configuration having any effect, defeating the purpose of the `timeout-minutes` field for cost management.

#### Alternative 2: Hardcode a Fixed Job-Level Timeout in the Compiler

The compiler could emit a constant `timeout-minutes: 60` on all generated jobs regardless of configuration. This would bound the full job without adding any new configuration surface. However, it is inflexible: workflows with legitimately long setup or long multi-step agentic runs (e.g., 180-minute persona-explorer workflows) would be arbitrarily killed, and there is no way for users to opt into a longer or shorter limit without changing the compiler itself.

#### Alternative 3: Separate Job Timeout from Step Timeout

Job-level and step-level timeouts could be independently configurable (e.g., `jobs.agent.timeout-minutes` for the job, `jobs.agent.steps.agentic_execution.timeout-minutes` for the step). This would give the most control but significantly increases configuration surface area and mental overhead. In practice, users want the job and its primary step to share the same budget so setup time does not silently "eat into" untracked time.

### Consequences

#### Positive
- All steps within `agent` and `detection` jobs, including setup and cleanup, are now bounded by the configured timeout; no more unbounded runner time due to non-agentic step hangs.
- Configurable at multiple granularities (per-job, per-workflow, org-level variable, 60-minute fallback), allowing different workflows with different expected durations to coexist.
- The `vars.GH_AW_DEFAULT_TIMEOUT_MINUTES` variable enables organization administrators to set a global default without modifying individual workflow frontmatter.

#### Negative
- Job-level timeout now encompasses setup cost: if setup takes 5 minutes and the job timeout is 30 minutes, the agent gets at most 25 minutes to run — previously the full 30 minutes were available to the agent step alone. Authors must account for setup time when setting `timeout-minutes`.
- Regenerating all lock files with updated timeout values produces a large PR surface (322 files changed), making code review harder and increasing the risk of unrelated merge conflicts in generated files.

#### Neutral
- Schema, frontmatter, and cost-management documentation must be updated to reflect the new configurable fields and their precedence order.
- The 60-minute fallback matches the previous hardcoded step timeout default, so existing workflows that relied on the default see no behavioral change at the job level beyond gaining an explicit bound.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
