---
description: Reusability patterns for GitHub Agentic Workflows — shared components, imports, import-schema, and the gh aw add/update lifecycle
---

# Imports & Reusability Patterns

This guide covers how to build modular, reusable agentic workflows using the `imports:` system. Use it when you need to share tool configurations, prompt instructions, MCP servers, or safe-output jobs across multiple workflows.

## Why Reuse?

Duplicating frontmatter across workflows creates maintenance burdens: a change to a shared MCP server config, a common safe-output job, or a standard prompt fragment must be replicated everywhere. Instead, extract the shared piece into a **shared component** (a markdown file under `.github/workflows/shared/`) and `import:` it.

Benefits:
- **Single source of truth** for tool configs, prompts, and safe-output jobs
- **Consistent behaviour** — all consumers get the same update when the shared file changes
- **Smaller individual workflows** — each workflow declares only what makes it unique
- **Composable** — mix multiple imports to assemble complex behaviour from simple building blocks

---

## Shared Component File Structure

Shared components live in `.github/workflows/shared/` and follow the same markdown-with-frontmatter format as regular workflows. Only certain frontmatter fields are merged when imported:

| Imported field | Merge behaviour |
|---|---|
| `tools:` | Deep-merged with importing workflow |
| `mcp-servers:` | Deep-merged |
| `safe-outputs:` | Deep-merged |
| `env:` | Merged; duplicate keys are a compile error |
| `network:`, `permissions:`, `runtimes:`, `services:`, `cache:`, `features:` | Deep-merged |
| `github-app:`, `on.github-app:` | First-wins across imports |
| `steps:`, `pre-steps:`, `pre-agent-steps:`, `post-steps:` | Appended in import order |
| Markdown body | Appended as additional prompt instructions |

Fields not in this list (e.g. `on:`, `engine:`, `timeout-minutes:`) are **ignored** in imported files — only the importing workflow controls those.

### Minimal shared component

```markdown
---
tools:
  web-fetch:
---

Always cite your sources with a link when using web search results.
```

### Mixed frontmatter + instructions

```markdown
---
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/mcp/"
    env:
      TAVILY_API_KEY: "${{ secrets.TAVILY_API_KEY }}"
    allowed:
      - search
      - extract
---

<!--
Tavily Search MCP — provides real-time web search.
Required secrets: TAVILY_API_KEY
Usage: imports: [shared/mcp/tavily.md]
-->

When searching the web, prefer Tavily for up-to-date results.
Summarise sources and include links.
```

---

## The `imports:` Field

Add `imports:` to any workflow's frontmatter to pull in shared components:

```yaml
---
on:
  issues:
    types: [opened]
imports:
  - shared/reporting.md
  - shared/gh.md
  - shared/mcp/tavily.md
---
```

### String form (simple)

```yaml
imports:
  - shared/common-tools.md
  - shared/security-notice.md
  - copilot-setup-steps.yml    # Special: merges copilot-setup-steps job steps
```

### Object form with inputs

When a shared component defines an `import-schema:`, pass values with `with:` (or `inputs:`):

```yaml
imports:
  - uses: shared/repo-memory-standard.md
    with:
      branch-name: "memory/issue-triage"
      description: "Issue triage historical data"
  - path: shared/tool-setup.md
    with:
      environment: staging
    env:
      MY_OVERRIDE: "value"        # Optional: env vars for the import's context
    checkout: main                # Optional: ref to check out for this import
```

`with:` / `inputs:` values are accessible inside the shared file via `${{ github.aw.import-inputs.<name> }}`.

---

## The `import-schema:` Field

Define typed parameters that consuming workflows must (or may) provide. Use this when a shared component is parameterised — for example, a repo-memory component where the branch name varies per workflow.

```yaml
---
import-schema:
  branch-name:
    type: string
    required: true
    description: "Branch name for storage (e.g. memory/my-workflow)"
  max-items:
    type: number
    default: 50
    description: "Maximum items to retain"
  environment:
    type: choice
    options: [dev, staging, prod]
    required: true
    description: "Target deployment environment"

tools:
  repo-memory:
    branch-name: ${{ github.aw.import-inputs.branch-name }}
---
```

