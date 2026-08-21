# ADR-46570: Add Top-Level `model` Field, Allow `engine.model` as a Per-Instance Override

**Date**: 2026-07-19
**Status**: Accepted
**Deciders**: @pelikhan

---

### Context

Workflow frontmatter currently requires users to nest a single scalar (the LLM model name) inside the `engine` object as `engine.model`. This forces users to understand the engine object structure just to set a model, even when they do not need to customize any other engine behavior. Other top-level scalar configuration fields (`max-turns`, `max-runs`, `max-tool-denials`) follow a flat, top-level convention. `engine.model` is therefore an ergonomic outlier that increases cognitive load and verbosity for a common operation. A top-level `model` field would align model selection with the existing convention and reduce friction.

A follow-up issue (github/gh-aw#54047) showed that unconditionally deprecating `engine.model` in favor of the top-level field breaks a legitimate use case: workflows that run a different engine (and therefore a differently-namespaced model) for `safe-outputs.threat-detection` than for the main agent. In that split-engine scenario the top-level `model:` can only hold one model ID, so `threat-detection.engine.model` is the *only* way to select a different model for the detection engine. Deprecating it, and nudging users toward the top-level field via `gh aw fix`, made that scenario unsupportable.

### Decision

We introduce a canonical top-level `model` field in workflow frontmatter for specifying the LLM model used by the agentic engine. `engine.model` remains a fully supported, non-deprecated way to set (or override) the model for a *specific* engine instance: when both are present, the nested `engine.model` takes precedence over the top-level `model` for that engine instance, since it is the more specific setting. This makes it possible to set a workflow-wide default via top-level `model` while still overriding it per engine instance (e.g. `safe-outputs.threat-detection.engine.model`). We removed the `engine-model-to-top-level` codemod and the compile-time deprecation warning, since automatically migrating `engine.model` to the top level would silently break the split-engine use case.

### Alternatives Considered

#### Alternative 1: Keep `engine.model` as the Canonical Location

Retain the existing `engine.model` syntax with no changes. Users who want to change the model must still nest it inside an engine object. This avoids migration complexity and any risk of naming conflicts at the top level. However, it perpetuates the ergonomic inconsistency: users must learn the engine object structure to perform a common operation, and documentation becomes harder to write because `model` and other engine fields are at different levels of the hierarchy.

#### Alternative 2: Use a Distinct Name (e.g., `llm-model`) for the Top-Level Field

Introduce a top-level `llm-model` field instead of `model` to avoid any potential ambiguity with other hypothetical uses of `model`. This eliminates the name-collision risk but introduces a less intuitive, hyphenated field name that diverges from both YAML conventions and common agentic tooling vocabulary. Given that `model` is already the universally understood name for LLM model selection, the disambiguation is not worth the usability cost.

#### Alternative 3: Deprecate `engine.model` and Auto-Migrate It to the Top Level

The original version of this ADR deprecated `engine.model`, emitted a compile-time warning, and shipped a `gh aw fix` codemod to move it to the top-level `model` field. This was reverted (github/gh-aw#54047) because it broke split-engine workflows (e.g. main agent on one engine, `safe-outputs.threat-detection.engine` on another) where the two engine instances legitimately need different models. There is no single top-level value that can represent both, so nudging users to migrate away from `engine.model` — and having `gh aw fix` do so automatically — actively broke working configurations.

### Consequences

#### Positive
- Simpler, flatter YAML: users can set `model: gpt-5.4` without understanding the engine object structure.
- Consistent with the existing convention for top-level scalar fields (`max-turns`, `max-runs`, etc.).
- `engine.model` remains available, without warnings, as the mechanism for overriding the model on a specific engine instance (e.g. `safe-outputs.threat-detection.engine.model`) independently of the top-level default.

#### Negative
- Two syntactically valid ways to configure the model coexist indefinitely (top-level default vs. per-instance override), which slightly increases precedence logic complexity and requires clear documentation of "nested wins over top-level".

#### Neutral
- The `FrontmatterConfig` struct has a `Model` field; consumers that serialize or deserialize frontmatter must account for it.
- The JSON schema has a top-level `model` property; `engine.model` is documented as a per-instance override with no `deprecated`/`x-deprecation-message` markers.

---

*ADR created by [adr-writer agent]. Amended following github/gh-aw#54047 to reverse the `engine.model` deprecation.*
