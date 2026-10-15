# ADR-69048: Reuse the logs date resolver for `health` time windows

**Date**: 2026-02-10
**Status**: Draft
**Deciders**: Unknown (drafted from PR #69048 by the Design Decision Gate)

---

### Context

`gh aw health` could only sample workflow runs over three fixed horizons (`--days 7|30|90`) counted backward from "now". That makes it impossible to answer bounded questions such as "how have workflows behaved since the latest release?", because the evidence window cannot be pinned to an explicit start or end instant. The `logs` command already supports precise windows through a shared relative-date resolver (`resolveLogsDateRange` / `parseFilterDate`) that accepts `YYYY-MM-DD` dates, RFC3339 timestamps, and relative offsets such as `-24h`. Health must also keep its resolved bounds stable across paginated GitHub API calls, so the window has to be computed once and carried through collection and reporting rather than recomputed per page. Health remains a bounded current-revision sample, not an exhaustive historical report.

### Decision

We will add `--start-date` and `--end-date` to `gh aw health` and resolve them with the existing `logs` date resolver instead of introducing a health-specific date syntax. A new `pkg/cli/health_time_window.go` owns window resolution: it validates `--days` (still 7/30/90), resolves both bounds to UTC, expands a date-only `--end-date` to cover the whole UTC day, and derives `start = end - days` when no explicit start is given. An explicit `--start-date` replaces the default horizon and is declared mutually exclusive with `--days` via `MarkFlagsMutuallyExclusive`; the resolved RFC3339 bounds are stored on `HealthConfig`/the collection struct so every API page, the text report, and the JSON report use identical boundaries.

### Alternatives Considered

#### Alternative 1: Extend the `--days` enum with more fixed horizons

Adding values such as 1, 14, or 180 would be a smaller change and keeps a single flag. It was rejected because no finite enum can express "since release tag X at 04:49:11Z"; the underlying need is an arbitrary instant, not a coarser grid. It would also keep health's vocabulary divergent from `logs`.

#### Alternative 2: Implement a health-specific date parser

A dedicated parser inside the health package would let health diverge (for example, rejecting relative offsets or defaulting differently). It was rejected because users already learned the `logs` syntax, and a second parser would duplicate RFC3339/relative-time handling and risk subtly different semantics between two commands that are frequently used together. The close call is that reuse now couples health to `logs` parsing changes.

### Consequences

#### Positive

- Health and logs share one date vocabulary (`YYYY-MM-DD`, RFC3339, relative such as `-24h`), so knowledge transfers between commands.
- Release-scoped and incident-scoped investigations become expressible, and the exact resolved bounds are emitted in both text and JSON output for reproducibility.
- Resolving the window once keeps paginated fetches consistent and avoids drift from a moving "now" during long collections.

#### Negative

- Health now depends on `logs` date-resolution internals; a change to `resolveLogsDateRange` or `parseFilterDate` can alter health behaviour without any change in the health package.
- The flag surface grew and carries non-obvious rules (`--days` mutually exclusive with `--start-date`, date-only `--end-date` silently extended to end-of-day, `--days` otherwise counted backward from `--end-date`), which must be documented and kept in sync with `docs/src/content/docs/setup/cli.md`.

#### Neutral

- `--days` remains restricted to 7/30/90 when no `--start-date` is supplied, so existing invocations behave exactly as before.
- Health stays a bounded current-revision sample; wide windows do not turn it into an exhaustive historical or compiler-version report.
- Window semantics are covered by regression tests in `pkg/cli/health_time_window_test.go` rather than by end-to-end health tests.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
