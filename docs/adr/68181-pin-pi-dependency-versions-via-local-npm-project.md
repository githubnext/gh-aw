# ADR-68181: Install the Pi CLI into a gh-aw-owned npm project with pinned `@earendil-works` overrides

**Date**: 2026-10-13
**Status**: Draft
**Deciders**: Unknown *(generated from PR #68181 — add the real deciders)*

---

### Context

gh-aw pins the Pi engine to `@earendil-works/pi-coding-agent@1.1.0` (`DefaultPiVersion` in `pkg/constants/version_constants.go`) and installs it globally with `npm install -g` and no lockfile. That package declares its sibling `@earendil-works/*` packages as caret ranges (`^1.1.0`). When upstream released the 1.2.0 set on 12 Oct 2026, npm began resolving those carets to 1.2.0, which contains breaking changes — notably the removal of `MCP_TYPESCRIPT_PREAMBLE` from `@earendil-works/pi-codemode/declarations`. Every `pi` engine workflow then failed roughly two seconds into the **Execute Pi CLI** step with an ESM link error (issue #68172). The breakage is not limited to `pi-codemode`: `pi-mcp`, `pi-tui` and `pi-coding-agent` each shipped breaking changes in the same 1.2.0 wave, so upstream minor releases cannot be treated as semver-safe for this package set.

The natural fix — npm `overrides` — is not available for global installs: npm only honours `overrides` from a root project manifest and ignores them entirely for `npm install -g`.

### Decision

We will stop installing the Pi CLI globally and instead install it into a gh-aw-owned npm project directory (`${RUNNER_TEMP}/gh-aw/pi-cli` on the host, `${RUNNER_TEMP}/gh-aw/engine-cli` in the microVM). The compiler writes a `package.json` containing `overrides` that pin every `@earendil-works` sibling (`chord`, `pi-agent-core`, `pi-ai`, `pi-codemode`, `pi-mcp`, `pi-telemetry`, `pi-tui`) to the same version as `DefaultPiVersion`, then runs `npm install --prefix` into that directory and symlinks the `pi` binary into `$(npm prefix -g)/bin` so PATH lookups keep working unchanged. The pinned set lives in a single new constant, `DefaultPiDependencyVersions`, so a future bump touches only that map plus `DefaultPiVersion`. Overrides are written only when the workflow uses `DefaultPiVersion`; a custom `engine.version` gets an empty manifest (`{"private":true}`) so gh-aw never forces mismatched pins onto a user-chosen release.

### Alternatives Considered

#### Alternative 1: Pin only `@earendil-works/pi-codemode`

A single override for the package whose removed export caused the crash would have been the smallest possible change and the fastest to ship. It was rejected because the 1.2.0 changelogs list breaking changes in `pi-mcp` (`CallToolResult.structuredContent` typing), `pi-tui` (editor/input cursor markers) and `pi-coding-agent` (`before_agent_start` semantics, moved theme schema) as well. Pinning one package would leave the remaining siblings free to drift and reproduce the same class of failure on the next upstream release.

#### Alternative 2: Bump `DefaultPiVersion` to the 1.2.0 set

The 1.2.0 packages are internally consistent, so simply moving the top-level pin forward would also have restored a working install. This was rejected as a fix for the outage: it swaps an unvetted dependency tree in under incident pressure, carries the breaking changes of four packages into gh-aw's runtime surface at once, and still leaves the underlying mechanism — unconstrained caret ranges in a global, lockfile-free install — in place. The bump is explicitly deferred to a follow-up once the pinning mechanism exists.

#### Alternative 3: Commit a lockfile and install from it

An `npm ci` from a committed `package-lock.json` would give fully reproducible, transitively-pinned installs. It was not chosen for this change because it requires generating and maintaining a lockfile for every supported platform/arch combination and a refresh process for routine bumps; the PR notes it as follow-up work. `overrides` achieves the needed determinism for the direct sibling set with far less machinery.

### Consequences

#### Positive
- Pi engine workflows become immune to breaking minor releases of the `@earendil-works` sibling packages; the installed tree is deterministic for a given `DefaultPiVersion`.
- All version pins are centralised in `pkg/constants/version_constants.go`, so a future upgrade is a one-map edit rather than a hunt through generated shell.
- The `pi` symlink into the global bin directory preserves existing PATH-based discovery, so AWF, `pi_runtime.cjs` and the NVX lookup in `nvx_assets.cjs` need no changes.
- Host and microVM install paths now share one pinning mechanism, removing a source of drift between the two environments.

#### Negative
- The pin map must be hand-maintained: a `DefaultPiVersion` bump that forgets to update `DefaultPiDependencyVersions` silently reintroduces a mismatched tree, and nothing currently verifies the installed versions post-install.
- Users who set an explicit `engine.version` get no pinning at all, so they remain exposed to exactly the failure mode this ADR fixes.
- `overrides` only constrain the listed direct siblings; deeper transitive dependencies are still resolved freely, so this is weaker than a lockfile.
- The generated install step is longer and less obvious than `npm install -g`, raising the cost of reading and debugging workflow logs.

#### Neutral
- `GH_AW_PI_PACKAGE_ROOT` now points at the new install directory instead of the global prefix; consumers that resolve pi through that variable follow the move automatically.
- All 36 pi `.lock.yml` files and the wasm golden (`pkg/workflow/testdata/TestWasmGolden_AllEngines/pi.golden`) are regenerated; the only semantic change is the install block.
- `@earendil-works/pi-telemetry` is pinned even though it is a transitive dependency of `pi-ai`, because it moved to 1.2.0 in the same release wave.
- Pinning for `engine.extensions` (installed via `pi install`) is out of scope and left for a follow-up.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
