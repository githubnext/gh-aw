# ADR-43587: Centralise `huh` Form Construction Wrappers in `pkg/console`

**Date**: 2026-07-05
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `huh` TUI form library requires callers to chain `.WithTheme(styles.HuhTheme).WithAccessible(console.IsAccessibleMode())` onto every `huh.NewForm(...)` call to apply the project's visual theme and respect the user's accessibility preferences. Before this change, this two-call chain was duplicated across at least a dozen callsites in `pkg/cli` (e.g., `add_interactive_auth.go`, `add_interactive_engine.go`, `engine_secrets.go`, `run_interactive.go`, `interactive.go`, and others). Any callsite that omitted the chain would silently render a form with incorrect styling or ignore the user's accessibility setting, creating an inconsistency that was invisible at compile time and hard to catch in review.

### Decision

We will introduce a small set of constructor wrappers in `pkg/console` (`NewForm`, `NewInputForm`, `NewSelectForm`, `NewMultiSelectForm`, `NewTextForm`, `NewConfirmForm`) that bake in `.WithTheme(styles.HuhTheme).WithAccessible(IsAccessibleMode())`. All interactive form callsites in `pkg/cli` and within `pkg/console` itself will migrate to these wrappers. The wrappers are gated to non-WASM builds with a `//go:build !js && !wasm` tag, preserving the existing WASM exclusion already present in the package.

### Alternatives Considered

#### Alternative 1: Keep per-callsite chaining (status quo)

Each callsite continues to call `huh.NewForm(...).WithTheme(styles.HuhTheme).WithAccessible(console.IsAccessibleMode())` directly. No new abstractions are added. This is trivially easy to understand but scales poorly: the theme/accessibility coupling must be remembered and applied by every future contributor adding a new form, with no compiler enforcement. A single missed chain silently degrades the UX.

#### Alternative 2: Configure a global default theme on `huh` at program startup

If the `huh` library supports setting a process-wide default theme, it could be set once in `main` and omitted at callsites. This would be even less code at callsites. However, at the time of this change, the `huh` library (`charm.land/huh/v2`) does not expose a global theme default — theme must be set per-form. A future library upgrade that adds this capability would allow this approach and could supersede this ADR.

### Consequences

#### Positive
- Theme and accessibility mode are applied consistently to all forms; a new callsite that uses a wrapper cannot forget to apply them.
- Callsite code is simpler and shorter, reducing the boilerplate that developers must write and review.
- The single theme/accessibility configuration point means future changes to defaults (e.g., switching themes, changing accessibility detection) require editing one place instead of N callsites.

#### Negative
- The wrappers add one extra layer of indirection between callsites and the `huh` API, which a reader must look up to understand what configuration is applied.
- The `//go:build !js && !wasm` build tag on `prompt_form.go` means WASM/JS builds still lack these wrappers; any WASM callsite that needs interactive forms requires a separate, manually maintained path.

#### Neutral
- Existing higher-level helpers (`ConfirmAction`, `PromptSecretInput`, `ShowInteractiveList`) in `pkg/console` are also migrated to use the new wrappers, eliminating the last direct `huh.NewForm` calls inside the package itself.
- The change is purely structural: no prompt behaviour, field definitions, or observable UX outcomes change.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
