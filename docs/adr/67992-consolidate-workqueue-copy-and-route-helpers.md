# ADR-67992: Consolidate workqueue copy and route-verification helpers

**Date**: 2026-10-12
**Status**: Draft
**Deciders**: Unknown (draft generated from PR #67992)

---

### Context

`pkg/workqueue` had accumulated three near-identical pieces of logic. Two separate map-copy helpers (`copyNativeVerifiers` for `map[string]NativeOutputVerifier` and `copyNativeEffectResource` for `EffectResource`) existed solely because Go's pre-generics code could not express "copy any string-keyed map while preserving its named type". Separately, `verifyWorkerRoutes` (policy-wide) and `verifyAssignmentWorkerRoutes` (assignment-scoped) each inlined the same dedup-then-verify-then-record sequence against an `identitySet`, so any change to deduplication keying or error ordering had to be applied twice and kept in sync by hand. Finally, `IssuesSettings` and `parseIssuesSettings` lived in their own `settings_issues.go` file while every other `work_queue` field parser lived in `settings_parse.go`. The non-negotiable constraint is that this consolidation must be behaviour-preserving: copy isolation (including the non-nil copy of a nil map), route verification order, and error precedence are security- and correctness-relevant and must not shift.

### Decision

We will replace the duplicated helpers with single shared implementations: a type-parameterised `copyMap[M ~map[string]V, V any](values M) M` that preserves named map types such as `EffectResource`, and a `Branch.verifyWorkerRouteOnce(ctx, profile, seen)` helper that owns the dedup key, verification call, and `seen` bookkeeping for both route-verification paths. We will also move `IssuesSettings` and `parseIssuesSettings` into `settings_parse.go` so all settings field parsers are co-located. The primary driver is removing duplicate invariants: one implementation means copy isolation and route-dedup semantics can only be changed in one place. Regression tests are added for map isolation, route deduplication counts, and assignment error precedence to pin the preserved behaviour.

### Alternatives Considered

#### Alternative 1: Keep the per-type copy helpers

Leaving `copyNativeVerifiers` and `copyNativeEffectResource` as-is avoids introducing generics into this package and keeps call sites maximally explicit about what is being copied. It was rejected because the two bodies are byte-for-byte identical apart from the element type, and each additional copied map type would have required a third and fourth clone. The `~map[string]V` constraint preserves the named return type, so call sites lose no type information.

#### Alternative 2: Deduplicate routes at the caller by pre-filtering profiles

Instead of a shared `verifyWorkerRouteOnce`, both callers could build a deduplicated slice of profiles first and then verify them in a plain loop. This was rejected because it changes when verification happens relative to the assignment-validity checks in `verifyAssignmentWorkerRoutes` (the `profile_invalid` check must still run per assignment, in order), and that reordering would alter which error surfaces when an invalid assignment follows a failing route — exactly the precedence the new test pins down.

#### Alternative 3: Leave `settings_issues.go` as a separate file

Keeping the Issues parser in its own file is defensible on "one concept per file" grounds. It was rejected because every sibling `work_queue` field parser already lives in `settings_parse.go`, so the split made the Issues parser harder to find and reviewed inconsistently with its peers.

### Consequences

#### Positive
- Copy isolation semantics (including nil-map handling) now have one implementation and one test, so they cannot drift between the verifier registry and effect resources.
- Route deduplication and verification ordering are defined once, so policy and assignment paths cannot diverge.
- Settings parsing is discoverable in a single file, matching the existing convention for other `work_queue` fields.
- New regression tests explicitly pin map isolation, dedup counts, and assignment error precedence, which were previously only implicitly covered.

#### Negative
- `copyMap` is generic over any `~map[string]V`, so it no longer documents at the signature level which specific maps are expected to be copied; misuse is caught only by review.
- `verifyWorkerRouteOnce` mutates the caller-supplied `seen` set, which is a less explicit contract than the previous inline `seen.add(key)` at each call site.
- Behaviour preservation is enforced by tests rather than by the type system; a future change to `verifyWorkerRouteOnce` now silently affects both call paths.

#### Neutral
- The package gains a dependency on Go generics in `delivery_host.go`; the repository's Go version already supports this.
- Net line count is roughly flat (≈107 additions, mostly tests) — the win is in duplicate-invariant removal, not code size.
- `settings_issues.go` is deleted; future `git log`/`git blame` on the Issues parser needs `--follow` across the move.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
