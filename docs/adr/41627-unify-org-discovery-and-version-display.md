# ADR-41627: Unify Org Discovery Strategy and Add Version Display to update/upgrade --org

**Date**: 2026-06-26
**Status**: Draft
**Deciders**: pelikhan (PR author: copilot-swe-agent)

---

### Context

The `gh aw update --org` and `gh aw upgrade --org` commands used different GitHub code-search queries to discover repositories: `update` searched for `.lock.yml` files while `upgrade` searched for any `.md` files in `.github/workflows` (excluding README). This divergence caused the two commands to operate on different repository sets, creating confusion when an org's repos appeared in one command's output but not the other's. Additionally, both commands showed opaque output: `update --org` displayed raw commit SHAs (e.g., `e15e57b`) instead of human-readable version tags (e.g., `v1.4.0`), and `upgrade --org` showed no current compiler version information at all — operators had no way to know what version a repo was running before initiating an upgrade.

### Decision

We will unify both `update --org` and `upgrade --org` to use the `path:.github/workflows filename:.lock.yml` GitHub code-search query for repository discovery. We will resolve commit SHAs to tag names via the GitHub tags API (with per-repo in-memory caching) for `update --org` version display. We will add a per-repo scan phase to `upgrade --org` that shallow-clones each discovered repo, counts agentic workflow `.md` files, and extracts the current compiler version from `.lock.yml` metadata headers — allowing the dry-run report to show `(v1.2.3 -> v1.4.0)` per repo.

### Alternatives Considered

#### Alternative 1: Keep separate discovery queries, add cross-reference deduplication

The `upgrade` command could continue using the broader `.md`-extension search but then intersect the result with a `.lock.yml` search to match `update`'s set. This avoids changing `upgrade`'s discovery scope but adds complexity (two API calls, set intersection logic). It was rejected because the unified `.lock.yml` query is the correct canonical signal that a repo has compiled agentic workflows — the `.md` search was overly broad and included repos with no source-managed workflows.

#### Alternative 2: Read current compiler version from GitHub API metadata instead of shallow-cloning

The compiler version could potentially be read by fetching the raw `.lock.yml` file content via the GitHub contents API (`/repos/{owner}/{repo}/contents/.github/workflows/*.lock.yml`) without cloning. This avoids the overhead of a local git clone. It was considered but rejected because `upgrade --org` already performs a shallow clone later (during the apply phase via `runUpgradeForTargetRepo`), and the scan phase reuses the same checkout infrastructure (`ensureUpdateTargetRepoGitignore`, `shallowCloneTargetRepo`) already established by the `update` command — making cloning during scan consistent and reusing existing abstractions.

#### Alternative 3: Use the old `.md + source:` search query for both commands

Both commands could use the original `update` query (`extension:md "source:"`). This was rejected because the `source:` field text-match can produce false positives (any `.md` with the string `source:`) and the `.lock.yml` filename match is a precise, purpose-built artifact of the compilation process that reliably identifies repos with compiled agentic workflows.

### Consequences

#### Positive
- Both `update --org` and `upgrade --org` now operate on identical repository sets, eliminating operator confusion.
- `update --org` dry-run output shows human-readable version tags (e.g., `ci-doctor: e15e57b -> v1.4.0`) instead of opaque short SHAs.
- `upgrade --org` dry-run output shows current and target compiler versions (e.g., `octo/api (v1.2.3 -> v1.4.0)`), enabling operators to triage urgency before applying.
- The per-repo version-label cache (`versionLabelCache`) avoids redundant GitHub tags API calls when multiple workflows share the same source repo.

#### Negative
- `upgrade --org` now performs a shallow clone of every discovered repo during the scan phase before the apply phase. This adds latency proportional to the number of repos and requires network access and a local git context (the command must be run inside a git repository). Operators upgrading large orgs will see increased wall-clock time during the scan phase.
- The version label resolution for `update --org` makes one additional GitHub API call per unique source repo (fetching the first 100 tags). For orgs with many distinct source repos this may approach API rate limits.

#### Neutral
- The `upgrade --org` scan phase reuses the existing `ensureUpdateTargetRepoGitignore` / `shallowCloneTargetRepo` infrastructure, so the cloned repos end up under the same `.github/workflows-updates/` directory as `update --org` clones. Operators should be aware this directory grows with org size.
- The `searchOrgAnyWorkflowRepos` function (`.md`-extension search) is now dead code and has been removed from `upgrade_org.go`; any callers or tests referencing it have been updated to `searchOrgLockWorkflowRepos`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
