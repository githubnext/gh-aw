# ADR-41159: Honor the active GitHub host for trial repository URLs

**Date**: 2026-06-24
**Status**: Draft
**Deciders**: pelikhan (PR author), gh-aw maintainers

---

### Context

`gh aw trial` (including `--clone-repo`) constructs several URLs for the host and source repositories: clone URLs, force-push remote URLs, displayed repository links, and Actions settings links. These were all hard-coded against `https://github.com/`, so trials run against a GitHub Enterprise Server (GHES) host failed even when `gh` was authenticated to the enterprise host. Worse, a fully-qualified GHES repo spec (`https://example.ghe.com/owner/repo`) was normalized to `owner/repo` and then rebuilt against public GitHub, silently targeting the wrong host. The codebase already exposes `getGitHubHost()`, which resolves the active host from `GITHUB_SERVER_URL` / `GITHUB_ENTERPRISE_HOST` / `GITHUB_HOST` / `GH_HOST`.

### Decision

We will centralize all trial-mode repository URL construction on the active GitHub host. We introduce three small helpers in `pkg/cli/trial_repository.go` — `trialRepositoryURL`, `trialRepositoryGitURL`, and `trialRepositoryActionsSettingsURL` — that build URLs from `getGitHubHost()` plus the repo slug, and we route every previously hard-coded `https://github.com/...` string in the trial flow through them. Clone-mode repo specs continue to parse down to `owner/repo`, but clone/push/display operations now rebuild URLs against the resolved host.

### Alternatives Considered

#### Alternative 1: Inline `getGitHubHost()` at each call site

Replace each hard-coded literal with an inline `fmt.Sprintf("%s/%s.git", getGitHubHost(), slug)` expression. This avoids new functions but duplicates the formatting (including the `.git` and `/settings/actions` suffixes) across six-plus call sites, inviting drift. Rejected because a single set of named helpers is clearer and gives the URL-shape tests one stable surface to assert against.

#### Alternative 2: Thread the host as an explicit parameter

Pass the resolved host (or a host-aware URL builder) down through `ensureTrialRepository`, `cloneTrialHostRepository`, and `cloneRepoContentsIntoHost` instead of reading it from ambient environment inside the helpers. This is more testable and explicit, but it widens several function signatures and is a larger change than the bug fix warrants. Rejected for scope; the helpers keep the existing `getGitHubHost()` resolution that the rest of the CLI already relies on.

### Consequences

#### Positive
- `gh aw trial --clone-repo` now works correctly against GHES hosts.
- A single source of truth for trial URL construction reduces the chance of a future hard-coded `github.com` regression.

#### Negative
- The helpers read the host from process environment via `getGitHubHost()`, so they are not pure functions and require env setup to test in isolation (as the new tests do with `t.Setenv`).
- Three helpers are added for what is essentially string formatting, a small surface-area increase.

#### Neutral
- Behavior on public GitHub is unchanged: with no host env set, `getGitHubHost()` resolves to `github.com`.
- The precedence order among `GITHUB_SERVER_URL`, `GITHUB_ENTERPRISE_HOST`, `GITHUB_HOST`, and `GH_HOST` is inherited unchanged from `getGitHubHost()`.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
