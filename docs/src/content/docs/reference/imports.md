---
title: Imports
description: Learn how to modularize and reuse workflow components across multiple workflows using the imports field in frontmatter for better organization and maintainability.
sidebar:
  order: 325
---

## Syntax

Use `imports:` in frontmatter or `{{#import ...}}` in markdown to share workflow components across multiple workflows.

```aw wrap
---
on: issues
engine: copilot
imports:
  - shared/common-tools.md
  - shared/mcp/tavily.md
---

# Your Workflow

Workflow instructions here...
```

### Parameterized imports (`uses`/`with`)

Shared workflows that declare an `import-schema` accept runtime parameters. Use the `uses`/`with` form to pass values:

```aw wrap
---
on: issues
engine: copilot
imports:
  - uses: shared/mcp/serena.md
    with:
      languages: ["go", "typescript"]
---
```

`uses` is an alias for `path`; `with` is an alias for `inputs`.

### Single-import constraint

A workflow file can appear at most once in an import graph. If the same file is imported more than once with identical `with` values it is silently deduplicated. Importing the same file with **different** `with` values is a compile-time error:

```
import conflict: 'shared/mcp/serena.md' is imported more than once with different 'with' values.
An imported workflow can only be imported once per workflow.
  Previous 'with': {"languages":["go"]}
  New 'with':      {"languages":["typescript"]}
```

In markdown, use the special `{{#import ...}}` directive:

```aw wrap
---
...
---

# Your Workflow

Workflow instructions here...

{{#import shared/common-tools.md}}
```

## Shared Workflow Components

Files without an `on` field are shared workflow components — validated but not compiled into GitHub Actions, only imported by other workflows. The compiler skips them with an informative message.

## Import Schema (`import-schema`)

Use `import-schema` to declare a typed parameter contract. Callers pass values via `with`; the compiler validates them and substitutes them into the shared file's frontmatter and body before processing.

```aw wrap
---
# shared/deploy.md — no 'on:' field, shared component only
import-schema:
  region:
    type: string
    required: true
  environment:
    type: choice
    options: [staging, production]
    required: true
  count:
    type: number
    default: 10
  languages:
    type: array
    items:
      type: string
    required: true
  config:
    type: object
    description: Configuration object
    properties:
      apiKey:
        type: string
        required: true
      timeout:
        type: number
        default: 30

mcp-servers:
  my-server:
    url: "https://example.com/mcp"
    allowed: ["*"]
---

Deploy ${{ github.aw.import-inputs.count }} items to ${{ github.aw.import-inputs.region }}.
API key: ${{ github.aw.import-inputs.config.apiKey }}.
Languages: ${{ github.aw.import-inputs.languages }}.
```

### Supported types

| Type | Description | Extra fields |
|------|-------------|--------------|
| `string` | Plain text value | — |
| `number` | Numeric value | — |
| `boolean` | `true`/`false` | — |
| `choice` | One of a fixed set of strings | `options: [...]` |
| `array` | Ordered list of values | `items.type` (element type) |
| `object` | Key/value map | `properties` (one level deep) |

Each field supports `required: true` and an optional `default` value.

### Accessing inputs in shared workflows

Use `${{ github.aw.import-inputs.<key> }}` to substitute a top-level value; use dotted notation for object sub-fields (e.g. `${{ github.aw.import-inputs.config.apiKey }}`). Substitution applies to both frontmatter and body, so inputs can drive any field such as `mcp-servers` or `runtimes`.

### Calling a parameterized shared workflow

```aw wrap
---
on: issues
engine: copilot
imports:
  - uses: shared/deploy.md
    with:
      region: us-east-1
      environment: staging
      count: 5
      languages: ["go", "typescript"]
      config:
        apiKey: my-secret-key
        timeout: 60
---
```

The compiler validates `required` fields, `choice` options, array element types, and object `properties`. Unknown keys are compile-time errors.

## Path Formats

Import paths support local files (`shared/file.md`, `../file.md`), remote repositories (`owner/repo/file.md@v1.0.0`), and section references (`file.md#SectionName`). Optional imports use `{{#import? file.md}}`. Paths are resolved relative to the importing file; nested and circular imports are supported.

## Remote Repository Imports

Import shared components from external repositories using the `owner/repo/path@ref` format:

```aw wrap
---
on: issues
engine: copilot
imports:
  - acme-org/shared-workflows/mcp/tavily.md@v1.0.0
  - acme-org/shared-workflows/tools/github-setup.md@main
---

# Issue Triage Workflow

Analyze incoming issues using imported tools and configurations.
```

Supported refs: semantic tags (`@v1.0.0`), branches (`@main`), or commit SHAs. See [Reusing Workflows](/gh-aw/guides/packaging-imports/) for installation and update workflows.

