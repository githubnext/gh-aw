# ADR-34037: Resolve Remote Workflow Filenames via Directory Listing

**Date**: 2026-05-22
**Status**: Draft
**Deciders**: Unknown (draft generated from PR #34037)

---

## Part 1 — Narrative (Human-Friendly)

### Context

`gh aw experiments analyze --repo` reads each experiment's workflow frontmatter to
recover overrides for `hypothesis`, `analysis_type`, `guardrail_metrics`, and
`min_samples`. Experiment branch names are sanitized via
`SanitizeWorkflowIDForCacheKey` — hyphens are stripped and the name is
lowercased — so the branch `experiments/cicoach` carries the experiment name
`cicoach`, while the corresponding workflow file is `.github/workflows/ci-coach.md`.
Because sanitization is lossy, the experiment name alone cannot deterministically
reconstruct the original filename, and `loadRemoteExperimentConfigs` was
404'ing on every hyphenated workflow and silently falling back to defaults.

### Decision

We will resolve the remote workflow filename by **listing
`.github/workflows/` via the GitHub contents API and matching files whose
`SanitizeWorkflowIDForCacheKey(basename)` equals the experiment name**, mirroring
the existing local-disk resolution in `findWorkflowFileForExperiment`. The bare
sanitized name is retained only as a last-resort fallback when the directory
listing is unavailable. The shared scan loop is extracted into the pure helper
`matchWorkflowFilenameByExperiment` so the same matching logic backs both local
and remote lookups and can be unit-tested in isolation.

### Alternatives Considered

#### Alternative 1: Enumerate hyphen-reinsertion candidates from the sanitized name

We could programmatically generate filename candidates by inserting hyphens at
every position of the sanitized name (`cicoach` → `c-icoach`, `ci-coach`,
`cic-oach`, …) and try each one against the contents API. Rejected because the
candidate count grows combinatorially with name length, every miss is a wasted
404, and the approach silently picks the *first* HTTP-200 result rather than the
authoritative filename — easy to fool when two similarly named workflows exist.

#### Alternative 2: Maintain a static experiment-name → workflow-filename map

A hand-maintained mapping (or a generated one committed alongside experiments)
would avoid the directory listing. Rejected because it requires every new
experiment to remember to update the map, drifts silently when files are
renamed, and reintroduces the exact class of bug this PR fixes — a name lookup
that disagrees with the real filesystem state.

#### Alternative 3: Require experiment branch names to preserve hyphens

We could change `SanitizeWorkflowIDForCacheKey` so the experiment name remains a
faithful slug of the filename. Rejected as out of scope: the sanitizer is shared
with cache-key logic that has its own constraints (filesystem-safe, case-folded
keys), and changing it would ripple through unrelated subsystems and invalidate
existing cache entries.

### Consequences

#### Positive

- Hyphenated workflow filenames now load correctly for `--repo` analyses, so
  `hypothesis`, `analysis_type`, `guardrail_metrics`, and `min_samples`
  overrides are actually honored in remote reports.
- Local and remote lookups share `matchWorkflowFilenameByExperiment`, removing a
  divergence where the local path was already correct.
- Ambiguous collisions (two files sanitizing to the same name) are logged with
  a warning instead of silently picking an arbitrary file.

#### Negative

- Each `loadRemoteExperimentConfigs` call now incurs one additional `gh api`
  request to list `.github/workflows/` before fetching the workflow markdown.
- The fix depends on the contents API being reachable and on the caller having
  read access to `.github/workflows/`; in degraded environments we silently fall
  back to the sanitized name and behave as before.
- When `.github/workflows/` is unusually large, the listing response grows
  proportionally; today this is bounded by repo size, but there is no paging.

#### Neutral

- `workflowFileCandidates` is now documented as a last-resort fallback rather
  than the primary resolution path; its previous "hyphen reinsertion" intent is
  removed.
- A new test, `TestMatchWorkflowFilenameByExperimentAmbiguous`, locks in
  "first-match wins" behavior on collisions.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**,
> **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this
> section are to be interpreted as described in
> [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Remote Workflow Filename Resolution

1. `loadRemoteExperimentConfigs` **MUST** attempt to resolve the workflow
   filename by listing `.github/workflows/` via the GitHub contents API before
   falling back to the bare sanitized experiment name.
2. The resolver **MUST** consider a workflow file a match if and only if
   `SanitizeWorkflowIDForCacheKey(basename(file))` equals the experiment name
   passed in.
3. The resolver **MUST NOT** generate hyphen-reinsertion candidates or any other
   speculative filename variants in place of the directory listing.
4. When the directory listing fails (network error, permission error, parse
   error), the resolver **MUST** log the failure and **MUST** fall through to
   the bare sanitized name fallback rather than aborting the analysis.
5. When more than one file sanitizes to the same experiment name, the resolver
   **MUST** log a warning identifying all colliding filenames and **MUST**
   return the first match deterministically (by the order returned from the
   listing).

### Shared Matching Helper

1. The matching logic **MUST** live in a pure helper
   (`matchWorkflowFilenameByExperiment`) that takes a list of filenames and an
   experiment name and returns a basename or the empty string.
2. The helper **MUST NOT** perform I/O, so it can be exercised by unit tests
   without GitHub credentials or filesystem fixtures.
3. Both local (`findWorkflowFileForExperiment`) and remote
   (`findRemoteWorkflowFilenameForExperiment`) resolvers **SHOULD** delegate to
   the same helper to keep their matching semantics aligned.

### Fallback Behavior

1. `workflowFileCandidates` **MUST** be treated as a last-resort fallback only;
   callers **SHOULD NOT** rely on it as the primary resolution mechanism.
2. New callers introducing remote workflow lookups **SHOULD** use
   `findRemoteWorkflowFilenameForExperiment` rather than calling
   `workflowFileCandidates` directly.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all
**MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or
**MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/26295141671) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
