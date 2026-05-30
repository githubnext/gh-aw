# ADR-26688: Configurable Expiration for Action Failure Issues via aw.json

**Date**: 2026-04-16
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The GitHub Agentic Workflows platform creates GitHub issues when workflow runs fail (via the conclusion job and `handle_agent_failure.cjs`). Previously the lifetime of these failure issues was hardcoded to 7 days (168 hours), with no mechanism for a repository to adjust that duration. As gh-aw adoption broadened, teams with shorter or longer incident-retention requirements had no recourse: short-lived demo repos accumulated stale issues, while compliance-sensitive repos wanted longer retention. A repository-level configuration mechanism was needed to let each repo tune this value without forking the platform.

### Decision

We will extend the `aw.json` repository-level configuration schema with a new integer field `maintenance.action_failure_issue_expires` (unit: hours, minimum: 1) that controls the lifespan of action failure issues — including grouped parent issues — created by the conclusion job. When the field is absent, the system defaults to `168` (7 days), preserving existing behavior for repos that do not configure it. During workflow compilation, `buildConclusionJob` reads the value from `aw.json` via `LoadRepoConfig` and injects it as the `GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS` environment variable into the generated conclusion-job YAML; `handle_agent_failure.cjs` reads this variable at runtime and applies it when creating or updating failure issues.

### Alternatives Considered

#### Alternative 1: Keep the hardcoded 7-day expiration

Retaining the hard-coded value is the simplest option and requires no code changes. It was rejected because it offers no flexibility to repositories with different operational or compliance requirements — the status quo that motivated this feature request.

#### Alternative 2: Expose expiration as a workflow-level input parameter

Each generated workflow could expose an `action_failure_issue_expires` input that callers set when invoking the workflow. This surfaces the knob at the call site but requires every repo that wants a non-default value to edit every workflow file individually, and provides no central place to audit the setting. It was rejected in favor of the single aw.json configuration point.

#### Alternative 3: Use a repository secret or variable

Setting `GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS` as a GitHub Actions repository variable would achieve the runtime effect without schema changes. However, this mixes operational configuration (issue retention) with the secrets/variables store, bypasses schema validation, and is harder to document and discover. It was rejected because aw.json is the established, validated configuration surface for repository-level gh-aw settings.

### Consequences

#### Positive
- Repository owners can tune failure-issue lifetime to match their operational or compliance needs without modifying platform code.
- The JSON schema enforces `minimum: 1`, preventing zero or negative values from reaching the runtime handler.
- Backward compatibility is fully preserved: omitting the field is equivalent to the previous hardcoded behavior.

#### Negative
- The compilation path now carries an additional piece of runtime configuration from `aw.json` through `buildConclusionJob` into generated workflow YAML, adding a new data-flow dependency to track and test.
- Any repo that sets `action_failure_issue_expires` now has a build-time dependency on `aw.json` being valid JSON; a malformed file causes the compiler to fall back to the default with a warning rather than failing loudly, which could silently mask misconfiguration.

#### Neutral
- The `DefaultActionFailureIssueExpiresHours` constant (`24 * 7 = 168`) is now an exported symbol, making it accessible to other packages that may need the canonical default.
- Documentation in `docs/src/content/docs/guides/ephemerals.md` is updated to describe the new field alongside the existing `runs_on` maintenance option.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Schema and Configuration

1. The `aw.json` schema **MUST** accept an `action_failure_issue_expires` integer field nested under `maintenance`, with a minimum value of `1`.
2. Implementations **MUST NOT** accept a value of `0` or any negative integer for `action_failure_issue_expires`; schema validation **MUST** reject such values with an error.
3. When `action_failure_issue_expires` is absent from `aw.json`, implementations **MUST** behave as if the value is `168` (the default expiration of 7 days).

### Compilation and Environment Variable Propagation

1. The workflow compiler **MUST** read `action_failure_issue_expires` from `aw.json` (via `LoadRepoConfig`) during `buildConclusionJob` and inject the resolved value as the `GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS` environment variable into the generated conclusion-job YAML.
2. If `aw.json` cannot be loaded or parsed, the compiler **MUST** log a warning and fall back to the default value of `168`; it **MUST NOT** fail the build.
3. The injected environment variable value **MUST** be a positive integer string representing hours.

### Runtime Handling

1. `handle_agent_failure.cjs` **MUST** read `GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS` from the process environment at runtime to determine the expiration for both grouped parent issues and per-run failure issues.
2. If the environment variable is absent, empty, or not a positive integer, the handler **MUST** fall back to the default of `168` hours.
3. The resolved expiration **MUST** be applied consistently to all failure issues created or updated within the same workflow run (parent and child issues **MUST NOT** use different expiration values).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24522565038) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
