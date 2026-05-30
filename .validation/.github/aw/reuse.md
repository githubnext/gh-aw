---
description: Imports, shared components, import-schema, and gh aw add/update for GitHub Agentic Workflows
---

# Imports & Reusability

Shared components eliminate duplication of tool configs, prompt instructions, MCP servers, and safe-output jobs across multiple workflows. Each consumer gets updates automatically when the shared file changes.

---

## Merged Fields

Only these frontmatter fields are merged when a file is imported:

| Field | Merge behaviour |
|---|---|
| `tools:`, `mcp-servers:`, `safe-outputs:`, `network:`, `permissions:`, `runtimes:`, `services:`, `cache:`, `features:` | Deep-merged |
| `env:` | Merged; duplicate keys → compile error |
| `github-app:`, `on.github-app:` | First-wins across imports |
| `steps:`, `pre-steps:`, `pre-agent-steps:`, `post-steps:` | Appended in import order |
| Markdown body | Appended as prompt instructions |

All other fields (`on:`, `engine:`, `timeout-minutes:`, …) are ignored in imported files.

---

## `imports:` Field

```yaml
# String form
imports:
  - shared/reporting.md
  - shared/mcp/tavily.md
  - copilot-setup-steps.yml   # merges copilot-setup-steps job steps

# Object form — pass values to import-schema:
imports:
  - uses: shared/repo-memory-standard.md
    with:
      branch-name: "memory/issue-triage"
      description: "Issue triage historical data"
  - path: shared/tool-setup.md
    with:
      environment: staging
    env:
      MY_OVERRIDE: "value"   # env vars for the import's context
    checkout: main            # ref to check out for this import
```

`with:` values are accessed inside the shared file as `${{ github.aw.import-inputs.<name> }}`.

---

## `import-schema:` Field

Declare typed parameters consumers must (or may) supply:

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

tools:
  repo-memory:
    branch-name: ${{ github.aw.import-inputs.branch-name }}
---
```

### Input types

| Type | Notes |
|---|---|
| `string` | Free-form text |
| `number` | Integer or float |
| `boolean` | `true` / `false` |
| `choice` | Enumerated; must supply `options:` |
| `array` | List of values |
| `object` | Sub-fields via `${{ github.aw.import-inputs.<name>.<subkey> }}` |

---

## Refactoring Patterns

### 1 — Extract shared MCP server / tool config

Create `.github/workflows/shared/mcp/tavily.md`:

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

Each workflow imports it with one line:

```yaml
imports:
  - shared/mcp/tavily.md
```

### 2 — Extract shared prompt instructions

```markdown
<!-- shared/keep-it-short.md -->
---
---

Keep all output concise. Use bullet points, not paragraphs.
Never repeat information already visible in the GitHub UI.
```

### 3 — Parameterise with `import-schema:`

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
```

```yaml
imports:
  - uses: shared/jira-mcp.md
    with:
      project-key: "ENG"
```

### 4 — Compose multiple imports

```yaml
---
on:
  schedule: weekly on monday
imports:
  - shared/mcp/tavily.md
  - shared/gh.md
  - shared/reporting.md
  - uses: shared/repo-memory-standard.md
    with:
      branch-name: "memory/weekly-research"
      description: "Weekly research snapshots"
---

Conduct weekly research on ${{ github.repository }} dependencies...
```

### 5 — Shared safe-output job

```markdown
<!-- shared/slack-notify.md -->
---
import-schema:
  channel:
    type: string
    required: true

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
              // post message to channel
---
```

```yaml
imports:
  - uses: shared/slack-notify.md
    with:
      channel: "#engineering-alerts"
```

---

## External Imports

### `gh aw add` — Install a remote shared component

```bash
gh aw add https://github.com/org/agentics/blob/main/workflows/shared/reporting.md
```

Downloaded files are stored under `.github/aw/imports/org/agentics/<sha>/`. Reference them in `imports:` by that local path. The `source:` field in the file tracks the origin for future updates.

MCP equivalent: `Use the add tool with url: "<url>"`

### `gh aw update` — Refresh all external imports

```bash
gh aw update
```

Re-fetches every file under `.github/aw/imports/` using its `source:` field. Follows `redirect:` and rewrites `source:` automatically.

MCP equivalent: `Use the update tool`

### Fields for publishable shared components

```yaml
---
source: "org/agentics/workflows/shared/my-component.md@main"
redirect: "org/agentics/workflows/shared/my-component-v2.md@main"
resources:
  - shared/mcp/dependency.md   # fetched alongside this file
private: false                 # true → prevent gh aw add from sharing
import-schema:
  # ...
---
```

---

## Recommended Directory Layout

```
.github/
└── workflows/
    ├── my-workflow.md
    ├── my-workflow.lock.yml        # auto-generated
    └── shared/
        ├── mcp/
        │   ├── tavily.md
        │   ├── notion.md
        │   └── github-mcp-app.md
        ├── reporting.md
        ├── gh.md
        ├── keep-it-short.md
        └── repo-memory-standard.md
.github/aw/
└── imports/                        # installed via gh aw add
    └── org/repo/<sha>/
        └── workflows_shared_component.md
```

---

## Compile-Time Behaviour

- Imports are resolved at **compile time**; the `.lock.yml` loads shared `.md` bodies at **runtime** — edits to shared files take effect on the next run without recompilation.
- **`inlined-imports: true`** — bundles all imported content at compile time (required for workflows used as repository ruleset status checks). Cannot be combined with `.github/agents/` file imports.
- Any change to the `imports:` list in frontmatter requires recompilation: `gh aw compile <workflow-name>`.
- Editing only the *body* of a shared `.md` file (not its frontmatter) does **not** require recompilation.

---

## Quick Checklist: Extracting a Shared Component

1. Identify the repeated frontmatter block or prompt section
2. Create `.github/workflows/shared/<name>.md` with the extracted content
3. Add `import-schema:` if values differ per consuming workflow
4. Replace the duplicated block in each workflow with an `imports:` entry
5. Recompile: `gh aw compile` (or `gh aw compile <name>`)
6. Verify: `gh aw compile --strict`
