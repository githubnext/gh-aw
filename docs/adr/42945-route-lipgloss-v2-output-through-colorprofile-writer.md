# ADR-42945: Route Lip Gloss v2 Output Through colorprofile.Writer

**Date**: 2026-07-02
**Status**: Draft
**Deciders**: Unknown

---

### Context

Lip Gloss v2 removed automatic terminal capability detection that was present in v1. As a result, styled output paths in `pkg/console` and `pkg/logger` began emitting raw ANSI escape codes regardless of `NO_COLOR`, `COLORTERM`, or terminal type — including piped output and limited-color terminals. This violates the [NO_COLOR](https://no-color.org) convention and can corrupt output on terminals that do not support truecolor. The codebase has multiple write sites that emit directly to `os.Stderr`, which bypasses any capability probing entirely.

### Decision

We will route all styled stderr output through a `colorprofile.Writer` (from `github.com/charmbracelet/colorprofile`) wrapper rather than writing to `os.Stderr` directly. A shared factory function `stderrWriter()` is introduced in `pkg/console` (with a passthrough stub for wasm builds) so all output paths consistently consult the environment for `NO_COLOR`, `COLORTERM`, and terminal capability before rendering ANSI sequences.

### Alternatives Considered

#### Alternative 1: Manual NO_COLOR guard at each write site

Each styled `fmt.Fprintf(os.Stderr, ...)` call could be preceded by an `if os.Getenv("NO_COLOR") == ""` check. This is the lowest-effort change but requires repetitive, error-prone guards scattered across every output site, with no coverage for terminal capability degradation (e.g., downgrading truecolor to 256-color on constrained terminals). It does not scale as new write sites are added.

#### Alternative 2: Configure a global Lip Gloss renderer with the correct profile

Lip Gloss v2 exposes a `Renderer` that can be constructed with a specific `colorprofile.Profile`. A single global renderer could be initialized once at startup and used for all style rendering. This would centralize capability probing but requires replacing every `lipgloss.NewStyle()` call with renderer-scoped style construction, a larger refactor surface. It also does not address `lipgloss.Fprintf` calls that accept an `io.Writer` rather than using a renderer.

### Consequences

#### Positive
- `NO_COLOR` is correctly honored across all `pkg/console` and `pkg/logger` output paths without per-call guards.
- Terminal capability is degraded appropriately (e.g., truecolor → 256-color → plain) based on the detected environment at write time.
- Wasm builds retain their original `os.Stderr` passthrough via a build-tagged stub, preserving existing behavior on that platform.

#### Negative
- `stderrWriter()` calls `colorprofile.NewWriter(os.Stderr, os.Environ())` on every invocation, allocating a new writer wrapper per write site call rather than reusing a singleton. This introduces minor per-call overhead that could be avoided with a cached writer.
- The `newColorProfileWriter`/`stderrWriter` helper functions are duplicated between `pkg/console` and `pkg/logger`, creating two independent copies that could drift in behavior. [TODO: verify] whether consolidating into a shared package is feasible.

#### Neutral
- Existing tests for `NO_COLOR` behavior in `pkg/logger` (`TestNoColorEnvironment`) continue to pass; new regression tests confirm ANSI stripping via the colorprofile writer path.
- The `pkg/styles/theme.go` comment update clarifying `adaptiveColor` vs `lipgloss.LightDark` usage is documentation-only and carries no runtime impact.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
