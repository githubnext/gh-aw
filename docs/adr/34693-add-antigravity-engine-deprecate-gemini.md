# ADR-34693: Add Antigravity Engine and Deprecate Gemini

**Date**: 2026-05-25
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

Google has begun positioning a successor CLI (`agy` / Antigravity) as the recommended path for users currently relying on the Gemini CLI engine. The `gh-aw` engine catalog ships `gemini` as a first-class agentic engine with its own constants, registry entry, MCP/logs/tools wiring, and AWF proxy target. Migrating users by simply renaming `gemini` → `antigravity` would silently break every existing workflow that pins `engine: gemini`. Removing `gemini` outright would do the same. The product constraint is to offer the new engine immediately while keeping every Gemini-using workflow running unchanged until authors migrate.

### Decision

We will introduce `antigravity` as a fully independent engine alongside `gemini`, register it through the existing engine catalog (`NewEngineRegistry`, `AgenticEngines`, `EngineOptions`, `data/engines/antigravity.md`), and mirror the Gemini constant family (`AntigravityEngine`, `AntigravityLLMGatewayPort`, `DefaultAntigravityVersion`, `AntigravityCLIModelEnvVar`, model env vars). Gemini remains fully functional, but the compiler emits a deprecation warning via `validateGeminiDeprecation()` whenever `engine: gemini` is detected, with a migration example pointing at `engine: antigravity`. AWF proxy configuration, default domains, and API target helpers populate both keys so workflows on either engine continue to route correctly while authors migrate.

### Alternatives Considered

#### Alternative 1: Replace Gemini Outright (Rename in Place)

Rename `gemini` → `antigravity` everywhere (engine identifier, constants, registry, AWF target key) and ship a one-shot codemod that rewrites `engine: gemini` to `engine: antigravity` in user workflows. This is the simplest end state — only one engine to maintain. Rejected because compiled `.lock.yml` files, third-party shared workflows, documentation, and pinned references in user repos cannot all be codemodded atomically; any workflow still saying `engine: gemini` after the cut would fail compilation, which violates the "no breaking changes" constraint stated in the PR description.

#### Alternative 2: Alias `gemini` to `antigravity` Silently

Keep `engine: gemini` as a valid identifier but transparently route it to the Antigravity binary, settings file, and env vars. No warning, no second registry entry. Rejected because it (a) hides which CLI is actually executing — confusing for logs, audit, and debugging; (b) couples the two engines' release cadences forever; (c) gives no signal to workflow authors that they should update their pins, so the deprecation never completes. The compile-time warning in this PR is the mechanism that drives the migration to its end state.

#### Alternative 3: Hard Deprecate with Removal Timer

Mark `gemini` deprecated and remove it on a fixed date / version. Rejected because `gh-aw` consumers update on their own schedule and a hard removal date would break workflows whose authors did not see (or could not act on) the warning in time. Soft deprecation with an indefinite warning matches how other engine transitions have been handled in this codebase (e.g., legacy aliases for API targets and domains are still present).

### Consequences

#### Positive
- Existing `engine: gemini` workflows continue to compile and run with no changes — backward compatibility is preserved end-to-end (deprecated `GetGeminiAPITarget`, `DefaultGeminiAPITarget`, `GeminiDefaultDomains` aliases retained).
- Workflow authors get an actionable compile-time warning with the exact migration snippet, so the path forward is discoverable without reading release notes.
- The catalog stays the source of truth for engines — both engines flow through the same `NewEngineRegistry` / `EngineOptions` / `data/engines/*.md` pipeline, so completions, validation, and docs stay consistent.
- AWF proxy keeps a single port (10003) for both engines, which means no new firewall rule or runner port allocation is required.

