# ADR-27626: Introduce sandbox.agent.version and Remove Deprecated network.firewall Field

**Date**: 2026-04-21
**Status**: Draft
**Deciders**: Unknown (copilot-swe-agent / pelikhan)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `network.firewall` frontmatter field was previously used to configure the Agent Workflow Firewall (AWF) in agentic workflow definitions. It was deprecated in favor of the unified `sandbox.agent` configuration block, but the migration codemod only handled the `true` case, leaving `false`, `null`, `"disable"`, and object-with-version forms unmigrated. Additionally, there was no mechanism to pin a specific AWF version via frontmatter — users who needed to run a particular AWF release had no stable, documented way to express that constraint. This PR addresses both gaps simultaneously: removing the deprecated field from the schema and expanding the codemod to cover all value variants.

### Decision

We will remove `network.firewall` from the frontmatter schema entirely and add `sandbox.agent.version` as a first-class string field for pinning the AWF version used during installation and runtime. The codemod will be expanded to migrate all `network.firewall` value forms — `true`, `false`, `null`, `"disable"`, and `{version: ...}` objects — to their `sandbox.agent` equivalents. This consolidates AWF configuration under a single `sandbox.agent` surface and ensures the migration path covers every variant that appears in the wild.

### Alternatives Considered

#### Alternative 1: Retain network.firewall as a Deprecated Alias

Keep `network.firewall` in the schema with a deprecation warning, parsing it alongside `sandbox.agent` and merging the values at runtime. This avoids a hard removal and gives teams more migration runway, but perpetuates two competing configuration surfaces indefinitely, increases parser complexity, and makes it harder to reason about precedence when both fields are set.

#### Alternative 2: Introduce a Flat sandbox.awf-version Field

Add a top-level sibling key `sandbox.awf-version` (or similar) rather than nesting the version under `sandbox.agent`. This is marginally more ergonomic to type but diverges from the `sandbox.agent` object model already established, creates a second place where AWF version information lives, and complicates the precedence rules for effective version resolution.

### Consequences

#### Positive
- All `network.firewall` value forms now have a deterministic, tested migration path to `sandbox.agent`.
- Users can pin an explicit AWF version via `sandbox.agent.version`, enabling reproducible builds without relying on the latest release.
- The schema surface is reduced by removing a deprecated field and its associated validation rules.

#### Negative
- Behavioral change: `network.firewall: false` previously produced no `sandbox` block; it now migrates to `sandbox.agent: false`. Workflows relying on the old no-op behavior will see a new block added by the codemod.
- The `normalizeFirewallVersion` helper must handle all numeric YAML types (int8 through uint64, float32/float64) because YAML parsers may unmarshal numeric version values into any of these types, increasing codemod surface area.

#### Neutral
- The codemod expansion requires updating existing tests that asserted `sandbox:` was *not* added for `false` and nested object cases; these expectations were inverted.
- The `aw-info` AWF version reporting now reads `sandbox.agent.version` in addition to the legacy `firewallVersion` field, requiring both paths to be tested.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Frontmatter Schema

1. Implementations **MUST NOT** include `network.firewall` in the frontmatter schema as a valid, non-deprecated field.
2. Implementations **MUST** expose `sandbox.agent.version` as an optional string field in the frontmatter schema for specifying an AWF version override.
3. The `sandbox.agent.version` field **MUST** be treated as a string type; numeric-like values written in YAML **MUST** be quoted at generation time to prevent YAML parsers from interpreting them as numbers.

### Codemod Migration

1. The `network.firewall` codemod **MUST** produce a `sandbox.agent` block for every non-absent value of `network.firewall`, including `true`, `false`, `null`, `"disable"`, and object forms.
2. When `network.firewall` is `true` or `null`, the codemod **MUST** emit `sandbox.agent: awf`.
3. When `network.firewall` is `false` or `"disable"`, the codemod **MUST** emit `sandbox.agent: false`.
4. When `network.firewall` is an object containing a `version` key, the codemod **MUST** emit a `sandbox.agent` object block with `id: awf` and `version: "<migrated-value>"`.
5. The codemod **MUST NOT** add a `sandbox` block when one already exists in the frontmatter.
6. Numeric version values encountered during migration **MUST** be normalized to their string representation before being written as `sandbox.agent.version`.

### AWF Version Resolution

1. When `sandbox.agent.version` is set, it **MUST** take precedence over any version derived from the legacy `network.firewall` configuration when resolving the effective AWF version for installation and runtime.
2. Implementations **SHOULD** surface the resolved AWF version in `aw-info` metadata so that workflow authors can verify which version is active.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24736713102) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
