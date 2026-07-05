---
description: Detailed trigger and escalation guidance referenced by create-agentic-workflow.md section 2.
---

## Reporting and digest guidance

For recurring reports, audits, and stakeholder digests, set these create-specific defaults:

- default to `create-issue`; use `create-discussion` only when the requester explicitly wants threaded discussion
- use `add-comment` only when updating an existing issue or pull request instead of creating a new report destination
- add `workflow_dispatch` when manual reruns, backfills, or preview runs should be possible

Recurring-report lifecycle checklist (compact):

- [ ] enable `close-older-issues: true` for issue-based recurring reports unless the requester explicitly wants parallel open threads
- [ ] define one explicit report window (for example `last 7 full days ending at run start (UTC)` or `since previous successful run`)
- [ ] define grouping dimensions that match audience decisions (for example team, area, owner, severity, status)
- [ ] derive one stable dedup key per scope and window (for example `stakeholder-digest:<scope>:<window-id>`) and search for it before creating a new issue

Follow [triggers.md](triggers.md) for the report window, grouping dimensions, deduplication key, and empty-window `noop` rule, and [workflow-patterns.md](workflow-patterns.md) for the digest/incident skeletons. When the digest depends on missing or inconsistent metadata, group by the next-best available dimension, use an explicit "Unclassified" bucket, and never invent classifications — call `noop` only when the window itself has zero events.

## Persona-oriented scenario map

Use these defaults when the requester frames the automation in non-engineering persona language:

| Persona or scenario | Trigger and scope | Typical tools and outputs | Required prompt details |
|---|---|---|---|
| Program Manager or information-worker digest | `schedule` plus `workflow_dispatch` for previews, reruns, and backfills | `github` (`gh-proxy`); `create-issue` by default | Define the report window, grouping dimensions, deduplication key, and `noop` behavior for empty windows |
| Designer or design-governance review | `pull_request` with `paths:` scoped to UI, design-token, copy, or asset files | `github` (`gh-proxy`); optional `playwright`; `add-comment` on the PR | State the review rubric (for example accessibility, token consistency, asset policy), and call `noop` when scoped files are unchanged |
| Legal / compliance / documentation-policy review | `pull_request` with scoped `paths:` or `schedule` for recurring audits | `github` (`gh-proxy`); `add-comment` for findings; `create-issue` only for violations needing follow-up | Classify findings against the policy, search for existing open issues before escalating, and call `noop` when there is no in-scope change or violation |

## Backend review guidance

For backend-focused PR automation (schema migrations and API compatibility):

- scope `pull_request.paths` to backend contract indicators instead of whole-repo review
- instruct the agent to classify changes as additive, backward-compatible, or breaking, then report only actionable risks
- include explicit `noop` criteria when no migration/API contract files changed

## PR analyzer escalation guidance

For PR-triggered automation that must decide between commenting, creating an issue, or doing nothing:

| Condition | Action |
|---|---|
| Findings affect only this PR (style, quality, risk) | `add-comment` on the PR |
| Finding is a cross-cutting or team-wide concern requiring follow-up beyond this PR | `create-issue` |
| No findings, or only docs/metadata changed outside scoped `paths:` | `noop` |

Rules:

- prefer `add-comment` over `create-issue` for PR-local findings; issues outlive the PR and create noise
- before creating an issue, search for an existing open issue covering the same concern (use a stable title prefix or label to avoid duplicates)
- if a matching open issue already exists, add a linked `add-comment` on the PR referencing it instead of opening a duplicate issue
- call `noop` explicitly whenever no actionable finding exists — do not comment with "no issues found" text

## Incident dedup-key templates (`workflow_run` and `deployment_status`)

For incident workflows, define one stable dedup key before creating output and search for an open issue containing that key.

Use and adapt these templates:

```text
# workflow_run incident key
incident:workflow_run:<workflow-name>:<job-name-or-unknown>:<error-signature>:<window-id>
example: incident:workflow_run:CI:lint:eslint-error:2026-07-05

# deployment_status incident key
incident:deployment_status:<environment-or-ref>:<provider-or-target>:<error-signature>:<window-id>
example: incident:deployment_status:production:vercel:build-timeout:2026-07-05
```

Template rules:

- keep `<error-signature>` stable (normalized failing step, error class, or provider error code)
- use `<window-id>` based on the selected reporting window (for example `2026-07-05` or `2026-W27`)
- create a new issue only when no open issue matches the same key
- call `noop` when the event is non-terminal, recovered, or already represented by an open issue with the same key

## Compliance review guidance

For dependency-license compliance and policy review on PRs:

- scope `pull_request.paths` to dependency manifest files (for example `package.json`, `go.mod`, `requirements.txt`, `Cargo.toml`, `pyproject.toml`, `composer.json`)
- classify each new dependency by license tier using the project's configured policy (the example tiers below represent a common MIT-compatible policy; adjust for your project): **allowed** (MIT, Apache-2.0, BSD, ISC), **needs-review** (unknown, dual-licensed, weak-copyleft), **blocked** (strong-copyleft such as GPL/AGPL, proprietary, or licenses incompatible with your project's license)
- publish per-tier findings with `add-comment` listing each dependency, its version, and detected license
- escalate to `create-issue` only when a **blocked** dependency was added or a policy violation requires team-wide follow-up beyond this PR
- before creating a new issue, search for an existing open issue with a stable key (for example `license-violation + dependency-name + version`) to avoid duplicates; if found, link to it from the PR comment instead
- call `noop` when no new dependencies were added or all additions are confirmed in the allowed tier

**Compliance escalation decision table:**

| Finding | Action |
|---|---|
| No dependency manifest files changed | `noop` immediately |
| All new dependencies in allowed tier | `noop` (or brief `add-comment` confirmation when the workflow prompt explicitly requests a confirmation comment) |
| Dependencies in needs-review tier | `add-comment` listing them with license details and requesting maintainer confirmation |
| Blocked dependency added | `add-comment` flagging the violation + `create-issue` for team-wide record (skip `create-issue` if a matching open issue already exists) |

## Coverage-analysis guidance

For workflows that read, analyze, or comment on test coverage (PR comments, trend tracking, coverage gates):

- **Prefer existing artifacts**: check for a coverage artifact from the current or parent CI run before recomputing; use `actions: read` via `gh-proxy` to list and download artifacts.
- **Prefer PR signals**: read existing check run annotations or coverage diff comments before fetching raw data; only recompute when no artifact or annotation is available.
- **Explicit fallback**: when no artifact exists, document the fallback computation step in the workflow prompt; never invent coverage values.
- call `noop` when no coverage data can be retrieved or computed and there is no meaningful output to report.

See [test-coverage.md](test-coverage.md) for the full coverage data strategy.
