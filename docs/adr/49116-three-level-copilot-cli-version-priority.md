# ADR-49116: Three-Level Version Priority for Copilot CLI Installation

**Date**: 2026-07-30
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

### Context

The `install_copilot_cli.sh` script supports multiple sources for the Copilot CLI version to install: an explicit positional argument, a compatibility window (`compat.json`), a toolcache lookup, and a baked-in default. The Go compiler (`GenerateCopilotInstallerSteps`) was always passing `DefaultCopilotVersion` as a positional argument when no `engine.version` was set in the workflow configuration. This meant `compat.json` resolution and toolcache lookup were permanently bypassed — every lock file contained a hardcoded version string even when the intent was to defer to the compatibility window. The result was that `compat.json`-driven version pinning had no practical effect, and all 266 compiled lock files required manual updates whenever the default version changed.

### Decision

We will enforce a strict three-level version priority in the compiler and install script:
1. **Explicit `engine.version`** — passed as a positional arg; the script uses it directly.
2. **No `engine.version`** — the compiler passes no positional arg and instead injects `GH_AW_COMPILED_VERSION` as an env var; the script then resolves via `compat.json` or toolcache.
3. **Compat unavailable** — the script falls back to a `DEFAULT_COPILOT_VERSION` constant baked into the script itself (replacing the prior `exit 1`).

The `WorkflowData` struct gains a `CompiledVersion` field so the compiler can propagate its own version as `GH_AW_COMPILED_VERSION` without coupling it to `DefaultCopilotVersion`. All 266 lock files are recompiled to remove the previously-hardcoded positional version argument.

### Alternatives Considered

#### Alternative 1: Keep the Hardcoded Positional Argument (Status Quo)

The compiler always passes `DefaultCopilotVersion` as a positional arg regardless of whether `engine.version` is set. Simple, predictable, but it makes `compat.json` dead code for the majority of workflows. Any default version bump requires recompiling all lock files, and `compat.json` cannot serve its intended purpose of decoupling version selection from the compiler.

#### Alternative 2: Single Env Var for All Cases

Remove the positional arg entirely and always convey the version (whether explicit or default) through an env var. This unifies the interface but loses the semantic distinction between "user explicitly pinned a version" and "use the best available version at install time." It would also require the script to decide which env var takes priority, re-creating a priority problem inside the script rather than at the compiler level.

### Consequences

#### Positive
- `compat.json` version resolution is now active for all workflows that do not pin `engine.version`, enabling compatibility-window-driven upgrades without recompilation.
- Lock files no longer embed a hardcoded Copilot CLI version when `engine.version` is unset, reducing churn on lock file recompilation when the default changes.
- The install script degrades gracefully (uses `DEFAULT_COPILOT_VERSION`) instead of exiting with an error when `compat.json` is unavailable.

#### Negative
- All 266 lock files required a bulk recompilation, producing a large PR diff that is hard to review in detail.
- The three-level priority adds conditional logic to both the Go compiler and the shell script, increasing the surface area for version-resolution bugs.

#### Neutral
- `WorkflowData` gains a new `CompiledVersion string` field; callers that do not use Copilot installation are unaffected but carry the extra struct field.
- The `GenerateCopilotInstallerSteps` function signature changed (added `compiledVersion string` parameter); any other callers outside the reviewed files must be updated.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
