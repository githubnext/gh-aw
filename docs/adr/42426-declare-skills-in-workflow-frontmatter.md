# ADR-42426: Declare External Skills in Workflow Frontmatter

**Date**: 2026-06-30
**Status**: Draft
**Deciders**: Unknown (copilot-swe-agent, pelikhan)

---

### Context

The gh-aw system compiles Markdown workflow definitions into GitHub Actions YAML. Agent engines (Claude, Copilot, etc.) support loadable "skills" — small instruction files that extend agent capabilities — stored in engine-specific directories (e.g., `.claude/skills/`). Previously, skill directory setup was gated entirely behind the `inline-agents` feature flag, with no mechanism for workflow authors to declare specific external skills their workflow requires. Authors who needed external skills had to manually add `run:` steps, which required knowledge of internal path conventions, bypassed activation artifact persistence, and provided no SHA-pinning for reproducibility. The `gh skill` CLI (available since v2.90) provides a standardized install interface that the compiler can invoke on behalf of declarative skill references.

### Decision

We will add a `skills` array to the workflow frontmatter schema, allowing workflow authors to declare external skill references pinned to 40-character commit SHAs (e.g., `owner/repo@<sha>` or `owner/repo/skill/path@<sha>`). During the activation job, the compiler will emit steps that upgrade `gh` to ≥2.90, then call `gh skill install` for each reference, routing installed files to the engine-specific skill directory. The installed skill directory will be uploaded in the activation artifact and restored in the main job, consistent with how inline sub-agent assets are persisted. The compiler will also emit the skill list via `GH_AW_INFO_SKILLS` so `aw_info.json` exposes configured skills to the running engine.

### Alternatives Considered

#### Alternative 1: Manual `run:` steps in workflow body

Workflow authors could add their own `run:` steps before the agent executes to call `gh skill install` manually. This was the status quo workaround. It was rejected because it requires authors to know internal skill directory paths per engine, provides no integration with activation artifact upload/restore (skills would be lost between jobs), and duplicates boilerplate across every workflow that needs skills. It also offers no compile-time validation of reference format.

#### Alternative 2: Bundle skills into the existing `inline-agents` feature flag

The `inline-agents` flag already establishes skill directory infrastructure. Skills could have been configured as a sub-option of that flag rather than an independent frontmatter key. This was rejected because `inline-agents` is a runtime behavioral flag for sub-agent spawning — it is conceptually orthogonal to which skills a workflow declares. Conflating the two would force authors to enable sub-agent spawning just to use skills, and would make the feature flag semantics confusing. Keeping skills as a first-class frontmatter field mirrors how other workflow-level configuration (tools, engine, metadata) is declared.

#### Alternative 3: Skills as a repository-level configuration file

A `.gh-aw-skills.yml` file at the repository root could list skills to install for all workflows in the repo. This was not pursued because it cannot express per-workflow skill sets, provides no per-ref pinning at the workflow level, and would complicate the existing single-file compilation model where each workflow Markdown file is self-contained.

### Consequences

#### Positive
- Workflow authors can declaratively specify skill dependencies with SHA-pinned references, making skill sets reproducible and auditable via git history.
- Engine skill directories are populated before the agent runs, enabling consistent skill discovery without engine-specific setup code.
- `aw_info.json` now carries the configured skill list, so engines can introspect which skills were requested without parsing frontmatter themselves.
- Activation artifact upload and main-job restore are unconditionally extended to cover the skill directory when skills are declared, preventing cross-job data loss without manual configuration.

#### Negative
- Skill installation requires `gh` CLI ≥2.90. Workflows using this feature will fail activation on runners with older versions; operators must ensure runner images are up to date.
- Activation job run time increases by at least one `gh skill install` invocation per reference, plus the version check step. High-skill-count workflows may see noticeable activation overhead.
- The `isRepositorySkillSpec` heuristic (counting `/` separators to determine `--all` install) is a fragile proxy for intent; a repo with exactly two path segments triggers bulk install even if the author wanted only one skill from that path.

#### Neutral
- GitHub Actions expressions (`${{ inputs.skill_ref }}`) are accepted as skill refs at compile time without SHA validation, deferring resolution to runtime. This is consistent with how other expression-valued fields (e.g., `max-daily-ai-credits`) are handled.
- The `GH_AW_INFO_SKILLS` environment variable follows the existing `GH_AW_INFO_*` naming convention used for features and other runtime metadata.
- Existing workflows with no `skills` key are unaffected; all new code paths are gated on `len(data.Skills) > 0`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
