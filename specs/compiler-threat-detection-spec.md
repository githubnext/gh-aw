---
title: GitHub Actions Compiler Threat Detection Specification
description: Formal W3C-style specification for compiler detection rules that identify and remediate unsafe generated workflow behavior
sidebar:
  order: 1001
---

# GitHub Actions Compiler Threat Detection Specification

**Version**: 1.0.6  
**Status**: Candidate Recommendation  
**Latest Version**: https://github.com/github/gh-aw/blob/main/specs/compiler-threat-detection-spec.md  
**Editors**: GitHub Next (GitHub, Inc.)

---

## Abstract

This specification defines the normative requirements for compiler-side threat detection rules in GitHub Agentic Workflows (gh-aw). The rules detect unsafe or non-compliant patterns in generated GitHub Actions workflows and enforce secure-by-default outcomes before runtime.

This specification is the source of truth for detection rule coverage, implementation obligations, and daily maintenance. Implementations MUST keep compiler behavior and this document synchronized.

## Status of This Document

This is a Candidate Recommendation specification. It may be revised based on operational evidence, threat-model updates, and conformance results.

**Publication Date**: May 15, 2026  
**Governance**: This specification is maintained by the gh-aw maintainers and governed by gh-aw security review processes.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [Threat Detection Rule Model](#3-threat-detection-rule-model)
4. [Normative Rule Requirements](#4-normative-rule-requirements)
5. [Daily Optimizer Maintenance Protocol](#5-daily-optimizer-maintenance-protocol)
6. [Implementation Mapping](#6-implementation-mapping)
7. [Compliance Testing](#7-compliance-testing)
8. [References](#8-references)
9. [Change Log](#9-change-log)

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

## 2. Conformance

An implementation conforms to this specification if it satisfies all MUST requirements in Sections 3-7.

### 2.1 Conformance Targets

- Compiler source in `pkg/workflow/`
- Related schema/validation sources in `pkg/parser/` and `actions/setup/` where applicable
- Daily optimizer workflow that enforces ongoing coverage

### 2.2 Requirement Keywords

The key words **MUST**, **MUST NOT**, **SHALL**, **SHOULD**, and **MAY** are to be interpreted as described in RFC 2119.

---

## 3. Threat Detection Rule Model

Each rule SHALL be represented with:

- **Rule ID** (e.g., `CTR-001`)
- **Threat Class** (permissions, sandbox, network, integrity, output safety)
- **Detection Condition**
- **Compiler Action** (reject, rewrite, warn)
- **Evidence** (error code/message and affected source location)
- **Implementation Mapping** (file/function reference)

Rule definitions SHOULD remain implementation-agnostic while preserving testability.

---

## 4. Normative Rule Requirements

### 4.1 Core Rule Catalog

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

### 4.2 Compiler Response Requirements

For each triggered rule, the compiler MUST:

1. Produce deterministic diagnostics.
2. Prevent insecure generation by failing compilation OR applying a safe rewrite.
3. Emit actionable remediation guidance.
4. Include stable identifiers so tests can assert rule behavior.

### 4.3 Rule Lifecycle Requirements

When a new threat class is identified:

- If implementation already covers the threat, the threat MUST be added to this specification with mapping and tests.
- If implementation does not cover the threat, detection/remediation MUST be implemented and then added to this specification.

#### 4.3.1 Deprecation Policy

When a compiler feature that a `CTR-*` rule depends on is removed, the rule MUST be formally retired:

- The rule's status MUST be updated to `Deprecated` in this specification in the same change set as the implementation removal.
- The rule catalog entry MUST be retained (not deleted) with a deprecation notice indicating the version in which the rule was retired and the reason.
- All test IDs mapped to the deprecated rule in Section 7 MUST be marked as `[DEPRECATED]` and MUST NOT be required for conformance after the deprecation version.
- The implementation mapping in Section 6.1 for the deprecated rule MUST be cleared; the row MUST remain in the table annotated with `[Deprecated in vX.Y.Z]`.
- A change-log entry MUST document the deprecation with the rule ID, deprecation version, and rationale.

---

## 5. Daily Optimizer Maintenance Protocol

A daily optimizer process MUST execute threat coverage reconciliation.

### 5.1 Daily Inputs

The optimizer MUST inspect at least:

- Recent compiler changes (`pkg/workflow/**/*.go`)
- Related validation/security code paths
- Open and recent security findings (issues, PRs, and code scanning context where available)
- Current rule catalog in this specification

### 5.2 Daily Decision Procedure

For each discovered or candidate threat:

1. Determine whether an implemented compiler rule already covers the threat.
2. If covered, update the specification (rule catalog/mapping/tests references).
3. If uncovered, implement detection/remediation in compiler code and tests, then update the specification.

### 5.3 Daily Output Requirements

The optimizer MUST produce one of:

- A pull request containing required spec and/or implementation updates, or
- A noop report explicitly stating no new threat coverage actions were required

---

## 6. Implementation Mapping

This specification maps primarily to:

- `pkg/workflow/` (compiler and validation logic)
- `pkg/parser/` (schema and frontmatter validation where relevant)
- `actions/setup/js/` (runtime validation helpers where required by rule semantics)

Implementations MUST maintain a clear mapping from each active `CTR-*` rule to concrete source locations and test coverage.

### 6.1 Baseline Rule Mapping

| Rule ID | Primary Implementation Areas | Test Coverage Targets |
|---------|-------------------------------|-----------------------|
| CTR-001 Privilege Escalation | `pkg/workflow/*permissions*validation*.go`, `pkg/workflow/strict_mode_permissions_validation.go`, `pkg/workflow/github_app_permissions_validation.go` | `pkg/workflow/*permissions*_test.go`, `pkg/workflow/*dangerous_permissions*_test.go` |
| CTR-002 Unpinned Action Integrity | `pkg/workflow/*action*.go`, `pkg/workflow/strict_mode_validation*.go` | `pkg/workflow/*action*_test.go`, `pkg/workflow/*strict_mode*_test.go` |
| CTR-003 Unsafe Tool Scope Expansion | `pkg/workflow/tools_validation*.go`, `pkg/workflow/strict_mode_validation*.go` | `pkg/workflow/*tools*_test.go` |
| CTR-004 Sandbox Bypass Configuration | `pkg/workflow/sandbox_validation*.go`, `pkg/workflow/strict_mode_sandbox_validation*.go` | `pkg/workflow/*sandbox*_test.go` |
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

The mappings above are pattern-based references and MUST be validated against concrete file paths whenever this specification is updated.

When mappings change, this table MUST be updated in the same change set as the implementation update.

---

## 7. Compliance Testing

A conforming implementation MUST provide tests that validate:

1. Rule detection triggers for malicious or unsafe inputs.
2. Expected compiler action (reject/rewrite/warn) per rule.
3. Stable diagnostics (rule IDs and actionable messages).
4. No regression in secure generation behavior.

Test updates SHOULD be included whenever rules are added or modified.

### 7.1 Test ID Catalog

The following test IDs map one-to-one to the CTR rules in Section 4.1. Each test case MUST exercise the described detection trigger and verify the expected compiler action.

| Test ID | Rule | Detection Trigger | Expected Compiler Action | Stable Diagnostic ID |
|---------|------|-------------------|--------------------------|----------------------|
| **T-CTR-001** | CTR-001 Privilege Escalation | Workflow frontmatter declares `permissions: contents: write` (or another write permission) in a non-safe-outputs job without `strict: false` override | Compilation failure with error identifying the unauthorized write permission and suggesting `safe-outputs` | `CTR-001` |
| **T-CTR-002** | CTR-002 Unpinned Action Integrity | A `jobs.*.steps[].uses` field references an action by tag (e.g., `actions/checkout@v6`) or branch name (`@main`) in strict mode | Compilation failure with error identifying the unpinned reference and providing SHA pinning instructions | `CTR-002` |
| **T-CTR-003** | CTR-003 Unsafe Tool Scope Expansion | Workflow grants wildcard tool permissions (e.g., `tools: bash: ["*"]`) in a context where policy forbids it, or an MCP server is granted broader than declared tool scope | Compilation failure or warning identifying the overbroad scope and suggesting a restricted permission set | `CTR-003` |
| **T-CTR-004** | CTR-004 Sandbox Bypass Configuration | Workflow configuration sets `sandbox: false` or equivalent field that disables required sandboxing | Compilation failure with error identifying the disabled sandbox control and referencing the required configuration | `CTR-004` |
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

### 7.2 Test Coverage Requirements

- Each active CTR rule MUST have at least one test ID in Section 7.1 that covers the primary detection trigger.
- Tests MUST be deterministic: given the same malicious or unsafe input, the compiler MUST always emit the same diagnostic.
- Tests MUST assert the stable diagnostic ID (e.g., `CTR-006`) appears in the compiler error output so that CI can mechanically verify rule coverage.
- When a new rule is added to Section 4.1, at least one new test ID MUST be added to Section 7.1 in the same change set.
- When a rule is deprecated per Section 4.3.1, its test IDs MUST be marked `[DEPRECATED]` and removed from the required compliance gate.

---

## 8. References

- RFC 2119: Key words for use in RFCs to Indicate Requirement Levels
- GitHub Actions syntax and permissions documentation
- gh-aw security architecture and safe outputs specifications

---

## 9. Change Log

### 1.0.6 (2026-05-15)

- Added CTR-016 Compile-Time Manifest Drift (compilation rejection when recompilation of an existing workflow would introduce new restricted secrets or unapproved action references beyond the previously approved lock file manifest baseline; detected by `EnforceSafeUpdate` in `safe_update_enforcement.go`, called from `compiler.go`)
- Added T-CTR-016 test ID entry in Section 7.1
- Extended Section 6.1 baseline rule mapping table with CTR-016 implementation references (`safe_update_enforcement.go`, `compiler.go`)

### 1.0.5 (2026-05-14)

- Added CTR-015 Allowed Label Glob Scope (compilation error when `safe-outputs.*.allowed-labels` contains a bare `"*"` wildcard that effectively disables label restrictions and may permit unintended label-driven automation; triggered by the new glob pattern support for `allowed-labels` introduced in gh-aw #32027)
- Added T-CTR-015 test ID entry in Section 7.1
- Extended Section 6.1 baseline rule mapping table with CTR-015 implementation references (`safe_outputs_allowed_labels_validation.go`)

### 1.0.4 (2026-05-13)

- Added CTR-014 Supply Chain Attack via Install Scripts (warn/reject when `run-install-scripts: true` is configured; protects against malicious npm pre/post install hooks)
- Added T-CTR-014 test ID entry in Section 7.1
- Extended Section 6.1 baseline rule mapping table with CTR-014 implementation references (`run_install_scripts_validation.go`)

### 1.0.3 (2026-05-11)

- Added CTR-013 Argument Injection via Package/Image Names (hyphen-prefix package/image name rejection for npm/npx, pip/uv, and Docker to prevent exec.Command argument injection)
- Added T-CTR-013 test ID entry in Section 7.1
- Extended Section 6.1 baseline rule mapping table with CTR-013 implementation references

### 1.0.2 (2026-05-09)

- Added CTR-012 Safe-Outputs Wildcard Push Scope (unconstrained write scope detection in safe-outputs push-to-pull-request-branch subsystem)
- Extended CTR-001 mapping with `github_app_permissions_validation.go` (GitHub App-only permission scope enforcement)
- Extended CTR-006 mapping with `heredoc_validation.go` (heredoc delimiter injection defense)
- Extended CTR-010 mapping with `expression_syntax_validation.go` (structural expression syntax validation)
- Extended CTR-011 rule description and mapping with `strict_mode_network_validation.go` (wildcard domain rejection in strict mode)
- Updated Section 6.1 baseline rule mapping table for CTR-001, CTR-006, CTR-010, CTR-011, and CTR-012

### 1.0.1 (2026-05-08)

- Extended CTR rule catalog from 5 to 11 rules to reflect existing compiler coverage
- Added CTR-006 Template Injection (template injection detection in shell run: steps)
- Added CTR-007 Markdown Content Security (unicode abuse, hidden content, HTML abuse, social engineering)
- Added CTR-008 Pull Request Target Safety (pwn-request prevention for pull_request_target trigger)
- Added CTR-009 Shell Expansion in Safe-Outputs (dangerous bash expansion detection at compile time)
- Added CTR-010 Expression Safety Allowlist (approved expression enforcement, multi-line rejection)
- Added CTR-011 Network Firewall Configuration (firewall dependency and domain pattern validation)
- Updated Section 6.1 baseline rule mapping table with concrete file references for CTR-006 through CTR-011

### 1.0.0 (2026-05-06)

- Initial W3C-style specification for compiler threat detection rule governance
- Defined daily optimizer reconciliation protocol
- Established baseline `CTR-*` rule catalog and conformance model
