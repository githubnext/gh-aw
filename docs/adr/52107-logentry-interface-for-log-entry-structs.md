# ADR-52107: LogEntry Interface for Heterogeneous Log-Entry Structs

**Date**: 2026-08-11
**Status**: Draft
**Deciders**: Unknown

---

### Context

`pkg/cli` contains four independently-defined structs — `AccessLogEntry`, `FirewallLogEntry`, `AuditLogEntry`, and `GatewayLogEntry` — each modelling a parsed log line from a different source. They share no common type, so any code that wants to handle "a log entry" generically (formatters, filters, report generators) must special-case every concrete type. The structs also have structurally incompatible fields: `Timestamp` is a `string` in three types but a `float64` in `AuditLogEntry`, and only `GatewayLogEntry` carries a `Level` field. These differences make a shared embedded base struct impractical without changing wire formats.

### Decision

We will define a `LogEntry` interface in `pkg/cli/log_entry.go` with four accessor methods — `EntryTimestamp()`, `EntrySource()`, `EntryLevel()`, and `EntryMessage()` — and implement it on all four existing log-entry types using value receivers. Timestamp normalisation (epoch seconds → RFC3339 UTC) is handled inside the implementations so callers see a uniform string regardless of the underlying field type. Compile-time conformance is enforced with blank-identifier assertions (`var _ LogEntry = AccessLogEntry{}`). A `FormatLogEntry` function serves as the first generic consumer.

### Alternatives Considered

#### Alternative 1: Embedded Base Struct

Define a shared `BaseLogEntry` struct and embed it in the four types. This would promote common fields directly and avoid the interface layer.

Rejected because the four types have incompatible field layouts: `AuditLogEntry.Timestamp` is `float64` while the others are `string`, and only `GatewayLogEntry` has `Level`. Adding these fields to a base struct would require changing the JSON tags or adding duplicate fields, breaking existing serialisation and parse call sites.

#### Alternative 2: Type Switch / Ad-Hoc Polymorphism

Continue the current pattern: any code that needs to act on "any log entry" performs an explicit type switch over all four concrete types.

Rejected because it duplicates the dispatch logic in every consumer, makes adding a fifth log-entry type a multi-site change, and provides no compile-time guarantee that all types are handled. The very motivation of the linked issue (#52091) was to eliminate this duplication.

### Consequences

#### Positive
- Formatting, filtering, and reporting code can operate uniformly over any `[]LogEntry` without knowing the underlying type.
- Adding a fifth log-entry type in the future requires only implementing four methods, with no changes to existing consumers.
- Compile-time `var _ LogEntry = ...` assertions catch interface drift immediately at build time.
- No existing struct fields, JSON tags, or parsers are touched, so serialisation and all current call sites are unaffected.

#### Negative
- Epoch-to-RFC3339 timestamp normalisation is now encapsulated inside each implementation, making the per-source conversion logic less visible to callers who might expect raw values.
- All four implementations use value receivers; callers passing large `[]LogEntry` slices by value incur copying overhead that would not exist with pointer receivers or a concrete slice type.

#### Neutral
- The `LogEntry` interface is defined in the same `cli` package as the concrete types, so there is no cross-package dependency change.
- Test coverage is added in `pkg/cli/log_entry_test.go` using table-driven tests; the interface itself is not exported beyond the `cli` package boundary at this point.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
