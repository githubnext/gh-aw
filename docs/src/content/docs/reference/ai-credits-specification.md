---
title: AI Credits Specification
description: Formal W3C-style specification for AI Credits (AIC) calculation, model pricing catalog format, and Copilot billing reference requirements.
sidebar:
  order: 1361
---

# AI Credits Specification

**Version**: 1.0.0  
**Status**: Draft  
**Publication Date**: 2026-06-05  
**Editor**: GitHub Agentic Workflows Team  
**This Version**: [ai-credits-specification](/gh-aw/reference/ai-credits-specification/)  
**Latest Published Version**: This document

---

## Abstract

This specification defines AI Credits (AIC) as the normative inference-cost metric for GitHub Agentic Workflows (gh-aw). It specifies the required calculation model from token usage and provider pricing, the canonical `models.json` catalog format used to store per-model pricing inputs, and the required external references for GitHub Copilot model and billing alignment.

## Status of This Document

This section describes the status of this document at the time of publication. This is a draft specification and may be updated, replaced, or made obsolete by other documents at any time.

This document is governed by the GitHub Agentic Workflows project specifications process.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [AI Credits Accounting Model](#3-ai-credits-accounting-model)
4. [Pricing Catalog Format (`models.json`)](#4-pricing-catalog-format-modelsjson)
5. [Catalog Provisioning and Synchronization](#5-catalog-provisioning-and-synchronization)
6. [Copilot Billing Reference Requirements](#6-copilot-billing-reference-requirements)
7. [Reporting Requirements](#7-reporting-requirements)
8. [Compliance Testing](#8-compliance-testing)
9. [Appendices](#appendices)
10. [References](#references)
11. [Change Log](#change-log)

---

## 1. Introduction

### 1.1 Purpose

AIC provides a single monetary-normalized metric for inference cost across supported model providers. This specification defines how conforming implementations compute AIC, how pricing data is represented, and how Copilot-specific pricing alignment is governed.

### 1.2 Scope

This specification covers:

- The normative AIC unit definition and conversion rules.
- The per-invocation and aggregated AIC calculation formulas.
- Required `models.json` data structure and field semantics.
- Requirements for how pricing catalog data is provided and mirrored in gh-aw.
- Required references to GitHub Copilot model and billing documentation.

This specification does NOT cover:

- GitHub Actions minutes billing.
- ET (Effective Tokens) normalization rules.
- Provider-side billing reconciliation and invoice dispute procedures.

### 1.3 Design Goals

The specification is designed to:

1. Provide a testable and deterministic calculation contract.
2. Keep pricing inputs explicit and auditable through structured catalog files.
3. Support model-name drift through normalized lookup and prefix fallback matching.
4. Maintain compatibility between CLI and setup runtime pricing catalogs.

---

## 2. Conformance

### 2.1 Conformance Classes

**Conforming implementation**: Satisfies all MUST/SHALL requirements in Sections 3 through 8.

**Partially conforming implementation**: Computes core AIC from token usage and model pricing (Section 3) but omits one or more optional reporting or synchronization requirements.

### 2.2 Requirements Notation

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

### 2.3 Compliance Levels

- **Level 1 – Calculation**: Implements AIC formulas and provider token handling (Section 3).
- **Level 2 – Catalog**: Implements required `models.json` structure and lookup semantics (Section 4).
- **Level 3 – Operational**: Implements catalog provisioning, Copilot reference alignment, and reporting requirements (Sections 5–7).

---

## 3. AI Credits Accounting Model

### 3.1 Unit Definition

A conforming implementation MUST define:

- `1 AIC = 0.01 USD`
- `AIC = USD / 0.01`

### 3.2 Token Classes

Implementations MUST support the following token classes for cost calculation when available:

- Input tokens
- Output tokens
- Cache read tokens
- Cache write tokens
- Reasoning tokens

### 3.3 Per-Invocation Cost Formula

For a single invocation, implementations MUST compute USD as:

```text
cost_usd =
  (input_tokens × input_price_per_token) +
  (output_tokens × output_price_per_token) +
  (cache_read_tokens × cache_read_price_per_token) +
  (cache_write_tokens × cache_write_price_per_token) +
  (reasoning_tokens × reasoning_price_per_token)
```

A conforming implementation MUST derive AIC as:

```text
aic = cost_usd / 0.01
```

### 3.4 Price Fallback Rules

If a model entry omits optional price fields, implementations MUST apply the following fallback behavior:

- `cache_read_price_per_token` defaults to `input_price_per_token`
- `cache_write_price_per_token` defaults to `input_price_per_token`
- `reasoning_price_per_token` defaults to `output_price_per_token`

### 3.5 Provider-Specific Input Handling

For providers that include cache-read tokens in total input tokens, implementations MUST subtract `cache_read_tokens` from `input_tokens` before applying input price and MUST NOT double-charge cache-read usage.

### 3.6 Aggregation

For grouped runs (for example, episodes), implementations MUST aggregate AIC by summing per-invocation AIC values.

---

## 4. Pricing Catalog Format (`models.json`)

### 4.1 Top-Level Structure

A conforming catalog MUST be valid JSON with this structure:

```json
{
  "providers": {
    "provider-name": {
      "models": {
        "model-id": {
          "cost": {
            "input": "...",
            "output": "...",
            "cache_read": "...",
            "cache_write": "...",
            "reasoning": "..."
          }
        }
      }
    }
  }
}
```

### 4.2 Required and Optional Fields

For each model:

- `cost.input` MUST be present.
- `cost.output` MUST be present.
- `cost.cache_read` MAY be present.
- `cost.cache_write` MAY be present.
- `cost.reasoning` MAY be present.

Cost values MUST be decimal numbers encoded as strings and interpreted as USD per token.

### 4.3 Provider Keys

Provider keys MUST be lowercase identifiers. For Copilot-backed pricing, the canonical provider key MUST be `github-copilot`.

### 4.4 Model Lookup Normalization

A conforming implementation MUST normalize provider and model identifiers for lookup by trimming whitespace and applying case-insensitive comparison. An implementation SHOULD support compatibility matching between punctuation variants (for example, `.` and `_` compared to `-`) and provider-scoped prefix fallback.

---

## 5. Catalog Provisioning and Synchronization

### 5.1 Embedded Runtime Catalogs

gh-aw implementations MUST provide synchronized pricing catalogs at:

- `pkg/cli/data/models.json`
- `actions/setup/js/models.json`

These files SHALL represent the same pricing dataset.

### 5.2 Source and Refresh Expectations

Catalog refresh processes SHOULD use normalized upstream model inventories and SHOULD validate Copilot entries against authoritative GitHub Copilot model and billing documentation.

### 5.3 Change Control

Catalog updates MUST preserve JSON validity and MUST maintain backward-safe handling for historical model IDs that remain in the catalog but are absent from current live inventories.

---

## 6. Copilot Billing Reference Requirements

Implementations and documentation that describe Copilot AIC behavior MUST reference:

- GitHub Copilot models documentation: <https://docs.github.com/en/copilot/concepts/about-github-copilot-models>
- GitHub Copilot models and pricing reference: <https://docs.github.com/en/copilot/reference/copilot-billing/models-and-pricing>
- GitHub Copilot plan and billing documentation: <https://docs.github.com/en/copilot/about-github-copilot/subscription-plans-for-github-copilot>

These references SHOULD be treated as the external billing-alignment sources for Copilot model pricing validation.

---

## 7. Reporting Requirements

A conforming implementation MUST expose AIC in runtime reporting outputs where cost metrics are emitted.

Implementations SHOULD provide:

- Per-run AIC values.
- Aggregated AIC values for grouped executions.
- Structured output fields suitable for machine parsing.

---

## 8. Compliance Testing

### 8.1 Test Suite Requirements

A conformance test suite MUST include at least the following test cases:

- **T-AIC-001**: Verify `1 AIC = 0.01 USD` conversion.
- **T-AIC-002**: Verify per-invocation AIC computation using all token classes.
- **T-AIC-003**: Verify cache-read subtraction behavior for providers that include cache-read tokens in input totals.
- **T-AIC-004**: Verify fallback pricing when optional cost fields are omitted.
- **T-AIC-005**: Verify `models.json` rejects missing required fields (`input`, `output`).
- **T-AIC-006**: Verify provider key normalization and `github` to `github-copilot` mapping behavior.
- **T-AIC-007**: Verify catalog mirror consistency between CLI and setup runtime paths.
- **T-AIC-008**: Verify reporting outputs include per-run AIC values.

### 8.2 Compliance Checklist

| Requirement | Test ID | Level | Status |
|-------------|---------|-------|--------|
| Unit conversion (`1 AIC = 0.01 USD`) | T-AIC-001 | 1 | Required |
| Full token-class formula | T-AIC-002 | 1 | Required |
| Cache-read non-double-charge behavior | T-AIC-003 | 1 | Required |
| Optional price fallback behavior | T-AIC-004 | 1 | Required |
| Catalog schema conformance | T-AIC-005 | 2 | Required |
| Provider/model normalization behavior | T-AIC-006 | 2 | Required |
| Mirrored catalog consistency | T-AIC-007 | 3 | Required |
| AIC reporting visibility | T-AIC-008 | 3 | Required |

---

## Appendices

### Appendix A: Worked Example

Given:

- Input: 1000 at $0.000003/token
- Output: 200 at $0.000015/token
- Cache read: 400 at $0.0000003/token
- Cache write: 50 at $0.00000375/token
- Reasoning: 25 at $0.000015/token

Result:

```text
cost_usd = 0.0054825
aic = 0.54825
```

### Appendix B: Error Conditions

Conforming implementations SHOULD surface explicit validation errors for:

- Invalid `models.json` structure.
- Non-numeric cost values.
- Missing required model cost fields.
- Unknown provider/model pairs with no fallback match.

### Appendix C: Security and Integrity Considerations

Pricing catalogs are configuration inputs. Implementations SHOULD:

- Treat catalog updates as controlled changes.
- Validate and review catalog source provenance.
- Avoid silently mutating cost values at runtime.

---

## References

### Normative References

- **[RFC 2119]** Key words for use in RFCs to Indicate Requirement Levels. <https://www.ietf.org/rfc/rfc2119.txt>

### Informative References

- **[GH-AW-COST]** Cost Management reference. <https://github.github.com/gh-aw/reference/cost-management/>
- **[GH-COPILOT-MODELS]** About GitHub Copilot models. <https://docs.github.com/en/copilot/concepts/about-github-copilot-models>
- **[GH-COPILOT-BILLING-MODELS]** GitHub Copilot models and pricing. <https://docs.github.com/en/copilot/reference/copilot-billing/models-and-pricing>
- **[GH-COPILOT-BILLING-PLANS]** Subscription plans for GitHub Copilot. <https://docs.github.com/en/copilot/about-github-copilot/subscription-plans-for-github-copilot>
- **[MODELS-DEV]** models.dev API index. <https://models.dev/api.json>

---

## Change Log

### Version 1.0.0 (Draft)

- Added initial AI Credits (AIC) normative definition and formulas.
- Added canonical `models.json` format and synchronization requirements.
- Added Copilot billing reference requirements and compliance test matrix.

---

Copyright © 2026 GitHub. All rights reserved.
