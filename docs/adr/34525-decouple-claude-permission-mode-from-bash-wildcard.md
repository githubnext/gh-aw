# ADR-34525: Decouple Claude Permission Mode from Bash Wildcard and Promote `engine.permission-mode` to a First-Class Setting

**Date**: 2026-05-24
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

The Claude engine previously derived its `--permission-mode` CLI flag implicitly from the presence of an unrestricted bash tool grant (`bash: ["*"]`, `bash: [":*"]`, or `bash: nil`). Whenever a workflow asked for unrestricted bash, the engine silently switched from the default `acceptEdits` mode to `bypassPermissions`, which causes Claude Code to ignore `--allowed-tools` as an MCP tool boundary. In firewall-enabled workflows that depend on `--allowed-tools` to enforce the declared MCP surface, this coupling silently nullified the security boundary. There was also no validated way for a workflow author to set the permission mode explicitly — the only escape hatch was passing `--permission-mode <value>` through `engine.args`, which had no schema validation and could produce duplicate `--permission-mode` flags on the final command line.

### Decision

We will promote `engine.permission-mode` to a first-class, schema-validated engine setting accepting one of `auto | acceptEdits | plan | bypassPermissions`, and we will remove the implicit derivation of `bypassPermissions` from bash wildcard presence. The default remains `acceptEdits`; when `tools.edit: false`, the default becomes `auto` because such workflows do not rely on edit auto-approval. When the workflow sets `engine.permission-mode`, that value takes precedence over both the default and any legacy `--permission-mode` flag passed through `engine.args`. The Claude CLI invocation is guaranteed to emit exactly one `--permission-mode` flag: legacy `engine.args` permission-mode flags (both `--permission-mode <v>` and `--permission-mode=<v>` forms) are stripped and, if `engine.permission-mode` is unset, the legacy value is used to replace the default.

### Alternatives Considered

#### Alternative 1: Keep the implicit bash-wildcard → `bypassPermissions` derivation and add an opt-out

Retain the existing behaviour where bash wildcard tools auto-switch the engine to `bypassPermissions` and introduce an opt-out flag for workflows that want to keep `acceptEdits`. Rejected because the implicit derivation is the source of the security regression: it makes the permission mode a hidden function of an unrelated tool configuration, and most workflow authors did not know that adding `bash: ["*"]` silently disabled `--allowed-tools` enforcement. An opt-out keeps the surprising default; making the mode explicit eliminates it.

#### Alternative 2: Continue using `engine.args: ["--permission-mode", "..."]` as the override mechanism

Document `engine.args` as the supported way to override permission mode and leave the implicit derivation in place. Rejected because `engine.args` is a freeform passthrough with no schema validation (typos like `acceptedits` would compile and only fail at Claude CLI invocation time), no precedence rules with the implicit derivation, and no protection against duplicate `--permission-mode` flags on the final command line. A first-class `engine.permission-mode` field gets enum validation, precedence semantics, and a single-flag emission guarantee for free.

#### Alternative 3: Drop the `tools.edit: false` → `auto` default and require `engine.permission-mode` explicitly

Make the default always `acceptEdits` and force workflows that disable the edit tool to set `engine.permission-mode: auto` themselves. Rejected because `tools.edit: false` is a strong signal that the workflow does not rely on edit-write auto-approval, and `acceptEdits` in that configuration has no useful effect beyond defaulting the agent into an "approve writes" stance that does not apply. Defaulting to `auto` in this configuration matches the workflow's intent without an extra opt-in.

### Consequences

#### Positive

- `--allowed-tools` is now a reliable MCP tool boundary in firewall-enabled workflows, even when the workflow grants unrestricted bash. Tool exposure no longer silently inverts when bash wildcards are present.
- `engine.permission-mode` is schema-validated against a fixed enum (`auto | acceptEdits | plan | bypassPermissions`), so typos are caught at compile time rather than at Claude CLI invocation.
- Workflow authors who previously relied on `engine.args: ["--permission-mode", ...]` keep working: legacy values are still picked up, and the engine now guarantees exactly one `--permission-mode` flag in the final command line.
- The `tools.edit: false` → `auto` default removes a meaningless `acceptEdits` default for workflows that have explicitly opted out of edits.

#### Negative

