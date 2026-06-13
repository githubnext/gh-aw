---
title: AI Credits Specification
description: Formal W3C-style specification for AI Credits (AIC) calculation, model pricing catalog format, and Copilot billing reference requirements.
sidebar:
  order: 1361
---

# AI Credits Specification

**Version**: 1.3.0  
**Status**: Draft  
**Publication Date**: 2026-06-09  
**Editor**: GitHub Agentic Workflows Team  
**This Version**: [ai-credits-specification](/gh-aw/specs/ai-credits-specification/)  
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
9. [Daily AI Credits Guardrail](#9-daily-ai-credits-guardrail)
10. [Per-Run AI Credits Budget](#10-per-run-ai-credits-budget)
11. [Appendices](#appendices)
12. [References](#references)
13. [Change Log](#change-log)

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

A conforming implementation MUST recognize the following provider name aliases and normalize them to `github-copilot` before catalog lookup:

| Input value | Normalized to |
|-------------|---------------|
| `github` | `github-copilot` |
| `copilot` | `github-copilot` |
| `github_models` | `github-copilot` |

The `github_models` alias is written by the AWF proxy for Copilot engine runs and MUST be recognized so that AIC is computed and emitted for all Copilot-backed engines.

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
- **T-AIC-006**: Verify provider key normalization and `github`, `copilot`, and `github_models` to `github-copilot` mapping behavior.
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

## 9. Daily AI Credits Guardrail

### 9.1 Purpose

The daily AI Credits guardrail is a per-workflow cumulative budget limit evaluated at action runtime. It prevents a single workflow from consuming excessive AI Credits across all runs in a rolling 24-hour window. This section specifies the normative resolution order for the guardrail threshold and the mechanism by which the runtime default is resolved.

### 9.2 Guardrail Configuration Field

The guardrail threshold is expressed through the `max-daily-ai-credits` frontmatter field in the workflow YAML source. When set, a conforming compiler MUST emit the resolved threshold as the `GH_AW_MAX_DAILY_AI_CREDITS` environment variable in the compiled workflow YAML. The daily guardrail check step MUST gate its execution on this variable being non-empty.

### 9.3 Normative Resolution Order

A conforming implementation MUST resolve the effective daily AI Credits threshold using the following precedence (highest to lowest):

1. **Frontmatter value** (`max-daily-ai-credits`): Resolved at compile time. Numeric values MUST be normalized to integer strings; suffix notation (`K`, `M`) MUST be expanded (e.g., `100M` → `100000000`). When present, the resolved value MUST be emitted literally as the `GH_AW_MAX_DAILY_AI_CREDITS` value.

2. **Imported workflow configuration** (`max-daily-ai-credits` from imported shared workflows): Resolved at compile time using a first-wins accumulation across all imported workflows. A conforming implementation MUST apply the first usable `max-daily-ai-credits` value found across imports when no frontmatter value is present on the main workflow. Imported values undergo the same normalization and validation rules as frontmatter values.

3. **Runtime organization variable** (`vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS`): A GitHub Actions `vars.*` expression resolved at action runtime by the GitHub Actions runner. A conforming implementation MUST NOT read this variable at compile time via the process environment; it MUST instead be embedded in the compiled YAML as a GitHub Actions expression evaluated by the runner.

4. **Built-in constant default** (`5000`): The fallback literal embedded in the GitHub Actions expression when the organization variable is unset. The value `5000` AIC represents the normative built-in default for the daily guardrail threshold.

A conforming implementation MUST NOT resolve any of these values from repository-local configuration files (e.g., `aw.json`).

### 9.4 Emitted Expression Form

When neither a frontmatter value nor an imported workflow configuration value is present, a conforming implementation MUST emit the following GitHub Actions expression as the `GH_AW_MAX_DAILY_AI_CREDITS` environment variable value in the compiled workflow:

```
${{ vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS || '5000' }}
```

This expression MUST be emitted verbatim (not pre-evaluated) so that the GitHub Actions runner resolves the organization variable at workflow execution time. The `'5000'` fallback ensures the guardrail is active by default when the organization variable is not configured.

### 9.5 Disable Sentinel

A `max-daily-ai-credits` frontmatter value of `-1` MUST disable the daily guardrail. When this sentinel is present:

- A conforming implementation MUST NOT emit `GH_AW_MAX_DAILY_AI_CREDITS` into the compiled workflow.
- The daily guardrail check step MUST be skipped at runtime (its `if:` condition evaluates to false).

### 9.6 Value Validation

A conforming implementation MUST enforce the following at compile time:

- Accept positive integers and positive numeric strings.
- Accept the suffix notation `K` and `M` (case-insensitive) as multipliers (×1,000 and ×1,000,000 respectively).
- Accept `-1` as the explicit disable sentinel.
- Accept GitHub Actions expression strings (verbatim, passed through for runtime evaluation).
- Reject integer values below `-1` with a compile-time validation error.
- Reject non-numeric, non-expression values.

### 9.7 Compliance Tests

| Test ID | Description | Requirement |
|---------|-------------|-------------|
| T-AIC-DG-001 | Frontmatter value emitted literally; no expression wrapper | §9.3 (1) |
| T-AIC-DG-002 | No frontmatter, no imports: emitted expression is `${{ vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS \|\| '5000' }}` | §9.4 |
| T-AIC-DG-003 | Frontmatter `-1` disables guardrail; no env var emitted | §9.5 |
| T-AIC-DG-004 | `K`/`M` suffix values expanded in emitted literal | §9.6 |
| T-AIC-DG-005 | Values below `-1` rejected at compile time | §9.6 |
| T-AIC-DG-006 | Runtime variable resolved by GitHub Actions runner, not compiler process | §9.3 (3) |
| T-AIC-DG-007 | Imported workflow `max-daily-ai-credits` used when no frontmatter value; frontmatter takes precedence over imports | §9.3 (2) |

---

## 10. Per-Run AI Credits Budget

### 10.1 Purpose

The per-run AI Credits budget is a per-invocation credit limit applied by the AWF firewall to each individual workflow run. It prevents a single run from consuming excessive AI Credits regardless of daily aggregate usage. This section specifies the normative resolution order for the budget threshold and the mechanism by which the runtime default is resolved.

### 10.2 Budget Configuration Field

The budget threshold is expressed through the `max-ai-credits` frontmatter field in the workflow YAML source. When set, a conforming compiler MUST bake the resolved integer threshold directly into the AWF firewall configuration JSON emitted by the compiled workflow. When not set, a conforming implementation MUST resolve the threshold at action runtime using the mechanism specified in §10.4.

### 10.3 Normative Resolution Order

A conforming implementation MUST resolve the effective per-run AI Credits budget using the following precedence (highest to lowest):

1. **Frontmatter value** (`max-ai-credits`): Resolved at compile time. Numeric values MUST be normalized to integers; suffix notation (`K`, `M`) MUST be expanded (e.g., `1M` → `1000000`). When present, the resolved integer MUST be baked into the AWF firewall config JSON at compile time.

2. **Imported workflow configuration** (`max-ai-credits` from imported shared workflows): Resolved at compile time using a first-wins accumulation across all imported workflows. A conforming implementation MUST apply the first usable `max-ai-credits` value found across imports when no frontmatter value is present on the main workflow. Imported values undergo the same normalization rules as frontmatter values.

3. **Runtime organization variable** (`vars.GH_AW_DEFAULT_MAX_AI_CREDITS`): A GitHub Actions `vars.*` expression resolved at action runtime by the GitHub Actions runner. A conforming implementation MUST NOT read this variable at compile time via the process environment; it MUST instead embed a GitHub Actions expression evaluated by the runner at execution time.

4. **Built-in constant default** (`1000`): The fallback literal embedded in the GitHub Actions expression when the organization variable is unset. The value `1000` AIC represents the normative built-in default for the per-run budget.

A conforming implementation MUST NOT resolve any of these values from repository-local configuration files (e.g., `aw.json`).

### 10.4 Runtime Resolution Mechanism

When neither a frontmatter value nor an imported workflow configuration value is present, a conforming implementation MUST emit a runtime patch step that applies the following GitHub Actions expression to the AWF firewall configuration JSON:

```
${{ vars.GH_AW_DEFAULT_MAX_AI_CREDITS || '1000' }}
```

This expression MUST be embedded in the compiled YAML (not pre-evaluated at compile time) so that the GitHub Actions runner resolves the organization variable at workflow execution time. The `'1000'` fallback ensures the budget is active by default when the organization variable is not configured.

### 10.5 Disable Sentinel

A `max-ai-credits` frontmatter value of `-1` MUST disable the per-run budget. When this sentinel is present at runtime (via the organization variable resolving to `-1`), a conforming implementation MUST disable AWF budget steering and omit the `maxAiCredits` field from the firewall configuration.

### 10.6 Value Validation

A conforming implementation MUST enforce the following at compile time when a frontmatter value is present:

- Accept positive integers and positive numeric strings.
- Accept the suffix notation `K` and `M` (case-insensitive) as multipliers (×1,000 and ×1,000,000 respectively).
- Accept `-1` as the explicit disable sentinel.
- Reject integer values below `-1` with a compile-time validation error.
- Reject non-numeric values.

### 10.7 Compliance Tests

| Test ID | Description | Requirement |
|---------|-------------|-------------|
| T-AIC-PR-001 | Frontmatter value baked into AWF config JSON at compile time | §10.3 (1) |
| T-AIC-PR-002 | No frontmatter, no imports: emitted expression is `${{ vars.GH_AW_DEFAULT_MAX_AI_CREDITS \|\| '1000' }}` | §10.4 |
| T-AIC-PR-003 | Org variable `-1` disables budget steering at runtime | §10.5 |
| T-AIC-PR-004 | `K`/`M` suffix values expanded in compile-time literal | §10.6 |
| T-AIC-PR-005 | Values below `-1` rejected at compile time | §10.6 |
| T-AIC-PR-006 | Runtime variable resolved by GitHub Actions runner, not compiler process | §10.3 (3) |
| T-AIC-PR-007 | Imported workflow `max-ai-credits` used when no frontmatter value; frontmatter takes precedence over imports | §10.3 (2) |

---

## Appendices

### Appendix A: Worked Example

This example assumes the provider bundles cache-read tokens in the reported input total, so §3.5 applies.

Given:

- Input (raw): 1000 tokens at $0.000003/token
- Output: 200 tokens at $0.000015/token
- Cache read: 400 tokens at $0.0000003/token
- Cache write: 50 tokens at $0.00000375/token
- Reasoning: 25 tokens at $0.000015/token

Step 1 — Apply §3.5: net input = 1000 − 400 = **600 tokens**

Step 2 — Per-class cost:

```text
input:       600 × 0.000003    = 0.0018000
output:      200 × 0.000015    = 0.0030000
cache_read:  400 × 0.0000003   = 0.0001200
cache_write:  50 × 0.00000375  = 0.0001875
reasoning:    25 × 0.000015    = 0.0003750
```

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

### Version 1.3.0 (2026-06-09)

- **Updated**: §9.3 — Expanded the Daily AI Credits Guardrail resolution order from three levels to four. Added "Imported workflow configuration" as the second priority (between frontmatter and the runtime org variable), documenting that a conforming implementation MUST apply the first usable `max-daily-ai-credits` value found across imported shared workflows when no frontmatter value is present on the main workflow.
- **Updated**: §9.4 — Clarified that the runtime expression is emitted only when neither frontmatter nor imported config provides a value.
- **Updated**: §9.7 — Added compliance test T-AIC-DG-007 asserting imported workflow `max-daily-ai-credits` resolution.
- **Updated**: §10.3 — Expanded the Per-Run AI Credits Budget resolution order from three levels to four. Added "Imported workflow configuration" as the second priority, documenting that a conforming implementation MUST apply the first usable `max-ai-credits` value found across imported shared workflows when no frontmatter value is present.
- **Updated**: §10.4 — Clarified that the runtime patch step is emitted only when neither frontmatter nor imported config provides a value.
- **Updated**: §10.7 — Added compliance test T-AIC-PR-007 asserting imported workflow `max-ai-credits` resolution.
- **Updated**: Version and publication metadata to 1.3.0.

### Version 1.2.0 (2026-06-09)

- **Added**: Section 10 — Per-Run AI Credits Budget. Specifies the normative three-level resolution order (frontmatter → `vars.GH_AW_DEFAULT_MAX_AI_CREDITS` at runtime → built-in constant `1000`) and the required runtime patch mechanism emitted by conforming compilers when no frontmatter value is set.
- **Added**: §10.7 compliance test matrix (T-AIC-PR-001 through T-AIC-PR-006).
- **Updated**: Table of contents and publication metadata to 1.2.0.

### Version 1.1.0 (2026-06-09)

- **Added**: Section 9 — Daily AI Credits Guardrail. Specifies the normative three-level resolution order (frontmatter → `vars.GH_AW_DEFAULT_MAX_DAILY_AI_CREDITS` at runtime → built-in constant `5000`) and the required GitHub Actions expression form emitted by conforming compilers.
- **Added**: §9.7 compliance test matrix (T-AIC-DG-001 through T-AIC-DG-006).
- **Updated**: Table of contents and publication metadata to 1.1.0.

### Version 1.0.0 (Draft)

- Added initial AI Credits (AIC) normative definition and formulas.
- Added canonical `models.json` format and synchronization requirements.
- Added Copilot billing reference requirements and compliance test matrix.

---

Copyright © 2026 GitHub. All rights reserved.
