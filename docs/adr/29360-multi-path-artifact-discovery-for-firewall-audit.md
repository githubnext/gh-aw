# ADR-29360: Multi-Path Artifact Discovery Strategy for Firewall Audit Command

**Date**: 2026-04-30
**Status**: Draft
**Deciders**: pelikhan

---

## Part 1 — Narrative (Human-Friendly)

### Context

The `gh aw audit` command relies on `detectFirewallAuditArtifacts` to locate `policy-manifest.json` and `audit.jsonl` before it can display firewall audit results. Artifact directories may arrive in several layouts depending on how they were obtained: some users run `gh run download` directly without passing the directory through `flattenUnifiedArtifact`, and the internal path structure of the downloaded artifact varies further by the version of `actions/upload-artifact` used (v4+ strips the `/tmp/gh-aw/` common prefix; older releases preserve it). On top of that, the artifact container itself may be named `agent`, `agent-artifacts`, or `<hash>-agent` depending on when the workflow was created. The original implementation only covered the post-`flattenUnifiedArtifact` path and the legacy `firewall-audit-logs` directory, leaving the non-flattened cases completely unsupported.

### Decision

We will extend `detectFirewallAuditArtifacts` to probe four candidate locations in a fixed priority order: (1) the post-flatten primary path, (2) the non-flattened new artifact structure, (3) the non-flattened old artifact structure, and (4) the legacy standalone `firewall-audit*` directory. The function stops as soon as both files are found and uses a shared `checkDir` closure to eliminate the repetitive stat-and-assign pattern across all search steps. This makes `gh aw audit` work correctly regardless of whether `flattenUnifiedArtifact` has been called and regardless of the `actions/upload-artifact` version in use.

### Alternatives Considered

#### Alternative 1: Require `flattenUnifiedArtifact` Before Audit

Reject non-flattened directories and document that `gh aw audit` requires a pre-processed directory. This would keep the function simple but breaks the manual download workflow — users who run `gh run download` and point the command at the result directory would get a silent failure (no audit output) rather than a useful error. That silent failure is worse than added complexity.

#### Alternative 2: Normalise on Download

Run the path-normalisation step (`flattenUnifiedArtifact`) automatically inside `gh aw audit` before probing for files. This would work but couples the audit command to the artifact download pipeline, is harder to test in isolation, and could mutate the caller's directory unexpectedly.

### Consequences

#### Positive
- `gh aw audit` works for both the automated pipeline path and the manual `gh run download` path without requiring any user-side workaround.
- Fully backward-compatible: all existing search paths (post-flatten primary and legacy `firewall-audit*`) are preserved at the same priority they had before.
- The `checkDir` closure removes duplicated stat+assign logic, making each search step a single readable call.

#### Negative
- `detectFirewallAuditArtifacts` now encodes knowledge of four distinct directory layouts; adding a fifth layout in the future requires understanding this ordering contract.
- The function implicitly depends on the naming conventions of `findArtifactDir` (suffixes `agent`, `agent-artifacts`, and `*-agent`); a rename of those artifacts without a corresponding update here would silently break step 2 and 3.

#### Neutral
- Four new table-driven sub-tests are added to `TestDetectFirewallAuditArtifacts` to cover each newly supported layout; the test file grows by ~68 lines.
- The search order is documented in a block comment on the function signature, establishing a reference point for future readers.

---

## Part 2 — Normative Specification (RFC 2119)

> The key words **MUST**, **MUST NOT**, **REQUIRED**, **SHALL**, **SHALL NOT**, **SHOULD**, **SHOULD NOT**, **RECOMMENDED**, **MAY**, and **OPTIONAL** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

### Artifact Discovery Search Order

1. Implementations **MUST** probe candidate directories in the following fixed priority order and stop as soon as both `policy-manifest.json` and `audit.jsonl` are found:
   1. `<runDir>/sandbox/firewall/audit/` — post-`flattenUnifiedArtifact` primary path
   2. `<agentDir>/sandbox/firewall/audit/` — non-flattened artifact, `actions/upload-artifact` v4+ (prefix stripped)
   3. `<agentDir>/tmp/gh-aw/sandbox/firewall/audit/` — non-flattened artifact, older upload action (prefix preserved)
   4. `<runDir>/firewall-audit*/` — legacy standalone audit artifact
2. Implementations **MUST** resolve `<agentDir>` by searching `<runDir>` for a subdirectory whose name is `agent`, `agent-artifacts`, or matches the pattern `*-agent` (workflow_call hash prefix).
3. Implementations **MUST NOT** mutate or reorder the caller-supplied `runDir` as a side-effect of artifact discovery.
4. Implementations **SHOULD** log the path of each file found and the search label where it was located to aid future debugging.

### Shared Discovery Helper

1. Implementations **MUST** use a single shared helper (closure or function) that checks one directory for both `policy-manifest.json` and `audit.jsonl`, so that the check logic is not duplicated across search steps.
2. The shared helper **MUST** only populate a result slot (manifest path or audit path) if that slot has not already been filled by an earlier search step.
3. The shared helper **MUST** return a boolean indicating whether both files have been found, so callers can short-circuit early.

### Conformance

An implementation is considered conformant with this ADR if it satisfies all **MUST** and **MUST NOT** requirements above. Failure to meet any **MUST** or **MUST NOT** requirement constitutes non-conformance.

---

*This is a DRAFT ADR generated by the [Design Decision Gate](https://github.com/github/gh-aw/actions/runs/25181134127) workflow. The PR author must review, complete, and finalize this document before the PR can merge.*