## Import Cache

Remote imports are cached in `.github/aw/imports/` by commit SHA, enabling offline compilation. The cache is git-tracked with `.gitattributes` for conflict-free merges. Local imports are never cached.

## Agent Files

Agent files are markdown documents in `.github/agents/` that add specialized instructions to the AI engine. Import them from your repository or from external repositories.

### Local Agent Imports

Import agent files from your repository's `.github/agents/` directory:

```yaml wrap
---
on: pull_request
engine: copilot
imports:
  - .github/agents/code-reviewer.md
---
```

### Remote Agent Imports

Import agent files from external repositories using the `owner/repo/path@ref` format:

```yaml wrap
---
on: pull_request
engine: copilot
imports:
  - githubnext/shared-agents/.github/agents/security-reviewer.md@v1.0.0
---

# PR Security Review

Analyze pull requests for security vulnerabilities using the shared security reviewer agent.
```

Remote agent imports support the same `@ref` versioning syntax as other remote imports.

### Constraints

- **One agent per workflow**: Only one agent file can be imported per workflow (local or remote)
- **Agent path detection**: Files in `.github/agents/` directories are automatically recognized as agent files
- **Caching**: Remote agents are cached in `.github/aw/imports/` by commit SHA, enabling offline compilation

## Frontmatter Merging

### Allowed Import Fields

Shared workflow files (without `on:` field) can define:

- `import-schema:` - Parameter schema for `with` validation and input substitution
- `tools:` - Tool configurations (bash, web-fetch, github, mcp-*, etc.)
- `mcp-servers:` - Model Context Protocol server configurations
- `services:` - Docker services for workflow execution
- `safe-outputs:` - Safe output handlers and configuration
- `mcp-scripts:` - MCP Scripts configurations
- `network:` - Network permission specifications
- `permissions:` - GitHub Actions permissions (validated, not merged)
- `runtimes:` - Runtime version overrides (node, python, go, etc.)
- `secret-masking:` - Secret masking steps

Agent files (`.github/agents/*.md`) can additionally define:

- `name` - Agent name
- `description` - Agent description

Other fields in imported files generate warnings and are ignored.

### Field-Specific Merge Semantics

Imports are processed using breadth-first traversal: direct imports first, then nested. Earlier imports in the list take precedence; circular imports fail at compile time.

| Field | Merge strategy |
|-------|---------------|
| `tools:` | Deep merge; `allowed` arrays concatenate and deduplicate. MCP tool conflicts fail except on `allowed` arrays. |
| `mcp-servers:` | Imported servers override same-named main servers; first-wins across imports. |
| `network:` | `allowed` domains union (deduped, sorted). Main `mode` and `firewall` take precedence. |
| `permissions:` | Validation only — not merged. Main must declare all imported permissions at sufficient levels (`write` ≥ `read` ≥ `none`). |
| `safe-outputs:` | Each type defined once; main overrides imports. Duplicate types across imports fail. |
| `runtimes:` | Main overrides imports; imported values fill in unspecified fields. |
| `services:` | All services merged; duplicate names fail compilation. |
| `steps:` | Imported steps prepended to main; concatenated in import order. |
| `jobs:` | Not merged — define only in the main workflow. Use `safe-outputs.jobs` for importable jobs. |
| `safe-outputs.jobs` | Names must be unique; duplicates fail. Order determined by `needs:` dependencies. |

Example — `tools.bash.allowed` merging:

```aw wrap
# main.md: [write]
# import:  [read, list]
# result:  [read, list, write]
```

### Importing Steps

Share reusable pre-execution steps — such as token rotation, environment setup, or gate checks — across multiple workflows by defining them in a shared file:

```aw title="shared/rotate-token.md" wrap
---
description: Shared token rotation setup
steps:
  - name: Rotate GitHub App token
    id: get-token
    uses: actions/create-github-app-token@v1
    with:
      app-id: ${{ vars.APP_ID }}
      private-key: ${{ secrets.APP_PRIVATE_KEY }}
---
```

Any workflow that imports this file gets the rotation step prepended before its own steps:

```aw title="my-workflow.md" wrap
---
on: issues
engine: copilot
imports:
  - shared/rotate-token.md
permissions:
  contents: read
  issues: write
steps:
  - name: Prepare context
    run: echo "context ready"
---

# My Workflow

Process the issue using the rotated token from the imported step.
```

Steps from imports run **before** steps defined in the main workflow, in import declaration order.

### Importing MCP Servers

Define an MCP server configuration once and import it wherever needed:

```aw title="shared/mcp/tavily.md" wrap
---
description: Tavily web search MCP server
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/mcp/?tavilyApiKey=${{ secrets.TAVILY_API_KEY }}"
    allowed: ["*"]
network:
  allowed:
    - mcp.tavily.com
---
```

