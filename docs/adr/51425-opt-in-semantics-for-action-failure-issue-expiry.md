# ADR-51425: Opt-In Semantics for Action-Failure Issue Expiry and Maintenance Generation

**Date**: 2026-08-08
**Status**: Draft
**Deciders**: pelikhan, Copilot SWE Agent

---

### Context

gh-aw always injects a `GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS` value into compiled lock files (defaulting to 168 hours), but `scanWorkflowsForExpires` only considered explicit safe-output expiry when deciding whether to generate `agentics-maintenance.yml`. This meant that repositories relying solely on the implicit action-failure default would get failure issues tagged with an expiration marker that no scheduled `close-expired-issues` job would ever enforce. Grouped parent issues received the same unenforceable marker. Because `safe-outputs.report-failure-as-issue` defaults to `true`, naively treating the 168-hour default as an unconditional maintenance trigger would force `agentics-maintenance.yml` into essentially every repository with a gh-aw workflow — violating the opt-in posture established for no-op issue expiry in #37965.

### Decision

We will treat `maintenance.action_failure_issue_expires` in `aw.json` as a pure opt-in. Only an explicitly configured value (detected via a secondary `map[string]json.RawMessage` pass in `UnmarshalJSON` that Go's `omitempty` cannot provide) causes `scanWorkflowsForExpires` to count action-failure expiry as a maintenance trigger and include it in the minimum-expires calculation. When `scanWorkflowsForExpires` returns `hasExpires=false` (no recognized expiry source, including no explicit action-failure config), the compiler patches already-generated lock files to replace the implicit `"168"` marker with `"0"`. The runtime treats `"0"` as "expiration disabled" and omits the marker from failure issues. When another recognized expiry source does trigger maintenance, the implicit marker is preserved because the generic `close-expired-issues` sweeper will enforce it. Parent issue reuse additionally checks the existing parent's expiration marker before sub-issue count, mirroring existing per-run issue logic.

### Alternatives Considered

#### Alternative 1: Make the implicit 168-hour default an unconditional maintenance trigger

Treat the implicit `action_failure_issue_expires` default the same as an explicit value: if any workflow could report failure as an issue, always generate `agentics-maintenance.yml`. This removes the producer/consumer gap without requiring users to touch `aw.json`. Rejected because `report-failure-as-issue` defaults to `true`, which would cause every repository with a gh-aw workflow to receive a scheduled maintenance workflow — exactly the behavior the existing opt-in posture for no-op issue expiry (#37965, #38627) was designed to prevent.

#### Alternative 2: Remove the implicit 168-hour default; require explicit configuration for any expiry

Eliminate the implicit default entirely and only write an expiry marker when `action_failure_issue_expires` is explicitly set. This is a simpler contract but is a breaking change: existing compiled lock files and any documentation referring to the 168-hour default would be invalidated. It would also break backwards compatibility with older lock files (which always contain a positive value). Rejected in favor of the current approach, which preserves the default for repositories that have another maintenance source.

### Consequences

#### Positive
- Failure issues no longer advertise expiration deadlines that no scheduled job will enforce, eliminating a silent correctness gap.
- Repositories that don't need scheduled maintenance are not forced to adopt it; the opt-in posture matches the precedent set for no-op issue expiry.
- Explicit `action_failure_issue_expires` values participate correctly in the minimum-expires calculation for the `close-expired-issues` cron schedule.
- Expired grouped parent issues are now detected and bypassed before sub-issue count is checked, preventing indefinite accumulation of sub-issues under an expired parent.

#### Negative
- `UnmarshalJSON` for `RepoConfig` now performs two JSON unmarshal passes on the maintenance object (one typed, one into `map[string]json.RawMessage`) to distinguish explicit from absent fields — a non-obvious pattern that future maintainers may find surprising.
- The runtime meaning of `GH_AW_ACTION_FAILURE_ISSUE_EXPIRES_HOURS="0"` changes from "fall back to 168" to "expiration disabled." The change is backwards-safe because compiler-generated lock files previously never contained `"0"`, but the new semantics must be documented and kept in sync with the compiler's patching logic.

#### Neutral
- Side-repository (`failure-issue-repo`) maintenance coverage is explicitly left as a follow-up; the current fix addresses only the primary-repository case.
- The `anyWorkflowMayReportFailureAsIssue` helper introduces a new scan over `workflowDataList` but is only called from within `scanWorkflowsForExpires`, which is already O(n) over the same list.

---

*ADR created by [adr-writer agent]. Review and finalize before changing status from Draft to Accepted.*
