---
title: AI Credit Specification
description: Formal specification for AI Credit (AIC) definition, Copilot pricing alignment, and models.json format.
sidebar:
  order: 1361
---

# AI Credit Specification

**Version**: 1.0.0  
**Status**: Draft  
**Publication Date**: 2026-06-05  
**Editor**: GitHub Agentic Workflows Team  
**This Version**: [ai-credit-specification](/gh-aw/reference/ai-credit-specification/)  
**Latest Published Version**: This document

---

## Abstract

This specification defines AI Credits (AIC) for GitHub Agentic Workflows. It establishes the Copilot-aligned credit unit, defines how total credit cost is computed from token-inference components, and specifies the `models.json` pricing registry format consumed by gh-aw.

---

## Status of This Document

This section describes the status of this document at the time of publication. This is a draft specification and may be updated, replaced, or made obsolete by other documents at any time.

This document is governed by the GitHub Agentic Workflows project specifications process.

---

## 1. Introduction

### 1.1 Purpose

AI Credits provide a provider-aligned, model-aware unit for reporting inference consumption in gh-aw logs, audits, and summaries.

### 1.2 Scope

This specification covers:

- The normative AI Credit unit definition
- Credit computation from token-inference pricing components
- The JSON format for the `models.json` pricing registry

This specification does NOT cover:

- Effective Tokens (ET) normalization logic
- Provider-side invoice generation or payment settlement
- Non-inference billing categories such as GitHub Actions minutes

---

## 2. Conformance

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

---

## 3. AI Credit Definition

### 3.1 Unit Value

One AI Credit is defined as one United States cent:

```
1 AIC = USD 0.01
```

### 3.2 Copilot Pricing and Billing Alignment

A conforming implementation MUST treat AIC as a normalized representation derived from GitHub Copilot pricing and billing documentation:

- [GitHub Copilot model pricing](https://docs.github.com/en/copilot/reference/copilot-billing/models-and-pricing)
- [GitHub Copilot billing](https://docs.github.com/en/copilot/about-github-copilot/subscription-plans-for-github-copilot)

When Copilot pricing updates, implementations SHOULD update `models.json` so AIC output remains aligned with current Copilot pricing semantics.

---

## 4. Credit Computation Model

### 4.1 Inference Cost Components

Total inference cost in USD MUST be the sum of all priced token-inference components reported for the invocation.

At minimum, implementations MUST support these pricing component keys when present:

- `prompt`
- `completion`
- `input_cache_read`
- `input_cache_write`
- `reasoning` and/or `internal_reasoning`

### 4.2 Total Credit Cost

For a single invocation:

```
cost_usd =
  (effective_input_tokens × prompt_price) +
  (output_tokens × completion_price) +
  (cache_read_tokens × input_cache_read_price) +
  (cache_write_tokens × input_cache_write_price) +
  (reasoning_tokens × reasoning_price)
```

Where:

- `effective_input_tokens` is provider-adjusted input token count
- each `*_price` is read from the model's `pricing` object in `models.json`

For multi-invocation execution, total credit cost MUST be the arithmetic sum of each invocation's `cost_usd`.

### 4.3 USD to AI Credits

AI Credits MUST be computed from USD using:

```
aic = cost_usd / 0.01
```

Equivalent form:

```
aic = cost_usd × 100
```

---

## 5. `models.json` Format Specification

### 5.1 Location

This specification applies to pricing catalogs stored as `models.json`, including:

- `pkg/cli/data/models.json`
- `actions/setup/js/models.json`

### 5.2 Top-Level Shape

The document MUST be a JSON object containing a `data` array:

```json
{
  "data": [
    {
      "id": "anthropic/claude-sonnet-4.6",
      "pricing": {
        "prompt": "0.000003",
        "completion": "0.000015",
        "input_cache_read": "0.0000003",
        "input_cache_write": "0.00000375",
        "internal_reasoning": "0.000015"
      }
    }
  ]
}
```

### 5.3 Entry Requirements

Each `data[]` entry MUST follow these rules:

- `id` MUST be a non-empty string in `provider/model` form.
- `pricing` MUST be a JSON object.
- `pricing` values MUST be decimal numbers encoded as either JSON numbers or numeric strings.

### 5.4 Pricing Keys

Implementations MUST recognize these keys when present:

- `prompt`
- `completion`
- `input_cache_read`
- `input_cache_write`
- `reasoning`
- `internal_reasoning`

Additional pricing keys MAY be present (for example, `web_search`, `audio`) and MUST NOT invalidate the file.

### 5.5 Parsing Behavior

A conforming parser:

- MUST ignore entries with invalid or missing `id`
- MUST parse numeric strings into finite decimal values
- MUST treat invalid numeric values as zero or absent
- SHOULD preserve unknown pricing keys for forward compatibility

---

## 6. References

- [Effective Tokens Specification](/gh-aw/reference/effective-tokens-specification/)
- [Cost Management](/gh-aw/reference/cost-management/)
- [GitHub Copilot subscription plans and billing](https://docs.github.com/en/copilot/about-github-copilot/subscription-plans-for-github-copilot)

---

## 7. Change Log

- **1.0.0 (2026-06-05)**: Initial draft defining AI Credit semantics and `models.json` format.
