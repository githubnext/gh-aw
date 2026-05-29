---
title: AWF Config Canonical Sources Specification
description: Canonical AWF configuration specification and schema sources that gh-aw agents MUST consult
sidebar:
  order: 1002
---

# AWF Config Canonical Sources Specification

**Version**: 0.1.0  
**Status**: Working Draft  
**Date**: 2026-05-10  
**Last Updated**: 2026-05-10  
**Editors**: GitHub gh-aw Team

---

## 1. Purpose

This document defines the canonical AWF configuration references in `github/gh-aw-firewall` that gh-aw agents and schema reconciliation workflows MUST use when generating or validating AWF config behavior.

## 2. Canonical sources (gh-aw-firewall)

The following documents are authoritative and MUST be consulted together:

### 2.1 Normative specification

- `docs/awf-config-spec.md` — processing model, precedence, CLI mapping, env merge semantics, credential isolation

### 2.2 JSON schemas

- `docs/awf-config.schema.json` — published schema for `.awf.json` / `.awf.yml`
- `src/awf-config-schema.json` — runtime schema source used by AWF CLI
- `schemas/audit.schema.json` — schema for firewall audit output
- `schemas/token-usage.schema.json` — schema for token usage output

### 2.3 Supporting docs

- `docs/environment.md` — environment variable configuration behavior
- `docs/authentication-architecture.md` — credential isolation architecture
- `schemas/README.md` — schema directory overview

## 3. Required coverage checks

When updating AWF config generation, schema sync, or validation in gh-aw, agents MUST verify:

1. Every relevant property in `docs/awf-config.schema.json` is represented in gh-aw logic.
2. CLI mapping behavior in `docs/awf-config-spec.md` is reconciled with schema-defined properties.
3. Config-only fields (without CLI flags) are still modeled where required by runtime behavior.

## 4. Known drift example (apiProxy)

The following fields previously existed in schema but were missed in spec CLI mapping checks:

| Config path | CLI flag |
|---|---|
| `apiProxy.anthropicAutoCache` | `--anthropic-auto-cache` |
| `apiProxy.anthropicCacheTailTtl` | `--anthropic-cache-tail-ttl` |
| `apiProxy.models` | config-only (model alias rewriting) |
| `apiProxy.modelMultipliers` | config-only (effective-token accounting) |
| `apiProxy.modelFallback` | config-only (model fallback policy; set `sandbox.agent.model-fallback: false` to prevent deployment-name rewriting for BYOK Azure) |
| `apiProxy.maxRuns` | config-only (LLM invocation hard cap) |
| `apiProxy.auth.*` | config-only (maps to `AWF_AUTH_*` env vars) |
| `apiProxy.targets.openai.authHeader` | `--openai-api-auth-header` (frontmatter: `sandbox.agent.targets.openai.authHeader`) |
| `apiProxy.targets.anthropic.authHeader` | `--anthropic-api-auth-header` (frontmatter: `sandbox.agent.targets.anthropic.authHeader`) |
| `container.dockerHostPathPrefix` | `--docker-host-path-prefix` |

Agents SHOULD treat this class of mismatch as a regression signal and open a corrective PR when detected.

---

## 3. Conformance Requirements

