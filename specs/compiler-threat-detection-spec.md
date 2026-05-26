---
title: GitHub Actions Compiler Threat Detection Specification
description: Formal W3C-style specification for compiler detection rules that identify and remediate unsafe generated workflow behavior
sidebar:
  order: 1001
---

# GitHub Actions Compiler Threat Detection Specification

**Version**: 1.0.12  
**Status**: Candidate Recommendation  
**Latest Version**: https://github.com/github/gh-aw/blob/main/specs/compiler-threat-detection-spec.md  
**Editors**: GitHub Next (GitHub, Inc.)

---

## Abstract

This specification defines the normative requirements for compiler-side threat detection rules in GitHub Agentic Workflows (gh-aw). The rules detect unsafe or non-compliant patterns in generated GitHub Actions workflows and enforce secure-by-default outcomes before runtime.

This specification is the source of truth for detection rule coverage, implementation obligations, and daily maintenance. Implementations MUST keep compiler behavior and this document synchronized.

## Status of This Document

This is a Candidate Recommendation specification. It may be revised based on operational evidence, threat-model updates, and conformance results.

**Publication Date**: May 26, 2026  
**Governance**: This specification is maintained by the gh-aw maintainers and governed by gh-aw security review processes.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Spec-to-Implementation Sync](#2-spec-to-implementation-sync)
3. [Conformance](#3-conformance)
4. [Threat Detection Rule Model](#4-threat-detection-rule-model)
5. [Normative Rule Requirements](#5-normative-rule-requirements)
6. [Daily Optimizer Maintenance Protocol](#6-daily-optimizer-maintenance-protocol)
7. [Implementation Mapping](#7-implementation-mapping)
8. [Compliance Testing](#8-compliance-testing)
9. [References](#9-references)
10. [Change Log](#10-change-log)

---

## 1. Introduction

### 1.1 Purpose

This specification defines how compiler detection rules are authored, implemented, and maintained to prevent unsafe generated workflow behavior.

### 1.2 Scope

This specification covers:

- Rule definitions for generated-code security threats
- Compiler obligations for detection and remediation
- Daily optimizer behavior for threat coverage review
- Rule-to-implementation mapping and conformance expectations

This specification does NOT cover:

- Runtime threat detection job internals
- External scanner rule ecosystems
- Non-compiler repositories

### 1.3 Design Principles

1. **Specification-first**: Rules MUST be defined in this specification.
2. **Security by default**: Unsafe generated behavior MUST be blocked or remediated.
3. **Bidirectional sync**: Implemented rules MUST appear in spec, and specified rules MUST map to implementation.
4. **Auditable evolution**: Rule additions and changes MUST be traceable.

---

## 2. Spec-to-Implementation Sync

This section anchors the specification version to the minimum gh-aw binary version expected to implement it and to the lock-file behavior that must remain compatible.

| Spec version | Minimum gh-aw binary version | Lock-file compatibility notes |
|--------------|------------------------------|-------------------------------|
| `1.0.12` | `v0.72.1` (or newer) | Threat-detection behavior must remain compatible with current `.lock.yml` compilation semantics, including manifest drift enforcement (`gh-aw-manifest` checks for CTR-016), update-check validation (`check-for-updates` handling for CTR-018), and cache-memory integrity enforcement (`update_cache_memory` gating for CTR-019). |
| `1.0.11` | `v0.72.1` (or newer) | Threat-detection behavior must remain compatible with current `.lock.yml` compilation semantics, including manifest drift enforcement (`gh-aw-manifest` checks for CTR-016), update-check validation (`check-for-updates` handling for CTR-018), and cache-memory integrity enforcement (`update_cache_memory` gating for CTR-019). |
| `1.0.10` | `v0.72.1` (or newer) | Threat-detection behavior must remain compatible with current `.lock.yml` compilation semantics, including manifest drift enforcement (`gh-aw-manifest` checks for CTR-016), update-check validation (`check-for-updates` handling for CTR-018), and cache-memory integrity enforcement (`update_cache_memory` gating for CTR-019). |
| `1.0.9` | `v0.72.1` (or newer) | Threat-detection behavior must remain compatible with current `.lock.yml` compilation semantics, including manifest drift enforcement (`gh-aw-manifest` checks for CTR-016) and update-check validation (`check-for-updates` handling for CTR-018). Top-level `sandbox: false` is no longer a valid workflow input; `sandbox.agent: false` is the supported field for CTR-004 detection. |
| `1.0.8` | `v0.72.1` (or newer) | Threat-detection behavior must remain compatible with current `.lock.yml` compilation semantics, including manifest drift enforcement (`gh-aw-manifest` checks for CTR-016) and update-check validation (`check-for-updates` handling for CTR-018). |

When this specification version changes, maintainers MUST update this table in the same pull request as any lock-file compatibility changes.

---

## 3. Conformance

An implementation conforms to this specification if it satisfies all MUST requirements in Sections 4-8.

### 3.1 Conformance Targets

- Compiler source in `pkg/workflow/`
- Related schema/validation sources in `pkg/parser/` and `actions/setup/` where applicable
- Daily optimizer workflow that enforces ongoing coverage

### 3.2 Requirement Keywords

The key words **MUST**, **MUST NOT**, **SHALL**, **SHOULD**, and **MAY** are to be interpreted as described in RFC 2119.

---

## 4. Threat Detection Rule Model

Each rule SHALL be represented with:

- **Rule ID** (e.g., `CTR-001`)
- **Threat Class** (permissions, sandbox, network, integrity, output safety)
- **Detection Condition**
- **Compiler Action** (reject, rewrite, warn)
- **Evidence** (error code/message and affected source location)
- **Implementation Mapping** (file/function reference)

Rule definitions SHOULD remain implementation-agnostic while preserving testability.

---

## 5. Normative Rule Requirements

### 5.1 Core Rule Catalog

A conforming implementation MUST include detection coverage for at least the following rules:

- **CTR-001 Privilege Escalation**: Detect generated jobs with unauthorized write permissions.
- **CTR-002 Unpinned Action Integrity**: Detect unpinned or weakly pinned action references in strict contexts.
- **CTR-003 Unsafe Tool Scope Expansion**: Detect wildcard or overbroad tool permissions that violate policy.
- **CTR-004 Sandbox Bypass Configuration**: Detect generated configurations that disable required sandboxing.
- **CTR-005 Unsafe Output Route**: Detect direct unsafe write paths that bypass safe-output controls.
- **CTR-006 Template Injection**: Detect GitHub Actions expressions used directly in `run:` shell commands where user-controlled data flows into shell execution context without environment variable indirection.
- **CTR-007 Markdown Content Security**: Detect dangerous or malicious content patterns in externally-sourced markdown workflow files, including unicode abuse, hidden content, obfuscated links, HTML abuse, embedded scripts, and social engineering.
- **CTR-008 Pull Request Target Safety**: Detect unsafe use of the `pull_request_target` trigger, which runs workflows with write permissions and secret access; enforce checkout restrictions to prevent pwn-request attacks.
- **CTR-009 Shell Expansion in Safe-Outputs**: Detect dangerous bash expansion patterns (`${var@op}`, `${!var}`, `$(...)`, backtick substitution) in safe-outputs `run:` scripts that would be blocked by the safe-outputs security harness at runtime.
- **CTR-010 Expression Safety Allowlist**: Enforce an allowlist of approved GitHub Actions expressions; reject unauthorized or multi-line expressions that could enable injection or exfiltration.
- **CTR-011 Network Firewall Configuration**: Validate network firewall configuration dependencies and domain patterns; reject configurations that declare firewall rules without required prerequisites (e.g., `allow-urls` without `ssl-bump`); reject wildcard `*` domains in strict mode.
- **CTR-012 Safe-Outputs Wildcard Push Scope**: Detect misconfiguration patterns when `safe-outputs.push-to-pull-request-branch: target: "*"` is used; warn when no wildcard fetch pattern is present in checkout (suppressed for public repos) and when no access constraints (`title-prefix` or `labels`) are configured.
- **CTR-013 Argument Injection via Package/Image Names**: Detect package or container image names that start with `-` (hyphen) in npm/npx, pip/uv, and Docker frontmatter configurations; reject these names before they are passed to `exec.Command` calls where they would be interpreted as CLI flags, enabling argument injection.
- **CTR-014 Supply Chain Attack via Install Scripts**: Detect when `run-install-scripts: true` is configured in workflow frontmatter (globally or per-runtime); warn in non-strict mode and reject in strict mode to protect against malicious npm pre/post install hooks that can exfiltrate secrets or corrupt the runner environment.
- **CTR-015 Allowed Label Glob Scope**: Detect bare `*` wildcard patterns in `safe-outputs.*.allowed-labels` fields (`create-issue`, `create-discussion`, `update-discussion`, `create-pull-request`, `merge-pull-request`); reject compilation when such a pattern is present because it renders the label restriction ineffective and may allow the agent to apply labels that trigger unintended label-driven automation in the repository.
- **CTR-016 Compile-Time Manifest Drift**: Detect when recompilation of an existing workflow would introduce new secrets or unapproved action references beyond what was previously approved in the lock file manifest; reject compilation when new restricted secrets or previously-absent action references appear, preventing adversarial workflow sources or prompt-injection from silently expanding the workflow's trust surface during routine updates.
- **CTR-017 Secret Leakage via Environment Variables**: Detect secrets expressions (`${{ secrets.* }}`) in the top-level `env:` section, in `engine.env` (excluding allowed engine-required vars), and in custom step fields (`pre-steps`, `steps`, `pre-agent-steps`, `post-steps`) outside controlled `env:` bindings and `with:` inputs for `uses:` action steps; these placements expose secrets to the agent container environment. Warn in non-strict mode; reject in strict mode.
- **CTR-018 Version Integrity Bypass**: Detect `check-for-updates: false` in workflow frontmatter, which disables the compile-agentic version update check that ensures the workflow was compiled with a supported version of gh-aw. Warn in non-strict mode; reject in strict mode.
- **CTR-019 Cache-Memory Integrity Enforcement**: Enforce that `update_cache_memory` job only runs when threat detection succeeds (not when skipped or failed); prevents cache pollution when agent outputs have not been validated or when detection was bypassed, ensuring cache-memory data integrity and preventing persistence of potentially malicious content from unvalidated agent sessions.

### 5.2 Compiler Response Requirements

For each triggered rule, the compiler MUST:

1. Produce deterministic diagnostics.
2. Prevent insecure generation by failing compilation OR applying a safe rewrite.
3. Emit actionable remediation guidance.
4. Include stable identifiers so tests can assert rule behavior.

### 5.3 Rule Lifecycle Requirements

When a new threat class is identified:

- If implementation already covers the threat, the threat MUST be added to this specification with mapping and tests.
- If implementation does not cover the threat, detection/remediation MUST be implemented and then added to this specification.

#### 4.3.1 Deprecation Policy

When a compiler feature that a `CTR-*` rule depends on is removed, the rule MUST be formally retired:

- The rule's status MUST be updated to `Deprecated` in this specification in the same change set as the implementation removal.
- The rule catalog entry MUST be retained (not deleted) with a deprecation notice indicating the version in which the rule was retired and the reason.
- All test IDs mapped to the deprecated rule in Section 7 MUST be marked as `[DEPRECATED]` and MUST NOT be required for conformance after the deprecation version.
- The implementation mapping in Section 7.1 for the deprecated rule MUST be cleared; the row MUST remain in the table annotated with `[Deprecated in vX.Y.Z]`.
- A change-log entry MUST document the deprecation with the rule ID, deprecation version, and rationale.

---

## 6. Daily Optimizer Maintenance Protocol

A daily optimizer process MUST execute threat coverage reconciliation.

### 6.1 Daily Inputs

The optimizer MUST inspect at least:

- Recent compiler changes (`pkg/workflow/**/*.go`)
- Related validation/security code paths
- Open and recent security findings (issues, PRs, and code scanning context where available)
- Current rule catalog in this specification

### 6.2 Daily Decision Procedure

For each discovered or candidate threat:

1. Determine whether an implemented compiler rule already covers the threat.
2. If covered, update the specification (rule catalog/mapping/tests references).
3. If uncovered, implement detection/remediation in compiler code and tests, then update the specification.

### 6.3 Daily Output Requirements

The optimizer MUST produce one of:

- A pull request containing required spec and/or implementation updates, or
- A noop report explicitly stating no new threat coverage actions were required

### 6.4 False-Positive Handling

False positives occur when a CTR rule triggers on a workflow input that is not actually unsafe. This section defines normative norms for suppressing, auditing, and resolving false-positive detections.

1. **Author suppression mechanism**: When a workflow author believes a compiler diagnostic is a false positive, they **MUST** add an inline suppression annotation in the workflow frontmatter using the `threat-detection-suppress` key. The value **MUST** be a list of objects, each with a `rule` field (the `CTR-*` identifier), a `reason` field (human-readable explanation of why the flagged pattern is safe in this context), and an optional `expires` field (ISO 8601 date after which the suppression is no longer valid). A suppression without a `reason` **MUST NOT** be accepted by the compiler; the compiler **MUST** emit a validation error if `reason` is absent or empty.

2. **Audit trail requirement**: Every active suppression annotation **MUST** be recorded in the compiled lock file (`.lock.yml`) manifest section so that reviewers can audit which rules are suppressed and why. The lock file **MUST** include the full `rule`, `reason`, and `expires` values for each suppression. Suppressions absent from the lock file manifest **MUST** be treated by subsequent compilations as unapproved and re-evaluated against the current CTR rule.

3. **SLA for resolution**: Suppressions marked as false positives that affect a `MUST`-level security control (as defined in Section 5.1 — specifically those rules whose compiler action is `reject` in non-strict mode) **SHOULD** be resolved within **10 business days** — either by confirming the suppression is correct and updating the rule's detection logic to eliminate the false positive, or by removing the suppression when the workflow is corrected.
4. **SLA enforcement in daily optimizer output**: The daily optimizer **MUST** compute suppression age from suppression creation date (or first-observed date when unavailable). Suppressions older than 10 business days **MUST** be emitted in daily output as `SLA_BREACH` entries that include `rule`, `reason`, `age_business_days`, `owner`, and `expires`.
5. **Escalation requirement**: Suppressions older than 20 business days for `MUST`-level controls **MUST** create a follow-up sync action in the same daily output (for example, PR task, issue task, or explicit policy exception record) so unresolved suppressions cannot remain silent.
6. **Expiration handling**: A suppression **MUST** be re-evaluated and explicitly renewed if the `expires` date passes; expired suppressions **MUST** be treated by the compiler as if they do not exist.

### 6.5 Threat Category Lifecycle

New threat categories do not immediately become normative rules. This section defines the lifecycle stages a threat category **MUST** pass through before it is added to the CTR rule catalog in Section 5.1.

1. **Experimental stage**: A threat class is identified (via security research, incident analysis, or operational observation) and a tracking issue is opened in `github/gh-aw`. An experimental prototype detection implementation **MAY** be added to the compiler behind a feature flag. The threat class **MUST NOT** appear in the normative CTR catalog while in Experimental stage; it **SHOULD** be documented in a separate scratchpad or issue thread. Experimental detections **MUST NOT** cause compilation failures in production.

2. **Candidate stage**: The threat class has a concrete detection trigger, an agreed compiler action (reject, rewrite, or warn), a stable diagnostic ID reserved in a draft spec update, and at least one test case demonstrating the detection. A Candidate threat **SHOULD** be deployed behind a feature flag for a minimum of one release cycle. During Candidate stage, maintainers **MUST** collect evidence (false-positive reports, affected workflow patterns) and document findings in the tracking issue. A Candidate threat **SHOULD NOT** be promoted to Normative without at least one successful deployment in a non-strict production workflow.

3. **Normative stage**: The threat class is formally added to Section 5.1 and Section 8.1 via a pull request that includes: the CTR rule definition, the implementation mapping in Section 7.1, at least one test ID in Section 8.1, and a change-log entry in Section 10. The pull request **MUST** be reviewed by at least one security-focused maintainer. Once merged, the rule **MUST** be enforced by all conforming implementations. Any feature flag used during Candidate stage **MUST** be removed in the same pull request that adds the Normative definition.

---

## 7. Implementation Mapping

This specification maps primarily to:

- `pkg/workflow/` (compiler and validation logic)
- `pkg/parser/` (schema and frontmatter validation where relevant)
- `actions/setup/js/` (runtime validation helpers where required by rule semantics)

Implementations MUST maintain a clear mapping from each active `CTR-*` rule to concrete source locations and test coverage.

### 7.1 Baseline Rule Mapping

| Rule ID | Primary Implementation Areas | Test Coverage Targets |
|---------|-------------------------------|-----------------------|
| CTR-001 Privilege Escalation | `pkg/workflow/*permissions*validation*.go`, `pkg/workflow/strict_mode_permissions_validation.go`, `pkg/workflow/github_app_permissions_validation.go` | `pkg/workflow/*permissions*_test.go`, `pkg/workflow/*dangerous_permissions*_test.go` |
| CTR-002 Unpinned Action Integrity | `pkg/workflow/*action*.go`, `pkg/workflow/strict_mode_validation*.go` | `pkg/workflow/*action*_test.go`, `pkg/workflow/*strict_mode*_test.go` |
| CTR-003 Unsafe Tool Scope Expansion | `pkg/workflow/tools_validation*.go`, `pkg/workflow/strict_mode_validation*.go` | `pkg/workflow/*tools*_test.go` |
| CTR-004 Sandbox Bypass Configuration | `pkg/workflow/sandbox_validation*.go`, `pkg/workflow/strict_mode_sandbox_validation*.go`, `pkg/workflow/strict_mode_permissions_validation.go` | `pkg/workflow/*sandbox*_test.go` |
| CTR-005 Unsafe Output Route | `pkg/workflow/compiler_safe_outputs*.go`, `pkg/workflow/safe_outputs*.go` | `pkg/workflow/*safe_outputs*_test.go` |
| CTR-006 Template Injection | `pkg/workflow/template_injection_validation.go`, `pkg/workflow/heredoc_validation.go` | `pkg/workflow/template_injection_validation_test.go`, `pkg/workflow/template_injection_validation_fuzz_test.go` |
| CTR-007 Markdown Content Security | `pkg/workflow/markdown_security_scanner.go` | `pkg/workflow/markdown_security_scanner_test.go`, `pkg/workflow/secure_markdown_rendering_test.go` |
| CTR-008 Pull Request Target Safety | `pkg/workflow/pull_request_target_validation.go` | `pkg/workflow/pull_request_target_validation_test.go` |
| CTR-009 Shell Expansion in Safe-Outputs | `pkg/workflow/safe_outputs_steps_shell_expansion_validation.go` | `pkg/workflow/safe_outputs_steps_shell_expansion_validation_test.go` |
| CTR-010 Expression Safety Allowlist | `pkg/workflow/expression_safety_validation.go`, `pkg/workflow/expression_syntax_validation.go` | `pkg/workflow/expression_extraction_test.go` |
| CTR-011 Network Firewall Configuration | `pkg/workflow/network_firewall_validation.go`, `pkg/workflow/firewall_validation.go`, `pkg/workflow/strict_mode_network_validation.go` | `pkg/workflow/network_firewall_validation_test.go` |
| CTR-012 Safe-Outputs Wildcard Push Scope | `pkg/workflow/push_to_pull_request_branch_validation.go` | `pkg/workflow/push_to_pull_request_branch_test.go`, `pkg/workflow/push_to_pull_request_branch_warning_test.go` |
| CTR-013 Argument Injection via Package/Image Names | `pkg/workflow/name_validation.go` (shared helper `rejectHyphenPrefixPackages`), `pkg/workflow/npm_validation.go`, `pkg/workflow/pip_validation.go`, `pkg/workflow/docker_validation.go` | `pkg/workflow/argument_injection_test.go` |
| CTR-014 Supply Chain Attack via Install Scripts | `pkg/workflow/run_install_scripts_validation.go` (`validateRunInstallScripts`, `resolveRunInstallScripts`) | `pkg/workflow/run_install_scripts_validation_test.go` |
| CTR-015 Allowed Label Glob Scope | `pkg/workflow/safe_outputs_allowed_labels_validation.go` (`validateSafeOutputsAllowedLabelsGlobScope`) | `pkg/workflow/safe_outputs_allowed_labels_validation_test.go` |
| CTR-016 Compile-Time Manifest Drift | `pkg/workflow/safe_update_enforcement.go` (`EnforceSafeUpdate`, `collectSecretViolations`, `collectActionViolations`, `collectRedirectViolations`), called from `pkg/workflow/compiler.go` | `pkg/workflow/safe_update_enforcement_test.go` |
| CTR-017 Secret Leakage via Environment Variables | `pkg/workflow/strict_mode_env_validation.go` (`validateEnvSecrets`, `validateEnvSecretsSection`), `pkg/workflow/strict_mode_steps_validation.go` (`validateStepsSecrets`, `validateStepsSectionSecrets`) | `pkg/workflow/env_secrets_validation_test.go`, `pkg/workflow/jobs_secrets_validation_test.go` |
| CTR-018 Version Integrity Bypass | `pkg/workflow/strict_mode_update_check_validation.go` (`validateUpdateCheck`) | `pkg/workflow/strict_mode_update_check_validation_test.go` |
| CTR-019 Cache-Memory Integrity Enforcement | `pkg/workflow/cache.go` (`buildUpdateCacheMemoryJob` using `buildDetectionSuccessCondition`), `pkg/workflow/expression_builder.go` (`buildDetectionSuccessCondition`) | `pkg/workflow/cache_memory_threat_detection_test.go`, `pkg/workflow/threat_detection_job_combinations_integration_test.go` |

The mappings above are pattern-based references and MUST be validated against concrete file paths whenever this specification is updated.

When mappings change, this table MUST be updated in the same change set as the implementation update.

### 7.2 Mapping Audit (2026-05-26)

Audit result: ✅ all listed `CTR-001` through `CTR-019` rows currently include non-empty implementation references and non-empty test coverage targets; no `TODO` placeholders were found in the mapping table. Review window: commits merged through 2026-05-26 (PR #34841 / commit 34e5154). Three security-related items were evaluated: (1) `pkg/workflow/heredoc_validation.go` (heredoc delimiter injection defense, `patch-fix-heredoc-delimiter-injection`) — already mapped under CTR-006 since version 1.0.2; no change required. (2) MCP server actor validation (`--validate-actor` runtime flag, `patch-add-mcp-actor-validation`) — runtime RBAC enforced by the MCP gateway container; not a compiler threat detection rule and requires no new CTR entry. (3) Cross-repository allowlist validation (`allowed-repos`/`target-repo` field hardening for SEC-005, `patch-cross-repo-allowlist-validation`) — compiler-side configuration parsing hardening for safe-output handlers; strengthens existing CTR-005 and CTR-012 enforcement boundaries but introduces no new threat class requiring a distinct CTR rule. No new uncovered threats were identified in this review cycle.

### 7.3 Sync Protocol for CTR Rule and Manifest Updates

When adding, removing, or materially changing any `CTR-*` rule, the same pull request **MUST** update all synchronized artifacts:

1. Section 5.1 rule catalog entry.
2. Section 7.1 implementation mapping row.
3. Section 8.1 test ID coverage entry.
4. Lock file manifest schema and compiler emission logic for any new or changed suppression/manifest fields tied to the rule.

If a CTR rule change lands without a matching lock file manifest update, CI policy **MUST** fail the change as out-of-sync. Recompilation of affected workflows **MUST** occur in the same change set when manifest shape changes.

---

## 8. Compliance Testing

A conforming implementation MUST provide tests that validate:

1. Rule detection triggers for malicious or unsafe inputs.
2. Expected compiler action (reject/rewrite/warn) per rule.
3. Stable diagnostics (rule IDs and actionable messages).
4. No regression in secure generation behavior.

Test updates SHOULD be included whenever rules are added or modified.

### 8.1 Test ID Catalog

The following test IDs map one-to-one to the CTR rules in Section 5.1. Each test case MUST exercise the described detection trigger and verify the expected compiler action.

| Test ID | Rule | Detection Trigger | Expected Compiler Action | Stable Diagnostic ID |
|---------|------|-------------------|--------------------------|----------------------|
| **T-CTR-001** | CTR-001 Privilege Escalation | Workflow frontmatter declares `permissions: contents: write` (or another write permission) in a non-safe-outputs job without `strict: false` override | Compilation failure with error identifying the unauthorized write permission and suggesting `safe-outputs` | `CTR-001` |
| **T-CTR-002** | CTR-002 Unpinned Action Integrity | A `jobs.*.steps[].uses` field references an action by tag (e.g., `actions/checkout@v6`) or branch name (`@main`) in strict mode | Compilation failure with error identifying the unpinned reference and providing SHA pinning instructions | `CTR-002` |
| **T-CTR-003** | CTR-003 Unsafe Tool Scope Expansion | Workflow grants wildcard tool permissions (e.g., `tools: bash: ["*"]`) in a context where policy forbids it, or an MCP server is granted broader than declared tool scope | Compilation failure or warning identifying the overbroad scope and suggesting a restricted permission set | `CTR-003` |
| **T-CTR-004** | CTR-004 Sandbox Bypass Configuration | Workflow configuration sets `sandbox.agent: false` in strict mode, disabling the agent sandbox firewall | Compilation failure with error identifying the disabled sandbox control and referencing the required configuration; note that the formerly supported top-level `sandbox: false` field is removed and now triggers a schema validation error rather than CTR-004 | `CTR-004` |
| **T-CTR-005** | CTR-005 Unsafe Output Route | Workflow uses a direct write path (e.g., `contents: write` with inline shell commands) that bypasses the safe-outputs subsystem | Compilation failure with error identifying the unsafe write route and requiring use of `safe-outputs` | `CTR-005` |
| **T-CTR-006** | CTR-006 Template Injection | A `run:` step embeds a GitHub Actions expression (`${{ github.event.issue.title }}`) directly in the shell command string without environment variable indirection | Compilation failure with error identifying the injected expression, the affected step, and providing the env-var indirection pattern | `CTR-006` |
| **T-CTR-007** | CTR-007 Markdown Content Security | An externally-sourced markdown workflow file contains a known dangerous pattern (e.g., unicode abuse, embedded HTML script tag, obfuscated link) | Compilation failure or error identifying the detected dangerous pattern, its location in the file, and recommending sanitization | `CTR-007` |
| **T-CTR-008** | CTR-008 Pull Request Target Safety | Workflow declares `on: pull_request_target` and a `checkout` step that references the PR head (`ref: ${{ github.event.pull_request.head.sha }}`) without an explicit fork-safety guard | Compilation failure with error identifying the unsafe checkout pattern, the pwn-request risk, and safe alternatives | `CTR-008` |
| **T-CTR-009** | CTR-009 Shell Expansion in Safe-Outputs | A `safe-outputs` `run:` step contains a dangerous bash expansion (e.g., `${var@Q}`, `${!var}`, `` `cmd` ``, `$(cmd)`) that the safe-outputs security harness would block at runtime | Compilation failure or error identifying the dangerous expansion pattern, the affected step, and safe alternatives | `CTR-009` |
| **T-CTR-010** | CTR-010 Expression Safety Allowlist | A workflow prompt or step uses a GitHub Actions expression not on the approved allowlist (e.g., `${{ github.event.comment.body }}`) or a multi-line expression that could enable exfiltration | Compilation failure with error identifying the disallowed expression, its location, and the approved allowlist | `CTR-010` |
| **T-CTR-011** | CTR-011 Network Firewall Configuration | Workflow declares `network: allowed: [some-domain]` with `ssl-bump: false` (or omits `ssl-bump` when required), or uses a wildcard `*` domain in strict mode | Compilation failure with error identifying the missing prerequisite or disallowed wildcard domain and providing the corrective configuration | `CTR-011` |
| **T-CTR-012** | CTR-012 Safe-Outputs Wildcard Push Scope | Workflow uses `safe-outputs.push-to-pull-request-branch: target: "*"` without a wildcard fetch pattern in checkout (for non-public repos) or without `title-prefix` or `labels` access constraints | Compilation warning identifying the unconstrained wildcard scope and the missing checkout fetch pattern or access constraint; suppressed for public repositories | `CTR-012` |
| **T-CTR-013** | CTR-013 Argument Injection via Package/Image Names | A workflow frontmatter declares an npm/npx package, a pip/uv package, or a Docker container image name that starts with `-` (e.g., `--privileged`, `-exploit`) | Compilation failure with error identifying the invalid name, the affected tool kind, and instructing the user to fix the package or image name | `CTR-013` |
| **T-CTR-014** | CTR-014 Supply Chain Attack via Install Scripts | A workflow frontmatter sets `run-install-scripts: true` (globally or under `runtimes.node`) | Compilation warning in non-strict mode identifying the supply chain risk and advising removal of `run-install-scripts: true`; compilation failure in strict mode | `CTR-014` |
| **T-CTR-015** | CTR-015 Allowed Label Glob Scope | A workflow frontmatter sets `safe-outputs.*.allowed-labels` to `["*"]` (bare wildcard) for any safe-output type that supports the field (`create-issue`, `create-discussion`, `update-discussion`, `create-pull-request`, `merge-pull-request`) | Compilation failure with error identifying the field name, explaining that `"*"` disables label restrictions and may permit unintended label-driven automation, and recommending specific names or narrower patterns | `CTR-015` |
| **T-CTR-016** | CTR-016 Compile-Time Manifest Drift | An existing workflow lock file has a `gh-aw-manifest` section recording approved secrets and action references; when recompiled, the new workflow body introduces a secret not in the approved manifest (e.g., `MY_NEW_SECRET`) or a new action reference not previously recorded | Compilation failure with error identifying each new restricted secret and each added or removed action reference beyond the previously approved manifest baseline, preventing silent trust-surface expansion | `CTR-016` |
| **T-CTR-017** | CTR-017 Secret Leakage via Environment Variables | A workflow frontmatter declares a secrets expression (e.g., `${{ secrets.MY_SECRET }}`) in the top-level `env:` section, in `engine.env` for a non-engine var, or in a custom step's `run:` field | Compilation warning in non-strict mode identifying the secrets expression and the section where it appears; compilation failure in strict mode | `CTR-017` |
| **T-CTR-018** | CTR-018 Version Integrity Bypass | A workflow frontmatter sets `check-for-updates: false` | Compilation warning in non-strict mode identifying the disabled version check and advising removal; compilation failure in strict mode | `CTR-018` |
| **T-CTR-019** | CTR-019 Cache-Memory Integrity Enforcement | A workflow has `cache-memory` enabled with `threat-detection: true`; the compiled lock file's `update_cache_memory` job condition must require `needs.detection.result == 'success'` (not accept `skipped`) | Compilation emits `update_cache_memory` job with condition `always() && needs.detection.result == 'success' && needs.agent.result == 'success'`; verified by integration tests | `CTR-019` |

### 8.2 Test Coverage Requirements

- Each active CTR rule MUST have at least one test ID in Section 8.1 that covers the primary detection trigger.
- Tests MUST be deterministic: given the same malicious or unsafe input, the compiler MUST always emit the same diagnostic.
- Tests MUST assert the stable diagnostic ID (e.g., `CTR-006`) appears in the compiler error output so that CI can mechanically verify rule coverage.
- When a new rule is added to Section 5.1, at least one new test ID MUST be added to Section 8.1 in the same change set.
- When a rule is deprecated per Section 5.3.1, its test IDs MUST be marked `[DEPRECATED]` and removed from the required compliance gate.

---

## 9. References

- RFC 2119: Key words for use in RFCs to Indicate Requirement Levels
- GitHub Actions syntax and permissions documentation
- gh-aw security architecture and safe outputs specifications

---

## 10. Change Log

### 1.0.12 (2026-05-26)

- Updated Section 7.2 mapping audit to 2026-05-26 confirming no new uncovered threats in this review cycle
- Evaluated three security items from PR #34841: heredoc delimiter injection defense (already covered by CTR-006), MCP actor validation runtime flag (not a compiler detection rule), and cross-repo allowlist validation for SEC-005 (strengthens CTR-005/CTR-012 boundaries; no new CTR rule required)
- Updated Section 2 spec-to-implementation sync table with version 1.0.12 entry

### 1.0.11 (2026-05-25)

- Corrected CTR-018 implementation mapping: `pkg/workflow/update_check_validation.go` → `pkg/workflow/strict_mode_update_check_validation.go` (the spec referenced a non-existent filename; the actual implementation and test file are `strict_mode_update_check_validation.go` and `strict_mode_update_check_validation_test.go`)
- Updated Section 7.2 mapping audit to 2026-05-25 noting the CTR-018 filename correction and confirming no new uncovered threats in this review cycle
- Updated Section 2 spec-to-implementation sync table with version 1.0.11 entry

### 1.0.10 (2026-05-22)

- Added CTR-019 Cache-Memory Integrity Enforcement (enforces that `update_cache_memory` job only runs when threat detection succeeds, not when skipped or failed; prevents cache pollution from unvalidated agent outputs; implemented in `cache.go` via `buildDetectionSuccessCondition` instead of `buildDetectionPassedCondition`)
- Added T-CTR-019 test ID entry in Section 8.1
- Extended Section 7.1 baseline rule mapping table with CTR-019 implementation references (`cache.go`, `expression_builder.go`)
- Updated Section 2 spec-to-implementation sync table to include CTR-019 in lock-file compatibility notes for version 1.0.10
- Updated Section 7.2 mapping audit timestamp to 2026-05-22 and notes to reflect CTR-019 addition

### 1.0.9 (2026-05-17)

- Updated T-CTR-004 detection trigger from deprecated `sandbox: false` (removed field) to `sandbox.agent: false` in strict mode; noted that the old top-level `sandbox: false` now triggers a schema validation error rather than CTR-004
- Extended CTR-004 implementation mapping with `strict_mode_permissions_validation.go`, which is the concrete enforcement site for `sandbox.agent: false` rejection in strict mode
- Updated Section 7.2 mapping audit timestamp and notes to reflect the CTR-004 mapping correction

### 1.0.8 (2026-05-16)

- Added CTR-018 Version Integrity Bypass (warn/reject when `check-for-updates: false` disables the compile-agentic version update check; implemented in `update_check_validation.go`)
- Added T-CTR-018 test ID entry in Section 8.1
- Extended Section 7.1 baseline rule mapping table with CTR-018 implementation references (`update_check_validation.go`)

### 1.0.7 (2026-05-16)

- Added CTR-017 Secret Leakage via Environment Variables (warn/reject when secrets expressions appear in top-level `env:`, `engine.env`, or in uncontrolled custom step fields; implemented in `strict_mode_env_validation.go` and `strict_mode_steps_validation.go`)
- Added T-CTR-017 test ID entry in Section 8.1
- Extended Section 7.1 baseline rule mapping table with CTR-017 implementation references (`strict_mode_env_validation.go`, `strict_mode_steps_validation.go`)
- Updated mapping audit note to cover CTR-001 through CTR-018

### 1.0.6 (2026-05-15)

- Added CTR-016 Compile-Time Manifest Drift (compilation rejection when recompilation of an existing workflow would introduce new restricted secrets or unapproved action references beyond the previously approved lock file manifest baseline; detected by `EnforceSafeUpdate` in `safe_update_enforcement.go`, called from `compiler.go`)
- Added T-CTR-016 test ID entry in Section 8.1
- Extended Section 7.1 baseline rule mapping table with CTR-016 implementation references (`safe_update_enforcement.go`, `compiler.go`)

### 1.0.5 (2026-05-14)

- Added CTR-015 Allowed Label Glob Scope (compilation error when `safe-outputs.*.allowed-labels` contains a bare `"*"` wildcard that effectively disables label restrictions and may permit unintended label-driven automation; triggered by the new glob pattern support for `allowed-labels` introduced in gh-aw #32027)
- Added T-CTR-015 test ID entry in Section 8.1
- Extended Section 7.1 baseline rule mapping table with CTR-015 implementation references (`safe_outputs_allowed_labels_validation.go`)

### 1.0.4 (2026-05-13)

- Added CTR-014 Supply Chain Attack via Install Scripts (warn/reject when `run-install-scripts: true` is configured; protects against malicious npm pre/post install hooks)
- Added T-CTR-014 test ID entry in Section 8.1
- Extended Section 7.1 baseline rule mapping table with CTR-014 implementation references (`run_install_scripts_validation.go`)

### 1.0.3 (2026-05-11)

- Added CTR-013 Argument Injection via Package/Image Names (hyphen-prefix package/image name rejection for npm/npx, pip/uv, and Docker to prevent exec.Command argument injection)
- Added T-CTR-013 test ID entry in Section 8.1
- Extended Section 7.1 baseline rule mapping table with CTR-013 implementation references

### 1.0.2 (2026-05-09)

- Added CTR-012 Safe-Outputs Wildcard Push Scope (unconstrained write scope detection in safe-outputs push-to-pull-request-branch subsystem)
- Extended CTR-001 mapping with `github_app_permissions_validation.go` (GitHub App-only permission scope enforcement)
- Extended CTR-006 mapping with `heredoc_validation.go` (heredoc delimiter injection defense)
- Extended CTR-010 mapping with `expression_syntax_validation.go` (structural expression syntax validation)
- Extended CTR-011 rule description and mapping with `strict_mode_network_validation.go` (wildcard domain rejection in strict mode)
- Updated Section 7.1 baseline rule mapping table for CTR-001, CTR-006, CTR-010, CTR-011, and CTR-012

### 1.0.1 (2026-05-08)

- Extended CTR rule catalog from 5 to 11 rules to reflect existing compiler coverage
- Added CTR-006 Template Injection (template injection detection in shell run: steps)
- Added CTR-007 Markdown Content Security (unicode abuse, hidden content, HTML abuse, social engineering)
- Added CTR-008 Pull Request Target Safety (pwn-request prevention for pull_request_target trigger)
- Added CTR-009 Shell Expansion in Safe-Outputs (dangerous bash expansion detection at compile time)
- Added CTR-010 Expression Safety Allowlist (approved expression enforcement, multi-line rejection)
- Added CTR-011 Network Firewall Configuration (firewall dependency and domain pattern validation)
- Updated Section 7.1 baseline rule mapping table with concrete file references for CTR-006 through CTR-011

### 1.0.0 (2026-05-06)

- Initial W3C-style specification for compiler threat detection rule governance
- Defined daily optimizer reconciliation protocol
- Established baseline `CTR-*` rule catalog and conformance model
