# ADR-54816: Bound Generated Agent and Detection Jobs with a Configurable Timeout Precedence Chain

**Date**: 2026-08-22
**Status**: Draft
**Deciders**: Unknown

---

### Context

Generated GitHub Actions workflows include `agent` and `detection` jobs that run potentially long-lived agentic execution steps. Previously, `timeout-minutes` in the gh-aw workflow configuration only bounded the `agentic_execution` step itself; setup steps and other non-agentic job steps had no explicit upper bound and could run up to GitHub Actions' default 6-hour job timeout. This created a risk where a slow or hung setup step (e.g., slow checkout, tool installation) could hold a runner indefinitely, consuming capacity and incurring uncontrolled costs. Users also needed a way to express different expected durations per workflow without a single global setting.

### Decision

We will emit an explicit job-level `timeout-minutes` on both `agent` and `detection` jobs, resolved independently from the agentic execution step timeout so a job budget covers setup and teardown without inheriting the step default:

- `agent` job: `jobs.agent.timeout-minutes` → `vars.GH_AW_DEFAULT_AGENT_JOB_TIMEOUT_MINUTES` → 60-minute fallback.
- `detection` job (and its execution step): `jobs.detection.timeout-minutes` → `vars.GH_AW_DEFAULT_DETECTION_JOB_TIMEOUT_MINUTES` → 10-minute fallback.
- `agentic_execution` step: top-level `timeout-minutes` → `vars.GH_AW_DEFAULT_TIMEOUT_MINUTES` → 20-minute fallback.

Every default is emitted as a `${{ vars.* || '<default>' }}` expression so organizations and enterprises can override it with GitHub Actions variables without recompiling. When top-level `timeout-minutes` explicitly requests a longer agentic step than the built-in agent job default, that built-in default is raised to match so an explicit step budget is never truncated.

### Alternatives Considered

#### Alternative 1: Keep Step-Level Timeout Only (Status Quo)

The `agentic_execution` step would remain the only bounded unit. Setup, cleanup, and other job steps would still be subject to GitHub Actions' 6-hour default. This is simple and requires no new configuration schema, but it means a hung setup step can silently exhaust runner resources without the workflow-author's timeout configuration having any effect, defeating the purpose of the `timeout-minutes` field for cost management.

#### Alternative 2: Hardcode a Fixed Job-Level Timeout in the Compiler

The compiler could emit a constant `timeout-minutes: 60` on all generated jobs regardless of configuration. This would bound the full job without adding any new configuration surface. However, it is inflexible: workflows with legitimately long setup or long multi-step agentic runs (e.g., 180-minute persona-explorer workflows) would be arbitrarily killed, and there is no way for users to opt into a longer or shorter limit without changing the compiler itself.

#### Alternative 3: Share a Single Timeout Between Job and Step

The job and its agentic execution step could share one resolved value, so top-level `timeout-minutes` would bound both. This keeps a single knob, but it conflates two different budgets: the step budget bounds model execution, while the job budget must also cover checkout, tool installation, artifact upload and log parsing. Sharing one value either starves setup or silently grants the agent more wall-clock time than the author configured.

### Consequences

#### Positive
- All steps within `agent` and `detection` jobs, including setup and cleanup, are now bounded by the configured timeout; no more unbounded runner time due to non-agentic step hangs.
- Configurable at multiple granularities (per-job, per-workflow, org-level variable, built-in fallback), allowing different workflows with different expected durations to coexist.
- Every default is overridable with a GitHub Actions variable, so organization administrators can set global agent step, agent job and detection job budgets without modifying individual workflow frontmatter.

#### Negative
- Job-level timeout now encompasses setup cost: if setup takes 5 minutes and the agent job timeout is 30 minutes, the agent gets at most 25 minutes to run. Authors must account for setup time when setting `jobs.agent.timeout-minutes`.
- Three defaults instead of one increase the configuration surface authors must learn.
- Regenerating all lock files with updated timeout values produces a large PR surface (322 files changed), making code review harder and increasing the risk of unrelated merge conflicts in generated files.

#### Neutral
- Schema, frontmatter, and cost-management documentation must be updated to reflect the new configurable fields and their precedence order.
- The 20-minute step fallback matches the previous hardcoded step timeout default, so existing workflows that relied on the default see no change to their agentic step budget; they only gain explicit job bounds.
- The detection job's 10-minute default also bounds its execution step, which previously inherited the top-level workflow timeout.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
