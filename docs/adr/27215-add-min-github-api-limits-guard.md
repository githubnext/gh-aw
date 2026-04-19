# ADR-27215: Add Pre-Execution GitHub API Rate Limit Guard to CLI Commands

**Date**: 2026-04-19
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `logs` and `audit` CLI commands issue many GitHub API calls during execution. When a user invokes one of these commands with the API budget nearly exhausted, the command fails partway through with a generic rate-limit error — leaving partial results on disk and providing no advance warning. Users running large batch jobs or CI automation need a way to prevent wasted work when the API budget is insufficient to complete the operation.

### Decision

We will add a `--min-github-api-limits` integer flag to the `logs` and `audit` commands. When set to a positive value, each command queries the GitHub API rate-limit endpoint before beginning its main work and cancels execution immediately with a clear error message if the remaining core-request budget is below the specified minimum. A value of `0` (the default) disables the guard entirely, preserving backward compatibility. A shared helper (`addMinGitHubAPILimitsFlag`) registers the flag consistently on every participating command, and a shared function (`guardGitHubAPIRateLimit`) encapsulates the check-and-cancel logic so it can be reused and unit-tested in isolation.

### Alternatives Considered

#### Alternative 1: Reactive Rate-Limit Error Handling

Catch `403 / rate-limit-exceeded` responses mid-execution and surface a better error message at that point, rather than checking upfront. This was not chosen because it does not prevent wasted work: partial downloads, partial report generation, and side-effects may have already occurred before the limit is hit. Pre-execution checks give users a fast, clean failure before any state is modified.

#### Alternative 2: Hard-Coded Rate-Limit Threshold

Apply a fixed minimum (e.g., 200 requests) checked automatically for every command invocation, without a user-visible flag. This was not chosen because the appropriate threshold varies widely by use case — a single-run audit needs far fewer API calls than a bulk logs download. A configurable flag lets callers right-size the guard to their workload; setting the default to `0` keeps existing invocations unaffected.

#### Alternative 3: Middleware / Cobra Pre-Run Hook

Register the guard as a global `PersistentPreRunE` on the root command or as Cobra middleware, so all subcommands inherit it automatically. This was not chosen because not all subcommands are API-intensive; applying the guard broadly would add latency to lightweight commands that do not need it. Explicit per-command registration via the shared helper is more targeted and easier to audit.

### Consequences

#### Positive
- Commands fail fast and cleanly before side-effects occur when the API budget is insufficient.
- The configurable minimum lets callers tune the guard to their workload without code changes.
- The injectable `fetchRateLimitFn` variable makes the guard fully unit-testable without network access.
- Verbose mode prints a confirmation message showing remaining budget, aiding debugging.

#### Negative
- Each guarded command makes one additional API call (to `/rate_limit`) at startup, consuming one request from the budget it is trying to protect.
- Adding the flag to future commands requires explicit wiring; there is no automatic coverage for new subcommands.
- Users who set an overly conservative threshold may see unnecessary cancellations on healthy API budgets.

#### Neutral
- The `fetchRateLimitFn` variable is package-level, meaning tests must restore it via `t.Cleanup`; this is a minor testing-hygiene convention to document for contributors.
- Flag registration is split between `flags.go` (helper) and each command file (call site); contributors must remember to call the helper when adding new API-intensive commands.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Rate Limit Guard Behavior

1. Implementations **MUST** treat a `--min-github-api-limits` value of `0` as "guard disabled" and **MUST NOT** query the rate-limit endpoint in that case.
2. Implementations **MUST** return an error and cancel execution before performing any primary work when the remaining GitHub API core-request budget is strictly less than the configured minimum.
3. Implementations **MUST** return an error when `--min-github-api-limits` is set to a negative integer.
4. Implementations **MUST NOT** modify any persistent state (files, GitHub resources) before the guard check passes.
5. Implementations **SHOULD** include the current remaining and total budget in the cancellation error message to aid diagnosis.
6. Implementations **SHOULD** print a confirmation message to stderr when verbose mode is active and the guard passes.

### Flag Registration

1. All CLI commands that make substantial GitHub API calls **SHOULD** register `--min-github-api-limits` via the shared `addMinGitHubAPILimitsFlag` helper.
2. The `addMinGitHubAPILimitsFlag` helper **MUST** register the flag with a default value of `0` and an integer type.
3. Implementations **MUST NOT** duplicate the flag registration logic outside of the shared helper.

### Testability

1. The rate-limit fetch function **MUST** be injectable (e.g., via a package-level function variable) so that unit tests can substitute a mock without network access.
2. Tests **MUST** restore the original fetch function after each test case (e.g., via `t.Cleanup`).

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24637462329) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
