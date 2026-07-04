# ADR-43195: Refactor Three `pkg/workflow` Largefunc Hotspots into Local Helpers

**Date**: 2026-07-04
**Status**: Draft
**Deciders**: Unknown

---

### Context

Three functions in `pkg/workflow` exceeded the `largefunc` linter threshold, each mixing unrelated concerns in a single body:

1. **`ActionResolver.ResolveSHA`** contained an inline loop over the embedded pin set to check for a semver-compatible pin before falling back to the GitHub API. The resolution sequence (failed-run short-circuit → cache hit → embedded pin → GitHub API) was correct but obscured by the in-body loop.

2. **`computeAntigravityToolsCore`** contained both wildcard detection and per-command string mapping for bash tool expansion in one flat body, making the two distinct concerns hard to read and test independently.

3. **`buildInputSchema`** contained per-input schema construction inline, including metadata extraction (`description`, `required`) and property assembly, making it harder to follow the logic for each input type (`number`, `boolean`, `choice`, string fallback).

### Decision

We will extract focused local helpers from each hotspot without changing observable behavior:

- **`resolveFromEmbeddedPins(repo, version string) (string, bool)`** — moves the semver-compatible pin scan out of `ResolveSHA`. The embedded-pin path intentionally does not write to the on-disk cache (to avoid creating root-owned files in Docker/Alpine CI environments), and the helper documents this invariant explicitly with a comment. The extracted helper is not registered in the on-disk cache; that remains the exclusive responsibility of `resolveFromGitHub`.

- **`appendAntigravityBashTools`** and a wildcard detector — separate wildcard detection from per-command `run_shell_command(...)` generation. The canonical output format (`run_shell_command(...)`) is unchanged.

- **`buildInputSchemaProperty`**, **`getInputSchemaMetadata`**, and **`newInputSchemaProperty`** — break per-input schema assembly into three focused helpers. `buildInputSchemaProperty` owns type dispatch; `getInputSchemaMetadata` owns description/required extraction with fallback; `newInputSchemaProperty` owns property map construction with optional default.

Targeted regression tests are added to lock in the behavioral invariants that motivated the extraction: embedded pins do not write to the cache, invalid `choice.options` shapes fall back to plain string properties, and non-string bash command entries are silently ignored during Antigravity tool expansion.

### Alternatives Considered

#### Alternative 1: Add Comments Only

Add explanatory comments inside each large function to demarcate sections without extracting helpers. This avoids function proliferation.

Rejected because comments do not enforce separation or reduce function length, the `largefunc` linter finding would remain, and the logic would continue to be tested only through the public surface (no targeted tests for the embedded behavior).

#### Alternative 2: Move Helpers to Separate Files

Move each extracted helper into a new file (e.g., `embedded_pins.go`) to make the extracted concern explicitly discoverable.

Not pursued for this change because the helpers are tightly coupled to their parent functions and have no callers outside the immediate context. Moving them to separate files would add file overhead without improving navigation. Future callers can motivate a move at that time.

### Consequences

#### Positive
- `largefunc` linter findings for all three hotspots are resolved without disabling the linter.
- Each extracted helper has a documented single responsibility, making future edits easier to scope.
- New regression tests make it harder to accidentally reintroduce the subtle embedded-pin caching bug (Docker/Alpine file ownership).

#### Negative
- Developers following a code path must now trace into helper calls even though each hop is simple.
- Three new unexported helpers in `build_input_schema.go` increase the surface of the file without reducing its line count.

#### Neutral
- No logic changes are introduced; all behavioral invariants are preserved.
- The existing test coverage for `ResolveSHA`, `computeAntigravityToolsCore`, and `buildInputSchema` continues to exercise the helpers indirectly through their callers.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
