# ADR-47690: Rewrite Local Skill Refs to Qualified Specs on `gh aw add`

**Date**: 2026-07-24
**Status**: Draft
**Deciders**: Unknown

---

### Context

The `gh aw` compiler requires all skill references in workflow frontmatter to be fully-qualified `owner/repo/path@sha` specs. During local development, workflow authors naturally write relative path references such as `.github/skills/my-skill` or `./my-skill` in the `skills:` frontmatter array. Without automatic transformation, a locally authored workflow cannot be installed via `gh aw add ./my-workflow.md` without the author first manually constructing the qualified spec — which requires knowing the current repo slug and the exact commit SHA, making iterative development awkward and error-prone.

### Decision

We will insert an `applyLocalSkillRefRewriting` step into the `processWorkflowContentModifications` pipeline in the `add` command. This step runs only when the source workflow is local (`sourceInfo.IsLocal == true`). It identifies skill entries with no `@` separator as local refs, resolves the current repository slug via `GetCurrentRepoSlug` and the HEAD commit SHA via `git rev-parse HEAD`, then rewrites matching entries to the `owner/repo/path@sha` form in-place using line-level YAML editing. If any lookup fails the step is skipped silently so that downstream compilation can surface a more actionable error.

### Alternatives Considered

#### Alternative 1: Full YAML Round-Trip Parsing

Parse the full YAML frontmatter, update the `skills` array, and serialize back to text. This is simpler algorithmically but was rejected because a full round-trip loses the original formatting — indentation, inline comments, and field ordering are not preserved by most YAML serializers. Workflow files are human-readable documents where formatting matters to authors.

#### Alternative 2: Require Authors to Always Write Qualified Refs

Do no rewriting at all and require all `skills:` entries to be fully-qualified `owner/repo/path@sha` refs before calling `gh aw add`. Rejected because this creates prohibitively high friction for local development: the author must know their exact repo slug, run `git rev-parse HEAD` to get the current SHA, and manually update the ref after every commit that changes skills. The ergonomic cost outweighs the simplicity benefit.

#### Alternative 3: Defer Rewriting to Compile Time

Resolve local refs during compilation rather than at `add` time. Rejected because the `add` command is the natural boundary where a workflow transitions from local development to an installed state. The compiler is designed to operate on fully-qualified refs as input; adding local-ref resolution there would blur the responsibility boundary and complicate compiler error messages.

### Consequences

#### Positive
- Local workflow development is frictionless: authors write relative paths while developing and have them automatically resolved when running `gh aw add`.
- Line-level YAML editing preserves indentation, inline comments, auth fields (e.g., `github-token:`), and all non-skill frontmatter unchanged.
- Already-qualified refs and GitHub Actions expression refs (e.g., `${{ vars.SKILL_REF }}`) are detected and left untouched.

#### Negative
- The rewritten SHA is the HEAD at add-time. If skill implementations are updated after the workflow is installed, the installed workflow continues to reference the old SHA until the workflow is re-added.
- In non-verbose mode, lookup failures (git root not found, repo slug unresolvable, HEAD SHA unresolvable) are silently skipped. The fallback error that surfaces during compilation may be less actionable than an explicit "local ref rewriting failed" message.

#### Neutral
- The new `applyLocalSkillRefRewriting` step is inserted before `applySourceAndIncludeModifications` in the pipeline, inheriting the same content-string threading pattern used by adjacent steps.
- The detection heuristic (no `@` in the spec, not starting with `${{`) intentionally excludes tag-style refs like `owner/repo@main` — those already have an `@` and are treated as remote refs.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
