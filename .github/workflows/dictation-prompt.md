---
name: Dictation Prompt Generator
on:
  workflow_dispatch:
  schedule:
    - cron: "0 6 * * 0"  # Weekly on Sundays at 6 AM UTC

permissions:
  contents: read
  actions: read

engine: copilot

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

## Task Steps

### 1. Scan Documentation for Project-Specific Glossary

Extract a comprehensive glossary of terms specific to this project from documentation files only.

**Analyze these sources:**

```bash
# Scan documentation markdown files for technical terms
find docs/src/content/docs -name "*.md" -type f -exec grep -hoE '\b[a-z][a-z0-9_-]{2,}\b' {} + | sort -u | head -200

# Extract headings from documentation
find docs/src/content/docs -name "*.md" -type f -exec grep -h "^#" {} + | sed 's/^#* //' | head -100

# Check key documentation files
cat docs/src/content/docs/reference/*.md | grep -oE '\b[a-z][a-z0-9_-]{3,}\b' | sort -u | head -150
```

**Extract terms from documentation covering:**

1. **Configuration terms**: safe-outputs, permissions, tools, cache-memory, toolset, frontmatter
2. **Engine types**: copilot, claude, codex, custom
3. **Actions and commands**: compile, audit, logs, mcp, recompile
4. **GitHub concepts**: workflow_dispatch, pull_request, issues, discussions, permissions
5. **Repository-specific**: agentic workflows, gh-aw, activation, MCP servers
6. **File types and formats**: markdown, lockfile (.lock.yml), YAML
7. **Tool types**: edit, bash, github, playwright, web-fetch, web-search
8. **Operations**: fmt, lint, test-unit, timeout_minutes, runs-on

**Goal**: Create a list of approximately 100 terms (95-105 is acceptable) that are most relevant to this project, extracted from documentation only.

**Exclude tooling-specific terms**: Do not include makefile, Astro, or starlight as these are tooling-specific and not user-facing concepts.

### 2. Create the Dictation Instructions File

Create the file `.github/instructions/dictation.instructions.md` with the following structure (NO EXAMPLES, NO PLANNING GUIDELINES):

```markdown
Fix text-to-speech errors in dictated text for creating agentic workflow prompts in the gh-aw repository.

## Project Glossary

[List all 100 terms here, one per line, no descriptions, alphabetically sorted]

## Technical Context

GitHub Agentic Workflows (gh-aw) - a Go-based GitHub CLI extension for writing agentic workflows in natural language using markdown files with YAML frontmatter, which compile to GitHub Actions workflows.

## Fix Speech-to-Text Errors

Replace speech-to-text ambiguities with correct technical terms from the glossary:

- "ghaw" → "gh-aw"
- "work flow" → "workflow"
- "front matter" → "frontmatter"
- "lock file" → "lockfile" or ".lock.yml"
- "co-pilot" → "copilot"
- "safe outputs" → "safe-outputs"
- "cache memory" → "cache-memory"
- "max turns" → "max-turns"
- "issue comment" → "issue_comment"
- "pull request" → "pull_request"
- "workflow dispatch" → "workflow_dispatch"
- "runs on" → "runs-on"
- "timeout minutes" → "timeout_minutes"
```

### 3. Generate the Actual Content

Use your analysis of the documentation to populate the glossary with approximately 100 relevant terms (95-105 is acceptable). Scan documentation files thoroughly to identify the most important and frequently used terms.

### 4. Create Pull Request

After creating the dictation instructions file:

1. Use the safe-outputs create-pull-request to submit your changes
2. Title: "[docs] Update dictation prompt instructions"
3. Include a PR description that explains the changes made

## Important Guidelines

- **Extract from documentation**: Scan only docs/src/content/docs/**/*.md files for terms
- **Ignore tooling-specific terms**: Exclude makefile, Astro, and starlight (tooling-specific, not user-facing)
- **Be Thorough**: Scan all documentation files to extract accurate terms
- **Be Precise**: The glossary should contain approximately 100 terms (95-105 is acceptable)
- **Prioritize Relevance**: Include terms that are actually used frequently in documentation
- **Avoid Duplicates**: Each term should appear only once in the glossary
- **Alphabetize**: Sort the glossary alphabetically for easy reference
- **No Descriptions**: The glossary should be a simple list of terms, no definitions
- **No Examples**: Keep the instructions concise without example transformations
- **No Planning Guidelines**: Focus only on fixing speech-to-text errors, not on restructuring or planning tasks
- **Focus on Specificity**: Prefer project-specific terms over generic software terms

## Success Criteria

Your task is complete when:
1. ✅ The file `.github/instructions/dictation.instructions.md` exists
2. ✅ It contains approximately 100 project-specific terms in the glossary (95-105 is acceptable)
3. ✅ Terms are extracted from documentation files only (docs/src/content/docs/**/*.md)
4. ✅ Tooling-specific terms (makefile, Astro, starlight) are excluded
5. ✅ The instructions focus on fixing speech-to-text errors only
6. ✅ NO planning guidelines are included (sentence structure, tone adjustment, context enhancement)
7. ✅ NO examples are included (keep it short)
8. ✅ File description clearly states focus on fixing text-to-speech errors
9. ✅ A pull request has been created with the updated file
