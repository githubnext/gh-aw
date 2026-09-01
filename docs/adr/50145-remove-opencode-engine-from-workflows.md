# ADR-50145: Remove opencode Engine from GitHub Agentic Workflows

**Date**: 2026-08-04
**Status**: Accepted
**Deciders**: pelikhan, Copilot

---

### Context

The GitHub Agentic Workflows platform supports multiple AI coding agent engines: `copilot`, `claude`, `codex`, `gemini`, and several experimental engines (`antigravity`, `opencode`, `pi`). The `opencode` engine was an experimental, provider-agnostic, open-source AI coding agent ("bring your own key") that defaulted to Copilot routing and supported 75+ models via a `provider/model` format.

Supporting `opencode` as a first-class engine option required each generated workflow lock file to include `.opencode` in its sparse-checkout folder list and `opencode.jsonc` in `GH_AW_AGENT_FILES`/`GH_AW_AGENT_FOLDERS`. With 280+ lock files, this creates substantial per-file maintenance overhead whenever opencode-specific configuration changes. The engine documentation in `syntax-engine.md` also listed `opencode` as a recognized engine identifier.

### Decision

We will remove `opencode` as a supported experimental engine from the gh-aw platform. This means:
- Removing `.opencode` from sparse-checkout folder lists in all workflow lock files
- Removing `opencode.jsonc` from `GH_AW_AGENT_FILES` and `.opencode` from `GH_AW_AGENT_FOLDERS` in all lock files
- Removing `opencode` from the list of recognized engine identifiers in `syntax-engine.md`

### Alternatives Considered

#### Alternative 1: Retain opencode as an experimental engine

Continue maintaining `opencode` references in all workflow lock files. This preserves the engine option for workflow authors who use `engine: opencode`. However, it perpetuates the per-file maintenance burden across 280+ lock files and requires every future lock-file regeneration to include opencode-specific configuration — even for workflows that will never use the engine.

#### Alternative 2: Centralize opencode into a shared workflow template

Move the opencode engine configuration to a shared/central workflow definition rather than embedding it in every individual lock file. This would reduce duplication while retaining the engine as an option. However, the decision to remove `opencode` from `syntax-engine.md`'s recognized identifier list (not just from per-file config) indicates the intent is to fully retire the engine, not merely to reorganize it.

### Consequences

#### Positive
- Reduced maintenance surface: each workflow lock file is smaller and does not need opencode-specific entries updated on every regeneration
- Simplified sparse-checkout in CI workflows: one fewer folder per checkout step across all 280+ workflows
- Cleaner engine documentation: `syntax-engine.md` no longer lists an engine that is no longer supported

#### Negative
- Workflow authors who have deployed workflows using `engine: opencode` will lose support and must migrate to another engine (copilot, claude, codex, gemini, antigravity, or pi)
- The opencode BYOK (bring your own key) capability and 75+ model support via `provider/model` format is no longer available through this platform
- Any `opencode.jsonc` or `.opencode/` configuration that workflow authors have stored in their repositories will be silently ignored by the workflow runner

#### Neutral
- The `opencode.jsonc` file and `.opencode/` directory are no longer included in the base-branch agent config save/restore cycle — repositories that have these files will retain them locally but they will not be restored by the workflow
- The number of supported experimental engines drops from 3 (`antigravity`, `opencode`, `pi`) to 2 (`antigravity`, `pi`)

---
