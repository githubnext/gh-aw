# ADR-54840: Extract Shared NetworkLogEntry Base Struct

**Date**: 2026-08-22
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

Three structs — `AccessLogEntry`, `FirewallLogEntry`, and `AuditLogEntry` — all represent parsed network request log records and share a common set of fields: client address, HTTP method, status code, routing decision, and URL. These fields were duplicated independently in each struct, sometimes with different names (`ClientIP` vs `ClientIPPort` vs `Client`) and incompatible types (`Status` was `string` in access/firewall logs but `int` in audit logs). The type mismatch required separate status-parsing branches in each consumer and made unified downstream processing (e.g., `isRequestAllowed`, timeline event construction) require ad-hoc conversions.

### Decision

We will extract a shared `NetworkLogEntry` struct containing the common fields (`ClientAddr`, `Method`, `Status string`, `Decision`, `URL`) and embed it in all three log entry types. `Status` is normalized to `string` everywhere; helpers `networkStatusCode` and `networkStatusFromJSON` centralize all parsing of numeric-or-string status values. `AuditLogEntry` adds a custom `UnmarshalJSON` to accept the legacy wire format where `status` is an undecorated JSON integer.

### Alternatives Considered

#### Alternative 1: Common Interface with Getter Methods

Define a `NetworkLogReader` interface with methods like `GetStatus() string`, `GetMethod() string`, etc., implemented by each struct separately. This allows polymorphic processing without struct embedding.

Rejected because it requires boilerplate getter implementations on all three types, doesn't eliminate field duplication in the structs themselves, and adds indirection without reducing the divergent naming (`ClientIP` vs `ClientIPPort` vs `Client`).

#### Alternative 2: Normalize Status Type Only

Change `AuditLogEntry.Status` from `int` to `string` without introducing a shared base struct, and add a single shared `networkStatusCode` helper.

Rejected because this fixes the type inconsistency but leaves the broader field duplication (`Method`, `Decision`, `URL`, `ClientAddr` under three different names) in place. Future cross-log consumers would still need to access identical concepts through different field names on each struct.

### Consequences

#### Positive
- Eliminates duplicated field declarations across three log entry types, reducing maintenance surface.
- Centralizes status parsing logic into `networkStatusCode`, `networkStatusCodeOrZero`, and `networkStatusFromJSON` — one place to fix if parsing rules change.
- Resolves the `int`/`string` type inconsistency for `Status`, enabling uniform comparison logic in `isEntryAllowed` and `isRequestAllowed`.

#### Negative
- Callers constructing `AuditLogEntry`, `FirewallLogEntry`, or `AccessLogEntry` literals must use the embedded struct syntax (`NetworkLogEntry: NetworkLogEntry{...}`), which is more verbose than setting flat fields.
- `AuditLogEntry.UnmarshalJSON` is now a custom implementation required to handle the legacy integer `status` wire format; this adds a deserialization code path that must be kept in sync if the struct changes.

#### Neutral
- Field name `ClientIPPort` (firewall) and `ClientIP` (access) are unified to `ClientAddr`, which is a broader rename visible in test assertions and any callers accessing the field directly.
- The `strconv` import is removed from `firewall_log.go` as status-to-int conversion is now delegated to `network_log_entry.go`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
