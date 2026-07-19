# ADR-26482: Peel Annotated Git Tags to Commit SHA When Resolving Action Pins

**Date**: 2026-04-15
**Status**: Draft
**Deciders**: pelikhan, Copilot

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh aw compile` command pins GitHub Actions references by SHA to produce reproducible, Renovate-compatible workflow files. The GitHub REST API endpoint `GET /repos/{owner}/{repo}/git/ref/tags/{tag}` returns two semantically different SHAs depending on whether the tag is *lightweight* (SHA points directly to a commit) or *annotated* (SHA points to a tag object, which in turn points to a commit). Before this decision, the compiler always emitted the SHA returned by this endpoint, which for annotated tags is the tag-object SHA rather than the underlying commit SHA. Renovate, and tooling that compares lock files against real commits, expects the commit SHA — so annotated tags caused Renovate to repeatedly rewrite the same `uses:` line, churning the lock file without actually changing anything meaningful.

### Decision

We will resolve annotated tags by making a second API call to `GET /repos/{owner}/{repo}/git/tags/{tag-object-sha}` to retrieve the underlying commit SHA (peeling). For lightweight tags the first API call already returns the commit SHA and no second call is needed. We distinguish the two cases by fetching both the object SHA and its type (`[.object.sha, .object.type] | @tsv`) in a single `gh api` invocation, parsing via a new exported helper `ParseTagRefTSV`. The same two-call strategy is applied consistently in both `action_resolver.go` (used by the compiler) and `getActionSHAForTag` in `update_actions.go` (used by `gh aw update-actions`), with code reuse through the shared `ParseTagRefTSV` helper.

### Alternatives Considered

#### Alternative 1: Use the `^{}` Dereferencing Suffix via `git ls-remote`

`git ls-remote` emits a peeled entry suffixed with `^{}` for annotated tags (e.g., `refs/tags/v1.0.0^{}`), which directly resolves to the commit SHA without a second API call. This was not chosen because the primary SHA resolution path already uses the GitHub REST API via `gh api`, not `git ls-remote`. Mixing two different resolution mechanisms would increase code complexity, introduce inconsistent behavior between the primary and fallback paths, and would require shelling out to `git` where the codebase currently relies on `gh`.

#### Alternative 2: Use the GitHub GraphQL `object { oid }` Query

The GitHub GraphQL API can resolve a tag ref to its target commit OID in a single query, handling peeling transparently. This approach was not chosen because the rest of the action-resolution code uses the REST API and replacing it with GraphQL would require a larger refactor. GraphQL access also adds a dependency on a different authentication scope and endpoint, whereas the existing `gh api` calls already have the necessary permissions.

#### Alternative 3: Accept the Tag Object SHA and Normalise at Diff Time

An alternative is to accept the tag-object SHA as the pin value and teach the diff/update logic to treat tag-object SHAs and commit SHAs as equivalent. This was not chosen because it would require invasive changes to multiple downstream consumers (Renovate config, lock-file diff logic) and doesn't fix the root cause — Renovate correctly expects a commit SHA.

### Consequences

#### Positive
- Renovate no longer rewrites the same `uses:` line for annotated-tag-based actions; lock files are stable.
- The new `ParseTagRefTSV` helper is unit-testable in isolation, improving test coverage of the critical parsing step.
- The fix is applied symmetrically in both the compiler path and the update-actions path, so both tools emit the same stable commit SHA.

#### Negative
- Annotated tags now require two sequential GitHub API calls instead of one, doubling network latency for those tags. In practice most GitHub Actions repositories use annotated tags, so this will be the common case.
- The two-call strategy is sensitive to rate-limiting: a rate-limit hit on the second (peeling) call results in a hard failure, with no partial-result fallback.

#### Neutral
- The `ParseTagRefTSV` function is exported from `pkg/workflow` (`github.com/github/gh-aw/pkg/workflow`), which is importable by external modules and is also documented in `pkg/workflow/README.md`. This means `ParseTagRefTSV` **is** a public symbol with SemVer implications: renaming or changing its signature without a major-version bump would be a breaking change for any external consumer that imports `pkg/workflow`. If the function is intended as an internal implementation detail, it should either be unexported (renamed to `parseTagRefTSV`) or moved to an `internal/` package. Until that refactoring occurs, any change to `ParseTagRefTSV`'s signature must be treated as a public API change.
- Integration tests require network access to validate the full two-call flow; unit tests cover only the TSV parsing.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Tag Resolution

1. Implementations **MUST** emit the underlying commit SHA when pinning a GitHub Actions reference, regardless of whether the tag is lightweight or annotated.
2. Implementations **MUST NOT** emit a tag-object SHA as the pin value for an action reference.
3. When querying `GET /repos/{owner}/{repo}/git/ref/tags/{tag}`, implementations **MUST** fetch both the object SHA and the object type in a single request.
4. Implementations **MUST** treat a response with `object.type == "tag"` as an annotated tag and **MUST** make a second API call to `GET /repos/{owner}/{repo}/git/tags/{tag-object-sha}` to retrieve the commit SHA.
5. Implementations **MUST** treat a response with `object.type == "commit"` as a lightweight tag and **MUST NOT** make a second API call for tag peeling.
6. Implementations **MUST** validate that any resolved SHA is exactly 40 hexadecimal characters; if not, resolution **MUST** fail with a descriptive error.

### Shared Parsing Helper

1. The tab-separated parsing of GitHub API tag-ref responses **MUST** be performed via the `workflow.ParseTagRefTSV` function (or a functional equivalent in the same package).
2. Implementations **MUST NOT** duplicate the tab-separated parsing logic inline in multiple call sites; use the shared helper.
3. The `ParseTagRefTSV` function **MUST** return an error for any of the following malformed inputs: empty string, missing tab separator, empty SHA field, empty type field, or SHA that is not exactly 40 characters.

### Consistency Across Resolution Paths

1. Both the compiler path (`ActionResolver.resolveFromGitHub`) and the update path (`getActionSHAForTag`) **MUST** apply the same two-call annotated-tag peeling strategy.
2. Implementations **SHOULD** share the peeling logic via the `ParseTagRefTSV` helper to avoid divergence between the two paths.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Specifically: it always emits a commit SHA (never a tag-object SHA), it validates SHA length, it does not duplicate TSV parsing logic, and it applies the two-call strategy consistently across both resolution paths. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

### Rate-Limit Handling Norms

1. When the second (peeling) API call to `GET /repos/{owner}/{repo}/git/tags/{tag-object-sha}` receives an HTTP 429 or a rate-limit-exceeded response, implementations **MUST** fail with a descriptive error rather than silently falling back to the tag-object SHA. Emitting a tag-object SHA as the pin value is explicitly prohibited by requirement 2 of the Tag Resolution section.
2. Implementations **SHOULD NOT** perform automatic retries on rate-limit responses during a single compile invocation; callers are responsible for retry logic at a higher level (e.g., CI re-run).
3. The error message produced on a rate-limit failure **MUST** include the tag reference and the HTTP status code so that operators can diagnose the cause without inspecting raw HTTP logs.

---

### Status Promotion

This ADR is currently **Draft**. To promote it to **Accepted**, all of the following criteria must be satisfied:

- [ ] The PR implementing annotated-tag peeling has been merged to the default branch.
- [ ] All CI checks (tests, linters, compilation) are green on the merged commit.
- [ ] An integration test for annotated-tag pinning exists in CI (`pkg/workflow/action_resolver_test.go` or equivalent).
- [ ] The rate-limit error path is covered by at least one unit test asserting the descriptive error message.
- [ ] No open issues reference a conformance failure against this ADR.

Once all boxes are checked, update the `**Status**` field at the top of this document from `Draft` to `Accepted`.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/24474969210) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
