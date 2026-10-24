# ADR-71681: Use Explicit Repository Endpoints for Remote `gh api` Calls

**Date**: 2026-10-24
**Status**: Draft
**Deciders**: pelikhan, Copilot [TODO: verify]

---

### Context

`gh aw experiments list/analyze --repo owner/repo` reads experiment branches, workflow files, and state/eval files from a remote repository by shelling out to `gh api`. The existing code built endpoints with the `repos/{owner}/{repo}/...` placeholder and passed `--repo <owner/repo>`, but `gh api` does not support `--repo`; the placeholder is resolved from the ambient checkout (or `GH_REPO`) instead, so remote requests failed or targeted the wrong repository. The same defect existed in the shared branch file reader (`readRemoteRepoBranchFileContext`) used for state/eval reads. Remote repositories may be on github.com or a GitHub Enterprise host, and branch names and file paths can contain characters such as `#`, spaces, `+`, and `{}` that must not be interpreted by `gh`.

### Decision

We will build every remote `gh api` request through a single helper, `repositoryAPIArgs`, which normalizes the `--repo` value with `repoutil.NormalizeRepoForAPI`, emits a fully explicit `repos/<owner>/<repo>/<endpoint>` path (with owner/repo and endpoint path escaping), and appends `--hostname` derived from an explicit hostname or the repository's host prefix. Git refs are passed as literal `--raw-field ref=...` values rather than `--field`, so `gh` does not perform placeholder or type expansion on them. Regression tests assert the exact `gh` argument vectors, reject `--repo` and `{owner}`/`{repo}` placeholders, and cover public and enterprise targets.

### Alternatives Considered

#### Alternative 1: Keep `{owner}/{repo}` placeholders and set `GH_REPO` per command

`gh api` resolves placeholders from `GH_REPO`, so exporting it per subprocess would have fixed resolution with minimal code changes. It was not chosen because it keeps repository selection implicit in environment state, is easy to break when ambient `GH_REPO`/`GH_HOST` values leak in (which the new tests explicitly guard against), and does not address host selection or literal ref handling.

#### Alternative 2: Call the GitHub REST API directly via a Go client library

Using a Go HTTP/REST client (e.g. go-gh's REST client) would avoid shell argument construction entirely. It was not chosen because the rest of the CLI consistently shells out via `workflow.ExecGH`, reusing `gh` authentication, host configuration, and `--paginate`/`--jq` behaviour; switching transports for this one feature would be a larger, inconsistent change for a bug fix.

### Consequences

#### Positive
- Remote experiment commands target exactly the requested repository and host, independent of the current checkout, `GH_REPO`, or `GH_HOST`.
- One shared helper centralizes repository normalization, escaping, and hostname handling for experiment fetches and the branch file reader, reducing duplicated argument construction.
- Literal `--raw-field` refs and escaped paths make branch names and file paths with special characters safe.

#### Negative
- `--hostname` is now emitted even for an explicit `github.com` host, slightly changing argument shape compared to before (previously github.com was omitted).
- Tests are tightly coupled to exact `gh` argument vectors, so harmless argument reordering will require test updates.
- Other call sites in the repository that still use `{owner}/{repo}` with `--repo` are not migrated by this decision. [TODO: verify whether any remain]

#### Neutral
- Pagination, legacy `state.json` fallback, and local (non-API) mode behaviour are preserved.
- The helper lives in `pkg/cli/branch_file_reader.go` and is package-private; broader reuse would require moving it.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
