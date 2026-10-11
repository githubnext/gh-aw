# ADR-67643: Share Scanner Finding Conversion via a Constructor Helper

**Date**: 2026-10-10
**Status**: Proposed
**Deciders**: pelikhan (PR author); gh-aw maintainers (review pending)

---

### Context

Four scanner integrations in `pkg/cli` — zizmor (`zizmor.go`), poutine (`poutine.go`), grype (`grype.go`),
and runner-guard (`runner_guard.go`) — each parse a third-party tool's own JSON schema into a
tool-specific wire struct, then hand-write a near-identical conversion into the shared
`scanfindings.Finding` type (`pkg/scanfindings/scanfindings.go`). The copy-pasted blocks repeated the same
`RuleID` / `ParseSeverity(...)` / `File` / `Line` / `Column` / `ContextLines(...)` mapping, so every new
scanner integration paid a copy-paste tax and risked diverging on severity normalization or context
attachment (issue #67202). The native wire structs themselves must stay separate because they mirror each
tool's external JSON contract, and the user-visible behaviour of each scanner — message formatting,
location selection, per-scanner defaults, filtering, and rendering — must not change.

### Decision

We will add a single shared constructor, `scanfindings.NewFinding(params)`, taking a
`scanfindings.NewFindingParams` value with named fields for the rule, severity label, message, display
location, and source lines. It owns the common mapping: severity normalization via `ParseSeverity` and
source-context attachment via `ContextLines`. Named fields make call sites self-documenting and prevent
same-typed values from being silently transposed. Each of the four adapters supplies values it has
already computed, so scanner-specific message formatting, location selection, and defaults remain in the
adapter where they belong. The helper centralizes only the mechanical, identical part of the conversion
and leaves the semantics of each scanner untouched. `AuditFinding` and `SecurityFinding` are explicitly
out of scope for this change.

### Alternatives Considered

#### Alternative 1: A `rawScanFinding` interface implemented by each wire struct

The original issue suggested defining an interface (e.g. `RuleID() string`, `Severity() string`,
`Location() (string, int, int)`) that each tool-specific struct implements, with one generic conversion
function. This was rejected because the per-scanner logic that actually differs — message composition from
rule tables, multi-location selection, fallback file paths, row/column offsets — is not uniform enough to
hide behind accessor methods; each struct would need bespoke methods that simply relocate the same code
behind more indirection, while making the adapters harder to read.

#### Alternative 2: Leave the duplication in place

Four short, obvious literal constructions are cheap to read, and duplication in adapters is a common,
defensible pattern. This was rejected because the duplication is exact rather than incidental: severity
normalization and context attachment are invariants of the shared `Finding` type, and letting each adapter
restate them invites silent drift as scanners are added — the concrete motivation recorded in issue #67202.

#### Alternative 3: Use positional constructor parameters

A constructor with one argument per field would avoid duplicating conversion logic, but adjacent
same-typed values could be accidentally transposed without a compiler error. A parameter struct was
chosen instead so each call site names the field it supplies.

### Consequences

#### Positive
- New scanner integrations get severity normalization and context attachment for free by calling one helper.
- Severity/context behaviour can be changed in one place instead of four, removing a class of drift bugs.
- Named parameter fields make scanner call sites clearer and avoid silent argument-order mistakes.
- Native-JSON regression tests added for all four adapters pin the existing output semantics, so future
  refactors of the helper are guarded.

#### Negative
- The helper always calls `ContextLines`; an adapter that wants a finding without context must pass `nil`,
  which couples all adapters to one context policy.

#### Neutral
- Adapters keep their own wire structs and message-formatting code, so the duplication reduction is partial
  by design.
- `AuditFinding` (`audit_report.go`) and `SecurityFinding` (`markdown_security_scanner.go`) remain separate
  cousin types; folding them in is deferred to a future decision.
- No public CLI behaviour or JSON schema changes; the change is internal to `pkg/cli` and `pkg/scanfindings`.

---

*ADR proposed by the PR author; acceptance is pending maintainer review.*
