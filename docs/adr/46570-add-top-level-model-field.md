# ADR-46570: Add Top-Level `model` Field, Deprecate `engine.model`

**Date**: 2026-07-19
**Status**: Draft
**Deciders**: Unknown

---

### Context

Workflow frontmatter currently requires users to nest a single scalar (the LLM model name) inside the `engine` object as `engine.model`. This forces users to understand the engine object structure just to set a model, even when they do not need to customize any other engine behavior. Other top-level scalar configuration fields (`max-turns`, `max-runs`, `max-tool-denials`) follow a flat, top-level convention. `engine.model` is therefore an ergonomic outlier that increases cognitive load and verbosity for a common operation. A top-level `model` field would align model selection with the existing convention and reduce friction.

### Decision

We will introduce a canonical top-level `model` field in workflow frontmatter for specifying the LLM model used by the agentic engine. The top-level field takes precedence over `engine.model` when both are present. We will simultaneously deprecate `engine.model`, emit a compile-time warning when it is encountered, and provide an automated codemod (`engine-model-to-top-level` via `gh aw fix`) to migrate existing workflows.

### Alternatives Considered

#### Alternative 1: Keep `engine.model` as the Canonical Location

Retain the existing `engine.model` syntax with no changes. Users who want to change the model must still nest it inside an engine object. This avoids migration complexity and any risk of naming conflicts at the top level. However, it perpetuates the ergonomic inconsistency: users must learn the engine object structure to perform a common operation, and documentation becomes harder to write because `model` and other engine fields are at different levels of the hierarchy.

#### Alternative 2: Use a Distinct Name (e.g., `llm-model`) for the Top-Level Field

Introduce a top-level `llm-model` field instead of `model` to avoid any potential ambiguity with other hypothetical uses of `model`. This eliminates the name-collision risk but introduces a less intuitive, hyphenated field name that diverges from both YAML conventions and common agentic tooling vocabulary. Given that `model` is already the universally understood name for LLM model selection, the disambiguation is not worth the usability cost.

### Consequences

#### Positive
- Simpler, flatter YAML: users can set `model: gpt-5.4` without understanding the engine object structure.
- Consistent with the existing convention for top-level scalar fields (`max-turns`, `max-runs`, etc.).
- Schema deprecation markers and a compile-time warning guide existing users toward migration automatically.
- The `gh aw fix` codemod enables zero-effort automated migration for the common case.

#### Negative
- A deprecation period means two syntactically valid ways to configure the model coexist, increasing implementation complexity (precedence logic) and potential user confusion.
- The inline-map engine syntax (`engine: { id: copilot, model: gpt-4 }`) cannot be automatically migrated by the codemod and requires manual intervention.
- Maintaining the codemod and deprecation warning adds ongoing test surface area that must be kept in sync.

#### Neutral
- The `FrontmatterConfig` struct gains a new `Model` field; consumers that serialize or deserialize frontmatter must account for it.
- The JSON schema gains a top-level `model` property and `deprecated`/`x-deprecation-message` markers on both `engine_config` variants — tooling that surfaces schema metadata will reflect this.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
