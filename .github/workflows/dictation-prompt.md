---
name: Dictation Prompt Generator
on:
  workflow_dispatch:
  schedule:
    - cron: "0 6 * * 0"  # Weekly on Sundays at 6 AM UTC

permissions:
  contents: read
  issues: read
  pull-requests: read

engine: copilot

network: defaults

imports:
  - shared/reporting.md

tools:
  edit:
  bash:
    - "*"
  github:
    allowed:
      - get_file_contents
      - search_code

safe-outputs:
  create-pull-request:
    title-prefix: "[docs] "
    labels: [documentation, automation]
    draft: false

timeout_minutes: 10
---

# Dictation Prompt Generator

Extract technical vocabulary from documentation files and create a concise dictation instruction file for fixing speech-to-text errors.

## Your Mission

Create a concise dictation instruction file at `.github/instructions/dictation.instructions.md` that:
1. Contains a glossary of approximately 100 project-specific terms extracted from documentation
2. Provides instructions for fixing speech-to-text errors (ambiguous terms, spacing, hyphenation)
3. Does NOT include planning guidelines or examples (keep it short and focused on error correction)
4. Includes guidelines to NOT plan or provide examples, just focus on fixing speech-to-text errors.

## Task Steps

### 1. Scan Documentation for Project-Specific Glossary

Scan documentation files in `docs/src/content/docs/` to extract approximately 100 project-specific technical terms (95-105 acceptable).

**Focus areas:**
- Configuration: safe-outputs, permissions, tools, cache-memory, toolset, frontmatter
- Engines: copilot, claude, codex, custom
- Commands: compile, audit, logs, mcp, recompile
- GitHub concepts: workflow_dispatch, pull_request, issues, discussions
- Repository-specific: agentic workflows, gh-aw, activation, MCP servers
- File formats: markdown, lockfile (.lock.yml), YAML
- Tool types: edit, bash, github, playwright, web-fetch, web-search
- Operations: fmt, lint, test-unit, timeout_minutes, runs-on

**Exclude**: makefile, Astro, starlight (tooling-specific, not user-facing)

### 2. Create the Dictation Instructions File

Create `.github/instructions/dictation.instructions.md` with:
- Title: Fix text-to-speech errors in dictated text
- Technical Context: Brief description of gh-aw
- Fix Speech-to-Text Errors: Common misrecognitions → correct terms
- Project Glossary: ~100 terms, alphabetically sorted, one per line
- Guidelines: General instructions as follows

```markdown
You do not have enough background information to plan or provide code examples.
- do NOT generate code examples
- do NOT plan steps
- focus on fixing speech-to-text errors only
```

### 3. Create Pull Request

Use the create-pull-request tool to submit your changes with:
- Title: "[docs] Update dictation prompt instructions"
- Description explaining the changes made

## Guidelines

- Scan only `docs/src/content/docs/**/*.md` files
- Extract ~100 terms (95-105 acceptable)
- Exclude tooling-specific terms (makefile, Astro, starlight)
- Prioritize frequently used project-specific terms
- Alphabetize the glossary
- No descriptions in glossary (just term names)
- Focus on fixing speech-to-text errors, not planning or examples

## Success Criteria

- ✅ File `.github/instructions/dictation.instructions.md` exists
- ✅ Contains ~100 project-specific terms (95-105 acceptable)
- ✅ Terms extracted from documentation only
- ✅ Focuses on fixing speech-to-text errors
- ✅ Pull request created with changes
