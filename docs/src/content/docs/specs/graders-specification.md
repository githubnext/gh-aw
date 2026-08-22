---
title: Graders Specification
description: Formal specification for deterministic gh-aw graders and grader-backed experiment metrics
sidebar:
  order: 1360
---

# Graders Specification

**Version**: 0.1.0  
**Status**: Draft Specification  
**Feature Status**: Experimental  
**Latest Version**: [graders-specification](/gh-aw/specs/graders-specification/)  
**Editor**: GitHub Agentic Workflows Team

---

## Abstract

This specification defines the `graders` feature in gh-aw: deterministic, post-agent metrics computed from execution traces and persisted as structured artifacts. It specifies configuration, built-in grader behavior, custom inline grader constraints, execution ordering, artifact outputs, experiment metric references, and conformance requirements.

## Status of This Document

This section describes the status of this document at the time of publication. This is a draft specification and may be updated, replaced, or made obsolete by other documents at any time.

This feature is experimental and implementations SHOULD expect iteration before final recommendation status.

## Table of Contents

1. [Introduction](#1-introduction)  
2. [Conformance](#2-conformance)  
3. [Architecture](#3-architecture)  
4. [Configuration Model](#4-configuration-model)  
5. [Built-in Graders](#5-built-in-graders)  
6. [Custom Inline Graders](#6-custom-inline-graders)  
7. [Execution and Artifacts](#7-execution-and-artifacts)  
8. [Experiment Metric References](#8-experiment-metric-references)  
9. [Security and Isolation](#9-security-and-isolation)  
10. [Compliance Testing](#10-compliance-testing)  
11. [Norms](#11-norms)  
12. [References](#12-references)  
13. [Change Log](#13-change-log)

---

## 1. Introduction

### 1.1 Purpose

The `graders` feature provides deterministic quality and behavior metrics derived from workflow trace artifacts without issuing additional LLM calls.

### 1.2 Scope

This specification covers:

- Frontmatter configuration under `graders`
- Built-in grader identifiers and semantics
- Custom inline grader script requirements
- Output artifact contracts
- Experiment metric integration for grader references

This specification does NOT cover:

- Non-deterministic evaluator models
- UI visualization requirements
- External metric backends

### 1.3 Design Goals

A conforming implementation:

1. MUST compute grader values deterministically from run artifacts.
2. MUST preserve stable grader IDs for experiment references.
3. SHOULD keep grading isolated from network-dependent behavior.
4. MUST emit machine-readable grader artifacts for downstream tooling.

---

## 2. Conformance

### 2.1 Conformance Classes

- **Conforming implementation**: Satisfies all MUST/SHALL requirements in this document.
- **Partially conforming implementation**: Supports built-in graders but omits custom inline grader execution.

### 2.2 Requirements Notation

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

### 2.3 Compliance Levels

- **Level 1 (Required)**: Built-in graders, manifest/results output.
- **Level 2 (Standard)**: Custom inline graders with validation and isolation.
- **Level 3 (Complete)**: Experiment metric references to graders with validation.

---

## 3. Architecture

`graders` executes as a post-agent step in the existing `agent` job:

1. Parse and validate frontmatter `graders`.
2. Build grader manifest and execution spec.
3. Preprocess trace artifacts once.
4. Execute enabled graders (built-in and custom inline).
5. Write normalized outputs to grader artifact files.

The grading step MUST run with `if: always()` semantics and SHOULD continue even when individual graders fail, recording per-grader errors in results.

---

## 4. Configuration Model

### 4.1 Frontmatter Key

The configuration key MUST be `graders`.

### 4.2 Enable/Disable Semantics

- If `graders` is omitted, grading MUST be disabled.
- If `graders: {}` is provided, all built-in graders MUST be enabled with defaults.
- If `graders` is present, at least one grader MUST be enabled; otherwise configuration MUST fail.

### 4.3 Entry Model

`graders` is a map of `<grader-id> -> <definition|null>`.

- Built-in grader entries MAY be `null` to enable defaults.
- Custom grader entries MUST be objects and MUST include `script`.

Supported object fields include `enabled`, `name`, `description`, `unit`, `direction`, `threshold`, `min`, `max`, `config`, and `script`.

---

## 5. Built-in Graders

The implementation MUST recognize the following built-in grader IDs:

- `tool-success-rate`
- `tool-failure-count`
- `retries`
- `loops`
- `trajectory-efficiency`
- `execution-step-count`
- `execution-duration`
- `working-set-rebuild-factor`
- `context-growth`
- `artifact-production`

These IDs are reserved for built-ins. A built-in grader MUST NOT accept a custom `script`.

---

## 6. Custom Inline Graders

A custom grader is any grader ID not in the built-in set.

### 6.1 Required Fields

A custom grader MUST define `script`.

### 6.2 Script Limits

- `script` MUST be non-empty.
- `script` MUST NOT exceed 4096 characters.

### 6.3 Forbidden Patterns

Inline scripts MUST be rejected if they contain any forbidden pattern, including:

- `require(`
- `import(`
- `import `
- `fetch(`
- `eval(`
- `process.exit`
- `child_process`
- `execSync`
- `spawnSync`
- `Function(`

---

## 7. Execution and Artifacts

### 7.1 Output Directory

Graders output MUST be written under:

`/tmp/gh-aw/agent/graders`

### 7.2 Required Files

The implementation MUST produce:

- `grader_manifest.json`
- `grader_results.json`

### 7.3 Artifact Inclusion

Both files MUST be included in the unified `agent` artifact.

### 7.4 Deterministic Output Contract

`grader_results.json` SHOULD include normalized run/result structures suitable for downstream programmatic reads, including per-grader value/status and run-level pass/fail/error counts.

---

## 8. Experiment Metric References

Experiment metric fields MAY reference grader outputs.

Supported forms include:

- `grader:<id>`
- `graders.<id>.value`

When a grader reference is used, `<id>` MUST resolve to a declared enabled grader. Unknown or empty grader references MUST fail validation.

---

## 9. Security and Isolation

- Grading MUST operate on local run artifacts and MUST NOT require outbound network access for built-ins.
- Custom inline graders MUST execute in a restricted context with blocked dangerous primitives.
- Implementations SHOULD enforce bounded execution time for inline scripts.
- Implementations SHOULD redact grader outputs when custom scripts are enabled to reduce secret leakage risk.

---

## 10. Compliance Testing

### 10.1 Test Suite Requirements

- **T-GRD-001**: Omitted `graders` key disables grading step emission.
- **T-GRD-002**: `graders: {}` enables all built-ins.
- **T-GRD-003**: Unknown custom grader without `script` is rejected.
- **T-GRD-004**: Custom `script` over 4096 chars is rejected.
- **T-GRD-005**: Forbidden script patterns are rejected.
- **T-GRD-006**: Built-in grader with `script` is rejected.
- **T-GRD-007**: `grader_manifest.json` is written to required path.
- **T-GRD-008**: `grader_results.json` is written to required path.
- **T-GRD-009**: Grader files are present in `agent` artifact.
- **T-GRD-010**: `experiments.*.metric` with `grader:<id>` validates declared enabled grader.
- **T-GRD-011**: `experiments.*.metric` with `graders.<id>.value` validates declared enabled grader.

### 10.2 Compliance Checklist

| Requirement | Test ID | Level | Status |
|---|---|---|---|
| Frontmatter key is `graders` | T-GRD-001 | 1 | Required |
| Empty map enables built-ins | T-GRD-002 | 1 | Required |
| Custom graders require script | T-GRD-003 | 2 | Required |
| Script safety constraints enforced | T-GRD-004, T-GRD-005 | 2 | Required |
| Required artifact files emitted | T-GRD-007, T-GRD-008 | 1 | Required |
| Experiment grader references validate | T-GRD-010, T-GRD-011 | 3 | Required |

---

## 11. Norms

- **N-GRD-001**: Implementations MUST treat `graders` as experimental.
- **N-GRD-002**: Implementations MUST preserve built-in grader ID stability across patch releases.
- **N-GRD-003**: Implementations SHOULD preserve deterministic output for identical trace inputs.
- **N-GRD-004**: Implementations MUST fail fast on invalid custom grader scripts.
- **N-GRD-005**: Implementations MUST keep grader artifact paths stable unless a major version change is issued.

---

## 12. References

### Normative References

- **[RFC 2119]** Key words for use in RFCs to Indicate Requirement Levels.  
  https://www.ietf.org/rfc/rfc2119.txt

### Informative References

- **[Graders Reference]** [Graders](/gh-aw/reference/trace-graders/)
- **[Experiments Specification]** [Experiments Specification](/gh-aw/experimental/experiments-specification/)

---

## 13. Change Log

### Version 0.1.0 (Draft Specification)

- Initial draft for gh-aw graders.
- Defines `graders` configuration semantics and built-in grader set.
- Defines custom inline grader constraints and forbidden patterns.
- Defines grader artifact output contract and experiment metric references.
