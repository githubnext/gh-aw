---
name: developer
description: Core developer rules and coding conventions for gh-aw changes.
---


# Developer Instructions

Use this reference for gh-aw technical specs and development guidelines across code organization, validation, security, and implementation patterns.

## Table of Contents

- [Operational Command Playbook](#operational-command-playbook)
- [Capitalization Guidelines](#capitalization-guidelines)
- [Sub-Skills](#sub-skills)


## Operational Command Playbook

Use this section for the detailed day-to-day command flow that was intentionally removed from `AGENTS.md` to keep first-run ambient context small.

### Validation checkpoints

Run validation in tiers — catch compile errors early, defer slow tests to the final pass only.

1. **After each significant code edit** (fast, <5s — catch compile errors immediately)
   ```bash
   make build && make fmt
   ```
2. **Before every intermediate `report_progress` call** (fast, <30s — no tests)
   ```bash
   make agent-report-progress-no-test
   ```
3. **Before the FINAL `report_progress` call** (change-scoped, includes impacted Go tests)
   ```bash
   make agent-report-progress
   ```
4. **Before final handoff when time allows**
   ```bash
   make agent-finish
   ```

> **Key rule:** Run `test-unit` only before the **final** `report_progress` call, not before intermediate saves. The pre-PR targets scope formatting, linting, tests, and workflow drift checks to the branch changes.

> **Timeout budget:** `make agent-report-progress` should normally finish in under 30 seconds. Workflow source or compiler changes additionally run the full workflow drift check. Set `TEST_UNIT_RUN_FULL=1` only when the full Go suite is required.

### Change-type command matrix

- Go file changes: `make fmt`
- Workflow markdown changes: `make recompile`
- JavaScript (`*.cjs`) changes: `make fmt-cjs && make lint-cjs`

### Merge-main playbook

When explicitly asked to merge main:

1. Run `make merge-main`.
2. If conflicts exist in `.go` or `.cjs`, resolve and stage files.
3. Run:
   ```bash
   make build
   make recompile
   git commit
   make fmt
   ```

## Capitalization Guidelines

The gh-aw CLI follows context-based capitalization to distinguish between the product name and generic workflow references.

### Capitalization Rules

| Context | Format | Example |
|---------|--------|---------|
| Product name | **Capitalized** | "GitHub Agentic Workflows CLI from GitHub Next" |
| Generic workflows | **Lowercase** | "Enable agentic workflows" |
| Technical terms | **Capitalized** | "Compile Markdown workflows to GitHub Actions YAML" |

This convention distinguishes between the product name (GitHub Agentic Workflows) and the concept (agentic workflows), following industry standards similar to "GitHub Actions" vs. "actions".

### Implementation

The capitalization rules are enforced through automated tests in `cmd/gh-aw/capitalization_test.go` that run as part of the standard test suite.



## Sub-Skills

The following sub-skills cover specific areas of the codebase. Load them lazily when the task requires the specific domain:

| Sub-skill | When to use |
|-----------|-------------|
| `.github/skills/developer-code-organization/SKILL.md` | Creating new files, refactoring, WASM stubs, file size decisions |
| `.github/skills/developer-security/SKILL.md` | Implementing new features, reviewing for security, template injection concerns |
| `.github/skills/developer-internals/SKILL.md` | Working on compiler internals, validation, safe outputs, MCP server, schema changes |
| `.github/skills/developer-release/SKILL.md` | Creating a release, evaluating breaking changes, firewall log analysis |