- This is a behavioural change: workflows that depended on the bash-wildcard → `bypassPermissions` auto-switch for smoother headless execution will see their effective mode drop to `acceptEdits` (or whichever value they set explicitly). Authors who actually wanted `bypassPermissions` must now set `engine.permission-mode: bypassPermissions` explicitly.
- `EngineConfig` and the workflow schema gain a new field, and the Claude engine now contains two small parsing helpers (`isEditToolExplicitlyDisabled`, `stripClaudePermissionModeArgs`) plus index-based mutation of the `claudeArgs` slice. The single-flag emission guarantee adds a small amount of stateful code on the hot compile path.
- Legacy `engine.args` permission-mode handling is preserved only for backward compatibility and increases the precedence rule surface (`engine.permission-mode` > legacy `engine.args` > default); this is one more layered semantics that future maintainers must remember.

#### Neutral

- The `tools.edit` schema is extended to accept a bare boolean in addition to `null` and the configuration object form. Existing object-form configurations continue to validate.
- The default permission mode for the common case (no `tools.edit: false`, no explicit override) remains `acceptEdits`, so most existing workflows see no change at compile time.
- The new logging lines (`Using engine.permission-mode override`, `tools.edit=false detected: using auto permission mode`, `Using legacy engine.args permission mode override`) surface the precedence resolution in compile output; this is informative but adds compile-time chatter.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Schema Surface

1. The workflow schema **MUST** accept `engine.permission-mode` as a string with the enum `["auto", "acceptEdits", "plan", "bypassPermissions"]`.
2. The workflow schema **MUST** reject any other string value for `engine.permission-mode` at frontmatter validation time.
3. The workflow schema **MUST** accept `tools.edit` as a boolean in addition to its existing `null` and object forms.
4. `engine.permission-mode` **MUST** be optional; omitting it **MUST NOT** cause a validation error.

### Engine Configuration Extraction

1. `EngineConfig` **MUST** carry a `PermissionMode string` field populated from `engine.permission-mode` when present.
2. The extractor **MUST** treat a missing `engine.permission-mode` as the empty string and **MUST NOT** synthesize a default value at extraction time.
3. The extractor **MUST** only populate `PermissionMode` when the frontmatter value is a string; non-string values **MUST** be ignored at extraction (validation rejects them earlier).

### Permission Mode Resolution (Claude Engine)

1. The Claude engine **MUST** start with a baseline permission mode of `"acceptEdits"`.
2. The Claude engine **MUST** override the baseline to `"auto"` when `tools.edit` is explicitly set to the boolean `false`, as determined by `isEditToolExplicitlyDisabled`.
3. `isEditToolExplicitlyDisabled` **MUST** return `true` only when `tools["edit"]` exists, is of Go type `bool`, and equals `false`. Any other type (including `nil`, string `"false"`, integer, or absent key) **MUST** return `false`.
4. The Claude engine **MUST** override any prior value when `EngineConfig.PermissionMode` is a non-empty string, replacing it with that value.
5. The Claude engine **MUST NOT** derive `"bypassPermissions"` from the presence of `bash: "*"`, `bash: ":*"`, `bash: nil`, or any other bash configuration.
6. The Claude engine **MUST** emit exactly one `--permission-mode <value>` flag in the assembled Claude CLI argument list.

### Legacy `engine.args` Compatibility

1. The Claude engine **MUST** filter `EngineConfig.Args` through `stripClaudePermissionModeArgs` before appending them to the Claude CLI argument list.
2. `stripClaudePermissionModeArgs` **MUST** remove every occurrence of `--permission-mode <value>` (two-token form) and `--permission-mode=<value>` (single-token form) from the supplied argument slice.
3. `stripClaudePermissionModeArgs` **MUST** return the last permission-mode value encountered when multiple are present, and the empty string when none are present.
4. When `EngineConfig.PermissionMode` is empty AND `stripClaudePermissionModeArgs` returns a non-empty value, the Claude engine **MUST** substitute that value into the previously emitted `--permission-mode` slot (at `permissionModeValueIndex`) rather than appending a second flag.
5. When `EngineConfig.PermissionMode` is non-empty, the engine **MUST NOT** substitute the legacy `engine.args` permission-mode value (the explicit setting wins).
6. A `--permission-mode` flag with no following value **MUST** be stripped without crashing and **MUST NOT** contribute to the returned permission-mode value.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26371950199) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
