# ADR-29776: Handle Real Filesystem Errors in filepath.Walk Log-Scanning Functions

**Date**: 2026-05-02
**Status**: Draft
**Deciders**: pelikhan (PR author)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `pkg/cli/` package contains 8+ functions that use `filepath.Walk` to locate log files (e.g., `events.jsonl`, `token-usage.jsonl`, `redacted-urls.log`) within CI run directories. All of these call sites previously used `_ = filepath.Walk(...)`, discarding the walk's return value entirely. This meant real OS-level errors — such as permission denied or I/O failures — were silently swallowed and treated identically to the expected early-stop sentinel values (`errors.New("stop")`, `errors.New("found")`). Additionally, each call site defined its own inline sentinel string, creating inconsistency and making the code harder to reason about.

### Decision

We will capture the return value of every `filepath.Walk` call in `pkg/cli/` and log any error that is not a recognized early-stop sentinel (`errWalkStop` or `filepath.SkipAll`). We will consolidate all early-stop sentinel values into a single package-level variable `errWalkStop` defined in `logs_parsing_core.go` and shared across all walk-based file-search functions in the package. Per-entry walk errors (passed as the `err` argument to the callback) will be logged and skipped rather than silently ignored.

### Alternatives Considered

#### Alternative 1: Continue Discarding Walk Errors (`_ = filepath.Walk(...)`)

The existing approach requires zero code changes and keeps the call sites minimal. It was rejected because it silently hides real filesystem failures — a permission-denied error on a run directory would produce no observable signal, making it difficult to diagnose why a log file was not found during debugging or incident response.

#### Alternative 2: Propagate Errors from Walk Functions to Callers

Walk functions could be changed to return an `error` so callers can react programmatically to filesystem failures. This was considered but rejected because all current callers treat "file not found" as a soft failure (falling back to an empty result), and plumbing errors through a large number of caller sites would require a broader refactor with limited immediate benefit. A log-only approach surfaces the information without disrupting call-site contracts.

### Consequences

#### Positive
- Real filesystem errors (permission denied, I/O failures) are now observable through the package logger, enabling faster diagnosis of missing-file issues in production and CI environments.
- Consolidating scattered inline `errors.New("stop")` / `errors.New("found")` literals into a single `errWalkStop` sentinel eliminates inconsistency and makes sentinel comparisons reliable across all walk callbacks.

#### Negative
- Walk failures only surface as log messages; calling code cannot react programmatically to failures. If a walk fails for a systemic reason (e.g., the entire run directory is inaccessible), the caller will silently receive an empty result with no indication beyond a log line.
- Two distinct sentinel mechanisms coexist (`errWalkStop` for functions that used inline literals, `filepath.SkipAll` for functions that already used the stdlib sentinel), which requires per-call-site awareness of which sentinel to test in the post-walk error check.

#### Neutral
- The `errors` package import is added to files that previously relied on it only indirectly; this is a minor dependency change with no behavioral impact.
- The change is purely internal to `pkg/cli/` and does not affect any public API, CLI output, or caller contracts.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Walk Error Handling

1. Implementations **MUST NOT** discard the return value of `filepath.Walk` using `_ = filepath.Walk(...)` in `pkg/cli/`.
2. Implementations **MUST** capture the return value of every `filepath.Walk` call and check whether it is non-nil and not a recognized early-stop sentinel.
3. Implementations **MUST** log any non-sentinel walk error using the package-local logger at `Printf` level, including the directory being walked and the error value.
4. Implementations **MUST NOT** return a fatal error or panic in response to a walk failure; the function **MUST** continue and return its best-effort result (empty string or zero value) when the walk fails.

### Sentinel Values

1. Implementations **MUST** use the package-level `errWalkStop` sentinel (defined in `logs_parsing_core.go`) as the early-exit return value for walk callbacks that previously used inline `errors.New("stop")` or `errors.New("found")` literals.
2. Implementations **MUST NOT** introduce new inline `errors.New(...)` sentinel literals in walk callbacks within `pkg/cli/`; the shared `errWalkStop` **MUST** be used instead.
3. Implementations **MAY** use `filepath.SkipAll` as the early-exit sentinel in walk callbacks where the intent is to skip all remaining entries (as opposed to stopping with a found result), provided the post-walk error check uses `errors.Is(walkErr, filepath.SkipAll)` to filter it out.

### Per-Entry Error Handling

1. Walk callback implementations **MUST** handle a non-nil `err` argument (per-entry OS error) explicitly, rather than combining the per-entry error check with a `nil` info check in a single condition.
2. Walk callback implementations **SHOULD** log per-entry errors using the package-local logger before returning `nil` to continue the walk.
3. Walk callback implementations **MUST NOT** return a non-nil, non-sentinel error from the callback in response to a per-entry OS error, as doing so would abort the entire walk and surface the per-entry error as the walk's return value.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25255300701) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
