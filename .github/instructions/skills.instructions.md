---
description: Using Skillz MCP Server with Docker
applyTo: ".github/workflows/*.md,.github/workflows/**/*.md"
---

# Skillz MCP Server Integration

Skillz is an MCP server that turns Claude-style skills (`SKILL.md` files plus optional resources) into callable tools for any MCP client. It discovers each skill, exposes instructions and resources, and can run bundled helper scripts.

**Repository**: https://github.com/intellectronica/skillz

> ⚠️ **Experimental proof-of-concept. Potentially unsafe. Treat skills like untrusted code and run in sandboxes/containers. Use at your own risk.**

## Quick Start

### Basic Docker Configuration

To use Skillz with GitHub Agentic Workflows, add it as an MCP server in your workflow frontmatter:

```aw
---
on: issues
engine: copilot
mcp-servers:
  skillz:
    container: "intellectronica/skillz"
    args:
      - "-v"
      - "/path/to/skills:/skillz"
      - "/skillz"
---

# Your workflow with skills

Use skills from the skillz server to accomplish tasks.
```

**Key points:**
- Replace `/path/to/skills` with the actual path to your skills directory
- The skills directory is mounted at `/skillz` inside the container
- Pass `/skillz` as the argument to tell skillz where to find skills

## Skills Directory Structure

Skillz looks for skills inside the root directory you provide (defaults to `~/.skillz`). Each skill lives in its own folder or zip archive that includes a `SKILL.md` file with YAML front matter.

### Example Directory Layout

```text
skills/
├── summarize-docs/
│   ├── SKILL.md
│   ├── summarize.py
│   └── prompts/example.txt
├── translate.zip
└── web-search/
    └── SKILL.md
```

### Skill Structure

Each skill folder must contain:
- **`SKILL.md`** - Required file with YAML frontmatter describing the skill
- **Helper scripts** - Optional Python, Node.js, or other scripts
- **Resources** - Optional datasets, examples, prompts, etc.

Example `SKILL.md`:

```markdown
---
name: summarize-docs
description: Summarize documentation files
---

# Document Summarization Skill

This skill helps summarize long documentation files.

Use the provided `summarize.py` script to process documents.
```

### Packaging Skills as Zip Files

Skills can be packaged as `.zip` archives:

```text
translate.zip
├── SKILL.md
└── helpers/
    └── translate.js
```

Or with a top-level directory:

```text
data-cleaner.zip
└── data-cleaner/
    ├── SKILL.md
    └── clean.py
```

## Skillz vs Claude Code Directory Structure

### Claude Code-Compatible Layout (Flat)

For compatibility with Claude Code, use a flat directory structure where every immediate subdirectory is a single skill:

```text
skills/
├── hello-world/
│   ├── SKILL.md
│   └── run.sh
└── summarize-text/
    ├── SKILL.md
    └── run.py
```

**Limitations**: No nested directories, no `.zip` files.

### Skillz-Only Layout (Flexible)

Skillz supports nested directories and `.zip` files:

```text
skills/
├── text-tools/
│   └── summarize-text/
│       ├── SKILL.md
│       └── run.py
└── image-processing.zip
```

**Note**: This layout is NOT compatible with Claude Code.

## Configuration Options

### Environment Variables

When using Docker, you can pass environment variables to skills:

```yaml
mcp-servers:
  skillz:
    container: "intellectronica/skillz"
    args:
      - "-v"
      - "/path/to/skills:/skillz"
      - "/skillz"
    env:
      API_KEY: "${{ secrets.SKILL_API_KEY }}"
```

### Verbose Logging

Enable verbose logging for debugging:

```yaml
mcp-servers:
  skillz:
    container: "intellectronica/skillz"
    args:
      - "-v"
      - "/path/to/skills:/skillz"
      - "/skillz"
      - "--verbose"
```

### Listing Available Skills

To verify which skills will be exposed before running your workflow:

```bash
docker run --rm -v /path/to/skills:/skillz intellectronica/skillz /skillz --list-skills
```

## Workflow Examples

### Issue Triage with Custom Skills

```aw
---
on:
  issues:
    types: [opened]
permissions:
  issues: write
engine: copilot
mcp-servers:
  skillz:
    container: "intellectronica/skillz"
    args:
      - "-v"
      - "${{ github.workspace }}/.github/skills:/skillz"
      - "/skillz"
---

# Issue Triage with Skills

Analyze issue #${{ github.event.issue.number }} using available skills.
Apply appropriate labels and provide helpful feedback.
```

### Research with Web Search Skill

```aw
---
on:
  workflow_dispatch:
permissions:
  contents: read
engine: copilot
mcp-servers:
  skillz:
    container: "intellectronica/skillz"
    args:
      - "-v"
      - "/home/runner/skills:/skillz"
      - "/skillz"
tools:
  web-search:
safe-outputs:
  create-issue:
    title-prefix: "[research] "
---

# Weekly Research

Use the web-search skill to research recent developments.
Create a summary issue with findings.
```

### Code Review with Analysis Skills

```aw
---
on:
  pull_request:
    types: [opened, synchronize]
permissions:
  pull-requests: write
engine: copilot
mcp-servers:
  skillz:
    container: "intellectronica/skillz"
    args:
      - "-v"
      - "${{ github.workspace }}/.github/skills:/skillz"
      - "/skillz"
safe-outputs:
  add-comment:
---

# Code Review Assistant

Use code analysis skills to review PR #${{ github.event.pull_request.number }}.
Provide constructive feedback as a comment.
```

## Additional Resources

- **Skillz Repository**: https://github.com/intellectronica/skillz
- **PyPI Package**: https://pypi.org/project/skillz/
- **Claude Skills Format**: https://github.com/anthropics/skills
- **MCP Protocol**: https://modelcontextprotocol.io/
