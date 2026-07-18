# ADR-46197: Hoist Palette Color Vars and Honor NO_COLOR on Stdout

**Date**: 2026-07-18
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `pkg/styles` and `pkg/console` packages use lipgloss v2 for terminal styling. Two related defects were surfaced during a lipgloss v2 Go Fan review:

1. `huh_theme.go` called `lipgloss.Color(hexConstant)` inside `HuhTheme(isDark)` on every invocation, re-parsing the same hex strings that `theme.go` already parsed once at package init for its `adaptiveColor` vars — duplicating work and creating two independent `color.Color` objects from the same source constants.

2. `applyStyle` in `pkg/console` gated on TTY presence only (`isTTY()`), but did not consult the color-profile environment (`NO_COLOR`, `COLORTERM`, `TERM`). A terminal session with `NO_COLOR=1` still received raw ANSI escape sequences from string-returning format helpers, while the equivalent stderr path already honored the color profile via `stderrWriter()`.

### Decision

We will make two coordinated changes to address both defects:

1. Declare 22 package-level `color.Color` vars in `pkg/styles/theme.go`, parsed once from the existing hex constants at init time, and have both the `adaptiveColor` structs and `huh_theme.go`'s `LightDark` calls reference these shared vars — eliminating duplicate hex parsing and establishing a single source of truth for the palette.

2. Add `colorwriter.Stdout()` and `colorwriter.Degrade(s, environ)` to `pkg/colorwriter`, then update `applyStyle` in `pkg/console` to call `colorwriter.Degrade` when stdout is a TTY, routing rendered ANSI through a `colorprofile.Writer` that honors `NO_COLOR`, `COLORTERM`, and `TERM` — matching the existing stderr behavior.

### Alternatives Considered

#### Alternative 1: Keep Inline `lipgloss.Color()` Calls in `huh_theme.go`

Leave `huh_theme.go` parsing hex strings on each `HuhTheme(isDark)` call. This is self-contained and requires no structural change. Rejected because it duplicates parsing work already done in `theme.go` and creates two independent `color.Color` objects from the same constants, violating the single-source-of-truth principle and adding unnecessary per-render overhead.

#### Alternative 2: Route All Stdout Output Through an `io.Writer` Pipeline

Replace the string-returning `applyStyle` pattern with an io.Writer pipeline where all styling writes directly to `colorwriter.Stdout()`. This would eliminate the need for a separate `Degrade` step. Rejected because it would require refactoring every caller from string-returning helpers to an io.Writer-passing API — a significantly larger change that does not align with the cleanup scope of this PR.

#### Alternative 3: Check `NO_COLOR` Inline in `applyStyle`

Manually check `os.Getenv("NO_COLOR") != ""` inside `applyStyle` before applying styles. Simpler than using `colorwriter.Degrade`, but covers only `NO_COLOR` and ignores the full color-profile spec (`COLORTERM`, `TERM`, capability-level downgrading). Rejected because `colorprofile.Writer` already encodes the full spec, and partial coverage would leave users of non-standard terminals without proper degradation.

### Consequences

#### Positive
- Single source of truth for palette colors: hex-to-`color.Color` parsing happens exactly once at package init, shared by both the `adaptiveColor` (startup-probe) path and the `LightDark` (per-render) path in huh themes.
- `NO_COLOR`, `COLORTERM`, and `TERM` are now honored for all stdout-bound styled strings, matching the existing behavior on the stderr path.
- Wasm build remains fully functional: platform stubs for `Stdout()` and `Degrade()` pass through unchanged.

#### Negative
- `applyStyle` now calls `colorwriter.Degrade` on every TTY invocation, adding one extra string allocation and a `colorprofile.Writer` pass compared to the previous direct `style.Render(text)` return.
- Adding 22 package-level `color.Color` vars to `pkg/styles` marginally increases module initialization footprint.

#### Neutral
- The two rendering paths (`adaptiveColor` structs for startup-probe selection vs `lipgloss.LightDark` for per-render selection in huh themes) remain architecturally distinct; this PR makes the shared palette the common underpinning but does not unify the selection strategies.
- `applyStyleWithTTY` (stderr-bound helpers) is unchanged; those paths already obtain colorprofile degradation at print time through `stderrWriter()`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