Import it into any workflow that needs web search:

```aw title="research.md" wrap
---
on: issues
engine: copilot
imports:
  - shared/mcp/tavily.md
permissions:
  contents: read
  issues: write
---

# Research Workflow

Search the web for relevant information and summarize findings in the issue.
```

### Importing Top-level `jobs:`

Top-level `jobs:` defined in a shared workflow are merged into the importing workflow's compiled lock file. The job execution order is determined by `needs` entries — a shared job can run before or after other jobs in the final workflow:

```aw title="shared/build.md" wrap
---
description: Shared build job that compiles artifacts for the agent to inspect

jobs:
  build:
    runs-on: ubuntu-latest
    needs: [activation]
    outputs:
      artifact_name: ${{ steps.build.outputs.artifact_name }}
    steps:
      - uses: actions/checkout@v6
      - name: Build
        id: build
        run: |
          npm ci && npm run build
          echo "artifact_name=build-output" >> "$GITHUB_OUTPUT"
      - uses: actions/upload-artifact@v4
        with:
          name: build-output
          path: dist/

steps:
  - uses: actions/download-artifact@v4
    with:
      name: ${{ needs.build.outputs.artifact_name }}
      path: /tmp/build-output
---
```

Import it so the `build` job runs before the agent and its artifacts are available as pre-steps:

```aw title="my-workflow.md" wrap
---
on: pull_request
engine: copilot
imports:
  - shared/build.md
permissions:
  contents: read
  pull-requests: write
---

# Code Review Workflow

Review the build output in /tmp/build-output and suggest improvements.
```

In the compiled lock file the `build` job appears alongside `activation` and `agent` jobs, ordered according to each job's `needs` declarations.

### Importing Jobs via `safe-outputs.jobs`

Jobs defined under `safe-outputs:` can be shared across workflows. These jobs become callable MCP tools that the AI agent can invoke during execution:

```aw title="shared/notify.md" wrap
---
description: Shared notification job
safe-outputs:
  notify-slack:
    description: "Post a message to Slack"
    runs-on: ubuntu-latest
    output: "Notification sent"
    inputs:
      message:
        description: "Message to post"
        required: true
        type: string
    steps:
      - name: Post to Slack
        env:
          SLACK_WEBHOOK: ${{ secrets.SLACK_WEBHOOK_URL }}
        run: |
          curl -s -X POST "$SLACK_WEBHOOK" \
            -H "Content-Type: application/json" \
            -d "{\"text\":\"${{ inputs.message }}\"}"
---
```

Import and use it in multiple workflows:

```aw title="my-workflow.md" wrap
---
on: issues
engine: copilot
imports:
  - shared/notify.md
permissions:
  contents: read
  issues: write
---

# My Workflow

Process the issue. When done, use notify-slack to send a summary notification.
```

### Error Handling

- **Circular imports**: Detected at compile time.
- **Missing files**: Use `{{#import? file.md}}` for optional imports; required imports fail if missing.
- **Conflicts**: Duplicate safe-output types across imports fail — define in main workflow to override.
- **Permissions**: Insufficient permissions fail with detailed error messages.

Remote imports are cached by commit SHA in `.github/aw/imports/`. Keep import chains shallow and consolidate related imports; every compilation records imports in the lock file manifest.


## Using Imports in Repository Rulesets (`inlined-imports: true`)

When a workflow is configured as a **required status check** in a [repository ruleset](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets), it runs in a restricted context that does not have access to other files in the repository. Shared files imported with the `imports:` field are loaded at runtime from the repository checkout, but this checkout is not available in the ruleset context, producing an error such as:

```
ERR_SYSTEM: Runtime import file not found: workflows/shared/file.md
```

Set `inlined-imports: true` to bundle all imported content directly into the compiled `.lock.yml` at compile time, so no file system access is needed at runtime:

```aw wrap
---
on: pull_request
engine: copilot
inlined-imports: true
imports:
  - shared/common-tools.md
  - shared/security-setup.md
---

# My Workflow

Workflow instructions here.
```

After adding `inlined-imports: true`, recompile the workflow:

```bash
gh aw compile my-workflow
```

> [!NOTE]
> With `inlined-imports: true`, any change to an imported file requires recompiling the workflow to take effect. The compiled `.lock.yml` must be committed and pushed for the updated content to run.
>
> `inlined-imports: true` cannot be combined with agent file imports (`.github/agents/` files). If your workflow imports a custom agent file, remove it before enabling inlined imports.

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to managing workflow imports
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options reference
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Safe output configuration details
- [Network Configuration](/gh-aw/reference/network/) - Network permission management
