---
name: 34526-ghe-shorthand-cross-host-fallback-for-public-orgs
description: Route shorthand workflow specs under known public orgs to github.com from GHE contexts and emit actionable 404 hints
---

# ADR-34526: GHE Shorthand Cross-Host Fallback for Known Public Orgs

**Date**: 2026-05-24
**Status**: Draft
**Deciders**: Unknown (PR #34526)

---

## Part 1 — Narrative (Human-Friendly)

### Context

`gh aw add-wizard` accepts shorthand workflow specs of the form `owner/repo/workflow-name[@version]`. When the active GitHub host is a GitHub Enterprise (GHE) server, shorthand resolution previously routed all owners through the enterprise host. Public workflow source repositories such as `github/*`, `githubnext/agentics`, and `microsoft/*` exist only on github.com, so resolving them on a GHE host produced misleading 404 / ref-resolution errors that gave the user no path forward. The codebase already had a narrow special-case for `github/gh-aw` and `githubnext/agentics`, but the set of "always public" sources needed to widen, and the failure mode needed user-actionable remediation guidance instead of an opaque 404.

### Decision

We will route shorthand specs whose owner is `github`, `githubnext`, or `microsoft` to `https://github.com` even when the caller is configured against a GHE host, and we will emit a 404-aware cross-host hint that includes a concrete `gh aw add-wizard https://github.com/...` command whenever ref→SHA resolution fails on a non-github.com host. The same owner allowlist is applied in both `pkg/cli` and `pkg/parser` host-resolution paths to prevent split behavior between the CLI command surface and the parser. `add-wizard` long help is updated to state plainly that shorthand resolves on the enterprise host and that full `https://github.com/...` URLs should be used for public github.com sources.

### Alternatives Considered

#### Alternative 1: Configurable allowlist (no hardcoded orgs)

Expose the "always public" owner allowlist via a config file or environment variable so enterprise administrators could extend or override it. This was not chosen because the current set of public source orgs is small, well-known, and tied to first-party workflow libraries shipped with `gh-aw`; introducing a config surface would add validation, precedence, and documentation complexity for a feature that has only three known consumers today. The allowlist can be promoted to configuration later if a fourth public org or an enterprise-specific override becomes necessary.

#### Alternative 2: Probe-then-fallback (try GHE, then github.com on 404)

Attempt resolution against the configured GHE host first and automatically retry against github.com on 404. This was not chosen because it doubles request latency on the common error path, surfaces unrelated GHE outages as confusing fallback noise, and silently hides the fact that the user's spec was ambiguous about which host it targeted. Surfacing an actionable hint and requiring the user to either use a known public org or supply a full URL keeps host intent explicit.

#### Alternative 3: Reject shorthand outright on GHE hosts

Require full URLs for any non-GHE source whenever the active host is GHE. This was not chosen because it breaks ergonomics for the most common public sources (`github/*`, `githubnext/agentics`) that GHE users routinely consume, and would force every example, doc, and tutorial to be rewritten in URL form.

### Consequences

#### Positive
- GHE users can run `gh aw add-wizard github/...`, `githubnext/...`, and `microsoft/...` shorthand specs without manual URL construction.
- When shorthand fails on a non-github.com host, the error message now contains a copy-pasteable `gh aw add-wizard https://github.com/...` command, turning a dead-end 404 into a one-step remediation.
- CLI and parser host-resolution paths share the same allowlist, eliminating the prior risk of one surface routing to github.com while the other routed to GHE for the same spec.

#### Negative
- The set of "always public" orgs is hardcoded in two files (`pkg/cli/github.go`, `pkg/parser/github.go`); adding a fourth org requires a coordinated edit in both locations.
- GHE deployments that legitimately mirror `github/<repo>`, `githubnext/<repo>`, or `microsoft/<repo>` on their enterprise host can no longer resolve those shorthand specs against the GHE mirror — they must use a different owner or a full URL.
- The 404-hint heuristic relies on substring matching against error text (`"not found"`, `"http 404"`), which is brittle if the underlying GitHub API client changes its error formatting.

#### Neutral
- The hardcoded allowlist replaces the previous narrower hardcoded list (`github/gh-aw`, `githubnext/agentics`); the mechanism is unchanged, only the membership widened.
- The hint is suppressed when the resolved host is empty or already `github.com`, so behavior on public-GitHub-only deployments is unchanged.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Host Resolution for Shorthand Specs

1. Implementations **MUST** route shorthand workflow specs whose owner is `github`, `githubnext`, or `microsoft` to `https://github.com`, regardless of the configured GitHub Enterprise host.
2. Implementations **MUST NOT** route shorthand specs for owners outside the public-org allowlist to `https://github.com` when a GHE host is configured; such specs **MUST** continue to resolve against the configured host.
3. The public-org allowlist used by the CLI host-resolution path (`pkg/cli`) and the parser host-resolution path (`pkg/parser`) **MUST** be identical.
4. Implementations **SHOULD** log a diagnostic message identifying the public org whenever the github.com fallback is applied.

### Cross-Host Failure Hints

1. When ref→SHA resolution fails with an error whose text contains `not found` or `http 404` (case-insensitive), and the resolved host is non-empty and not `github.com`, implementations **MUST** emit a user-facing hint that:
   1. names the host on which resolution was attempted, and
   2. includes a concrete `gh aw add-wizard https://github.com/<owner>/<repo>/blob/<ref>/<workflow-path>` command using the original spec components.
2. Implementations **MUST NOT** emit the cross-host hint when the resolved host is `github.com` or when the resolved host is empty.
3. Implementations **MUST NOT** emit the cross-host hint for `nil` errors or for errors whose text does not match the 404 substrings above.
4. Implementations **MUST** normalize a leading slash from the workflow path before constructing the suggested URL so that the URL does not contain `blob/<ref>//<path>`.

### User-Facing Documentation

1. The `add-wizard` command long help **MUST** state that, in GitHub Enterprise repositories, shorthand specs resolve on the enterprise host and that full `https://github.com/...` URLs are required when sourcing public github.com workflows.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26372124288) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
