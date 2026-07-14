# ADR-45301: Add `bytesbufferstring` Linter — Flag `string(buf.Bytes())` in Favour of `buf.String()`

**Date**: 2026-07-13
**Status**: Draft
**Deciders**: Unknown (automated PR by linter-miner)

---

### Context

The `gh-aw` repository maintains a custom suite of Go static-analysis linters under `pkg/linters/`. Each linter detects a specific sub-optimal code pattern and provides an auto-applicable suggested fix. The pattern `string(buf.Bytes())` — where `buf` is `*bytes.Buffer` — creates an unnecessary intermediate `[]byte` allocation before the final `string` conversion. `bytes.Buffer.String()` already returns the buffer's contents as a string directly, making the intermediate conversion redundant and wasteful. The linter-miner agent identified this pattern by scanning non-test Go source files in `pkg/` and `cmd/`.

### Decision

We will add a new Go analysis pass, `bytesbufferstring`, to the custom linter suite. The analyzer flags any `string(x.Bytes())` call where `x` is statically typed as `*bytes.Buffer` or `bytes.Buffer`, and offers a single suggested fix that rewrites the expression to `x.String()`. The analyzer skips test files and respects `//nolint:bytesbufferstring` suppressions. It is registered in `cmd/linters/main.go` alongside all other custom analyzers.

### Alternatives Considered

#### Alternative 1: Rely on Human Code Review

Leave detection of this pattern entirely to human reviewers during pull request review. No tooling change needed, and no new linter to maintain.

This was rejected because human review is inconsistent and does not scale: the same sub-optimal pattern can be introduced repeatedly across a large codebase. An automated linter provides reliable, zero-effort enforcement.

#### Alternative 2: Use an Existing Third-Party Linter (`staticcheck` / `golangci-lint`)

Configure an external linter tool that already covers this pattern, rather than writing a custom one.

This was not viable in this context: the existing tooling infrastructure is built around custom `golang.org/x/tools/go/analysis` passes in the `pkg/linters/` package. Introducing a new external tool would require changes to the CI pipeline, build configuration, and version-pinning strategy, which exceeds the scope of adding a single focused rule. No existing linter in the project's current set covers this pattern.

### Consequences

#### Positive
- Eliminates unnecessary intermediate `[]byte` allocations wherever the pattern is found, producing modest runtime improvements.
- The suggested fix is auto-applicable (zero manual effort for the author once the linter runs).
- Zero false-positive risk: the match is purely structural — any `string(x.Bytes())` where `x` is `*bytes.Buffer` is unambiguously replaceable.
- Follows the established linter conventions in the repository (`largefunc` layout, `nolint` suppression, test fixture with golden file).

#### Negative
- Adds one more analyzer to the custom linter suite, increasing maintenance surface.
- Each additional linter marginally increases the time for a full lint pass.
- Any existing occurrences of `string(buf.Bytes())` in the codebase that are not suppressed will become lint failures, requiring authors to update code or add `//nolint:bytesbufferstring`.

#### Neutral
- The pattern is detected at analysis time only; no runtime behaviour or API contracts change.
- Suppression via `//nolint:bytesbufferstring` is available for cases where the conversion is intentional (e.g., operating on a copy, not the original buffer).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