#### Negative
- Two engines with nearly identical wiring (`antigravity_engine.go` plus `_logs.go`, `_mcp.go`, `_tools.go` mirror the Gemini files) — every future Gemini/Antigravity engine change must be made in two places until Gemini is removed.
- Shared port (`AntigravityLLMGatewayPort = 10003 = GeminiLLMGatewayPort`) means a single workflow cannot run both engines concurrently in the same job; this is implicit and not statically enforced today.
- Domain and target aliasing (`GeminiDefaultDomains` → `AntigravityDefaultDomains`, dual keys in `awf_config.go`) is correct now but is a latent footgun: changes to one set must be mirrored to the other or one engine silently diverges.
- Carrying the deprecation warning indefinitely means CI logs for Gemini workflows will accumulate warning noise; there is no end-of-life date in this ADR.

#### Neutral
- `DefaultAntigravityVersion` is initially pinned to the same value as `DefaultGeminiVersion` (`0.39.1`). This is intentional during the parallel period but the versions will diverge as soon as Antigravity ships an independent release.
- `EngineOption` for Gemini is relabeled to "Gemini (deprecated)" in interactive prompts; users picking from the menu now see the deprecation, but scripted `--engine gemini` invocations do not.
- Both `antigravity` and `gemini` are valid `awf-config.schema.json` proxy target keys; downstream tooling that enumerates target keys must accept both.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Engine Registration

1. The compiler **MUST** register `antigravity` as a distinct `EngineName` in `pkg/constants/engine_constants.go` and include it in `AgenticEngines`, `EngineOptions`, and the registry returned by `NewEngineRegistry()`.
2. The compiler **MUST NOT** remove `gemini` from `AgenticEngines`, `EngineOptions`, `NewEngineRegistry()`, or `computeAllowedDomainsForSanitization` while this ADR is in effect.
3. The compiler **MUST** accept `engine: antigravity` and `engine: gemini` as valid top-level engine identifiers in workflow frontmatter.

### Gemini Deprecation Behavior

1. The compiler **MUST** emit a non-fatal deprecation warning when `engine: gemini` is detected during compilation (`validateGeminiDeprecation()`).
2. The deprecation warning **MUST** include a migration example that points to `engine: antigravity`.
3. The compiler **MUST NOT** fail compilation, alter generated workflow semantics, or change exit status solely because `engine: gemini` was used.
4. Workflows that pin `engine: gemini` **MUST** produce a `.lock.yml` that executes the Gemini CLI with `GEMINI_API_KEY` and `.gemini/settings.json`, unchanged from the pre-deprecation behavior.

### Antigravity Runtime Wiring

1. The Antigravity engine **MUST** invoke the `agy` CLI and **MUST** read its API key from the `ANTIGRAVITY_API_KEY` environment variable.
2. The Antigravity engine **MUST** read its settings from `.antigravity/settings.json`.
3. The Antigravity engine **MUST** expose its LLM gateway on the port defined by `AntigravityLLMGatewayPort` and **MUST NOT** hard-code a numeric literal at the call site.
4. The Antigravity engine **MUST** honor the `ANTIGRAVITY_MODEL` environment variable (`AntigravityCLIModelEnvVar`) for model selection.
5. The default Antigravity CLI version **MUST** be sourced from `DefaultAntigravityVersion` in `pkg/constants/version_constants.go`.

### AWF Proxy and Domain Configuration

1. `awf_config.go` **MUST** populate both `antigravity` and `gemini` target keys whenever either engine is in use.
2. `awf-config.schema.json` **MUST** accept `antigravity` as a valid proxy target key.
3. `GeminiDefaultDomains` **MUST** remain available as an alias of `AntigravityDefaultDomains` (or vice versa) so that existing references in the codebase compile.
4. `GetGeminiAPITarget` and `DefaultGeminiAPITarget` **MUST** remain exported as deprecated aliases and **MUST** return the same values they did before this change.

### Future Removal

1. Any future change that removes `gemini` from the engine catalog **MUST** be accompanied by a superseding ADR that records the removal decision and the cutoff condition.
2. A removal change **SHOULD NOT** be merged until the deprecation warning has been in place for at least one release.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26409186027) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
