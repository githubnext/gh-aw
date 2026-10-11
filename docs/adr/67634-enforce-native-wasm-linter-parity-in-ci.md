# ADR-67634: Enforce Native/WebAssembly Custom-Linter Parity via YAML-Parsing CI Sync Tests

**Date**: 2026-10-11
**Status**: Draft
**Deciders**: pelikhan (PR #67634 author), gh-aw maintainers

---

### Context

The custom analyzers in `pkg/linters` are enforced in CI through `LINTER_FLAGS` lists in `.github/workflows/cgo.yml`, which runs the linters twice: once natively and once with `GOOS=js GOARCH=wasm`. Some analyzers (notably `consolestderr`) were enforced only on the native target, so WebAssembly-only regressions could land undetected (see #67609). The previous guard in `pkg/linters` relied on a hard-coded expected step count, which was brittle: it broke on unrelated workflow edits while still failing to detect analyzer-set drift between the two targets. A known, deliberate exception exists — `contextcancelnotdeferred` is native-only pending the work tracked in #65753 — so the guard must tolerate an explicitly enumerated exception set rather than demanding exact equality.

### Decision

We will enforce native/WebAssembly analyzer parity by parsing the custom-linter job in `.github/workflows/cgo.yml` and comparing the *combined* analyzer sets (bulk step plus dedicated single-analyzer steps) per target, instead of asserting on a hard-coded step count. The parity test tracks native-only exceptions in an explicit allow-list keyed to tracking issues, and fails both when a target is missing an enforced analyzer and when a stale exception remains after parity is restored. As part of this decision, the `consolestderr -test=false` step is mirrored onto the wasm target using the existing wasm package scope.

### Alternatives Considered

#### Alternative 1: Keep the hard-coded step-count guard and just add the missing wasm step

Minimal diff: add the `consolestderr (wasm)` step and bump the expected step count. This was rejected because it fixes only the current instance of the bug class — the next analyzer added to one target but not the other would still slip through — and the step count keeps producing false failures on unrelated workflow edits.

#### Alternative 2: Generate the CI linter steps from the analyzer registry

Rather than testing the workflow, derive the `LINTER_FLAGS` lists (and therefore both target steps) from `pkg/linters`' registry at compile/generation time, making drift structurally impossible. This is the stronger guarantee, but it requires a generator plus workflow regeneration plumbing and does not accommodate intentional per-target differences (such as the #65753 exception) without additional metadata. It remains a reasonable future step; parity testing was chosen as the lower-risk increment.

#### Alternative 3: Collapse to a single linter invocation covering both targets

Run the analyzers once over a union package set and drop per-target steps entirely. Rejected because `GOOS=js GOARCH=wasm` build constraints change which files are analyzed, so a single invocation would lose genuine wasm-specific coverage.

### Consequences

#### Positive

- Native-only or wasm-only enforcement gaps now fail fast in `go test ./pkg/linters` instead of reaching `main`.
- `consolestderr` violations are caught in WebAssembly-constrained code paths, closing #67609.
- Removing the hard-coded step count eliminates a recurring source of unrelated CI failures when `cgo.yml` is edited.
- Deliberate exceptions are documented in code and tied to tracking issues, and stale exceptions are rejected automatically once parity is restored.

#### Negative

- The test parses CI YAML with string/structure matching, so it is coupled to the current step formatting (bulk flag layout, quoting, `GOOS=js GOARCH=wasm` prefix); reformatting the workflow can require test updates.
- The native-only exception list is a manually maintained allow-list that must be kept honest by reviewers.
- Analyzer flags are still duplicated between the native and wasm steps, so additions require touching two places — the test only detects the mistake rather than preventing it.

#### Neutral

- CI runtime grows slightly with the extra dedicated wasm step.
- `pkg/linters/README.md` becomes the canonical description of which analyzers are enforced per target and why exceptions exist.
- Regression coverage added in `pkg/linters/ci_sync_test.go` and `pkg/linters/doc_sync_test.go` (missing steps, bulk-flag drift, restored parity, step environment targets, single-quoted flags, commented-out enforcement).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
