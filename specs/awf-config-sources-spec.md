---
title: AWF Config Canonical Sources Specification
description: Canonical AWF configuration specification and schema sources that gh-aw agents MUST consult
sidebar:
  order: 1002
---

# AWF Config Canonical Sources Specification

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

Agents SHOULD treat this class of mismatch as a regression signal and open a corrective PR when detected.
