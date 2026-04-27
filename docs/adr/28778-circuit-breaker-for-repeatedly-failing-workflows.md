# ADR-28778: Circuit Breaker for Repeatedly Failing Agentic Workflows

**Date**: 2026-04-27
**Status**: Draft
**Deciders**: lpcox, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

Agentic workflows compiled by gh-aw run on GitHub Actions runners on a scheduled or event-driven basis. When a workflow is misconfigured, encounters a broken dependency, or has a systematic runtime error, it will continue triggering on every event and consuming runner resources indefinitely, because GitHub Actions has no native mechanism to suppress a workflow that fails repeatedly. This is a resource-exhaustion risk catalogued as OWASP Agentic Top-10 item ASI-08. The system needed an opt-in guard that workflow authors could add with a single frontmatter field and that did not introduce external infrastructure dependencies beyond what GitHub Actions already provides.

### Decision

We will implement the classic closed → open → half-open circuit breaker state machine as an opt-in frontmatter feature (`circuit-breaker: true` or a full object form). State is persisted across runs exclusively via GitHub Actions artifacts: a pre-activation job reads the previous state artifact from the most recent completed run, evaluates the state machine, and gates the `activated` output. After agent execution, two `if: always()` steps update the state and re-upload the artifact. This approach requires no infrastructure beyond a standard GitHub Actions token with `actions: read` permission.

### Alternatives Considered

#### Alternative 1: External State Store (Redis / Database)

A persistent external datastore (Redis, PostgreSQL, or similar) could track failure counts reliably across runs without artifact expiry or manual deletion risks. This was rejected because it introduces an infrastructure dependency that does not exist in the current gh-aw deployment model and would require every adopting repository to provision and credential-manage an external service, dramatically increasing the adoption barrier for what is an optional resilience feature.

#### Alternative 2: GitHub Repository/Environment Variables via the API

Failure counts could be written back to a GitHub Actions environment variable or repository variable via the REST API on each run. This avoids binary artifact management. It was rejected because updating repository or environment variables requires elevated `repo` or `admin` scopes that violate the principle of least privilege; environment variables also lack the structured JSON schema needed to store timestamps and state transitions cleanly.

#### Alternative 3: No Built-in Circuit Breaker (External Monitoring)

Teams could configure external alerting (Datadog, PagerDuty, etc.) to detect repeated failures and disable workflows manually. This was rejected because it places the burden on each team to set up monitoring, does not enforce a cooldown programmatically, and provides no self-healing (half-open) behaviour that allows automatic recovery once the root cause is fixed.

### Consequences

#### Positive
- Prevents resource exhaustion from runaway failing workflows with zero external infrastructure.
- Backward-compatible opt-in design: existing workflows are unaffected unless `circuit-breaker` frontmatter is added.
- Follows standard resilience engineering pattern (closed/open/half-open) that operators already understand.
- Self-healing: the half-open state allows automatic recovery after the cooldown period without manual intervention.
- Configurable thresholds (`max-consecutive-failures`, `time-window`, `cooldown`) let teams tune behaviour for their failure tolerance.

#### Negative
- State is stored in GitHub Actions artifacts, which are subject to repository retention policies and can be manually deleted, inadvertently resetting the circuit.
- Artifact I/O (list runs, list artifacts, download, upload) adds latency and API calls to every workflow execution that enables the feature.
- The `actions: read` permission is added automatically to the pre-activation job; teams with strict minimal-permission policies must be aware of this implicit grant.
- The feature relies on GitHub's artifact API being available; artifact service degradation could prevent state updates and cause the circuit to silently fail open.

#### Neutral
- The circuit breaker state is identified by workflow ID, so renaming a workflow file effectively resets the breaker.
- State inspection currently requires downloading the `circuit-breaker-state` artifact manually; no UI or CLI command is provided to read current state.
- Manual reset is performed by deleting the `circuit-breaker-state` artifact from the most recent run.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Feature Activation

