# ADR-54288: Add Pinned Agent Plugins Support via Top-Level `plugins` Frontmatter

**Date**: 2026-08-20
**Status**: Draft
**Deciders**: pelikhan, app/copilot-swe-agent

---

### Context

Agentic engines hosted in gh-aw (Copilot CLI, Claude Code, and imported engines such as Cursor and Kiro) need a standardized way to extend their capabilities with third-party Agent Plugins. Prior to this change, extension mechanisms were entirely engine-specific (e.g., Pi's `engine.extensions` npm packages) and not portable across engines. There was no compile-time guarantee that plugins would be installed from an immutable ref, creating a supply-chain risk if a moving branch or tag were used. The project needed a uniform, cross-engine plugin delivery mechanism that integrates with the existing compilation and validation pipeline.

### Decision

We will introduce a top-level `plugins` frontmatter field that accepts `owner/repository[/path]@ref` entries. During compilation, gh-aw resolves every branch or tag ref to a pinned commit SHA; compilation fails if any ref cannot be resolved. At runtime, each plugin is checked out at its pinned SHA, then made available to the engine either by staging it in the engine's plugin directory (imported engines via `behaviors.plugins.directory`) or by invoking the engine CLI's installation command (Copilot uses `copilot plugin install`, Claude uses `--plugin-dir`). Engines that declare no `behaviors.plugins` block reject `plugins:` at compile time. Duplicate identical plugin refs are deduplicated; compatible semver ranges are merged to the highest version.

### Alternatives Considered

#### Alternative 1: Per-engine extension fields (status quo)

Each engine exposes its own extension mechanism (e.g., `engine.extensions` for Pi with npm packages). This already exists and requires no new abstractions. It was not chosen because it is not portable across engines, provides no uniform validation, and allows installing from moving refs without SHA pinning—exposing workflows to supply-chain attacks when an upstream package changes.

#### Alternative 2: Plugin installation via `pre-agent-steps`

Users could install plugins by adding arbitrary shell steps under `pre-agent-steps`. This is maximally flexible but provides no standardized plugin format, no compile-time ref validation, no SHA pinning, and no cross-engine compatibility. It was not chosen because it shifts all security and compatibility responsibility onto workflow authors and cannot be validated or enforced by the compiler.

### Consequences

#### Positive
- Pinned commit SHAs prevent supply-chain attacks; generated workflows never install a plugin from a moving ref.
- A single, uniform `plugins:` field works across all supported engines (Copilot, Claude, and any imported engine that declares `behaviors.plugins`), replacing ad-hoc per-engine extension patterns.
- Compile-time deduplication and semver merging prevent duplicate or conflicting plugin installations when shared workflows each declare the same plugin.

#### Negative
- The feature is experimental and explicitly emits a compile-time warning; the interface may change in future releases, requiring consumers to update.
- Only two built-in engines support Agent Plugins natively (Copilot CLI and Claude Code); all other engines require an explicit `behaviors.plugins` block in their definition, adding a maintenance burden for imported engine authors.
- Using a plugin from a non-semver ref (e.g., a raw branch SHA) that conflicts with another import fails compilation, which may surprise authors who mix plugin sources across shared imports.

#### Neutral
- The `plugins` field is merged during import processing using breadth-first traversal, consistent with how other mergeable fields (like `mcp-servers`) are handled.
- The engine capability flag `Plugins bool` on `EngineCapabilities` follows the established pattern of feature-flagging optional engine capabilities (e.g., `BashCommandAllowlist`, `WebSearch`, `BareMode`).

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