### Supported input types

| Type | Notes |
|---|---|
| `string` | Free-form text |
| `number` | Integer or float |
| `boolean` | `true` / `false` |
| `choice` | Enumerated values; must supply `options:` |
| `array` | List of values |
| `object` | One-level sub-fields: `${{ github.aw.import-inputs.<name>.<subkey> }}` |

### Accessing inputs inside the shared file

```yaml
---
import-schema:
  model-name:
    type: string
    default: "gpt-4o"
env:
  SELECTED_MODEL: ${{ github.aw.import-inputs.model-name }}
---

Use the model specified by the importing workflow.
```

---

## Refactoring Patterns

### Pattern 1 — Extract shared tool config

**Before** (duplicated in three workflows):

```yaml
# workflow-a.md, workflow-b.md, workflow-c.md — all repeat:
tools:
  web-fetch:
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/mcp/"
    env:
      TAVILY_API_KEY: "${{ secrets.TAVILY_API_KEY }}"
    allowed: [search, extract]
```

**After** — extract to `shared/mcp/tavily.md`:

```markdown
---
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/mcp/"
    env:
      TAVILY_API_KEY: "${{ secrets.TAVILY_API_KEY }}"
    allowed: [search, extract]
---
```

Each workflow imports it with a single line:

```yaml
imports:
  - shared/mcp/tavily.md
```

### Pattern 2 — Extract shared prompt instructions

Move boilerplate prompt sections (security notices, output formatting, citation guidelines) into shared instruction files:

```markdown
<!-- shared/keep-it-short.md -->
---
---

Keep all output concise. Use bullet points, not paragraphs.
Never repeat information already visible in the GitHub UI.
```

Import alongside other shared files:

```yaml
imports:
  - shared/mcp/tavily.md
  - shared/keep-it-short.md
  - shared/reporting.md
```

### Pattern 3 — Parameterise with `import-schema:`

When multiple workflows need the same component but with different values, add `import-schema:` instead of hardcoding:

```markdown
<!-- shared/jira-mcp.md -->
---
import-schema:
  project-key:
    type: string
    required: true
    description: "Jira project key (e.g. ENG, INFRA)"

mcp-servers:
  jira:
    container: "mcp/jira"
    version: "latest"
    env:
      JIRA_TOKEN: "${{ secrets.JIRA_TOKEN }}"
      JIRA_PROJECT: ${{ github.aw.import-inputs.project-key }}
    allowed: [search_issues, get_issue, list_sprints]
---

When referencing Jira issues, always include the issue key and a link.
```

Consume it with different project keys per workflow:

```yaml
imports:
  - uses: shared/jira-mcp.md
    with:
      project-key: "ENG"
```

### Pattern 4 — Compose multiple imports

Build complex workflows from focused, single-purpose shared components:

```yaml
---
on:
  schedule:
    - cron: "0 9 * * 1"   # weekly Monday 9am
imports:
  - shared/mcp/tavily.md          # web search
  - shared/gh.md                  # gh CLI tool
  - shared/reporting.md           # output formatting rules
  - shared/repo-memory-standard.md  # (with: branch-name, description)
  - uses: shared/repo-memory-standard.md
    with:
      branch-name: "memory/weekly-research"
      description: "Weekly research snapshots"
---

Conduct weekly research on ${{ github.repository }} dependencies...
```

### Pattern 5 — Shared safe-output jobs

Extract common safe-output job definitions (e.g. a Slack notification job used by multiple workflows) into a shared file:

```markdown
<!-- shared/slack-notify.md -->
---
import-schema:
  channel:
    type: string
    required: true
    description: "Slack channel name"

safe-outputs:
  jobs:
    send-slack-notification:
      description: "Post a message to Slack"
      runs-on: ubuntu-latest
      output: "Slack notification sent"
      inputs:
        message:
          description: "Message text"
          required: true
          type: string
      permissions:
        contents: read
      steps:
        - name: Post to Slack
          uses: actions/github-script@v7
          env:
            SLACK_TOKEN: "${{ secrets.SLACK_TOKEN }}"
            CHANNEL: ${{ github.aw.import-inputs.channel }}
          with:
            script: |
              // Read and process agent output...
---
```