1. The circuit breaker feature **MUST** be opt-in: workflows that do not declare `circuit-breaker` frontmatter **MUST NOT** have any circuit breaker steps injected into their compiled output.
2. A workflow **MAY** enable the circuit breaker with `circuit-breaker: true` (boolean shorthand using all defaults) or with a `circuit-breaker` object specifying explicit configuration fields.
3. A workflow **MAY** also enable the circuit breaker via `features.circuit-breaker: true`; this form **MUST** behave identically to `circuit-breaker: true`.
4. Setting `circuit-breaker: false` **MUST** be treated as explicitly disabled and **MUST NOT** inject any steps.

### State Machine

1. Implementations **MUST** implement the three-state closed → open → half-open circuit breaker state machine.
2. In the **CLOSED** state, implementations **MUST** allow workflow execution to proceed (output `circuit_breaker_ok=true`).
3. A circuit **MUST** transition from CLOSED to OPEN when `consecutive_failures` reaches or exceeds `max-consecutive-failures` within the configured `time-window`.
4. In the **OPEN** state, implementations **MUST** block workflow execution (output `circuit_breaker_ok=false`) until the `cooldown` period has elapsed since the last recorded failure.
5. When the `cooldown` period has elapsed, the circuit **MUST** transition to the **HALF-OPEN** state and allow exactly one probe execution (output `circuit_breaker_ok=true`).
6. A successful probe in HALF-OPEN **MUST** reset `consecutive_failures` to zero and return the circuit to CLOSED.
7. A failed probe in HALF-OPEN **MUST** increment `consecutive_failures` and return the circuit to OPEN with an updated `last_failure` timestamp.
8. If no previous state artifact is found, implementations **MUST** treat the circuit as CLOSED (fail-open for availability).

### State Persistence

1. Circuit breaker state **MUST** be persisted as a JSON artifact named `circuit-breaker-state` uploaded to GitHub Actions after every execution, regardless of job outcome.
2. The update step **MUST** use `if: always()` to ensure state is written even when the agent job fails or is cancelled.
3. The state JSON **MUST** include at minimum: `consecutive_failures` (integer), `last_failure` (ISO 8601 timestamp or null), and `circuit_opened_at` (ISO 8601 timestamp or null).
4. Implementations **MUST** use `overwrite: true` when uploading the artifact so that only the latest state is retained per run.
5. Implementations **SHOULD** use `if-no-files-found: ignore` on the upload step to tolerate cases where the state file could not be written.

### Configuration Defaults

1. When not explicitly specified, `max-consecutive-failures` **MUST** default to `5`.
2. When not explicitly specified, `time-window` **MUST** default to `24h`.
3. When not explicitly specified, `cooldown` **MUST** default to `1h`.
4. When not explicitly specified, `notify` **MUST** default to `true`.
5. Duration values for `time-window` and `cooldown` **MUST** be parseable by Go's `time.ParseDuration`. Implementations **MUST NOT** accept arbitrary string formats that cannot be parsed by this function.

### Permissions

1. When the circuit breaker is enabled, the pre-activation job **MUST** be granted `actions: read` permission to allow listing workflow runs and downloading artifacts.
2. Implementations **MUST NOT** require permissions beyond `actions: read` and the standard `GITHUB_TOKEN` for circuit breaker operation.

### Notifications

1. When `notify: true` (the default), implementations **MUST** emit a workflow error annotation via `core.error()` when the circuit is OPEN and blocking execution.
2. When `notify: false`, implementations **MUST NOT** emit error annotations for circuit breaker state changes.
3. Implementations **SHOULD** emit a GitHub Actions step summary when the circuit is OPEN, describing the failure count, time window, and estimated retry time.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance. Specifically: (a) the circuit breaker **MUST** be opt-in, (b) the three-state machine transitions **MUST** be correctly implemented, (c) state **MUST** be persisted via artifact after every execution, and (d) the pre-activation job **MUST** gate on `circuit_breaker_ok=true` when the feature is enabled.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25013046554) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
