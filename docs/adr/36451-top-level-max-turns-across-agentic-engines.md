# ADR-36451: Promote `max-turns` to a Top-Level AWF Control Across Agentic Engines

**Date**: 2026-06-02
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

`max-turns` caps the number of chat iterations an agentic run may perform, bounding both runaway loops and cost. Historically it existed only as an engine-scoped field (`engine.max-turns`) and was treated as a Claude-only capability, so experiment-driven turn budgets could not be expressed once and applied uniformly across engines (Claude, Codex, Copilot, Antigravity). This blocked an experiment campaign that needed an AWF-level, templatable turn budget — including GitHub Actions expression values like `${{ inputs.max-turns }}` for reusable `workflow_call` workflows. The repository had already solved the analogous problem for `max-runs` (see [ADR-31418](31418-move-engine-max-runs-to-top-level-with-awf-enforcement.md)), establishing a precedent for lifting budget controls to the top level.

### Decision

We will make `max-turns` a **top-level workflow frontmatter field** that AWF enforces consistently across all agentic engines, rather than a Claude-only, engine-nested option. The field reuses the existing `templatable_integer` schema so it accepts both integer literals and GitHub Actions expressions, and the compiled value is exported as `GH_AW_MAX_TURNS` for each supporting engine's runtime. `engine.max-turns` is retained as a **deprecated backward-compatible alias**, and a `gh aw fix` codemod (`engine-max-turns-to-top-level`) migrates existing usage to the top-level field. The top-level value is preserved through built-in engine import/defaulting paths and through shared agentic-workflow import merges (first-wins), so it survives engine resolution.

### Alternatives Considered

#### Alternative 1: Keep `max-turns` engine-scoped and Claude-only

Leave the field under `engine.max-turns` and continue documenting it as unsupported for non-Claude engines. Rejected because it prevents a single experiment-controlled turn budget from being expressed in frontmatter and applied uniformly, and it diverges from the precedent already set for `max-runs`, leaving the budget-control surface inconsistent.

#### Alternative 2: Add a brand-new top-level field name (e.g. `turn-budget`) with no alias

Introduce a differently named top-level control and drop `engine.max-turns` outright. Rejected because it breaks existing workflows immediately, fragments terminology away from the well-understood `max-turns` name, and forfeits the smooth migration that a deprecated alias plus codemod provides.

### Consequences

#### Positive
- A single top-level `max-turns` value applies consistently across Claude, Codex, Copilot, and Antigravity engines.
- Templatable values (`${{ inputs.max-turns }}`) enable experiment- and `workflow_call`-driven turn budgets.
- Existing `engine.max-turns` workflows keep compiling, and `gh aw fix` automates migration to the new form.

#### Negative
- Two valid spellings (`max-turns` and deprecated `engine.max-turns`) coexist during the deprecation window, increasing surface area for parser, import, and codemod logic that must be tested.
- Enabling `MaxTurns` capability on previously-unsupported engines (Antigravity, Codex) creates a runtime expectation that each engine actually honors `GH_AW_MAX_TURNS`.

#### Neutral
- Enterprise default override `GH_AW_DEFAULT_MAX_TURNS` is now documented against the top-level field rather than `engine.max-turns`.
- Schema, error/help text, and reference docs were updated to present `max-turns` as a top-level key.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Turn-Cap Configuration Surface

1. The compiler **MUST** accept `max-turns` as a top-level workflow frontmatter key validated as a templatable integer (integer literal or GitHub Actions expression).
2. The compiler **MUST** continue to accept `engine.max-turns` as a deprecated alias for backward compatibility.
3. When both top-level `max-turns` and `engine.max-turns` are present, the compiler **MUST** prefer the top-level value.
4. The schema **MUST** mark `engine.max-turns` as deprecated and **SHOULD** direct users to the top-level field.

### Propagation and Enforcement

1. The top-level `max-turns` value **MUST** be preserved through built-in engine import and defaulting paths so it is not lost during engine resolution.
2. The top-level `max-turns` value imported from a shared agentic workflow **MUST** be preserved via a first-wins merge across imports.
3. For each agentic engine that declares `MaxTurns` capability, the compiled workflow **MUST** export the resolved value as `GH_AW_MAX_TURNS`.
4. The `gh aw fix` codemod `engine-max-turns-to-top-level` **MUST** migrate `engine.max-turns` to top-level `max-turns` and **MUST NOT** corrupt surrounding frontmatter, comments, or body content.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26827797756) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
