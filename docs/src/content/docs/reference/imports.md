---
title: Imports
description: Learn how to modularize and reuse workflow components across multiple workflows using compile-time imports in frontmatter and markdown for better organization and maintainability.
sidebar:
  order: 325
---

Compile-time imports in frontmatter or markdown allow you to modularize and reuse workflow components across multiple workflows. Imports are processed during compilation and merged into the final workflow configuration.

For runtime content inclusion (files/URLs loaded during workflow execution), see [Runtime Imports in Templating](/gh-aw/reference/templating/#runtime-imports).

## Syntax

Compile-time imports can be specified either in frontmatter or in markdown:

### Frontmatter Imports

Use the `imports:` field to declare compile-time imports:

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

### Markdown Import Macros

Use the `{{#import}}` and `{{#import?}}` macros for compile-time imports in markdown:

```aw wrap
---
on: issues
engine: copilot
---

# Your Workflow

Workflow instructions here...

{{#import shared/common-tools.md}}
{{#import? optional-config.md}}
```

Both syntaxes process imports during compilation, merging frontmatter configurations and including markdown content into the final workflow.

## Shared Workflow Components

Workflows without an `on` field are shared workflow components. These files are validated but not compiled into GitHub Actions - they're meant to be imported by other workflows. The compiler skips them with an informative message, allowing you to organize reusable components without generating unnecessary lock files.

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

Version references support semantic tags (`@v1.0.0`), branch names (`@main`, `@develop`), or commit SHAs for immutable references. See [Packaging & Distribution](/gh-aw/guides/packaging-imports/) for installation and update workflows.

## Import Cache

Remote imports are cached in `.github/aw/imports/` to enable offline compilation. First compilation downloads and caches the import by commit SHA; subsequent compilations use the cached file. The cache is git-tracked with `.gitattributes` configured for conflict-free merges. Local imports are never cached.

## Agent Files

Import custom agent files from `.github/agents/` to customize AI engine behavior. Agent files are markdown documents with specialized instructions that modify how the AI interprets and executes workflows.

```yaml wrap
---
on: pull_request
engine: copilot
imports:
  - .github/agents/code-reviewer.md
---
```

Only one agent file can be imported per workflow.

## Frontmatter Merging

Imported files can define specific frontmatter fields that merge with the main workflow's configuration. The merge behavior varies by field type and follows specific precedence rules detailed below.

### Allowed Import Fields

Shared workflow files (without `on:` field) can define:
- `tools:` - Tool configurations (bash, web-fetch, github, mcp-*, etc.)
- `mcp-servers:` - Model Context Protocol server configurations
- `services:` - Docker services for workflow execution
- `safe-outputs:` - Safe output handlers and configuration
- `safe-inputs:` - Safe input configurations
- `network:` - Network permission specifications
- `permissions:` - GitHub Actions permissions (validated, not merged)
- `runtimes:` - Runtime version overrides (node, python, go, etc.)
- `secret-masking:` - Secret masking steps

Agent files (`.github/agents/*.md`) can additionally define:
- `name` - Agent name
- `description` - Agent description

Other fields in imported files generate warnings and are ignored.

### Merge Algorithm Overview

The compiler processes imports using a **breadth-first search (BFS) traversal** to ensure deterministic ordering and cycle detection:

1. **Queue initialization**: Main workflow's imports are added to processing queue
2. **BFS traversal**: Each import is processed in order
3. **Nested imports**: Imported files' own imports are added to queue
4. **Cycle detection**: Prevents circular import chains
5. **Configuration merging**: Accumulated configurations are merged into main workflow
6. **Validation**: Final configuration is validated for conflicts and requirements

**Visual representation of BFS processing**:

```
Processing Queue (left to right):
┌─────────────┐   ┌─────────────┐   ┌─────────────┐   ┌─────────────┐
│  import-a   │ → │  import-b   │ → │  nested-1   │ → │  nested-2   │
│  (direct)   │   │  (direct)   │   │  (from a)   │   │  (from b)   │
└─────────────┘   └─────────────┘   └─────────────┘   └─────────────┘
      ↓                  ↓                  ↓                  ↓
   Parse &           Parse &           Parse &           Parse &
   Extract           Extract           Extract           Extract
      ↓                  ↓                  ↓                  ↓
   Add to            Add to            Add to            Add to
   Merged            Merged            Merged            Merged
   Config            Config            Config            Config
```

**Data flow through merge stages**:

```
Main Workflow      Import-A         Import-B         Final Config
─────────────      ────────         ────────         ────────────
tools:             tools:           tools:           tools:
  bash:              github:          web-fetch:       bash:
    allowed: [w]       toolsets: [i]    {}               allowed: [w,r]
                       │                                github:
                       └─ bash:                           toolsets: [i]
                            allowed: [r]                web-fetch: {}
                            
mcp-servers:       mcp-servers:                      mcp-servers:
  local:             remote:                           remote:
    url: x             url: y                            url: y (override)
                                                       local:
                                                         url: x

network:           network:         network:         network:
  allowed:           allowed:         allowed:         allowed:
    - github.com       - api.com        - cdn.com        - api.com
                                                         - cdn.com
                                                         - github.com
```

### Field-Specific Merge Semantics

#### Tools (`tools:`)

**Merge Strategy**: Deep merge with array concatenation and deduplication

```aw wrap
# shared/tools.md
---
tools:
  bash:
    allowed: [read, list]
  github:
    toolsets: [issues]
---

# main.md
---
on: issues
engine: copilot
imports:
  - shared/tools.md
tools:
  bash:
    allowed: [write]  # Merges with imported, becomes [read, list, write]
  web-fetch: {}       # Added alongside imported tools
---
```

**Merge rules**:
- New tool keys are added
- Duplicate tool keys trigger deep merge
- `allowed` arrays are concatenated and deduplicated
- Map properties are recursively merged
- MCP tools detect conflicts (except for `allowed` arrays)

#### MCP Servers (`mcp-servers:`)

**Merge Strategy**: Imported servers override top-level servers with same name

```aw wrap
# shared/mcp.md
---
mcp-servers:
  tavily:
    url: "https://mcp.tavily.com/..."
    allowed: ["*"]
---

# main.md
---
on: issues
imports:
  - shared/mcp.md
mcp-servers:
  tavily:  # Imported definition takes precedence
    # This definition is discarded
    url: "https://old-url.com"
  github:  # New server, kept
    mode: remote
---
```

**Merge rules**:
- Imported MCP servers **override** top-level servers with identical names
- Main workflow servers are kept if not defined in imports
- Multiple imports defining same server use first-wins ordering

#### Network Permissions (`network:`)

**Merge Strategy**: Union of allowed domains, deduplicated and sorted

```aw wrap
# shared/network.md
---
network:
  allowed:
    - example.com
    - api.example.com
---

# main.md
---
on: issues
imports:
  - shared/network.md
network:
  allowed:
    - github.com
    - example.com  # Deduplicated
  firewall: true
---
# Result: allowed = [api.example.com, example.com, github.com] (sorted)
```

**Merge rules**:
- All `allowed` domains are accumulated across main and imports
- Duplicates are removed
- Final list is sorted alphabetically for consistency
- Network `mode` and `firewall` from main workflow take precedence

#### Permissions (`permissions:`)

**Merge Strategy**: Validation only - main workflow must satisfy imported requirements

```aw wrap
# shared/permissions.md
---
permissions:
  issues: write
  contents: read
---

# main.md
---
on: issues
imports:
  - shared/permissions.md
permissions:
  issues: write    # Satisfies imported requirement
  contents: write  # write >= read, satisfies imported requirement
  actions: read    # Additional permission, allowed
---
```

**Validation rules**:
- Imported permissions are **not merged** into main workflow
- Main workflow must explicitly declare all imported permissions
- Permission levels must be sufficient: `write` >= `read` >= `none`
- Missing or insufficient permissions cause compilation to fail with detailed error

#### Safe Outputs (`safe-outputs:`)

**Merge Strategy**: Main workflow overrides imported types; conflicts fail compilation

```aw wrap
# shared/outputs.md
---
safe-outputs:
  create-issue:
    title-prefix: "[shared] "
    labels: [imported]
  add-comment:
    max: 3
---

# main.md
---
on: issues
imports:
  - shared/outputs.md
safe-outputs:
  create-issue:  # Main workflow overrides imported definition
    title-prefix: "[main] "
    labels: [local]
  # add-comment is inherited from import
---
```

**Merge rules**:
- Each safe-output type (create-issue, add-comment, etc.) can be defined **once** across all imports
- Main workflow definitions **override** imported definitions for same type
- Multiple imports defining same type cause compilation error
- Meta fields (allowed-domains, staged, env, github-token, max-patch-size, runs-on) use first-wins merging (main > imports)
- Messages configuration merges at field level with main taking precedence
- Jobs are merged separately with conflict detection

#### Runtimes (`runtimes:`)

**Merge Strategy**: Main workflow versions override imported versions

```aw wrap
# shared/runtimes.md
---
runtimes:
  node:
    version: "18"
  python:
    version: "3.11"
---

# main.md
---
on: issues
imports:
  - shared/runtimes.md
runtimes:
  node:
    version: "20"  # Overrides imported version
  # python 3.11 is inherited
---
```

**Merge rules**:
- Runtime versions from main workflow override imported versions
- Imported runtimes are used if not specified in main workflow
- Each runtime can only have one version in final configuration

#### Services (`services:`)

**Merge Strategy**: Deep merge with conflict detection

```aw wrap
# shared/services.md
---
services:
  redis:
    image: redis:alpine
    ports: [6379:6379]
---

# main.md
---
on: issues
imports:
  - shared/services.md
services:
  postgres:
    image: postgres:15
    ports: [5432:5432]
---
# Result: Both redis and postgres services available
```

**Merge rules**:
- Service names must be unique across main and imports
- Duplicate service names cause compilation error
- All services are available to workflow jobs

#### Steps (`steps:`)

**Merge Strategy**: Array prepend - imported steps run before main workflow steps

```aw wrap
# shared/setup.md
---
steps:
  - name: Configure environment
    run: echo "Setting up environment"
  - uses: actions/checkout@v4
---

# main.md
---
on: issues
imports:
  - shared/setup.md
steps:
  - name: Run custom action
    run: echo "Main workflow step"
---
# Result: Imported steps run first, then main workflow steps
```

**Merge rules**:
- Imported steps are **prepended** to main workflow steps
- Steps execute in order: imported steps → main workflow steps
- Action pinning is applied to all steps (both imported and main)
- Steps from multiple imports are concatenated in import order

#### Jobs (`jobs:`)

**Merge Strategy**: No merging - jobs field is not importable

```yaml wrap
# shared/jobs.md
---
jobs:
  notify:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Notification"
---

# main.md
---
on: issues
imports:
  - shared/jobs.md  # jobs field is ignored
jobs:
  build:
    runs-on: ubuntu-latest
---
```

**Important**: The `jobs:` field in imported files is **not merged**. Custom jobs can only be defined in the main workflow's frontmatter. Use `safe-outputs.jobs` for importable job definitions.

#### Safe Output Jobs (`safe-outputs.jobs`)

**Merge Strategy**: Conflict detection - job names must be unique

```aw wrap
# shared/notification.md
---
safe-outputs:
  jobs:
    notify:
      name: Send notification
      steps:
        - run: echo "Notifying team"
---

# main.md
---
on: issues
imports:
  - shared/notification.md
safe-outputs:
  jobs:
    cleanup:
      name: Cleanup resources
      steps:
        - run: echo "Cleaning up"
---
# Result: Both notify and cleanup jobs available
```

**Merge rules**:
- Safe-job names must be **unique** across main workflow and all imports
- Duplicate job names fail compilation with clear error
- Main workflow jobs and imported jobs are both included
- Job execution order determined by `needs:` dependencies
- Each safe-job can access safe-output artifacts and GitHub token

### Import Processing Order

Imports are processed in **breadth-first order** to ensure consistent and predictable merging:

```
Main Workflow
├── import-a.md (processed 1st)
│   ├── nested-1.md (processed 3rd)
│   └── nested-2.md (processed 4th)
└── import-b.md (processed 2nd)
    └── nested-3.md (processed 5th)
```

This ordering ensures:
- Earlier imports in the main workflow's list take precedence over later ones
- Nested imports are processed after their parent import
- Circular imports are detected and prevented
- Deterministic results regardless of file system order

### Error Handling and Edge Cases

#### Circular Import Detection

The compiler detects and prevents circular import chains:

```aw wrap
# workflow-a.md imports workflow-b.md
# workflow-b.md imports workflow-a.md
# Result: ERROR - Circular import detected
```

#### Missing Import Files

Optional imports use `?` suffix to gracefully handle missing files:

```markdown wrap
{{#import? optional-config.md}}  # No error if file doesn't exist
```

Required imports (frontmatter or without `?`) fail compilation if file is missing.

#### Conflicting Configurations

Some conflicts cause compilation to fail:

```aw wrap
# import-1.md defines safe-outputs: create-issue
# import-2.md also defines safe-outputs: create-issue
# Result: ERROR - Safe output type defined in multiple imports
```

To resolve: Define the configuration in main workflow (overrides both imports) or remove from one import.

#### Permission Validation Failures

Insufficient permissions produce detailed error messages:

```
ERROR: Imported workflows require permissions that are not granted.

Missing permissions:
  - actions: read

Insufficient permissions:
  - contents: has read, requires write

Suggested fix:
permissions:
  actions: read
  contents: write
```

### Performance Considerations

**Import Caching**: Remote imports are cached in `.github/aw/imports/` by commit SHA. First compilation downloads the file; subsequent compilations use the cache even if referencing the same commit via different refs (tags, branches).

**Compilation Time**: Each import adds parsing overhead. For optimal performance:
- Keep import chains shallow (avoid deep nesting)
- Use shared workflows for genuinely reusable configurations
- Consider consolidating related imports into single files

**Manifest Generation**: Every compilation records imported files in the lock file's manifest section, enabling accurate dependency tracking.

## Common Pitfalls and Best Practices

### ❌ Avoid: Conflicting Safe Output Definitions

```yaml wrap
# shared-1.md
safe-outputs:
  create-issue:
    title-prefix: "[bot] "

# shared-2.md  
safe-outputs:
  create-issue:  # ERROR: Conflict!
    labels: [automated]

# main.md
imports:
  - shared-1.md
  - shared-2.md
```

**Fix**: Define in main workflow (overrides both) or consolidate into one shared file.

### ✅ Best Practice: Layer Configurations by Scope

```yaml wrap
# shared/base-tools.md - Core tools everyone needs
tools:
  github:
    toolsets: [repos, issues]
  bash:
    allowed: [read, list]

# shared/advanced-tools.md - Extended capabilities
imports:
  - base-tools.md  # Nested import
tools:
  web-fetch: {}
  web-search: {}

# workflow.md - Specific needs
imports:
  - shared/advanced-tools.md
tools:
  bash:
    allowed: [write]  # Extends base
```

### ❌ Avoid: Assuming Permission Inheritance

```yaml wrap
# shared.md
permissions:
  contents: read
  issues: write

# main.md
on: issues
imports:
  - shared.md
# ERROR: Permissions not automatically inherited
```

**Fix**: Explicitly declare all required permissions in main workflow:

```yaml wrap
on: issues
imports:
  - shared.md
permissions:
  contents: read
  issues: write
```

### ✅ Best Practice: Use Semantic Versioning for Stability

```yaml wrap
# Development workflow - uses latest
imports:
  - acme-org/shared/tools.md@main

# Production workflow - uses stable version
imports:
  - acme-org/shared/tools.md@v2.1.0
```

### ❌ Avoid: Deeply Nested Import Chains

```yaml wrap
# Creates brittle dependency chain
workflow.md → shared-a.md → shared-b.md → shared-c.md → shared-d.md
```

**Fix**: Flatten structure and use direct imports:

```yaml wrap
workflow.md → shared-tools.md
            → shared-config.md
            → shared-mcp.md
```

## Related Documentation

- [Packaging and Updating](/gh-aw/guides/packaging-imports/) - Complete guide to managing workflow imports
- [Frontmatter](/gh-aw/reference/frontmatter/) - Configuration options reference
- [MCPs](/gh-aw/guides/mcps/) - Model Context Protocol setup
- [Safe Outputs](/gh-aw/reference/safe-outputs/) - Safe output configuration details
- [Network Configuration](/gh-aw/reference/network/) - Network permission management
