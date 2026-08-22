# ADR-54690: Shared Finding/SeverityLevel Type Across Scanner Integrations

**Date**: 2026-08-22
**Status**: Accepted
**Deciders**: copilot-swe-agent (PR author), gh-aw maintainers

---

### Context

The codebase integrates nine security scanners (zizmor, poutine, grype, grant, runner-guard, yamllint, audit findings, validation issues, markdown security scanner). Each integration defined its own finding struct with a different severity vocabulary (`High`, `error`, `Negligible`, `note`, …) and its own location shape. As a result, severity classification, message formatting, context-line extraction, and console rendering were reimplemented per tool. This made adding new scanners expensive and meant severity inconsistencies (e.g., `low`-severity zizmor findings rendered as `warning` while grype rendered equivalent findings as `info`) accumulated silently.

### Decision

We will introduce a new `pkg/scanfindings` package that provides a shared `SeverityLevel` enum (`unknown` < `info` < `low` < `medium` < `high` < `critical`) and a shared `Finding` struct (`RuleID`, `Severity`, `Message`, `File`, `Line`, `Column`, `Context`). Each scanner integration retains its own native structs for JSON decoding, then maps onto `scanfindings.Finding` via a small `…FindingsToShared` adapter function. Severity classification, message formatting (`FormatMessage`), context-line extraction (`ContextLines`), rendering (`Render`), sorting (`Sort`), and counting (`CountAtLeast`) are implemented once in `pkg/scanfindings`.

### Alternatives Considered

#### Alternative 1: Extract a shared severity mapping function only

Extract a single `ParseSeverity(raw string) string` helper returning a normalized string, keeping each tool's finding struct and rendering loop separate. This eliminates only the severity inconsistency while leaving context-line extraction, message building, and rendering duplicated across nine files. It is a smaller change but does not solve the maintenance problem that motivated this PR.

#### Alternative 2: Define a `Finding` interface instead of a concrete struct

Define a `Scanner` or `Finding` interface and let each integration implement it via its own native type. This avoids the explicit adapter functions and the coupling to a single shared struct, but adds indirection without eliminating boilerplate: every integration would still implement the same set of methods. The concrete-struct adapter approach chosen here produces less code overall and makes the mapping explicit and testable in isolation.

### Consequences

#### Positive
- Adding a scanner now requires only a field-mapping adapter; severity classification, rendering, sorting and counting are handled once.
- The severity vocabulary is normalized through `ParseSeverity`, eliminating per-tool inconsistencies in how identical native labels were classified.
- `audit_report.go`'s `Finding.Severity` is now a typed `SeverityLevel` rather than an unvalidated `string`, enabling compile-time checks in severity comparisons.
- Two duplicated rendering loops in `poutine.go` collapse into a single `poutineFindingsToShared` adapter.

#### Negative
- All scanner integrations are now coupled to `pkg/scanfindings`; changes to the shared type's API propagate to every integration.
- The `Severity` field type on `audit_report.go`'s `Finding` struct changes from `string` to `SeverityLevel` — callers that compared `finding.Severity == "critical"` must now use `finding.Severity == scanfindings.SeverityCritical` or `.String()`.

#### Neutral
- Low-severity zizmor and runner-guard findings now render as `info` (matching grype's behavior) rather than `warning` — this is an intentional alignment, not a regression.
- The `yamllintIssue` internal struct is removed; `parseYamllintLine` returns `scanfindings.Finding` directly.
- JSON serialization of `SeverityLevel` is unchanged: the underlying type is `string`, so existing JSON consumers reading `"severity"` fields see the same lowercase values.

---

*Accepted on 2026-08-22 after validating the shared type and all scanner adapters.*
