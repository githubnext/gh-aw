# ADR-28486: Protect Any Top-Level Dot-Folder in Safe Outputs Handlers

**Date**: 2026-04-25
**Status**: Draft
**Deciders**: pelikhan, copilot-swe-agent

---

## Part 1 — Narrative (Human-Friendly)

### Context

The safe outputs system controls which files AI agents may write when creating pull requests or pushing to PR branches. Sensitive configuration directories at the repository root — such as `.github/`, `.agents/`, `.githooks/`, `.husky/`, and `.codex/` — were previously protected via an explicit hardcoded list in the handler config. The ecosystem of tool-configuration dot-directories is growing rapidly (`.cursor/`, `.vscode/`, `.devcontainer/`, `.claude/`, etc.), and each new convention required a manual update to every workflow config that embedded the protected path prefix list.

### Decision

We will replace the per-workflow hardcoded list of protected dot-directory path prefixes with a general rule (`protect_top_level_dot_folders: true`) that automatically blocks agent writes to any top-level directory whose name starts with `.`. The rule is opt-out: workflow owners who need agents to write into a specific dot-folder may add it to the `exclude:` list, which is forwarded to the runtime as `protected_dot_folder_excludes`.

### Alternatives Considered

#### Alternative 1: Continue Expanding the Hardcoded List

Each new dot-directory that should be protected (e.g., `.cursor/`) could be appended to the `protected_path_prefixes` array in every workflow config. This was the prior approach and requires coordinated changes across all generated lock files whenever a new tool convention emerges. It is error-prone because any workflow not updated would silently leave new dot-directories unprotected.

#### Alternative 2: Opt-In Allowlist (Protect Nothing by Default)

Invert the model: protect only explicitly listed directories and leave everything else writable. This is more permissive and easier for workflows that legitimately need to write into dot-directories, but it provides no safety net against accidental or adversarial writes into new tool-configuration directories that are not yet on any team's radar.

### Consequences

#### Positive
- New dot-directories (e.g., `.cursor/`, `.vscode/`) are protected automatically without any code or config change.
- Security posture is consistent across all agentic workflows by default rather than being patchwork.
- The surface area for "forgot to update the list" gaps is eliminated.

#### Negative
- Agents that legitimately need to write into a dot-directory (e.g., updating `.cursor/` rules) now require an explicit `exclude:` entry in the workflow config, adding friction for legitimate use cases.
- Existing workflows with dot-directories in their `exclude:` lists that overlap with the new general rule may exhibit unexpected interaction until the deduplication logic is verified.

#### Neutral
- All generated lock files (`.github/workflows/*.lock.yml`) must be regenerated to embed the new `protect_top_level_dot_folders: true` flag; this PR includes those regenerated files.
- The Go compiler layer gains a `getDotFolderExcludes()` helper and config propagation for `protected_dot_folder_excludes`; the JavaScript runtime gains `checkForTopLevelDotFolders()`.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### File Protection at Runtime

1. When `protect_top_level_dot_folders` is `true`, implementations **MUST** reject any file path whose first path segment starts with `.` (e.g., `.cursor/settings.json`), unless that segment appears in `protected_dot_folder_excludes`.
2. Implementations **MUST NOT** silently skip the dot-folder check when `protect_top_level_dot_folders` is `true`; a protection violation **MUST** produce an error or fallback consistent with the handler's `protected_files_policy`.
3. Implementations **SHOULD** deduplicate entries in `protected_dot_folder_excludes` before comparing them against the file path to avoid redundant checks.
4. Implementations **MAY** retain existing explicit `protected_path_prefixes` entries (e.g., `.github/`) alongside `protect_top_level_dot_folders: true`; the two mechanisms are additive and not mutually exclusive.

### Configuration Generation (Compiler Layer)

1. The compiler **MUST** propagate all dot-directory entries from the workflow's `exclude:` list into `protected_dot_folder_excludes` when generating handler config with `protect_top_level_dot_folders: true`.
2. The compiler **MUST NOT** include non-dot-directory entries from the `exclude:` list in `protected_dot_folder_excludes`.
3. The compiler **SHOULD** emit `protected_dot_folder_excludes` as an empty array rather than omitting the field when no dot-directory exclusions are present, to make the intent explicit.

### Workflow Configuration

1. All agentic workflow handler configs that use `create_pull_request` or `push_to_pull_request_branch` **MUST** set `protect_top_level_dot_folders: true` unless a documented exception is approved.
2. Workflow owners **SHOULD** prefer adding targeted entries to the `exclude:` list over disabling `protect_top_level_dot_folders` entirely.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24937660894) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
