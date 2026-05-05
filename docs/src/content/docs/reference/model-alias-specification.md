---
title: Model Alias Format Specification
description: Formal W3C-style specification defining the model alias format, naming conventions, URL-style parameter encoding, and fallback resolution strategies for GitHub Agentic Workflows
sidebar:
  order: 1360
---

# Model Alias Format Specification

**Version**: 1.1.0  
**Status**: Draft  
**Publication Date**: 2026-05-03  
**Editor**: GitHub Agentic Workflows Team  
**This Version**: [model-alias-specification](/gh-aw/reference/model-alias-specification/)  
**Latest Published Version**: This document

---

## Abstract

This specification defines the Model Alias Format (MAF) for GitHub Agentic Workflows (AWF). It establishes normative requirements for model identifier syntax, URL-style parameter encoding for model configuration (such as reasoning effort and temperature), and the multi-layer fallback resolution algorithm that AWF applies when selecting a concrete model at compile time. The specification is intended for workflow authors, AWF implementors, and tool integrations that consume or produce model name strings.

## Status of This Document

This section describes the status of this document at the time of publication. This is a draft specification and may be updated, replaced, or made obsolete by other documents at any time.

This document is governed by the GitHub Agentic Workflows project specifications process.

## Table of Contents

1. [Introduction](#1-introduction)
2. [Conformance](#2-conformance)
3. [Terminology](#3-terminology)
4. [Model Identifier Syntax](#4-model-identifier-syntax)
5. [URL-Style Parameter Encoding](#5-url-style-parameter-encoding)
6. [Defined Parameters](#6-defined-parameters)
7. [Alias Map Format](#7-alias-map-format)
8. [Fallback Resolution Algorithm](#8-fallback-resolution-algorithm)
9. [Builtin Aliases](#9-builtin-aliases)
10. [Merge Precedence](#10-merge-precedence)
11. [Validation Rules](#11-validation-rules)
12. [Compliance Testing](#12-compliance-testing)
13. [Appendices](#appendices)
14. [References](#references)
15. [Change Log](#change-log)

---

## 1. Introduction

### 1.1 Purpose

AWF workflows must specify the LLM model that an AI engine invokes during execution. Model names are version-specific, provider-specific, and vary substantially across the Copilot, Anthropic, OpenAI, and Google ecosystems. The Model Alias Format addresses three interrelated problems:

1. **Portability**: A workflow written against `sonnet` runs on whatever the current Anthropic Sonnet model is, without requiring edits when models are updated.
2. **Configurability**: Model-level knobs such as reasoning effort and sampling temperature must be expressible inline, without adding new frontmatter fields for every parameter.
3. **Resilience**: When a preferred model is unavailable, AWF MUST attempt alternative candidates in a deterministic order before failing.

### 1.2 Scope

This specification covers:

- Syntax of a single model identifier string (bare name, provider-scoped name, glob pattern, URL-encoded parameters)
- Encoding rules for URL-style query parameters on a model name
- The alias map YAML format used in workflow frontmatter (`models:` key)
- The multi-layer merge and recursive resolution algorithm
- Semver-aware glob match ranking
- Builtin aliases shipped with AWF
- Normative validation rules applied at compile time and at runtime

This specification does NOT cover:

- Engine-specific API call construction (how parameters are forwarded to the provider REST API)
- Token budgets, cost accounting, or the Effective Tokens metric (see [Effective Tokens Specification](/gh-aw/reference/effective-tokens-specification/))
- Model capability detection at runtime
- Model routing logic within the Copilot gateway

### 1.3 Design Goals

The Model Alias Format:

1. Uses familiar URL query-string syntax so parameters are human-readable and require no new YAML keys.
2. Is backward-compatible: a plain model name without parameters is a valid identifier.
3. Supports recursive alias resolution to allow layered abstractions (`auto` → `large` → `sonnet`).
4. Is extensible: new parameters MAY be added without changing the core syntax.
5. Preserves the `vendor/model` convention already established in the AWF engine layer.

---

## 2. Conformance

### 2.1 Conformance Classes

**Conforming AWF implementation**: An implementation that satisfies all MUST/SHALL requirements in this specification, including correct parsing, resolution, and validation of model alias strings.

**Partially conforming implementation**: An implementation that correctly parses and resolves model identifiers (Sections 4–8) but does not implement the full parameter set defined in Section 6.

### 2.2 Requirements Notation

The key words "MUST", "MUST NOT", "REQUIRED", "SHALL", "SHALL NOT", "SHOULD", "SHOULD NOT", "RECOMMENDED", "NOT RECOMMENDED", "MAY", and "OPTIONAL" in this document are to be interpreted as described in [RFC 2119](https://www.ietf.org/rfc/rfc2119.txt).

### 2.3 Compliance Levels

- **Level 1 – Syntax**: Correct parsing of model identifier strings including parameters (Sections 4–5)
- **Level 2 – Resolution**: Full alias map format, merge precedence, and fallback resolution (Sections 7–10)
- **Level 3 – Complete**: Full parameter set, validation, and error reporting (Sections 6, 11)

---

## 3. Terminology

**Model Identifier**: A string that names a concrete or abstract LLM model. May include URL-style parameters.

**Bare Identifier**: A model identifier that has no slash and no query string. Examples: `gpt-5`, `sonnet`, `auto`.

**Provider-Scoped Identifier**: A model identifier of the form `provider/model-id`. The provider segment is a short token identifying the routing gateway or vendor. Examples: `copilot/gpt-5`, `anthropic/claude-opus-4`.

**Glob Pattern**: A provider-scoped identifier containing one or more `*` wildcard characters. Used in alias list entries to match a family of concrete model names. Examples: `copilot/*sonnet*`, `openai/gpt-5*mini*`.

**Alias Name**: A bare identifier that resolves to an ordered list of candidate patterns or other alias names. Examples: `sonnet`, `large`, `auto`.

**Model Parameter**: A key=value pair encoded in the query string of a model identifier that supplies additional configuration to the model invocation. Example: `effort=high`.

**Alias Map**: The YAML map under the `models:` frontmatter key. Keys are alias names (or the empty string for the default policy). Values are ordered lists of model patterns or other alias names.

**Builtin Aliases**: The set of alias definitions shipped with AWF, covering the major model families across supported vendors.

**Default Policy** (`""`): The alias entry whose key is the empty string. When a workflow does not specify a model, the default policy governs which model family is selected.

**Resolution**: The process of converting a model identifier or alias name into a concrete, provider-scoped model name by following the fallback list and expanding aliases recursively.

---

## 4. Model Identifier Syntax

### 4.1 Grammar

A model identifier string MUST conform to the following ABNF grammar:

```abnf
model-identifier  = base-identifier [ "?" query-string ]

base-identifier   = bare-name
                  / provider-scoped
                  / glob-pattern

bare-name         = 1*( ALPHA / DIGIT / "-" / "_" / "." )
                    ; MUST NOT start with "-" or "."

provider-scoped   = provider-token "/" model-token

provider-token    = ALPHA 0*( ALPHA / DIGIT / "-" )
                    ; starts with a letter; hyphens allowed but not at end

model-token       = model-char 0*( model-char / "." model-char )
                    ; segments separated by "."; each segment starts with ALPHA or DIGIT

model-char        = ALPHA / DIGIT / "-" / "_"
                    ; the underscore is permitted inside model tokens only

glob-pattern      = provider-token "/" model-glob-token

model-glob-token  = 1*( model-char / "." / "*" )
                    ; "*" is a wildcard that matches zero or more non-"/" characters

query-string      = param *( "&" param )

param             = param-key "=" param-value

param-key         = ALPHA 0*( ALPHA / DIGIT / "-" )
                    ; starts with a letter; no digits or hyphens at start

param-value       = 1*( ALPHA / DIGIT / "-" / "_" / "." )
```

**Allowed character set summary:**

| Segment | Allowed characters | Notes |
|---------|-------------------|-------|
| Provider token | `[A-Za-z0-9-]` | MUST start with a letter; MUST NOT end with `-` |
| Model token | `[A-Za-z0-9-_.]` | Dot separates version segments; underscore allowed within a segment only |
| Bare name | `[A-Za-z0-9-_.]` | MUST NOT start with `-` or `.` |
| Glob wildcard | `*` | Only in model-glob-token; provider token MUST NOT contain `*` |
| Parameter key | `[A-Za-z0-9-]` | MUST start with a letter |
| Parameter value | `[A-Za-z0-9-_.]` | |

Characters explicitly PROHIBITED in all segments: whitespace, `@`, `!`, `#`, `$`, `%`, `^`, `&` (except as separator), `(`, `)`, `+`, `=`, `[`, `]`, `{`, `}`, `|`, `\`, `:`, `;`, `,`, `<`, `>`, `/` (except as provider/model separator), `?` (except as parameter separator), `"`, `'`.

**Notes:**

- The `?` character separates the base identifier from the query string.
- An implementation MUST NOT percent-encode or percent-decode the model identifier string; the syntax is self-contained.
- Multiple parameters are separated by `&`.
- A bare identifier with no `?` and no `/` is treated as an alias name during resolution (see Section 8).

### 4.2 Examples

```text
# Bare alias name
sonnet

# Bare alias name with effort parameter
opus?effort=high

# Provider-scoped exact name
copilot/gpt-5

# Provider-scoped name with effort parameter
copilot/claude-opus-4.5?effort=medium

# Provider-scoped name with multiple parameters
openai/o3?effort=high&temperature=0.2

# Glob pattern (used in alias list entries only — not in engine.model)
copilot/*sonnet*
```

### 4.3 Parsing Rules

An implementation MUST parse a model identifier as follows:

1. Split on the first occurrence of `?`. The left side is the `base-identifier`; the right side (if present) is the `query-string`.
2. If the `base-identifier` contains `/`, it is a provider-scoped identifier or glob pattern. The segment before the first `/` is the provider token; the segment after is the model token.
3. If the `base-identifier` contains no `/`, it is a bare name. It MAY be an alias name or a concrete model name depending on context.
4. Parse the `query-string` as a sequence of `key=value` pairs separated by `&`.
5. Implementations MUST reject model identifiers where a parameter key or value contains characters outside the allowed set, with a compile-time validation error.

---

## 5. URL-Style Parameter Encoding

### 5.1 Motivation

Model invocation knobs such as reasoning effort, temperature, and top-p are provider-specific and frequently change as new models are released. Encoding them as query parameters on the model name keeps the alias map clean, avoids proliferating frontmatter keys, and makes parameters visible at the point of use.

### 5.2 Attachment Point

Parameters MUST be attached to the model identifier string. They apply to the model they are attached to after alias resolution terminates in a concrete provider-scoped name.

Parameters attached to an alias name MUST be forwarded to the first successfully resolved concrete model. A later section (Section 8.5) defines the forwarding semantics precisely.

### 5.3 Parameter Inheritance

When an alias list entry itself carries parameters (e.g., `opus?effort=high`), and the caller also specifies parameters (e.g., `sonnet?temperature=0.3`), the following precedence rules apply:

1. Parameters explicitly set on the **caller** identifier (the engine.model or alias list entry) take highest precedence.
2. Parameters inherited from a **resolved alias** fill in any key not set by the caller.
3. Parameters set in the **builtin alias** fill in any key not set by layers 1–2.

This allows callers to override specific parameters while inheriting the rest from the alias definition.

---

## 6. Defined Parameters

### 6.1 `effort`

Controls the reasoning depth or thinking budget for models that support extended reasoning (e.g., `claude-opus-4.*`, `o1`, `o3`, `o4` series).

| Value | Description |
|-------|-------------|
| `low` | Minimal reasoning tokens; fastest and least expensive |
| `medium` | Balanced reasoning budget (provider default where applicable) |
| `high` | Maximum reasoning tokens; highest quality, highest cost |

**Type**: enumerated string  
**Allowed values**: `low`, `medium`, `high`  
**Default**: provider-defined (typically `medium`)

Implementations MUST map these values to the provider's native reasoning-control API parameter. For Anthropic models this is the `thinking.budget_tokens` field; for OpenAI `o`-series models this is the `reasoning_effort` field. The exact mapping is engine-specific and outside the scope of this specification.

An implementation MUST emit a compile-time warning when `effort` is set on a model known not to support extended reasoning.

**Examples:**

```yaml
# Workflow using opus with high effort
engine:
  id: copilot
  model: opus?effort=high

# Alias map entry with effort baked in
models:
  deep-think:
    - opus?effort=high
    - gpt-5?effort=high
```

### 6.2 `temperature`

Controls the sampling temperature for models that support it. Lower values produce more deterministic output; higher values produce more varied output.

| Value range | Description |
|-------------|-------------|
| `0.0`–`2.0` | Floating-point value (decimal notation, e.g. `0.7`) |

**Type**: decimal float string  
**Allowed values**: `0.0` through `2.0` inclusive  
**Default**: provider-defined

Implementations MUST convert the string to a floating-point number before forwarding to the provider API.

Implementations MUST emit a compile-time error if the value is outside the range `[0.0, 2.0]` or cannot be parsed as a decimal float.

**Examples:**

```yaml
engine:
  id: codex
  model: openai/gpt-5?temperature=0.2

models:
  deterministic:
    - gpt-5?temperature=0.0
    - sonnet?temperature=0.0
```

### 6.3 Future Parameters

This specification is designed to be extended. Future versions MAY define additional parameters. An implementation MUST silently ignore unrecognized parameter keys but SHOULD emit a warning to assist authors in identifying typos.

The following parameter names are reserved for future use and MUST NOT be used for other purposes:

- `top-p`
- `top-k`
- `max-tokens`
- `seed`
- `stop`

---

## 7. Alias Map Format

### 7.1 YAML Representation

The alias map is expressed under the `models:` key in workflow frontmatter:

```yaml
models:
  <alias-name>:
    - <model-pattern-or-alias>
    - <model-pattern-or-alias>
    ...
```

Each key is an alias name (a bare identifier). The special empty-string key (`""`) defines the **default policy** — the ordered list tried when no model is specified.

Each list entry is either:

- A **glob pattern** (provider-scoped, may contain `*`)
- A **provider-scoped model name** (no wildcards)
- Another **alias name** (bare identifier, resolved recursively)

Any of the above MAY carry URL-style parameters.

### 7.2 Example

```yaml
models:
  # Override the builtin sonnet alias to prefer a custom gateway
  sonnet:
    - mygateway/*sonnet-v3*
    - copilot/*sonnet*

  # New alias for high-effort reasoning tasks
  deep-think:
    - opus?effort=high
    - gpt-5?effort=high

  # Default policy: try deep-think first, fall back to sonnet
  "":
    - deep-think
    - sonnet
```

### 7.3 Constraints

- Alias names MUST match `bare-name` as defined in Section 4.1.
- Alias names MUST NOT contain `/`, `?`, or `&`.
- The same alias name MUST NOT appear more than once as a key within the same alias map layer. Duplicate keys are a compile-time error.
- Circular alias references (direct or transitive) are PROHIBITED and MUST be detected and reported as compile-time errors.
- An alias list MUST contain at least one entry.

---

## 8. Fallback Resolution Algorithm

### 8.1 Overview

Resolution converts a model identifier (which may be an alias, a glob, or a concrete name) into a single **concrete provider-scoped model name** by following the fallback list until a match is found in the engine's model catalog.

### 8.2 Resolution Input

The resolution procedure takes:

- A **target**: the model identifier string from `engine.model`, or the alias name `""` if no model is specified.
- The **merged alias map**: the result of the three-layer merge described in Section 10.
- The **engine's model catalog**: the set of concrete model names available to the configured engine (provider-scoped, no wildcards).

### 8.3 Resolution Procedure

```
Resolve(target, aliasMap, catalog):
  1. Strip parameters from target → (base, params)
  2. If base is found as a key in aliasMap:
     a. Retrieve the ordered list L for base.
     b. For each entry E in L:
        i.  Strip parameters from E → (eBase, eParams)
        ii. Merge params ← MergeParams(params, eParams)   // caller wins
        iii.If eBase is a key in aliasMap → recurse: Resolve(eBase+MarshalParams(eParams), ...)
        iv. If eBase is a glob pattern → match against catalog; if any match, return first match + params
        v.  If eBase is a provider-scoped name (no wildcards) → if present in catalog, return eBase + params
     c. If no entry in L resolved → continue to next step (as if target were a bare name)
  3. If base matches catalog entry exactly → return base + params
  4. Resolution FAILS: emit a compile-time error naming the unresolved alias/model.
```

### 8.4 Glob Matching

Glob patterns MUST be matched using the following rules:

- `*` matches zero or more characters that do not include `/`.
- Matching is case-insensitive.
- The provider prefix MUST be matched exactly (no wildcards in provider segment).

When multiple catalog entries match a glob pattern, the implementation MUST select the entry with the **highest semantic version** — it MUST NOT return the first lexicographic match. This ensures that alias-based resolution always resolves to the latest available model in a matched family.

#### 8.4.1 Semver Extraction

To rank matches, an implementation MUST extract the version string from a catalog entry using the following procedure:

1. Take the model-token portion (after the first `/`).
2. Scan right-to-left for the last occurrence of a version segment matching the pattern `\d+(\.\d+)*` (one or more dot-separated integers).
3. The extracted version string is used for comparison as a semver tuple.
4. If no version segment is found, the entry is treated as version `0.0.0` for ranking purposes.

Examples of version extraction:

| Catalog entry | Extracted version |
|---------------|------------------|
| `copilot/claude-opus-4.5` | `4.5` |
| `copilot/claude-opus-4` | `4` |
| `copilot/claude-sonnet-4.5-20250514` | `4.5` (scans to find last numeric sequence pattern; date portion treated separately — see §8.4.2) |
| `openai/gpt-5` | `5` |
| `copilot/gemini-2.5-pro` | `2.5` |
| `copilot/model-without-version` | `0.0.0` |

#### 8.4.2 Semver Comparison

Versions are compared as ordered tuples of non-negative integers. A shorter tuple is left-padded with zeros before comparison. Examples:

- `4.5` > `4` (i.e., `4.5` > `4.0`)
- `2.5` > `2.0`
- `5` > `4.5` (i.e., `5.0` > `4.5`)

When a model token contains a date suffix (e.g., `claude-sonnet-4.5-20250514`), the date is treated as a secondary sort key (higher date wins) only when the primary version tuples are equal.

When two entries have identical versions and identical date suffixes (or both lack dates), the implementation SHOULD preserve catalog ordering as a tiebreaker.

#### 8.4.3 Selection

```
SelectLatestMatch(pattern, catalog):
  matches = [ entry for entry in catalog if GlobMatch(pattern, entry) ]
  if matches is empty → return nil
  sort matches descending by (semver, date-suffix)
  return matches[0]
```

### 8.5 Parameter Forwarding

When recursing into an alias, parameters accumulate using caller-wins semantics:

```
MergeParams(callerParams, aliasParams):
  result = aliasParams
  for each (key, value) in callerParams:
    result[key] = value   // caller overwrites alias
  return result
```

The resulting merged parameter set is attached to the resolved concrete model name.

### 8.6 Loop Detection

Circular alias references are strictly PROHIBITED. An implementation MUST detect and report cycles at both compile time and at runtime.

#### 8.6.1 Compile-Time Detection

When the alias map is built (after the three-layer merge described in Section 10), the implementation MUST perform a full cycle check over all alias keys before any resolution is attempted.

Algorithm: for each alias key, perform a depth-first traversal of its list entries. Maintain a set of alias names on the current DFS path. If any traversal reaches an alias key already on the current path, a cycle is detected and MUST be reported as a compile-time error that names every alias involved in the cycle.

Compilation MUST be aborted when a cycle is detected. A workflow with a cyclic alias map is invalid and MUST NOT produce a lock file.

**Example error:**

```
Error: circular alias reference detected: deep-think → opus → deep-think
  models:
    deep-think: [opus]
    opus: [deep-think]   ← cycle back to deep-think
```

#### 8.6.2 Runtime Detection

At runtime, an implementation MUST also guard against cycles that evade compile-time detection (e.g., when alias maps are constructed or extended dynamically, or when catalog entries alias-resolve at engine initialization).

The runtime resolver MUST maintain a per-resolution-call set of alias names visited on the current resolution path. If `Resolve` is invoked with an alias name already in the visited set, the implementation MUST:

1. Immediately abort the resolution of that model.
2. Log an error that includes the chain of aliases that led to the cycle.
3. Treat the failing model as unavailable (skip to the next fallback entry if one exists).
4. If no fallback remains, fail the workflow run with a descriptive error message.

The runtime guard is an additional safety net and does NOT replace compile-time detection.

### 8.7 Empty-String Default Policy

When `engine.model` is absent or empty, the implementation MUST use `""` (the empty string) as the target for resolution. If the `""` key is not present in the merged alias map, the implementation MUST pass through to the engine's own default model selection.

---

## 9. Builtin Aliases

AWF ships the following builtin aliases. Workflow frontmatter definitions (and imported workflow definitions) are merged on top of these; see Section 10 for merge precedence.

### 9.1 Vendor Family Aliases

| Alias | Patterns (in order) |
|-------|---------------------|
| `sonnet` | `copilot/*sonnet*`, `anthropic/*sonnet*` |
| `haiku` | `copilot/*haiku*`, `anthropic/*haiku*` |
| `opus` | `copilot/*opus*`, `anthropic/*opus*` |
| `gpt-4.1` | `copilot/gpt-4.1*`, `openai/gpt-4.1*` |
| `gpt-5` | `copilot/gpt-5*`, `openai/gpt-5*` |
| `gpt-5-mini` | `copilot/gpt-5*mini*`, `openai/gpt-5*mini*` |
| `gpt-5-nano` | `copilot/gpt-5*nano*`, `openai/gpt-5*nano*` |
| `gpt-5-codex` | `copilot/gpt-5*codex*`, `openai/gpt-5*codex*` |
| `reasoning` | `copilot/o1*`, `copilot/o3*`, `copilot/o4*`, `openai/o1*`, `openai/o3*`, `openai/o4*` |
| `gemini-flash` | `copilot/gemini-*flash*`, `google/gemini-*flash*` |
| `gemini-pro` | `copilot/gemini-*pro*`, `google/gemini-*pro*` |

### 9.2 Meta-Aliases

| Alias | Resolves to (in order) |
|-------|------------------------|
| `small` | `mini` |
| `mini` | `haiku`, `gpt-5-mini`, `gpt-5-nano`, `gemini-flash` |
| `large` | `sonnet`, `gpt-5`, `gemini-pro` |
| `auto` | `large` |

Meta-aliases reference other aliases and are resolved recursively. They allow workflow authors to express capability tiers (`mini`, `large`) without committing to a specific vendor.

### 9.3 Absence of Default Policy

The builtin alias set does NOT define a `""` entry. If a workflow does not define a `""` entry either, the engine's own default model is used.

---

## 10. Merge Precedence

### 10.1 Three-Layer Merge

The final alias map used for resolution is built from three layers, applied in order from lowest to highest priority:

| Priority | Layer | Rule |
|----------|-------|------|
| 1 (lowest) | Builtin aliases | Always present; shipped with AWF |
| 2 | Imported workflow aliases | First import to define a key wins among imports |
| 3 (highest) | Main workflow frontmatter `models:` | Always wins; overwrites any lower-layer entry for the same key |

### 10.2 Models Payload Merge Algorithm

The following pseudocode defines the exact merge procedure that produces the unified alias map from the three layers:

```
MergeAliasMap(builtins, importedMaps[], frontmatterMap):

  // Step 1: Start from builtins (layer 1 — always present)
  merged = copy(builtins)

  // Step 2: Apply imported maps in BFS order (layer 2 — first-wins among imports)
  for importedMap in importedMaps:         // BFS traversal order
    for key, list in importedMap:
      if key NOT IN merged:               // first import to define this key wins
        merged[key] = list
      // if key already present, ignore this import's definition

  // Step 3: Apply frontmatter map (layer 3 — always wins)
  for key, list in frontmatterMap:
    merged[key] = list                    // unconditionally overwrites any prior layer

  return merged
```

**Key properties of this algorithm:**

- The merge is performed at the **key level** — the entire list for a key is replaced, never merged entry-by-entry.
- Within the imported layer (step 2), the strategy is **first-wins** among peers: once a key is set by an earlier import, later imports defining the same key are silently ignored.
- The main workflow (step 3) uses **last-wins** relative to all other layers: it always overwrites.
- The algorithm runs **once at compile time** after all imports are resolved. The result is a stable, frozen map used for all subsequent resolution calls.

### 10.3 Key-Level Override

Override is performed at the **key level**: if the main workflow defines `sonnet`, the entire list from the main workflow replaces the entire builtin `sonnet` list. There is no list-level merge.

### 10.4 Import Priority

When multiple imported workflows define the same alias key, the first import encountered in BFS traversal order wins. This is consistent with the `features:` merge semantics elsewhere in AWF.

### 10.5 Transitivity

When a resolved alias entry references another alias (e.g., `"" → sonnet`), the resolved `sonnet` is looked up in the **same merged map**. The merged map is computed once at compile time and remains stable throughout resolution.

---

## 11. Validation Rules

### 11.1 Syntax Validation

At compile time, an implementation MUST:

- **V-MAF-001**: Reject model identifiers that do not conform to the grammar in Section 4.1 with a parse error.
- **V-MAF-002**: Reject parameter values for `effort` that are not one of `low`, `medium`, `high`.
- **V-MAF-003**: Reject parameter values for `temperature` that cannot be parsed as a decimal float in `[0.0, 2.0]`.
- **V-MAF-004**: Reject glob patterns used in `engine.model` (glob patterns are valid only in alias list entries).
- **V-MAF-005**: Reject alias keys that contain `/`, `?`, or `&`.
- **V-MAF-006**: Reject any model identifier segment that contains a character outside the allowed set defined in Section 4.1. The implementation MUST name the offending character and the segment type (provider, model, alias, parameter key, or parameter value) in the error message.

### 11.2 Semantic Validation

At compile time, an implementation MUST:

- **V-MAF-010**: Detect and report all circular alias references using the DFS algorithm described in Section 8.6.1. Compilation MUST be aborted on any detected cycle.
- **V-MAF-011**: Emit a warning for unrecognized parameter keys (after applying Section 6 known parameters).
- **V-MAF-012**: Emit a warning when `effort` is attached to a model known not to support extended reasoning.

At runtime, an implementation MUST:

- **V-MAF-013**: Guard against cycles not caught at compile time using the per-call visited-set guard described in Section 8.6.2. A runtime cycle MUST cause an immediate resolution failure with a descriptive error that includes the cycle chain.

### 11.3 Resolution Validation

At compile time, an implementation SHOULD:

- **V-MAF-020**: Warn when a model alias resolves to zero entries in the engine's catalog, indicating the alias may be misconfigured or the engine does not support those models.

---

## 12. Compliance Testing

### 12.1 Test Suite Requirements

#### 12.1.1 Syntax Tests

- **T-MAF-001**: Parse bare alias name `sonnet` → base=`sonnet`, params=`{}`
- **T-MAF-002**: Parse `opus?effort=high` → base=`opus`, params=`{effort: high}`
- **T-MAF-003**: Parse `copilot/gpt-5` → provider=`copilot`, model=`gpt-5`, params=`{}`
- **T-MAF-004**: Parse `openai/o3?effort=low&temperature=0.2` → provider=`openai`, model=`o3`, params=`{effort: low, temperature: 0.2}`
- **T-MAF-005**: Reject `copilot/*sonnet*` when used as `engine.model` (glob not allowed there)
- **T-MAF-006**: Reject `effort=extreme` (unknown effort value)
- **T-MAF-007**: Reject `temperature=3.0` (out of range)
- **T-MAF-008**: Reject `my model` (whitespace in identifier)
- **T-MAF-009**: Reject `my:model` (colon in identifier); error message MUST identify the offending character

#### 12.1.2 Resolution Tests

- **T-MAF-020**: `sonnet` resolves to first catalog match for `copilot/*sonnet*` or `anthropic/*sonnet*`
- **T-MAF-021**: `auto` transitively resolves through `large` → `sonnet` → concrete model
- **T-MAF-022**: `opus?effort=high` propagates `effort=high` to resolved concrete model
- **T-MAF-023**: Caller `opus?effort=high` + alias entry `opus?effort=medium` → resolved with `effort=high` (caller wins)
- **T-MAF-024**: Custom alias `deep-think: [opus?effort=high]` resolves via `opus` builtin alias
- **T-MAF-025**: Default policy `""` is used when `engine.model` is absent
- **T-MAF-026**: Given catalog `[copilot/claude-opus-4, copilot/claude-opus-4.5]` and pattern `copilot/*opus*`, resolution returns `copilot/claude-opus-4.5` (latest semver wins)
- **T-MAF-027**: Given catalog `[copilot/claude-sonnet-4.5-20250310, copilot/claude-sonnet-4.5-20250514]` and pattern `copilot/*sonnet*`, resolution returns `copilot/claude-sonnet-4.5-20250514` (latest date wins when versions are equal)
- **T-MAF-028**: Given catalog entries with no version string, they are ranked as `0.0.0` and lose to entries with a version

#### 12.1.3 Merge Precedence Tests

- **T-MAF-030**: Main workflow `sonnet` key overwrites builtin `sonnet` key entirely
- **T-MAF-031**: First imported workflow to define `mini` wins over subsequent imports
- **T-MAF-032**: Main workflow always wins over imported workflow for the same key
- **T-MAF-033**: Key defined only in builtins and absent from all imports and frontmatter is preserved in merged map

#### 12.1.4 Validation Tests

- **T-MAF-040**: Circular alias `a: [b]`, `b: [a]` is detected and reported as a compile-time error; compilation is aborted
- **T-MAF-041**: Longer cycle `a: [b]`, `b: [c]`, `c: [a]` is detected; error message names all three aliases
- **T-MAF-042**: Runtime cycle guard triggers when a dynamic alias expansion creates a cycle at engine startup; resolution falls back to next entry if available
- **T-MAF-043**: Unrecognized parameter key `?foo=bar` produces a compile-time warning
- **T-MAF-044**: Alias with no matching catalog entries produces a compile-time warning

### 12.2 Compliance Checklist

| Requirement | Test ID | Level | Status |
|-------------|---------|-------|--------|
| Bare identifier parsing | T-MAF-001 | 1 | Required |
| Parameter parsing | T-MAF-002, 004 | 1 | Required |
| Glob rejection in engine.model | T-MAF-005 | 1 | Required |
| Invalid effort value rejection | T-MAF-006 | 1 | Required |
| Temperature range validation | T-MAF-007 | 1 | Required |
| Whitespace rejection (V-MAF-006) | T-MAF-008 | 1 | Required |
| Illegal character rejection with message | T-MAF-009 | 1 | Required |
| Single-hop alias resolution | T-MAF-020 | 2 | Required |
| Transitive alias resolution | T-MAF-021 | 2 | Required |
| Parameter propagation | T-MAF-022 | 2 | Required |
| Caller-wins parameter merge | T-MAF-023 | 2 | Required |
| Default policy (`""`) | T-MAF-025 | 2 | Required |
| Semver-aware glob selection (latest wins) | T-MAF-026 | 2 | Required |
| Date-suffix tiebreaker | T-MAF-027 | 2 | Required |
| Main workflow wins merge | T-MAF-030 | 2 | Required |
| Compile-time circular alias detection | T-MAF-040, 041 | 3 | Required |
| Runtime circular alias guard | T-MAF-042 | 3 | Required |
| Unrecognized param warning | T-MAF-043 | 3 | Recommended |
| Empty catalog warning | T-MAF-044 | 3 | Recommended |

---

## Appendices

### Appendix A: Complete Resolution Example

#### A.1 Scenario

A workflow specifies:

```yaml
engine:
  id: copilot
  model: deep-think?temperature=0.1

models:
  deep-think:
    - opus?effort=high
    - gpt-5?effort=high
```

The Copilot catalog contains: `copilot/claude-opus-4.5`, `copilot/gpt-5`.

#### A.2 Resolution Trace

```
Resolve("deep-think?temperature=0.1", mergedMap, catalog)
  base = "deep-think", params = {temperature: 0.1}
  "deep-think" found in map → list: ["opus?effort=high", "gpt-5?effort=high"]

  Entry 1: "opus?effort=high"
    eBase = "opus", eParams = {effort: high}
    MergeParams({temperature: 0.1}, {effort: high})
      → {effort: high, temperature: 0.1}   // both keys, no conflict
    "opus" found in map → list: ["copilot/*opus*", "anthropic/*opus*"]

    Entry 1.1: "copilot/*opus*"
      Glob match against catalog:
        copilot/claude-opus-4.5 matches copilot/*opus*  ✓
      Return "copilot/claude-opus-4.5?effort=high&temperature=0.1"
```

#### A.3 Result

The engine is invoked with model `copilot/claude-opus-4.5` and parameters:
- `effort = high`
- `temperature = 0.1`

### Appendix B: Schema Reference

The `models:` field in AWF frontmatter is defined in the main workflow JSON Schema (`pkg/parser/schemas/main_workflow_schema.json`) as follows (informative excerpt):

```json
{
  "models": {
    "type": "object",
    "description": "Named model alias definitions with ordered fallback lists...",
    "additionalProperties": {
      "type": "array",
      "items": {
        "type": "string",
        "description": "A vendor/modelid glob pattern or alias name"
      }
    }
  }
}
```

The `engine.model` field is defined as:

```json
{
  "model": {
    "type": "string",
    "description": "Optional specific LLM model to use (e.g., 'claude-3-5-sonnet-20241022', 'gpt-4')."
  }
}
```

This specification extends both fields by normalizing the string format they accept.

### Appendix C: Extending with New Parameters

To propose a new model parameter for inclusion in Section 6:

1. The parameter key MUST be a valid `param-key` per the grammar in Section 4.1.
2. The parameter MUST have a clearly defined value set or range.
3. The parameter MUST map to a concrete provider API field for at least one supported engine.
4. The parameter name MUST NOT conflict with any reserved name listed in Section 6.3.

Until a parameter is formally added to this specification, workflow authors MAY use custom parameters with the understanding that AWF will emit a warning and forward the value as-is to the engine.

### Appendix D: Security Considerations

Model parameters are compile-time configuration values and are not derived from untrusted user input during workflow execution. However, implementations MUST:

- Reject parameter values that fall outside their defined range to prevent unexpected provider API behavior.
- Not expose parameter values in public logs, as they may reveal information about the reasoning strategy or cost envelope of a workflow.
- Treat model names and parameters as configuration (not executable) to avoid injection into shell commands.

---

## References

### Normative References

- **[RFC 2119]** Bradner, S., "Key words for use in RFCs to Indicate Requirement Levels", BCP 14, RFC 2119, March 1997. <https://www.ietf.org/rfc/rfc2119.txt>
- **[RFC 3986]** Berners-Lee, T. et al., "Uniform Resource Identifier (URI): Generic Syntax", RFC 3986, January 2005. <https://www.ietf.org/rfc/rfc3986.txt>
- **[AWF-SCHEMA]** GitHub Agentic Workflows Main Workflow JSON Schema. `pkg/parser/schemas/main_workflow_schema.json`

### Informative References

- **[AWF-ENGINES]** GitHub Agentic Workflows — AI Engines reference. <https://gh-aw.pages.dev/reference/engines/>
- **[AWF-ET-SPEC]** GitHub Agentic Workflows — Effective Tokens Specification. <https://gh-aw.pages.dev/reference/effective-tokens-specification/>
- **[ANTHROPIC-THINKING]** Anthropic — Extended Thinking documentation. <https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking>
- **[OPENAI-REASONING]** OpenAI — Reasoning models guide. <https://platform.openai.com/docs/guides/reasoning>

---

## Change Log

### Version 1.1.0 (Draft)

- **Added**: Semver-aware glob matching (§8.4): when multiple catalog entries match a glob, the entry with the highest semantic version is selected; date suffix used as tiebreaker.
- **Added**: Strict character restriction table and prohibited-character list for all identifier segments (§4.1); new validation rule V-MAF-006 and test cases T-MAF-008/009.
- **Added**: Runtime cycle detection (§8.6.2): resolver MUST maintain a per-call visited set and fail fast on re-entrant alias expansion; added V-MAF-013 and T-MAF-042.
- **Enhanced**: Compile-time cycle detection (§8.6.1): expanded from a single sentence to a full DFS algorithm with error-message requirements.
- **Added**: Models payload merge algorithm pseudocode (§10.2) making the three-layer merge semantics explicit.
- **Added**: Merge precedence test T-MAF-033 (builtin-only keys are preserved).

### Version 1.0.0 (Draft)

- Initial specification defining model identifier syntax, URL-style parameter encoding, alias map format, fallback resolution algorithm, builtin aliases, merge precedence, and compliance tests.
- Defined `effort` parameter with `low | medium | high` values.
- Defined `temperature` parameter with `[0.0, 2.0]` range.
- Reserved future parameter names: `top-p`, `top-k`, `max-tokens`, `seed`, `stop`.

---

*Copyright © 2026 GitHub Agentic Workflows Team. All rights reserved.*
