# ADR-68718: Prefer Root `go.mod` for Generated Go Setup Steps

**Date**: 2026-02-18
**Status**: Draft
**Deciders**: Unknown (originating PR [#68718](https://github.com/github/gh-aw/pull/68718), fixes #68678)

---

### Context

The compiler emits an `actions/setup-go` step whenever a workflow needs the Go runtime. Until now that step always carried a hard-coded default (`go-version: '1.26'`) unless the workflow author set `runtimes.go.version` explicitly. That constant is independent of the Go version declared in the consuming repository's root `go.mod`, so a repository requiring a newer toolchain would get a mismatched Go installation. Combined with `GOTOOLCHAIN=local`, which forbids on-demand toolchain downloads, Go commands failed outright instead of silently upgrading. The compiler needs a version source that tracks the repository, but it can only do so when the generated job actually checks out that repository at the workspace root before the setup step runs.

### Decision

We will have the compiler emit `go-version-file: go.mod` instead of a pinned `go-version` when three conditions all hold: the workflow does not set `runtimes.go.version`, a regular `go.mod` file exists at the compiler's detected git root, and the generated job performs a root checkout of the workflow's own repository before runtime setup. The primary driver is correctness of the toolchain selection: the repository's own module file is the authoritative statement of its minimum Go version. In every other case the compiler keeps emitting the existing default version, so no workflow loses a working setup step.

The "root checkout" predicate is evaluated from both the compiler's own checkout configuration (`shouldAddCheckoutStep`, excluding external-root and wiki checkouts) and from user-supplied custom steps via a new `customCheckoutHasRootGoMod` helper, which accepts only an unconditional, non-sparse, same-repository checkout with an empty or `.` path.

### Alternatives Considered

#### Alternative 1: Read `go.mod` at compile time and inline the parsed version

The compiler could parse the `go` directive from the root `go.mod` and emit `go-version: '<parsed>'`. This keeps the generated YAML self-contained and works even without a checkout, since the version is resolved at compile time. It was rejected because the pinned value immediately goes stale: bumping `go.mod` would silently leave every previously compiled `.lock.yml` on the old toolchain until recompiled, reintroducing the exact drift this change is meant to remove. `go-version-file` defers resolution to run time, where the checked-out module file is always current.

#### Alternative 2: Always emit `go-version-file: go.mod` unconditionally

Dropping the checkout and file-existence guards would make the compiler simpler and remove the new helper entirely. It was rejected because `actions/setup-go` hard-fails when the referenced file is absent, which would break workflows that intentionally skip checkout (`checkout: false`), check out a different repository, check out into a subdirectory, use a wiki checkout, or run in repositories with no root module. The guard logic is the cost of preserving those cases, and the new test matrix enumerates each of them.

#### Alternative 3: Raise the hard-coded default to the repository's current Go version

A one-line constant bump would unblock the immediate failure reported in #68678. It was rejected as a non-fix: it resolves one snapshot of the mismatch while guaranteeing the same class of breakage on the next toolchain bump, and it still ignores the Go version of any other repository consuming gh-aw-generated workflows.

### Consequences

#### Positive
- Generated Go setup tracks the repository's declared minimum Go version automatically, so `GOTOOLCHAIN=local` jobs stop failing after a `go.mod` bump without requiring workflow edits.
- Recompiling is no longer required to pick up a Go version change; the existing `.lock.yml` resolves the version at run time from the checked-out module file.
- Explicit `runtimes.go.version` still wins, preserving the escape hatch for workflows that deliberately pin a different toolchain.

#### Negative
- The compiler now has a compile-time dependency on filesystem state (`c.gitRoot/go.mod`), so the generated output of the same workflow source differs between repositories and between environments where the git root is or is not detected — making lock files less reproducible from source alone.
- The "root checkout" predicate duplicates checkout reasoning across configured checkouts and raw custom-step YAML; `customCheckoutHasRootGoMod` must stay in sync with checkout semantics (path, repository, `if`, sparse-checkout) or it will silently emit a setup step that fails at run time.
- A workflow that reorders steps so that checkout no longer precedes runtime setup, or that adds a conditional to its checkout, will change the emitted Go version source in a way that is not obvious from the workflow source.

#### Neutral
- Two in-repo lock files (`architecture-guardian.lock.yml`, `dead-code-remover.lock.yml`) changed as generated output; the diff for Go-using workflows is a one-line `with:` key swap.
- The documented default for the `go` runtime becomes conditional ("root `go.mod`, or 1.26 if absent") rather than a single constant, which the frontmatter reference now spells out.
- `constants.DefaultGoVersion` remains the fallback and is still exercised by the majority of the new test cases, so the constant continues to need maintenance.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
