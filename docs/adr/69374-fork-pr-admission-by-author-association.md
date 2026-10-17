# ADR-69374: Admit Fork Pull Requests by Explicit Author Association

**Date**: 2026-10-16
**Status**: Draft
**Deciders**: Unknown (PR #69374, issue #69313)

---

### Context

`on.pull_request.forks` currently accepts only repository or owner glob patterns (for example `trusted/repo`, `org/*`, or `*`), and the default admits same-repository pull requests only. In private and internal repositories where collaborators work from personal forks, the set of fork owners is open-ended, so no finite pattern list covers them; the only workable option is `forks: ["*"]`, which also admits fork PRs from authors who have no relationship to the base repository. Editing the compiled `.lock.yml` by hand is not viable because `gh aw compile` overwrites it. GitHub already computes `github.event.pull_request.author_association` from the author's relationship to the *base* repository, which is exactly the trust signal that is missing. This matters most where fork PR runs can reach secrets (for example an internal repository where an administrator enabled sending secrets to fork pull request workflows).

### Decision

We will extend `on.pull_request.forks` to accept an object form with an explicit `author-association` list, alongside the existing string and list pattern forms. The compiler ORs a `contains(fromJSON('[...]'), github.event.pull_request.author_association)` branch onto the existing same-repository check (`pkg/workflow/filters.go`), and the frontmatter schema validates values against GitHub's uppercase enum (`OWNER`, `MEMBER`, `COLLABORATOR`, `CONTRIBUTOR`, `FIRST_TIMER`, `FIRST_TIME_CONTRIBUTOR`, `MANNEQUIN`, `NONE`) in `pkg/parser/schemas/main_workflow_schema.json`. The primary driver is expressiveness without over-admission: membership is explicit and opt-in, no association is implied, and an empty list admits no forks. Admission is kept strictly separate from permissions and secret availability, and the rule is formalized as CTR-034 with a TLA+ model (`specs/workflow-security/PullRequestForkAdmission.tla`).

### Alternatives Considered

#### Alternative 1: Keep pattern-only configuration and use `forks: ["*"]`

Existing behavior already supports allowing all forks with a wildcard, requiring no compiler or schema change. It was rejected because it is strictly broader than the need: it admits PRs from `NONE`/`FIRST_TIME_CONTRIBUTOR` authors in the same breath as trusted collaborators, which is unacceptable where fork runs can authenticate.

#### Alternative 2: Post-process the compiled lock file

A repository could edit the generated `if:` condition in `.lock.yml` to add the association check. This was considered because it requires no upstream change, but it is overwritten by the next `gh aw compile`, is invisible to schema validation, and leaves the security-relevant condition unreviewed and undocumented.

#### Alternative 3: Resolve collaborator permission at runtime via the GitHub API

Instead of trusting `author_association`, the activation job could query the collaborators API for the author's permission level. This was a closer call — it reflects real access rather than an event-supplied field — but it requires an authenticated API call before admission, adds runtime failure modes and rate-limit exposure, and moves trust decisions out of the statically reviewable `if:` expression.

#### Alternative 4: A boolean "trusted authors" shortcut

A single flag such as `forks: {trusted: true}` expanding to a fixed set (`OWNER`, `MEMBER`, `COLLABORATOR`) would be terser. It was rejected because organization membership does not imply repository access in every organization; an explicit list lets a repository omit `MEMBER`.

### Consequences

#### Positive
- Repositories can admit fork PRs from collaborators working on personal forks without allowing all forks.
- The allowed set is explicit and schema-validated, so typos, lowercase values, and wildcards fail at compile time rather than silently widening admission.
- The filter is expressed in the generated `if:` condition, so it stays reviewable in the lock file and survives recompilation.
- The behavior is pinned by a spec rule (CTR-034), a TLA+ model with negative-control faults, and unit tests in `pkg/workflow` and `pkg/parser`.

#### Negative
- `author_association` comes from the webhook event payload and describes a relationship, not an access grant; `CONTRIBUTOR` or `MEMBER` can be configured by mistake and admit more authors than intended.
- Adds a third accepted shape for a single frontmatter field (`string`, `list`, object), increasing parsing and documentation surface in `filters.go` and the schema.
- Association semantics are defined by GitHub and may change outside this repository's control, so the enum must be kept in sync.

#### Neutral
- Same-repository PRs remain admitted and the default (forks denied) is unchanged; existing pattern configurations keep their behavior.
- The `on:` section cleanup now comments out nested compiler-only keys by indentation (`frontmatter_on_section_cleanup.go`), so the association list does not leak into the generated Actions `on:` block.
- The change does not alter GitHub Actions permissions or secret availability; it is useful only where fork runs can already authenticate.
- Documentation updates span `fork-support.md`, `frontmatter-full.md`, and `triggers.md`, and the threat-detection spec version moves to `1.0.53`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
