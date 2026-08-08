# ADR-51455: Allow Non-SHA Refs in Skills Frontmatter, Pinned at Compile Time

**Date**: 2026-08-08
**Status**: Draft
**Deciders**: pelikhan (PR author), copilot-swe-agent (implementation)

---

### Context

The `skills:` frontmatter field in workflow markdown files previously required every remote skill reference to be pinned to a full 40-character lowercase commit SHA (e.g., `owner/repo@abc123...def456`). This forced authors to manually look up the current SHA for a branch or tag, paste it in, and update it by hand whenever they wanted to upgrade a dependency. The requirement was intended to guarantee reproducible, tamper-resistant builds — the same `.lock.yml` would always activate the exact same skill code. However, the manual SHA-management burden reduced the ergonomics of skill authoring without adding a meaningful security benefit beyond what compile-time pinning already provides, since the compiler already pins `uses:` action references using a shared GitHub API + cache resolver.

### Decision

We will relax the validation of the `skills:` frontmatter field to accept branch names, tag names, or full commit SHAs as the `<ref>` portion of `owner/repo@<ref>` and `owner/repo/skill/path@<ref>` entries. Non-SHA refs are resolved to their current commit SHA at compile time using the compiler's existing `ActionResolver` (the same infrastructure used to pin `uses:` action references), and the resolved SHA is written into the compiled `.lock.yml`. The original source frontmatter retains the human-readable branch/tag name for authoring convenience. Ambiguous SHA-like strings (hex chars, 7–39 chars) are still rejected to prevent ref confusion. GitHub Actions expressions (`${{ ... }}`) remain unsupported as skill refs. Entries with no ref (`owner/repo@`) are permitted but emit a compiler warning recommending an explicit ref.

### Alternatives Considered

#### Alternative 1: Keep Requiring Full SHA-Only Refs (Status Quo)

Authors continue to specify a 40-character lowercase SHA for every remote skill. The compiler does not need a ref-resolution pass; validation remains a simple regexp.

Rejected because: the ergonomic cost is high — updating a skill pin requires looking up the SHA via `gh api` or browsing GitHub, and there is no automation to help. The security guarantee is not meaningfully stronger than compile-time pinning, since the `.lock.yml` would still be the authoritative artifact used at runtime.

#### Alternative 2: Accept Branch/Tag Refs Without Pinning at Compile Time

Accept any branch/tag ref in frontmatter and pass it through to the `.lock.yml` as-is, relying on the runtime skill installer to resolve the ref at activation time.

Rejected because: this breaks reproducibility — two activations of the same `.lock.yml` at different times can pick up different code if the branch has moved. It would also undermine the security posture that SHA pinning provides, since a compromised branch could silently change what code runs in a workflow.

#### Alternative 3: Resolve Refs at a Separate Pre-Compile Step (CI/CD Automation)

A separate CI job or bot resolves branch/tag refs to SHAs and opens a PR to update the frontmatter, similar to Dependabot-style pin management.

Rejected because: it requires additional infrastructure, introduces lag between authoring and pinning, and does not improve the immediate authoring experience. The compile-time resolution already has access to the resolver and cache, making a separate step redundant.

### Consequences

#### Positive
- Authors can reference skill dependencies by branch or tag name (e.g., `@main`, `@v1.2.3`, `@release/1.0`), eliminating the need to manually look up and maintain SHA strings.
- The compiled `.lock.yml` always contains fully-pinned SHAs, preserving the existing reproducibility and tamper-resistance guarantee at the artifact level.
- Reuses the existing `ActionResolver` + cache infrastructure, keeping the implementation surface small and consistent with how `uses:` action pinning already works.
- Ambiguous SHA-like strings (truncated or malformed SHAs) are explicitly rejected, preventing a class of ref-confusion bugs.

#### Negative
- Resolution failures (e.g., no network access, missing GitHub auth) degrade gracefully to a compiler warning rather than a hard error, which means a `.lock.yml` can be emitted with an unpinned ref in offline or restricted-auth build environments.
- The new `resolveFrontmatterSkillRefs` compiler pass adds a GitHub API call per non-SHA skill ref during compilation; in environments where the cache is cold this adds latency.
- The `owner/repo@` (no-ref) form is now syntactically valid, which may be accidentally used by authors who omit a ref, producing non-reproducible builds that only show a warning.

#### Neutral
- The validation regexp is replaced by structured parsing (split on `@`, validate repo path and ref separately), which is slightly more complex but allows clearer, per-field error messages.
- Existing workflows using full SHA refs are unaffected — they pass through the new validation unchanged and are not re-resolved.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
