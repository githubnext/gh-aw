# ADR-32805: Non-Handler Top-Level Keys in Safe-Outputs Handler Config

**Date**: 2026-05-17
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG` environment variable is a JSON object emitted by the Go compiler (`addHandlerManagerConfigEnvVar` in `pkg/workflow/compiler_safe_outputs_config.go`) and consumed at runtime by `safe_output_handler_manager.cjs`. Historically every top-level key corresponded to a registered handler (`add_comment`, `create_issue`, etc.), and the map was strongly typed as `map[string]map[string]any`. The handler manager already contained forwarding logic that copied the top-level `mentions` block into the `add_comment` handler's per-handler config — but the compiler never emitted that `mentions` key, so the forwarding path was dead code. As a result, `mentions.allowed` entries (e.g. `@copilot` in `pr-sous-chef`) were unconditionally escaped in `add_comment` output even when the workflow had explicitly allowed them.

### Decision

We will allow `GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG` to contain non-handler top-level keys ("global runtime knobs") that the handler manager forwards into specific handlers at startup. Concretely, the compiler now emits a top-level `"mentions"` key when `safeOutputs.Mentions` is configured, and the Go config map type is widened from `map[string]map[string]any` to `map[string]any` to accommodate both per-handler maps and non-handler scalars/objects. The forwarding contract (which top-level keys map to which handlers) lives in `safe_output_handler_manager.cjs` and is documented in `safe_outputs_config_generation.go`.

### Alternatives Considered

#### Alternative 1: Duplicate the mentions block into every per-handler config

Instead of a top-level `mentions` key, the compiler could embed a copy of the mentions config directly inside `add_comment` (and any future consumer's) per-handler block. This keeps the strongly-typed `map[string]map[string]any` shape intact, but duplicates data across handlers, makes the source of truth ambiguous if values diverge during compilation, and forces every new mentions consumer to be touched at compile time rather than declaratively forwarded at runtime.

#### Alternative 2: Pass mentions via a separate environment variable

Emit `GH_AW_SAFE_OUTPUTS_MENTIONS_CONFIG` as its own env var and read it independently from the handler manager. This avoids touching the handler config shape but fragments the configuration surface: workflow authors and reviewers would need to inspect multiple env vars to understand what's in scope, and any future global knob (`max`, `allowContext`, etc.) would multiply that fragmentation. The handler manager's existing forwarding logic already expects one config blob, so this would also require throwing away working runtime code.

#### Alternative 3: Promote mentions to a first-class handler in `handlerRegistry`

Register `mentions` as a pseudo-handler so it fits the existing per-handler typing. This is misleading — `mentions` does not produce a safe output, it parameterizes one — and would distort the meaning of `handlerRegistry`. It would also require adding shim plumbing in every part of the handler dispatch pipeline that walks the registry.

### Consequences

#### Positive
- The dead forwarding path in `safe_output_handler_manager.cjs` becomes live; `mentions.allowed` aliases are no longer escaped in `add_comment` output, fixing the user-visible bug.
- The pattern is reusable: future global runtime knobs (additional sanitization policies, rate-limit overrides, etc.) can be added as top-level keys without changing the env-var contract or duplicating data across handlers.
- Single source of truth for global knobs survives in the compiler and is forwarded once at runtime, instead of being copied per handler.

#### Negative
- The Go config map is no longer strongly typed as `map[string]map[string]any`; readers must know which top-level keys are handler entries (objects shaped like a handler config) and which are non-handler globals. The compiler now relies on a doc comment plus `handlerRegistry` membership rather than the type system to communicate that distinction.
- Adding a new global knob requires touching three files in lockstep: the Go config builder, the JS handler manager (for forwarding), and `safe_outputs_config_generation.go` (for documentation). There is no compile-time check that the three stay aligned.

#### Neutral
- Per-handler `handlerConfig` values are now cast through `map[string]any(handlerConfig)` when written into the outer map; this is a syntactic change with no semantic effect.
- The lock file `pr-sous-chef.lock.yml` now carries an extra `"mentions":{"allowed":["copilot"]}` block, which will appear on regeneration in any workflow that configures `safe-outputs.mentions`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Handler Config Map Shape

1. The compiler **MUST** emit `GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG` as a JSON object whose top-level keys are either (a) names registered in `handlerRegistry` or (b) names of documented global runtime knobs.
2. The Go in-memory representation of this config **MUST** be `map[string]any` so that both per-handler maps and non-handler globals can coexist.
3. Per-handler entries **MUST** be JSON objects (maps) at the top level; non-handler globals **MAY** be any JSON-compatible value documented in `safe_outputs_config_generation.go`.
4. The compiler **MUST NOT** emit a top-level key that is neither a registered handler nor a documented global runtime knob.

### Mentions Configuration

1. When `safeOutputs.Mentions` is non-nil and `buildMentionsHandlerConfig` returns a non-empty map, the compiler **MUST** include a top-level `"mentions"` key in `GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG`.
2. When `safeOutputs.Mentions` is nil or empty, the compiler **MUST NOT** include a top-level `"mentions"` key.
3. The `safe_output_handler_manager.cjs` runtime **MUST** forward the top-level `mentions` block into the `add_comment` handler's per-handler config when that handler is dispatched and the handler config does not already contain a `mentions` entry.
4. The runtime **MUST NOT** overwrite an existing per-handler `mentions` entry with the top-level one (per-handler config wins).

### Documentation Contract

1. Any non-handler top-level key that the compiler emits **MUST** be documented in `safe_outputs_config_generation.go` (or a successor documentation site referenced from it) before it is added to the compiler output.
2. The forwarding semantics from a non-handler key to a specific handler **SHOULD** be implemented in `safe_output_handler_manager.cjs` and covered by a unit test that asserts the top-level key reaches the target handler.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25991308912) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
