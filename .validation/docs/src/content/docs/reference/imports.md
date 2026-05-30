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

In markdown, use `{{#runtime-import filepath}}` to inject the content of another file directly into the body at that position. This is useful for sharing reusable prompt snippets, tone instructions, or reference material across workflows.

```aw wrap
---
on: schedule

engine: copilot
---

{{#runtime-import .github/shared/editorial.md}}

# Daily Report

Generate the daily report.
```

Use `{{#runtime-import? filepath}}` to silently skip a missing file instead of failing:

```aw wrap
{{#runtime-import .github/shared/editorial.md}}    # required — fails if missing
{{#runtime-import? .github/shared/optional.md}}    # optional — skipped if missing
```

Paths are resolved within the `.github` folder. You can specify paths with or without the `.github/` prefix — both `.github/shared/editorial.md` and `shared/editorial.md` refer to the same file. See [Runtime Imports](/gh-aw/reference/templating/#runtime-imports) for URLs, line ranges, and security details.

## Shared Workflow Components

Files without an `on` field are shared workflow components — validated but not compiled into GitHub Actions, only imported by other workflows. Shared components may also define import-safe `on` keys (`skip-if-match`, `skip-if-no-match`, `skip-roles`, `skip-bots`, `github-token`, `github-app`) for reuse through imports.

### Common bundles

Use bundled shared components when you regularly import the same pair together:

```aw wrap
---
on:
  schedule: daily

engine: copilot

imports:
  - shared/reporting-otlp.md
---
```

`shared/reporting-otlp.md` combines `shared/reporting.md` and `shared/otlp.md` for telemetry-enabled reporting workflows.

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

## Path Resolution

Import paths are resolved using one of three modes depending on their format.

### Relative paths (default)

Paths that do not start with `.github/`, `/`, or an `owner/repo/` prefix are resolved relative to the importing workflow's directory. When compiling with the default `--dir` value, that directory is `.github/workflows/`.

```aw wrap
---
on: issues

engine: copilot

imports:
  - shared/common-tools.md        # → .github/workflows/shared/common-tools.md
  - ../agents/helper.md           # → .github/agents/helper.md (.. goes up from .github/workflows/)
---
```

### Repo-root-relative paths

Paths starting with `.github/` or `/` are resolved from the repository root. Absolute paths (`/`) must point inside `.github/` or `.agents/`; any other prefix is rejected at compile time for security.

```aw wrap
---
on: pull_request

engine: copilot

imports:
  - .github/agents/code-reviewer.md   # resolved from repo root
  - .github/workflows/shared/app.md   # resolved from repo root
---
```

This form is required when workflows in different directories need to import the same shared file using a stable path, and is the supported way to import files from the `.github/agents/` directory.

### Cross-repo imports

Paths matching `owner/repo/path@ref` are fetched from GitHub at compile time. The `@ref` suffix pins to a semantic tag (`@v1.0.0`), branch (`@main`), or commit SHA. Remote imports are cached in `.github/aw/imports/` by commit SHA, enabling offline compilation; local imports are never cached. See [Reusing Workflows](/gh-aw/guides/reusing-workflows/) for installation and update flows.

```aw wrap
---
on: issues

engine: copilot

imports:
  - acme-org/shared-workflows/shared/reporting.md@v2.1.0   # pinned to a tag
  - acme-org/shared-workflows/shared/tools.md@main         # track a branch
  - acme-org/shared-workflows/shared/helpers.md@abc1234    # locked to a SHA
---
```

### Section references and optional imports

Append `#SectionName` to any path to import a single section from a markdown file:

```
imports:
  - shared/tools.md#WebSearch
```

Use `?` after `import` to mark an import as optional — missing files are skipped silently instead of failing compilation. This applies to both frontmatter imports and body-level directives:

```yaml
# Frontmatter — optional
imports:
  - shared/optional-tools.md?
```

```aw wrap
# Body — optional content injection
{{#runtime-import? .github/shared/optional.md}}
```

## Agent Files

Agent files are markdown documents in `.github/agents/` that add specialized instructions to the AI engine. Import them as either local or remote paths — files under `.github/agents/` are automatically recognized as agent files, and only **one agent file** may be imported per workflow.

```yaml wrap
---
on: pull_request
engine: copilot
imports:
  - .github/agents/code-reviewer.md                                       # local
  - githubnext/shared-agents/.github/agents/security-reviewer.md@v1.0.0   # remote, pinned
---
```

Remote agent imports support the same `@ref` versioning and SHA-keyed caching as other remote imports.

## Frontmatter Merging

### Allowed Import Fields

Shared workflow files (without `on:` field) can define the fields below. Other fields generate warnings and are ignored. Agent files (`.github/agents/*.md`) may additionally define `name` and `description`.

| Field | Purpose |
|-------|---------|
| `import-schema` | Parameter schema for `with` validation and input substitution |
| `tools` | Tool configurations (`bash`, `web-fetch`, `github`, `mcp-*`, etc.) |
| `mcp-servers` | Model Context Protocol server configurations |
| `mcp-scripts` | MCP Scripts configurations |
| `services` | Docker services for workflow execution |
| `safe-outputs` | Safe output handlers and configuration |
| `network` | Network permission specifications |
| `permissions` | GitHub Actions permissions (validated, not merged) |
| `runtimes` | Runtime version overrides (node, python, go, etc.) |
| `secret-masking` | Secret masking steps |
| `env` | Workflow-level environment variables |
| `pre-agent-steps` | Steps that run after artifacts download, before engine execution |
| `post-steps` | Steps that run after engine execution |
| `github-app` | GitHub App credentials for token minting |
| `checkout` | Checkout configuration for the agent job |
| `engine.mcp` | MCP gateway settings (`tool-timeout`, `session-timeout`); engine identifier itself is always inherited from the importing workflow |

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
| `github-app:` | Main workflow's `github-app` takes precedence; first imported value fills in if main does not define one. |
| `checkout:` | Imported checkout entries are appended after the main workflow's entries. For duplicate (repository, path) pairs, the main workflow's entry takes precedence: first-seen wins for `ref`, and auth is mutually exclusive — once `github-token` or `github-app` is set by the main workflow, an imported duplicate cannot add the other auth method. `checkout: false` in the main workflow disables all checkout including imported entries. |
| `engine.mcp` | First-wins across imports. Shared files may define `engine:` with only `mcp.tool-timeout` and/or `mcp.session-timeout` (no engine identifier). The importing workflow's own engine setting always takes precedence; the first imported value fills in if the main workflow does not set a value. |
| `steps:` | Imported steps prepended to main; concatenated in import order. |
| `pre-agent-steps:` | Imported pre-agent-steps prepended to main; concatenated in import order. |
| `post-steps:` | Imported post-steps appended after main; concatenated in import order. |
| `jobs:` | Not merged — define only in the main workflow. Use `safe-outputs.jobs` for importable jobs. |
| `safe-outputs.jobs` | Names must be unique; duplicates fail. Order determined by `needs:` dependencies. |
| `env:` | Main workflow env vars take precedence over imports. Duplicate keys across different imports fail compilation — move to the main workflow to override imported values. |

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
      client-id: ${{ vars.APP_ID }}
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

Consumers import it with `imports: [shared/mcp/tavily.md]`.

### Importing MCP Gateway Settings

Shared workflow files can export `engine.mcp.tool-timeout` and `engine.mcp.session-timeout` without specifying an engine identifier — the engine itself is always inherited from the importing workflow.

```aw title="shared/mcp/slow-backend.md" wrap
---
description: MCP gateway settings for slow-backend MCP servers
engine:
  mcp:
    tool-timeout: 5m     # Allow up to 5 minutes per tool call
    session-timeout: 2h  # Keep MCP sessions alive for long-running workflows
---
```

The importing workflow's own `engine.mcp` settings take precedence. Among imports, the first file that declares a timeout wins for that setting.

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

Consumers import it with `imports: [shared/notify.md]` and instruct the agent to call `notify-slack` when appropriate.

## Self-Contained Lock Files (`inlined-imports: true`)

Setting `inlined-imports: true` embeds all imported content directly into the compiled `.lock.yml` at compile time. The resulting lock file is **self-contained** — it requires no file-system access or cross-repository checkout at runtime.

Enable it whenever runtime import resolution would fail:

- **Cross-organization `workflow_call`** — a trigger in Org A calling a workflow in Org B cannot check out Org B's `.github` folder with the caller's `GITHUB_TOKEN`, producing `fatal: repository '...' not found`.
- **Repository rulesets** — workflows used as a [required status check](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets) run in a restricted context that cannot access other files in the repo, producing `ERR_SYSTEM: Runtime import file not found`.

Both cases are solved by bundling imports into the lock file at compile time:

```aw wrap
---
on:
  workflow_call:

engine: copilot

inlined-imports: true

imports:
  - shared/common-tools.md
  - shared/security-setup.md
---

# Platform Gateway Workflow

Workflow instructions here.
```

After adding the flag, recompile:

```bash
gh aw compile my-workflow
```

**Trade-off**: the compiled `.lock.yml` is larger because imported content is embedded inline.

> [!NOTE]
> With `inlined-imports: true`, any change to an imported file requires recompiling the workflow to take effect. The compiled `.lock.yml` must be committed and pushed for the updated content to run.
>
> `inlined-imports: true` cannot be combined with agent file imports (`.github/agents/` files). If your workflow imports a custom agent file, remove it before enabling inlined imports.

## Related Documentation

- [Reusing Workflows](/gh-aw/guides/reusing-workflows/) - Adding and updating installed workflows with `gh aw add` and `gh aw update`
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options reference
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Safe output configuration details
- [Network Configuration](/gh-aw/reference/network/) - Network permission management
