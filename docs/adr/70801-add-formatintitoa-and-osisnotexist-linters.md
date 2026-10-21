# ADR-70801: Add `formatintitoa` and `osisnotexist` analysis-pass linters

**Date**: 2026-10-21
**Status**: Draft
**Deciders**: pelikhan, Unknown

---

### Context

The repository keeps its custom `golang.org/x/tools/go/analysis` linters under `pkg/linters/`. Each linter lives in its own package and is registered in `pkg/linters/registry.go`. Two patterns were requested for automated detection (issues #70313 and #70579). The first is wordy decimal formatting of `int` values via `strconv.FormatInt(int64(x), 10)`, where `strconv.Itoa(x)` does the same thing more simply. The second is the use of legacy `os.IsNotExist`, `os.IsExist`, `os.IsPermission`, and `os.IsTimeout` predicates. These predicates do not check errors that were wrapped with arbitrary wrappers, such as `fmt.Errorf("...: %w", err)`, so they can silently miss real errors. Running both checks on production packages surfaced 116 existing findings. That means turning on enforcement immediately would break CI.

### Decision

We will add two new standalone analyzers, `pkg/linters/formatintitoa` and `pkg/linters/osisnotexist`. Each is built on the shared `analyzerutil`, `astutil`, `filecheck`, and `nolint` helpers and registered in `pkg/linters/registry.go`. `formatintitoa` uses type information to match only `strconv.FormatInt(int64(x), 10)` where `x` is exactly `int` and the base is the constant 10. It offers a suggested fix only when the call is package-qualified and has no overlapping comments. `osisnotexist` reports the four legacy `os` predicates. It recommends `errors.Is` with the matching `fs.Err*` sentinel. For `os.IsTimeout`, it gives separate guidance to use `errors.As` with `interface{ Timeout() bool }`, because `os.ErrDeadlineExceeded` does not cover every timeout. Both analyzers will stay out of CI enforcement until the existing findings are fixed.

### Alternatives Considered

#### Alternative 1: Rely on off-the-shelf linters (e.g. `perfsprint`, `errorlint`, `gocritic`) through golangci-lint

Third-party linters already cover similar ground. `errorlint` flags some error-comparison patterns, and `perfsprint` and `gocritic` flag some formatting patterns. They were not chosen for three reasons. Their exact rule coverage and suggested fixes do not match the narrow, type-exact behavior wanted here (for example, the `int`-only match and the timeout-specific guidance). They would bring in broader rule sets. And they would not follow the repository's convention of one package per in-house rule, with shared `nolint` and generated-file handling. [TODO: verify whether these upstream linters were evaluated.]

#### Alternative 2: Fix existing call sites manually, or with a one-off rewrite, without adding linters

The 116 findings could be fixed directly without adding any analyzer. This was rejected because nothing would stop the patterns from coming back, which is why the issues asked for linters.

#### Alternative 3: Enable CI enforcement immediately alongside the analyzers

Turning the analyzers on as blocking checks would stop new violations right away. This was not chosen because the 116 existing findings would fail CI until all of them were fixed in the same PR, which would greatly widen its scope.

### Consequences

#### Positive
- Wrapped-error-unaware `os.Is*` checks and wordy `FormatInt` calls can now be detected automatically and repeatably, which keeps them from spreading.
- Type-exact matching avoids false positives on defined integer types, non-decimal bases, and indirect calls. Suggested fixes for `formatintitoa` handle parentheses and import/type aliases.
- Both analyzers follow the existing one-package-per-linter convention and come with analysistest and golden-fix fixtures covering aliases, shadowing, suppression, and generated files.

#### Negative
- The analyzers are not enforced in CI yet, so new violations can still be merged until remediation is done and enforcement is enabled.
- Two more custom analyzers must be maintained in-house, and the documentation counts and registry contracts (`doc.go`, `README.md`, `spec_test.go`) must be kept in sync.
- `osisnotexist` only reports diagnostics and offers no automatic fix. Each of the 116 existing findings, and any future ones, must be fixed by hand.

#### Neutral
- The `os.IsTimeout` guidance deliberately differs from the other three predicates, which reflects real semantic differences rather than a uniform rewrite.
- The recorded analyzer count in the README/spec contract increases from 82 to 84.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
