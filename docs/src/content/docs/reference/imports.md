---
title: Imports
description: Learn how to modularize and reuse workflow components across multiple workflows using the imports field in frontmatter for better organization and maintainability.
sidebar:
  order: 325
---

Using imports in frontmatter or markdown allows you to modularize and reuse workflow components across multiple workflows.

## Syntax

Imports can be specified either in frontmatter or in markdown. In frontmatter the `imports:` field is used:

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

The `uses` key is an alias for `path`, and `with` is an alias for `inputs`. Both forms work interchangeably.

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

Workflows without an `on` field are shared workflow components. These files are validated but not compiled into GitHub Actions - they're meant to be imported by other workflows. The compiler skips them with an informative message, allowing you to organize reusable components without generating unnecessary lock files.

## Import Schema (`import-schema`)

Shared workflows can declare a typed parameter contract with `import-schema`. This allows callers to pass values via `with` that are validated at compile time and substituted into the shared file's frontmatter and markdown body before processing.

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

Use `${{ github.aw.import-inputs.<key> }}` to substitute a top-level value. For object sub-fields use dotted notation: `${{ github.aw.import-inputs.config.apiKey }}`.

Substitution is applied to both the YAML frontmatter and the markdown body of the shared file before any further parsing, so input values can drive any frontmatter field (e.g. `mcp-servers`, `runtimes`).

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

The compiler validates that all `required` fields are provided, that `choice` values are in `options`, that array element types match, and that object `properties` constraints are satisfied. Unknown keys produce a compile-time error.

## Path Formats

Import paths support local files (`shared/file.md`, `../file.md`), remote repositories (`owner/repo/file.md@v1.0.0`), and section references (`file.md#SectionName`). Optional imports use `{{#import? file.md}}` syntax in markdown.

Paths are resolved relative to the importing file, with support for nested imports and circular import protection.

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

Version references support semantic tags (`@v1.0.0`), branch names (`@main`, `@develop`), or commit SHAs for immutable references. See [Reusing Workflows](/gh-aw/guides/packaging-imports/) for installation and update workflows.

## Import Cache

Remote imports are cached in `.github/aw/imports/` to enable offline compilation. First compilation downloads and caches the import by commit SHA; subsequent compilations use the cached file. The cache is git-tracked with `.gitattributes` configured for conflict-free merges. Local imports are never cached.

## Agent Files

Import custom agent files to customize AI engine behavior. Agent files are markdown documents with specialized instructions that modify how the AI interprets and executes workflows. Agent files can be imported from local `.github/agents/` directories or from external repositories.

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

Remote agent imports support the same versioning as other imports:

- Semantic tags: `@v1.0.0`, `@v2.1.3`
- Branch names: `@main`, `@develop`
- Commit SHAs: `@abc123def456` (immutable references)

### Constraints

- **One agent per workflow**: Only one agent file can be imported per workflow (local or remote)
- **Agent path detection**: Files in `.github/agents/` directories are automatically recognized as agent files
- **Caching**: Remote agents are cached in `.github/aw/imports/` by commit SHA, enabling offline compilation

## Frontmatter Merging

Imported files can define specific frontmatter fields that merge with the main workflow's configuration. The merge behavior varies by field type and follows specific precedence rules detailed below.

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

### Merge Algorithm Overview

The compiler processes imports using **breadth-first search (BFS) traversal**. Direct imports are processed first, then their nested imports, preventing circular dependencies and ensuring deterministic ordering. Configurations accumulate during traversal and merge into the main workflow using field-specific rules.

### Field-Specific Merge Semantics

#### Tools (`tools:`)

Deep merge with array concatenation. New tool keys are added, duplicate keys trigger deep merge. `allowed` arrays concatenate and deduplicate. MCP tools detect conflicts except for `allowed` arrays.

```aw wrap
# main.md tools.bash.allowed: [write]
# import tools.bash.allowed: [read, list]
# Result: [read, list, write]
```

#### MCP Servers (`mcp-servers:`)

Imported servers override main workflow servers with the same name. Main workflow servers not defined in imports are kept. Multiple imports defining the same server use first-wins ordering.

#### Network Permissions (`network:`)

Union of `allowed` domains, deduplicated and sorted alphabetically. Network `mode` and `firewall` from main workflow take precedence.

#### Permissions (`permissions:`)

Validation only - imported permissions are not merged. Main workflow must explicitly declare all imported permissions with sufficient levels (`write` >= `read` >= `none`). Missing or insufficient permissions fail compilation.

#### Safe Outputs (`safe-outputs:`)

Each safe-output type can be defined once across all imports. Main workflow definitions override imported definitions for the same type. Multiple imports defining the same type fail compilation. Meta fields use first-wins merging (main > imports).

#### Runtimes (`runtimes:`)

Main workflow runtime versions override imported versions. Imported runtimes are used if not specified in main workflow.

#### Services (`services:`)

Service names must be unique across main and imports. Duplicate service names fail compilation. All services are available to workflow jobs.

#### Steps (`steps:`)

Imported steps are prepended to main workflow steps (imported first, then main). Action pinning applies to all steps. Steps from multiple imports concatenate in import order.

#### Jobs (`jobs:`)

The `jobs:` field in imported files is not merged. Custom jobs can only be defined in the main workflow's frontmatter. Use `safe-outputs.jobs` for importable job definitions.

#### Safe Output Jobs (`safe-outputs.jobs`)

Safe-job names must be unique across main workflow and all imports. Duplicate job names fail compilation. Job execution order is determined by `needs:` dependencies.

### Import Processing Order

Imports are processed in breadth-first order: direct imports first, then nested imports. Earlier imports in the main workflow's list take precedence. Circular imports are detected and prevented, ensuring deterministic results.

### Error Handling

**Circular imports**: Detected and prevented during compilation.

**Missing files**: Optional imports use `{{#import? file.md}}` to handle missing files gracefully. Required imports fail compilation if missing.

**Conflicts**: Multiple imports defining the same safe-output type fail compilation. Resolution: Define in main workflow (overrides imports) or remove from one import.

**Permission validation**: Insufficient permissions produce detailed error messages with suggested fixes.

### Performance Considerations

Remote imports are cached by commit SHA in `.github/aw/imports/`. Keep import chains shallow, use shared workflows for reusable configurations, and consolidate related imports. Every compilation records imports in the lock file manifest for dependency tracking.


## Using Imports in Repository Rulesets (`inlined-imports: true`)

When a workflow is configured as a **required status check** in a [repository ruleset](https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/about-rulesets), it runs in a restricted context that does not have access to other files in the repository. Shared files imported with the `imports:` field are loaded at runtime from the repository checkout, but this checkout is not available in the ruleset context, producing an error such as:

```
ERR_SYSTEM: Runtime import file not found: workflows/shared/file.md
```

The fix is to enable `inlined-imports: true` in your workflow frontmatter. This causes the compiler to bundle all imported content directly into the compiled `.lock.yml` at compile time, so no file system access is needed at runtime:

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
