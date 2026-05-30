# ADR-34874: Inline Skill Syntax Mirroring Inline Sub-Agents

**Date**: 2026-05-26
**Status**: Draft
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

## Part 1 — Narrative (Human-Friendly)

### Context

The gh-aw workflow system supports *skills* — reusable, named prompt fragments that an engine CLI (Claude, Codex, Gemini, Copilot) discovers and invokes by name from an engine-specific directory (e.g. `.claude/skills/<name>.md`, `.github/skills/<name>/SKILL.md`). Until now, every skill had to live in its own file under the engine's skill directory, separate from the workflow that uses it. This forced workflow authors to manage scattered files for small, single-use skills, and mirrored the same friction we already solved for sub-agents (see [ADR-29668](29668-inline-sub-agent-syntax-using-h2-heading-delimiters.md)). With inline sub-agents shipping and proving useful, authors asked for the same co-location capability for skills, and we wanted a single mental model for both inline artifacts.

### Decision

We will support inline skills using level-2 Markdown headings of the form `` ## skill: `name` `` as block delimiters, mirroring the existing inline sub-agent syntax. At runtime (after `{{#runtime-import}}` macros are resolved), each block is extracted and written to an engine-specific skills directory under `/tmp/gh-aw/`, then restored into the workspace on the main job via `restore_inline_skills.sh`. The Go compiler (`pkg/parser/inline_skill_extractor.go`) and JS runtime extractor (`actions/setup/js/extract_inline_skills.cjs`) share the same syntax, name rules, and H2-boundary semantics as their sub-agent counterparts, so authors learn one pattern.

### Alternatives Considered

#### Alternative 1: Separate skill files only (status quo)

Continue requiring every skill to live in its own file under the engine's skills directory. This requires no new parser logic and keeps the skill loading path identical to the engine's native expectation. It was not chosen because it fragments tightly-coupled, single-use skills across multiple files — the exact friction inline sub-agents were introduced to solve — and forces workflow authors to context-switch between the workflow file and one-or-more skill files during authoring and review.

#### Alternative 2: Reuse the inline sub-agent block with a frontmatter discriminator

Inline blocks could use a single `` ## agent: `name` `` marker plus a frontmatter field like `kind: skill` to distinguish skills from sub-agents. This would reduce the parser surface to a single block type. It was not chosen because conflating sub-agents and skills under one heading hides the artifact's true nature in the rendered Markdown outline, complicates frontmatter validation (different supported fields per kind), and makes engine-specific output-path routing depend on parsed body content rather than the heading itself.

#### Alternative 3: Engine-agnostic single output location

Inline skills could always be written to one canonical location (e.g. `.github/skills/<name>/SKILL.md`) regardless of engine, and engines other than Copilot could be taught to read from that path. This would simplify the extractor — one path, no engine branching. It was not chosen because Claude, Codex, and Gemini have established, documented conventions (`.claude/skills/`, `.codex/skills/`, `.gemini/skills/`) that users and external tooling already rely on; forcing a non-native path would break compatibility and surprise users coming from those engines' own ecosystems.

### Consequences

#### Positive

- Workflow authors can define small, single-use skills inline with the workflow that uses them, eliminating extra files for trivial cases.
- The syntax, name rules, and H2-boundary semantics are identical to inline sub-agents, so authors learn one mental model that covers both inline artifact types.
- Engine-specific output paths preserve native discovery: each engine still finds skills exactly where its own conventions say they live, with no engine-side configuration change.

#### Negative

- Extraction logic must now be maintained in two languages and three places — `pkg/parser/inline_skill_extractor.go` (Go, compile-time), `actions/setup/js/extract_inline_skills.cjs` (JS, runtime), and `actions/setup/sh/restore_inline_skills.sh` (Bash, artifact restore) — and any syntax or routing change must land in all three.
- The supported skill frontmatter is intentionally narrow (`description` only); authors familiar with full skill files may be surprised that fields like `model`, `tools`, or `engine` are stripped with a warning rather than honored.
- Inline skills share revision history with the parent workflow; there is no independent per-skill commit log, which can complicate auditing of a skill's evolution.

#### Neutral

- The `` ## skill: `name` `` heading appears in the rendered Markdown outline alongside the workflow's own H2 sections. This is both helpful (discoverability) and noisy (extra navigation entries); documentation must clarify this dual role.
- Skill file content is written under `/tmp/gh-aw/<engine-skill-dir>/...` for activation-artifact handoff and then restored into the workspace on the main job; this two-stage flow is invisible to authors but adds a step to the activation/main job pipeline.
- The default (non-Claude/Codex/Gemini) engine path uses a directory-style layout (`.github/skills/<name>/SKILL.md`) while named engines use a flat file (`.<engine>/skills/<name>.md`); authors switching engines may notice the layout difference.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Syntax

1. An inline skill block **MUST** begin with a level-2 Markdown heading matching the pattern `` ## skill: `name` `` where `name` starts with a lowercase letter (`a–z`) and contains only lowercase letters, digits, hyphens (`-`), and underscores (`_`).
2. An inline skill block **MUST** end at the next level-2 Markdown heading (`##`) or end of file — whichever comes first. No explicit closing marker is required or permitted.
3. Skill names within a single workflow file **MUST** be unique. Duplicate names **SHALL** be treated as a parse error.
4. Skill names **MUST NOT** start with a digit, contain uppercase letters, spaces, or path-separator characters (`/`, `\`).
5. Inline skill frontmatter **MUST** support only the `description` field. Any other top-level frontmatter key **SHALL** be stripped during extraction, and implementations **SHOULD** emit a warning naming each stripped field.

### Extraction and Output

1. Inline skill extraction **MUST** occur after all `{{#runtime-import}}` macros in the workflow file have been fully resolved, so that imports inside a skill block are inlined before the block is written to disk.
2. The compile-time extractor (`pkg/parser/inline_skill_extractor.go`) **MUST** strip inline skill sections from the effective markdown before include expansion, workflow-name extraction, and prompt generation.
3. The runtime extractor (`actions/setup/js/extract_inline_skills.cjs`) **MUST** write each extracted skill to the engine-specific skills directory under the configured base directory (`/tmp/gh-aw/` in production).
4. Engine-specific output paths **MUST** be selected by engine identifier as follows: `claude` → `.claude/skills/<name>.md`; `codex` → `.codex/skills/<name>.md`; `gemini` → `.gemini/skills/<name>.md`; any other or unset engine identifier → `.github/skills/<name>/SKILL.md`.
5. Each written skill file **MUST** end with a newline character.
6. Implementations **SHOULD** create the target skills directory recursively if it does not already exist rather than failing with an error.

### Artifact Handoff

1. The activation artifact **MUST** include the engine-specific inline skill staging directory so that the main job can restore inline skills from it.
2. The main job **MUST** restore inline skill files from the activation artifact into the workspace at the same engine-specific path, via the `restore_inline_skills.sh` step (or an equivalent restore mechanism), before the engine CLI starts.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26434651638) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