Importing workflow:

```yaml
imports:
  - uses: shared/slack-notify.md
    with:
      channel: "#engineering-alerts"
```

---

## External Imports with `gh aw add` and `gh aw update`

Shared components can be published and consumed across repositories. The `gh aw add` / `gh aw update` commands manage this lifecycle.

### `gh aw add` — Install a shared component

```bash
gh aw add <workflow-url>
```

Fetches a remote shared component and stores it under `.github/aw/imports/`. The `source:` field in the downloaded file records the origin for future updates:

```bash
# Install from a GitHub URL
gh aw add https://github.com/org/agentics/blob/main/workflows/shared/reporting.md

# MCP equivalent (restricted environments)
Use the add tool with url: "https://github.com/org/agentics/blob/main/workflows/shared/reporting.md"
```

After adding, reference the installed component via its local path in `imports:`:

```yaml
imports:
  - .github/aw/imports/org/agentics/<sha>/workflows_shared_reporting.md
```

### `gh aw update` — Refresh all external imports

```bash
gh aw update
```

Checks the `source:` field of every file under `.github/aw/imports/` and downloads updates. If a file defines a `redirect:` field, `gh aw update` follows the new location and rewrites `source:` automatically.

```bash
# MCP equivalent
Use the update tool
```

### Supporting fields for publishable shared components

When creating a shared component that others will import via `gh aw add`:

```yaml
---
source: "org/agentics/workflows/shared/my-component.md@main"  # origin tracking
redirect: "org/agentics/workflows/shared/my-component-v2.md@main"  # forward to new location on update
resources:
  - shared/mcp/dependency.md   # co-located files fetched alongside this one
private: false                 # set true to prevent gh aw add from sharing this file
import-schema:
  # ...
---
```

### Recommended directory layout for shared components

```
.github/
└── workflows/
    ├── my-workflow.md              # Your workflow
    ├── my-workflow.lock.yml        # Compiled output (auto-generated)
    └── shared/
        ├── mcp/                    # MCP server wrappers
        │   ├── tavily.md
        │   ├── notion.md
        │   └── github-mcp-app.md
        ├── reporting.md            # Output formatting instructions
        ├── gh.md                   # gh CLI mcp-script tool
        ├── keep-it-short.md        # Brevity instructions
        └── repo-memory-standard.md # Parameterised repo-memory setup
.github/aw/
└── imports/                        # External imports installed via gh aw add
    └── org/repo/<sha>/
        └── workflows_shared_component.md
```

---

## Compile-Time Behaviour

All imports are resolved at **compile time** by default. The compiled `.lock.yml` contains macros that load import content at **runtime** from the checked-out repository files (so edits to shared `.md` files take effect on the next run without recompilation).

### Inlined imports

For workflows that act as **required status checks in repository rulesets**, use `inlined-imports: true` to bundle all imported content at compile time:

```yaml
---
inlined-imports: true
imports:
  - shared/security-notice.md
---
```

> ⚠️ `inlined-imports: true` cannot be combined with `.github/agents/` file imports.

### Recompile after adding or changing imports

Any change to the `imports:` list in the frontmatter requires recompilation:

```bash
gh aw compile <workflow-name>
```

Editing only the *body* of a shared `.md` file (not its frontmatter) takes effect at runtime without recompilation.

---

## Quick Checklist: Extracting a Shared Component

Use this when you spot duplication across two or more workflow files:

1. **Identify** the repeated frontmatter block or prompt section
2. **Create** `.github/workflows/shared/<name>.md` with the extracted content
3. **Parameterise** with `import-schema:` if values differ per workflow
4. **Replace** the duplicated block in each workflow with an `imports:` entry
5. **Recompile** affected workflows: `gh aw compile` (or `gh aw compile <name>`)
6. **Verify** with `gh aw compile --strict`
