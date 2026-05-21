# ADR-33658: First-Class `engine.permission-mode` for Claude Engine

**Date**: 2026-05-21
**Status**: Draft
**Deciders**: Unknown (draft generated from PR #33658 diff)

---

## Part 1 — Narrative (Human-Friendly)

### Context

The Claude CLI accepts a `--permission-mode` flag that gates whether tool requests are auto-approved (`bypassPermissions`), interactively confirmed (`auto`), silently accepted for edits (`acceptEdits`), or restricted to planning (`plan`). gh-aw selects this flag implicitly: the `hasBashWildcardInTools` heuristic forces `bypassPermissions` whenever a `bash: ["*"]` entry is detected in the tool list. The network-egress firewall injects exactly such a wildcard at compile time, which means any workflow running with the firewall enabled was silently switched to `bypassPermissions` — making the `--allowed-tools` boundary configured by `safe-outputs` and `tools` effectively a no-op. The only known override was a fragile `engine.args` workaround that relied on the Claude CLI's last-flag-wins parsing to overwrite the auto-selected mode. There was no explicit, validated configuration surface for workflow authors to decouple the permission mode from the firewall.

### Decision

We will add `permission-mode` as a first-class field on `EngineConfig` for the Claude engine. The field is extracted from frontmatter (`engine.permission-mode`) in both the standard and inline-engine code paths, validated by a new `validateEnginePermissionMode` against the closed set `{auto, acceptEdits, plan, bypassPermissions}`, and surfaced in the workflow JSON schema (`engine_config` variants 1 and 2) with an `enum` constraint plus a regenerated `autocomplete-data.json`. At compile time, `claude_engine.go` consults the explicit field first; only when it is unset does the existing `hasBashWildcardInTools` fallback apply. The primary driver is restoring the integrity of `--allowed-tools` enforcement under the firewall by making the permission boundary an explicit, validated workflow concern rather than an implicit side effect of bash-tool wildcards.

### Alternatives Considered

#### Alternative 1: Keep the `engine.args` workaround as the supported override

Workflow authors could continue to override the auto-selected permission mode by appending `--permission-mode auto` (or similar) to `engine.args`, relying on Claude CLI's documented last-flag-wins behavior. This was rejected because it is undocumented in gh-aw, has no schema or validation (typos like `--permission-modes` produce silent breakage), and depends on internal CLI parsing semantics that are not part of any stability contract. It also offers no way for the compiler or downstream tooling to reason about the effective permission boundary.

#### Alternative 2: Remove the `hasBashWildcardInTools` auto-selection entirely

The auto-selection could be deleted so that the firewall's `bash: ["*"]` injection no longer flips the permission mode, leaving `auto` as the unconditional default. This was rejected for backwards compatibility: existing workflows that intentionally request unrestricted bash and rely on the implicit `bypassPermissions` behavior (without setting any explicit `permission-mode`) would suddenly hit interactive permission prompts that hang non-interactive CI runs. Layering explicit-wins-over-implicit preserves the safety hatch while letting authors opt into the stricter mode.

#### Alternative 3: Tie the permission mode to firewall state directly

The compiler could detect that the firewall feature is enabled and force a stricter default in that case (e.g. `auto` when firewall is on, current behavior otherwise). This was rejected because it couples two orthogonal concerns — network egress policy and Claude tool-permission policy — making the effective configuration depend on the interaction of two features rather than a single, locally visible workflow field. It also leaves no escape hatch for authors who legitimately want `bypassPermissions` under the firewall.

### Consequences

#### Positive
- `--allowed-tools` enforcement is once again meaningful for workflows that opt into a non-bypass mode, even with the firewall enabled.
- Schema and validation make typos and unsupported values fail fast at compile time with a documented error message, instead of silently producing a different runtime behavior.
- Editor autocomplete (`autocomplete-data.json`) and schema (`main_workflow_schema.json`) now advertise the field and its allowed values, improving discoverability.
- The new field is local and self-contained — workflow authors can audit a single frontmatter line to know the effective permission boundary instead of reasoning about the bash-wildcard heuristic.

#### Negative
- Workflow authors gain a new field that interacts subtly with `hasBashWildcardInTools`: the explicit value always wins, so an author who sets `permission-mode: auto` together with a `bash: ["*"]` tool produces a stricter mode than the silent default, and may be surprised by interactive prompts in a non-interactive runner if they forget about the implication.
- The Claude CLI's permission-mode vocabulary (`auto`, `acceptEdits`, `plan`, `bypassPermissions`) is now part of gh-aw's stable surface; if upstream Claude renames or removes a mode, the schema enum and validator will need a coordinated update.
- Two compiler entry points (`compiler_orchestrator_workflow.go` and `compiler_string_api.go`) and two parsing paths (standard frontmatter vs inline engine definition) must each plumb the new field and call the validator — drift between them would let invalid configs through in one path and not the other.

#### Neutral
- The field applies only to the Claude engine; other engines (Codex, OpenCode, etc.) ignore the value. Workflow authors targeting multiple engines must understand that `permission-mode` is Claude-specific.
- The autocomplete data file is regenerated from the schema; future schema changes to `engine_config` variants must regenerate this artifact or the editor experience drifts from runtime validation.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Configuration Surface

1. The `EngineConfig` struct **MUST** expose a `PermissionMode string` field that is populated from the frontmatter key `engine.permission-mode` in both the standard frontmatter and inline-engine-definition parsing paths.
2. The workflow JSON schema (`main_workflow_schema.json`) **MUST** define `permission-mode` on the `engine_config` object variants that accept the Claude engine, and the field **MUST** be constrained to an `enum` whose members are exactly `auto`, `acceptEdits`, `plan`, and `bypassPermissions`.
3. The editor `autocomplete-data.json` artifact **MUST** be regenerated whenever the schema enum membership changes, so that completion suggestions match the validator.
4. Implementations **MUST NOT** silently coerce or rewrite the field value; an unrecognized value is a configuration error, not a value to be normalized.

### Validation

1. The compiler **MUST** invoke `validateEnginePermissionMode` from both compiler orchestration entry points (`compiler_orchestrator_workflow.go` and `compiler_string_api.go`) before producing the workflow YAML.
2. If `engine.permission-mode` is set to a value outside `{auto, acceptEdits, plan, bypassPermissions}`, the compiler **MUST** return a validation error that names the offending value and lists the accepted set.
3. If `engine.permission-mode` is unset or empty, the validator **MUST** treat it as absent and **MUST NOT** raise an error.

### Runtime Effect on Claude CLI Invocation

1. The Claude engine builder **MUST** emit `--permission-mode <value>` exactly once on the Claude CLI command line.
2. When `engine.permission-mode` is set, the emitted value **MUST** equal that configured value, regardless of whether `hasBashWildcardInTools` would otherwise return true.
3. When `engine.permission-mode` is unset, the builder **MUST** preserve the prior behavior: emit `bypassPermissions` if `hasBashWildcardInTools` is true, otherwise the documented default (`auto`).
4. Implementations **MUST NOT** introduce a third mechanism (additional flag rewriting, environment-variable override, post-hoc patching of `engine.args`) to influence `--permission-mode` once the explicit field and the bash-wildcard fallback have been resolved.

### Scope

1. The `engine.permission-mode` field **SHALL** apply only to the Claude engine. Other engines **MAY** ignore the value without error, and **MUST NOT** repurpose it for unrelated configuration.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26203507133) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
