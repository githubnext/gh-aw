# ADR-67649: Centralize Work-Queue Receipt Projections in `pkg/workqueue`

**Date**: 2026-10-11
**Status**: Draft
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

### Context

`pkg/workqueue` owns the durable work-queue domain types `Assignment` and `AssignmentClaim`, which carry identity fields (`DispatchID`, `RequestID`, `CommitID`, `Handle`, `ClaimID`, `WorkID`) alongside sensitive or bulky data such as work payloads, result descriptors, and actor credentials. The CLI reporting layer (`pkg/cli/logs_work_queue_current.go`) needed a payload-free view of the same facts for its JSON snapshot output, and historically hand-duplicated ~60–70% of those field names in its own `WorkQueueAssignmentReceipt`/`WorkQueueClaimReceipt` structs. Any change to core identity fields therefore required a second, easy-to-forget hand edit in the CLI, creating a silent-drift risk between the domain model and the reported schema (issue #67200). The CLI's published JSON field names (`dispatch_id`, `claim_handle`, `delivery_barrier`, …) are a user-visible contract and must not change.

### Decision

We will define payload-free receipt types (`workqueue.AssignmentReceipt`, `workqueue.ClaimReceipt`) and projection methods `Assignment.ToReceipt()` / `AssignmentClaim.ToReceipt()` in `pkg/workqueue`, and reduce the CLI receipt types to type aliases over them. The projections deliberately map **identity only**; lifecycle facts (`State`, `Released`, `Barrier`) remain the caller's responsibility, populated from the caller's validated, captured projection. The primary driver is eliminating duplicate field declarations so the domain package is the single source of truth for receipt shape, while keeping the CLI's existing JSON schema byte-for-byte identical.

### Alternatives Considered

#### Alternative 1: Factor out a shared identity sub-struct embedded in both types

Introduce a small `identity` struct embedded in `Assignment`/`AssignmentClaim` and in the CLI receipt types, so the fields are declared once. This was considered because it removes duplication without adding new methods. It was rejected because embedding changes the JSON encoding shape of the durable domain types (or requires `,inline`-style workarounds Go's `encoding/json` does not support), risking the on-disk transaction-log format and the CLI output contract simultaneously.

#### Alternative 2: Leave the CLI structs duplicated and add a compile-time/round-trip conformance test

Keep both declarations and add a test asserting field-name parity, failing CI on drift. This is the least invasive option and was a plausible close call. It was rejected because it preserves the maintenance burden (two edits per change) and only detects drift after the fact, whereas aliasing makes drift structurally impossible.

#### Alternative 3: Have the CLI marshal `Assignment` directly with `json:"-"` tags on sensitive fields

Rejected: it couples the public CLI schema to the durable domain struct, so any new internal field would leak into user-visible output unless explicitly suppressed — a fail-open default for payloads and credentials.

### Consequences

#### Positive
- Receipt identity fields are declared exactly once; changes to `Assignment`/`AssignmentClaim` identity propagate to the CLI automatically.
- Payload, result-descriptor, and credential omission is enforced by construction in the projection rather than by reviewer vigilance in the CLI.
- New regression tests pin the enriched CLI JSON schema, claim ordering, nil-vs-empty claim slices, and receipt/assignment non-aliasing.

#### Negative
- The projection is intentionally partial: `State`, `Released`, and `Barrier` must still be filled in by each caller, so a caller that forgets produces a silently empty lifecycle view.
- `pkg/cli` now depends on `pkg/workqueue` for its public reporting types; a breaking change in the domain receipt types becomes an immediate breaking change in CLI JSON output.

#### Neutral
- Type aliases (`type WorkQueueAssignmentReceipt = workqueue.AssignmentReceipt`) keep the existing CLI identifiers and JSON field names, so no consumer-visible rename occurs.
- The CLI now mutates a value receipt in place (`receipt.Claims[i].State = …`) instead of appending constructed structs; behaviour is equivalent but the code style differs from the previous build-up loop.
- `ToReceipt()` preserves nil vs empty claim slices, so `claims` serializes as `null` or `[]` exactly as the source assignment dictates.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
