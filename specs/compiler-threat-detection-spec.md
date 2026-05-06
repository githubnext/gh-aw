---
title: GitHub Actions Compiler Threat Detection Specification
description: Formal W3C-style specification for compiler detection rules that identify and remediate unsafe generated workflow behavior
sidebar:
  order: 1001
---

# GitHub Actions Compiler Threat Detection Specification

**Version**: 1.0.0  
**Status**: Candidate Recommendation  
**Latest Version**: https://github.com/github/gh-aw/blob/main/specs/compiler-threat-detection-spec.md  
**Editors**: GitHub Next (GitHub, Inc.)

---

## Abstract

This specification defines the normative requirements for compiler-side threat detection rules in GitHub Agentic Workflows (gh-aw). The rules detect unsafe or non-compliant patterns in generated GitHub Actions workflows and enforce secure-by-default outcomes before runtime.

This specification is the source of truth for detection rule coverage, implementation obligations, and daily maintenance. Implementations MUST keep compiler behavior and this document synchronized.

## Status of This Document

This is a Candidate Recommendation specification. It may be revised based on operational evidence, threat-model updates, and conformance results.

**Publication Date**: May 6, 2026  
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
| CTR-001 Privilege Escalation | `pkg/workflow/*permissions*validation*.go`, `pkg/workflow/strict_mode_permissions_validation.go` | `pkg/workflow/*permissions*_test.go`, `pkg/workflow/*dangerous_permissions*_test.go` |
| CTR-002 Unpinned Action Integrity | `pkg/workflow/*action*.go`, `pkg/workflow/strict_mode_validation*.go` | `pkg/workflow/*action*_test.go`, `pkg/workflow/*strict_mode*_test.go` |
| CTR-003 Unsafe Tool Scope Expansion | `pkg/workflow/tools_validation*.go`, `pkg/workflow/strict_mode_validation*.go` | `pkg/workflow/*tools*_test.go` |
| CTR-004 Sandbox Bypass Configuration | `pkg/workflow/sandbox_validation*.go`, `pkg/workflow/strict_mode_sandbox_validation*.go` | `pkg/workflow/*sandbox*_test.go` |
| CTR-005 Unsafe Output Route | `pkg/workflow/compiler_safe_outputs*.go`, `pkg/workflow/safe_outputs*.go` | `pkg/workflow/*safe_outputs*_test.go` |

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

---

## 8. References

- RFC 2119: Key words for use in RFCs to Indicate Requirement Levels
- GitHub Actions syntax and permissions documentation
- gh-aw security architecture and safe outputs specifications

---

## 9. Change Log

### 1.0.0 (2026-05-06)

- Initial W3C-style specification for compiler threat detection rule governance
- Defined daily optimizer reconciliation protocol
- Established baseline `CTR-*` rule catalog and conformance model
