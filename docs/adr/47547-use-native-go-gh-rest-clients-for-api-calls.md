# ADR-47547: Use Native go-gh REST Clients for GitHub API Calls

**Date**: 2026-07-23
**Status**: Draft
**Deciders**: Unknown

---

### Context

`gh-aw` had two gaps in its use of the go-gh library. First, update-check code called `client.Get(...)` which does not accept a context, preventing cancellation from propagating to in-flight GitHub API requests. Second, remote ref-to-SHA resolution used `gh.Exec("api", ...)` — a subprocess call — which required string-parsing of stderr to detect authentication failures and was not unit-testable without spawning a process. Both paths needed to be brought in line with the rest of the codebase, which already used `DoWithContext` and native REST clients elsewhere. The go-gh library's `api.RESTClient.DoWithContext` provides context propagation and returns structured `*api.HTTPError` values instead of unstructured text.

### Decision

We will replace all remaining `client.Get(...)` calls and `gh.Exec("api", ...)` subprocess invocations with `client.DoWithContext(ctx, http.MethodGet, path, nil, &result)` using go-gh native REST clients. Auth failures in the ref-resolution fallback chain will be detected via `errors.As(err, &httpErr)` on `*api.HTTPError` status codes (401/403) rather than stderr string matching. A shared `githubapi.ClientOptions(host, authToken)` helper will seed all `api.ClientOptions` structs with the repository-standard HTTP timeout, eliminating duplicated inline `api.ClientOptions{}` literals.

### Alternatives Considered

#### Alternative 1: Keep subprocess-based `gh.Exec("api", ...)` calls

`gh.Exec` is already a supported go-gh entrypoint and handles `GH_HOST` and authentication automatically. Retaining it avoids refactoring the fallback chain. It was rejected because: (a) subprocess output cannot carry typed error values — auth detection relied on fragile stderr substring matching, (b) context cancellation cannot propagate into a subprocess the caller does not own, and (c) the invocation is not unit-testable without spawning a real `gh` binary.

#### Alternative 2: Switch to a third-party GitHub SDK (e.g. `google/go-github`)

A dedicated GitHub SDK offers richer typed APIs for releases, commits, and repositories. It was not chosen because: (a) go-gh is already a first-party dependency and integrates transparently with `gh`'s auth credential store and enterprise `GH_HOST` routing, and (b) adding a second HTTP client library for GitHub would increase dependency surface without a clear benefit for the narrow set of endpoints used here.

### Consequences

#### Positive
- Context cancellation now reaches in-flight GitHub API calls; commands that are cancelled mid-update-check will not leave goroutines blocked on network I/O.
- Auth errors in ref resolution are detected via typed `*api.HTTPError` status codes (401/403), eliminating fragile stderr substring matching and making the fallback logic deterministic.
- The `releaseRESTClient` and `restCommitResolver` interfaces allow unit tests to inject fake clients, giving focused coverage of endpoint selection and fallback ordering without requiring real network access or a `gh` binary.
- The `githubapi.ClientOptions` helper ensures the repository-standard HTTP timeout is applied consistently across all REST client construction sites.

#### Negative
- `DoWithContext` has a longer call signature than `client.Get`; every call site now explicitly passes `ctx`, `http.MethodGet`, a `nil` body, and a response pointer, which is more verbose.
- The new `pkg/githubapi` package is a thin wrapper; callers must import an additional package for what amounts to a two-field struct constructor.

#### Neutral
- The `getLatestRelease` function signature gained a `ctx context.Context` parameter, requiring all call sites (including the `getLatestOrgReleaseFunc` variable) to be updated to pass `context.Background()` at the top level or thread a real context from callers.
- The `resolveRefToSHAWithFallbacks` function was extracted from `resolveRefToSHA` to accept injected fallback functions, which is a testability refactor with no change in observable behaviour.
- HTTP verb string literals (`"GET"`) across the codebase were normalized to `http.MethodGet`; this is a cosmetic consistency change with no runtime effect.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
