# ADR-35778: Extend `aw.yml` Package Manifest to Support Skills and Agents

**Date**: 2026-05-29
**Status**: Draft
**Deciders**: Unknown

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `aw.yml` package manifest previously described only one kind of installable artifact: workflow files (via the `files:` array or auto-discovery of `workflows/` and `.github/workflows/`). As the project grew agentic-engine *skills* (directories containing a `SKILL.md` marker plus supporting files) and *sub-agents* (single `.md` files), there was no first-class way to ship them inside a package — authors had to copy these artifacts into a consumer repository by hand, and the `gh aw add` install pipeline had no concept of routing them to the correct engine-specific destination (`.github/skills/`, `.github/agents/`, or their `.claude/`/`.codex/` overrides). This forced manual, error-prone installation and meant a single package could not distribute a complete, ready-to-run set of workflows, skills, and agents together.

### Decision

We will extend the `aw.yml` manifest with two optional, dedicated array fields — `skills` (directory paths under `skills/`, each qualified by a `SKILL.md` marker) and `agents` (`.md` file paths under `agents/`) — and teach the install pipeline to treat them as distinct artifact types alongside workflows. When a field is omitted, the corresponding directory is auto-discovered: `skills/` subdirectories are accepted only when they contain a `SKILL.md` marker, and `agents/` is scanned for `.md` files. Each artifact type carries its own resolution, path-validation, and install routing: skill files install to `GetEngineSkillDir("")/<skill-name>/` and agent files to `GetEngineSubAgentDir("")`, so engine-specific overrides are respected automatically. Invalid manifest entries are skipped with a warning rather than aborting the install.

### Alternatives Considered

#### Alternative 1: Reuse the existing `files:` array for all artifact types

Skills and agents could be listed as ordinary entries in `files:` and copied verbatim. Rejected because `files:` entries are treated as workflows: they are routed to the workflows directory, not to the engine skill/agent directories, and they would bypass the `SKILL.md` marker check and the per-type path validation (`isSupportedSkillDirPath`, `isSupportedAgentFilePath`). Overloading one field with three install behaviors would make the destination depend on opaque heuristics rather than the author's explicit intent.

#### Alternative 2: A separate manifest file or a separate `gh aw add-skill` command

We could keep `aw.yml` workflow-only and introduce a parallel manifest (or dedicated subcommands) for skills and agents. Rejected because it fragments the package model: a consumer would need multiple install invocations to get a coherent bundle, and the resolution/download/tracking machinery (`ResolveWorkflows`, `FileTracker`, GitHub-API-with-git-clone fallback) would have to be duplicated. Extending the existing manifest reuses that machinery and keeps "one package, one install" intact.

#### Alternative 3: Require explicit listing only, with no auto-discovery

We could require every skill directory and agent file to be enumerated in the manifest. Rejected as the sole behavior because it adds boilerplate and drifts easily out of sync as a package gains artifacts. Auto-discovery (gated on the `SKILL.md` marker for skills and the `.md` extension for agents) is provided as the default, while explicit listing remains available for authors who want precise control.

### Consequences

#### Positive
- A single package can distribute workflows, skills, and agents together; `gh aw add <pkg>` installs all three in one operation.
- Auto-discovery removes manifest boilerplate for the common case, while explicit `skills:`/`agents:` lists remain available when authors need exact control.
- Install destinations resolve through `GetEngineSkillDir` / `GetEngineSubAgentDir`, so engine-specific layouts (`.claude/`, `.codex/`, etc.) are honored without per-call configuration.
- Invalid or missing entries degrade gracefully (warning + skip) instead of failing the whole install, matching the existing manifest-parsing philosophy.

#### Negative
- The install pipeline gains new artifact-type branches (`IsPackageSkillFile`, `IsPackageAgentFile`) and two new path-validation rules, increasing the surface area and the number of code paths each future change must consider.
- Auto-discovery issues additional GitHub API calls (directory listing plus a `SKILL.md` marker probe per candidate subdirectory), adding latency and rate-limit pressure for packages that rely on scanning rather than explicit lists.
- The `SKILL.md`-marker convention and the "direct child of `skills/`, no nesting" / "`.md` directly under `agents/`" path rules are now load-bearing contracts that package authors must follow but that are only documented implicitly in validation code.

#### Neutral
- `WorkflowSpec` / `ResolvedWorkflow` carry new fields (`IsPackageSkillFile`, `IsPackageAgentFile`, `SkillName`); workflow entries leave these unset and are unaffected.
- Remote resolution adds `ListDirAllFilesForHost` and `ListDirSubdirsForHost` (GitHub API with git-clone fallback), parallel to the existing workflow-file listing helpers.
- The feature is scoped to the `skills/` and `agents/` top-level directories with single-level nesting; deeper hierarchies or additional artifact types would require extending the validation predicates and resolver.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Manifest Schema

1. The `aw.yml` manifest **MUST** accept two optional array fields, `skills` and `agents`, in addition to `files`.
2. Each `skills` entry **MUST** be a directory path that is a direct child of `skills/` (e.g. `skills/my-skill`); entries with deeper nesting, a different prefix, or path-traversal segments **MUST** be rejected.
3. Each `agents` entry **MUST** be a `.md` file path that is a direct child of `agents/` (e.g. `agents/my-agent.md`); entries with a non-`.md` extension, deeper nesting, a different prefix, or path-traversal segments **MUST** be rejected.
4. A manifest entry that fails validation **MUST** be skipped with a warning and **MUST NOT** abort resolution of the remaining package contents.
5. Duplicate entries within `skills` or within `agents` **MUST** be deduplicated.
6. A `skills` or `agents` value that is not a list of strings **MUST** be ignored with a warning.

### Artifact Resolution

1. When `skills` is non-empty, the listed directories **MUST** be used as the skill set; when `skills` is absent, the `skills/` directory **MUST** be auto-scanned and a subdirectory **MUST** be treated as a skill if and only if it contains a `SKILL.md` marker file.
2. When `agents` is non-empty, the listed files **MUST** be used as the agent set; when `agents` is absent, the `agents/` directory **MUST** be auto-scanned and only `.md` files **MUST** be included.
3. A skill directory referenced explicitly but not found in the package **MUST** produce a warning and be skipped, not an error.
4. Absence of the `skills/` or `agents/` directory during auto-scan **MUST** be treated as "no artifacts of that type", not as an error.

### Install Routing

1. A resolved skill file **MUST** be installed under the engine skill directory resolved via `GetEngineSkillDir("")`, within a subdirectory named for its skill (`<skill-dir>/<skill-name>/`).
2. A resolved agent file **MUST** be installed under the engine sub-agent directory resolved via `GetEngineSubAgentDir("")`.
3. The installer **MUST NOT** route skill or agent files to the workflows directory.
4. When a destination skill or agent file already exists, the installer **MUST** skip it unless the force option is set, in which case it **MUST** overwrite and record the file as modified.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26661810526) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
