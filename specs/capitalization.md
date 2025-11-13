---
title: Capitalization Guidelines
description: Understanding capitalization conventions for "agentic workflows" vs "Agentic Workflows" in command descriptions.
sidebar:
  order: 900
---

The gh-aw CLI follows context-based capitalization to distinguish between the product name and generic workflow references. This approach provides clarity and maintains professional consistency across all user-facing text.

## Capitalization Rules

The CLI uses different capitalization based on context:

| Context | Format | Example |
|---------|--------|---------|
| Product name | **Capitalized** | "GitHub Agentic Workflows CLI from GitHub Next" |
| Generic workflows | **Lowercase** | "Enable agentic workflows" |
| Technical terms | **Capitalized** | "Compile Markdown to YAML workflows" |

## Design Rationale

This capitalization convention distinguishes between the product name (GitHub Agentic Workflows) and the concept (agentic workflows), following industry standards similar to "GitHub Actions" vs. "actions". The consistent pattern across all commands, help text, and documentation simplifies both user comprehension and contributor decision-making.

## Examples

### Product References (Capitalized)

When referring to the gh-aw tool or GitHub Agentic Workflows as a product:

```
GitHub Agentic Workflows CLI from GitHub Next
GitHub Agentic Workflows from GitHub Next
```

### Generic Workflow References (Lowercase)

When describing actions performed on workflows:

```
Enable agentic workflows
Initialize repository for agentic workflows
Download and analyze agentic workflow logs
Add an MCP tool to an agentic workflow
```

### Technical Terms (Capitalized)

Proper nouns and acronyms maintain their standard capitalization:

```
Compile Markdown to YAML workflows
MCP helpers
```

## Implementation

The capitalization rules are enforced through automated tests in `cmd/gh-aw/capitalization_test.go`. These tests verify:

- Product name uses "GitHub Agentic Workflows" (capitalized)
- Generic workflow references use "agentic workflows" (lowercase)
- Technical terms maintain proper capitalization

The tests run as part of the standard test suite and prevent inconsistencies from being introduced.