The key words **MUST**, **MUST NOT**, **SHOULD**, and **MAY** in this section are to be interpreted as described in [RFC 2119](https://www.rfc-editor.org/rfc/rfc2119).

**CR-01**: Agents and schema reconciliation workflows MUST consult **both** the normative specification (`docs/awf-config-spec.md`) and the published JSON schema (`docs/awf-config.schema.json`) before generating or validating AWF config behavior. Consulting only one source is insufficient.

**CR-02**: When a property exists in the JSON schema but has no corresponding entry in the normative spec CLI mapping table, agents MUST treat this as a drift condition and flag it for corrective action.

**CR-03**: Agents MUST NOT generate AWF config fields that are absent from both the normative spec and all JSON schemas. Undocumented fields are out of scope and may be silently ignored or rejected by the AWF CLI.

**CR-04**: Schema reconciliation workflows SHOULD verify coverage of all top-level properties in `docs/awf-config.schema.json` against the CLI mapping table in `docs/awf-config-spec.md` on every run.

**CR-05**: When drift is detected, the detecting agent or workflow SHOULD open a corrective pull request with specific field paths and suggested remediation.

**CR-06**: Drift categorized as "missing in gh-aw" or "spec mismatch" MUST be remediated (merged or explicitly waived with rationale) within **5 business days** of detection. For this requirement, business days are Monday-Friday in UTC, excluding weekends. If this SLA is missed, maintainers MUST open (or update) an escalation tracking issue within 1 business day. The escalation issue MUST include an owner, unblock plan, and revised ETA.

**CR-06a (Escalation Owner Assignment)**: When opening or updating an escalation tracking issue under CR-06, the assignee **SHOULD** be determined as follows: (a) the maintainer who merged the last change to the drifted property's corresponding implementation file in `pkg/workflow/` or `actions/setup/` is the **default escalation owner** (implementation guidance: this can be determined via `git log` on the relevant file, or through PR merge history); (b) if no such maintainer is identifiable (e.g., the property has never been implemented), the escalation owner **SHOULD** default to the on-call maintainer for the `github/gh-aw` repository at the time of escalation; (c) the assigned owner **MUST** be recorded in the `Owner` field of the escalation issue template and **MUST** acknowledge the assignment by commenting on the issue within 1 business day of assignment. The escalation issue **MUST NOT** be left unassigned.

---

## 4. Drift Detection Procedure

This section describes the concrete steps for detecting schema drift between `gh-aw-firewall` and `gh-aw`.

### 4.1 When to Run

Drift detection MUST be triggered when:

1. A pull request modifies `docs/awf-config.schema.json`, `src/awf-config-schema.json`, or `docs/awf-config-spec.md` in `github/gh-aw-firewall`.
2. A scheduled workflow runs the reconciliation check (RECOMMENDED: daily or weekly).
3. An agent is asked to generate or validate AWF config behavior.

### 4.2 Step-by-Step Procedure

1. **Fetch the canonical sources** from `github/gh-aw-firewall`:
   - `docs/awf-config.schema.json` — published schema
   - `src/awf-config-schema.json` — runtime schema
   - `docs/awf-config-spec.md` — normative specification

2. **Extract the property inventory** from both schema files:
   - List all top-level and nested property keys.
   - Note which properties have corresponding CLI flags (as documented in `docs/awf-config-spec.md`).
   - Note which properties are config-only (no CLI flag).

3. **Compare against gh-aw implementation**:
   - For each schema property, check whether `pkg/workflow/` or `actions/setup/` in `github/gh-aw` references it.
   - For each CLI-mapped property, check whether the CLI flag is tested in `pkg/workflow/` tests.

4. **Identify drift categories**:
   - **Missing in gh-aw**: Property exists in schema but `gh-aw` has no coverage.
   - **Missing in schema**: `gh-aw` generates a field not present in either schema.
   - **Spec mismatch**: CLI mapping in `gh-aw` disagrees with the normative spec description.

5. **Produce a drift report** listing:
   - Each drifted property path (e.g., `apiProxy.anthropicAutoCache`).
   - Drift category (missing in gh-aw / missing in schema / spec mismatch).
   - Suggested corrective action (add coverage, open PR, update spec).

6. **Open a corrective PR** when any drift of category "missing in gh-aw" or "spec mismatch" is found. The PR description MUST include the drift report and reference this procedure.

### 4.3 Example Drift Check (CLI)

```bash
# Requires GH_TOKEN (or GITHUB_TOKEN) with repo read access
: "${GH_TOKEN:?Set GH_TOKEN (or map GITHUB_TOKEN to GH_TOKEN) before running this check}"

# Fetch both schema files from gh-aw-firewall
gh api /repos/github/gh-aw-firewall/contents/docs/awf-config.schema.json \
  --jq '.content' | base64 -d > /tmp/published-schema.json

gh api /repos/github/gh-aw-firewall/contents/src/awf-config-schema.json \
  --jq '.content' | base64 -d > /tmp/runtime-schema.json

# Extract nested schema property paths
jq -r '
  def walk_props(prefix):
    (.properties // {} | to_entries[]) as $p
    | ($p.key) as $k
    | ((if prefix == "" then $k else prefix + "." + $k end)),
      ($p.value | walk_props(if prefix == "" then $k else prefix + "." + $k end));
  walk_props("")
' /tmp/published-schema.json | sort -u > /tmp/schema-keys.txt

# Compare against awf-config references in gh-aw implementation
rg --no-heading --no-filename --only-matching 'apiProxy\.[A-Za-z0-9_.]+' pkg/workflow actions/setup \
  | sort -u > /tmp/ghaw-refs.txt

# Review diff for drift
# Keep command non-fatal so investigators can review drift output before deciding whether to fail the run.
diff -u /tmp/schema-keys.txt /tmp/ghaw-refs.txt || true
```

### 4.4 Automation

A scheduled GitHub Actions workflow in `github/gh-aw` SHOULD automate this procedure. The workflow SHOULD:

- Run on a weekly schedule and on pull requests that touch AWF config handling.
- Fail the check (non-zero exit) when any "missing in gh-aw" drift is found.
- Post a summary comment on PRs with the drift report.
- Create a tracking issue when drift is detected on the scheduled run.

Current implementation reference: [`/.github/workflows/schema-consistency-checker.md`](../.github/workflows/schema-consistency-checker.md) (scheduled daily) is the tracked drift-detection workflow path for schema consistency checks and SHOULD include AWF config source drift checks from this section.

#### 4.4.1 Drift SLA tracking (CR-06)

To satisfy CR-06 tracking obligations, drift escalation records SHOULD use:

- **Label(s)**: `workflow` + `bug` (both exist in `github/gh-aw`)
- **Escalation issue title prefix**: `[Schema Drift SLA]`
- **Escalation template** (minimum required fields):

```markdown
## Schema Drift SLA Escalation

- Drift detected on: <YYYY-MM-DD>
- Source workflow run: <run-url>
- Owner: <github-handle>
- Unblock plan:
  1. ...
  2. ...
- Revised ETA (UTC): <YYYY-MM-DD>
- Waiver rationale (if any): <text>
```

The scheduled schema consistency workflow SHOULD open or update one such issue when drift remains unresolved beyond 5 business days.

### 4.5 DriftRecord Entity Schema

A `DriftRecord` represents a single detected schema drift item produced by the drift detection procedure (Section 4.2, Step 5). All automation and agents that produce or consume drift reports **MUST** use this schema for structured drift output.

#### 4.5.1 Formal Schema (JSON Schema)

```json
{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "DriftRecord",
  "description": "A single detected configuration drift item between gh-aw-firewall canonical sources and gh-aw implementation.",
  "type": "object",
  "required": ["property_path", "drift_category", "suggested_action", "detected_at"],
  "properties": {
    "property_path": {
      "type": "string",
      "description": "Dot-notation path to the drifted configuration property (e.g., 'apiProxy.anthropicAutoCache').",
      "examples": ["apiProxy.anthropicAutoCache", "container.dockerHostPathPrefix"]
    },
    "drift_category": {
      "type": "string",
      "enum": ["missing_in_ghaw", "missing_in_schema", "spec_mismatch"],
      "description": "Classification of the drift condition. 'missing_in_ghaw': property exists in canonical schema but gh-aw has no coverage. 'missing_in_schema': gh-aw generates a field not present in either schema. 'spec_mismatch': CLI mapping in gh-aw disagrees with the normative spec description."
    },
    "suggested_action": {
      "type": "string",
      "description": "Human-readable remediation recommendation for this drift item (e.g., 'Add coverage for apiProxy.anthropicAutoCache in pkg/workflow/ and reconcile with docs/awf-config-spec.md CLI mapping table').",
      "minLength": 1
    },
    "detected_at": {
      "type": "string",
      "format": "date-time",
      "description": "ISO 8601 timestamp (UTC) when this drift item was first detected in the current run."
    }
  },
  "additionalProperties": false
}
```

#### 4.5.2 Field Reference

| Property | Type | Required | Description |
|----------|------|----------|-------------|
| `property_path` | `string` | **MUST** | Dot-notation config property path (e.g., `apiProxy.anthropicAutoCache`) |
| `drift_category` | `enum` | **MUST** | One of `missing_in_ghaw`, `missing_in_schema`, or `spec_mismatch` (see Section 4.2, Step 4) |
| `suggested_action` | `string` | **MUST** | Actionable remediation text; **MUST NOT** be empty |
| `detected_at` | `string` (ISO 8601) | **MUST** | UTC timestamp of detection; filesystem-safe format **SHOULD** use `YYYY-MM-DDTHH:MM:SSZ` |

#### 4.5.3 Usage

The drift detection procedure (Section 4.2, Step 5) **MUST** produce a list of zero or more `DriftRecord` objects. When any record has `drift_category` of `missing_in_ghaw` or `spec_mismatch`, the detecting automation **MUST** open a corrective PR (CR-05) and, if the SLA window is exceeded, an escalation issue (CR-06). The corrective PR description **MUST** embed the full `DriftRecord` list as JSON.

**Example output (Step 5 of the drift detection procedure):**

```json
[
  {
    "property_path": "apiProxy.anthropicAutoCache",
    "drift_category": "missing_in_ghaw",
    "suggested_action": "Add coverage for apiProxy.anthropicAutoCache in pkg/workflow/ and reconcile CLI mapping in docs/awf-config-spec.md.",
    "detected_at": "2026-05-17T16:00:00Z"
  }
]
```

## 5. Safeguards

When canonical sources in `github/gh-aw-firewall` are unavailable (GitHub outage, auth failure, transient fetch errors), agents and automation MUST apply the following safeguards:

1. The workflow **MUST** attempt to use the last-known validated local snapshot (for example cached schema/spec artifacts from the previous successful run) to keep checks deterministic.
2. The workflow **SHOULD** emit a warning that canonical source retrieval failed, including the failing source path(s) and timestamp.
3. The workflow **MUST** skip destructive validation actions (for example failing required checks, auto-opening corrective PRs, or auto-creating drift issues from stale snapshots) when canonical data cannot be refreshed, and mark the run as degraded instead of silently passing.
4. The workflow **SHOULD** open or update a tracking issue when canonical source unavailability persists for more than one consecutive scheduled run.
