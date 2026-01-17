---
title: Schema Design Philosophy
description: Security model and design philosophy behind main workflows and included files, explaining why certain properties are restricted and how to choose the right workflow type.
sidebar:
  order: 350
---

GitHub Agentic Workflows uses two distinct JSON schemas to validate workflow files: one for main workflows (entry points with triggers) and one for included files (reusable components). The schemas enforce different capabilities based on security and semantic requirements.

## Security Model

The schema architecture separates workflows by trust level and execution context:

### Main Workflows

Main workflows are **trusted entry points** that define when and how automation runs. They have full capabilities including:

- Defining workflow triggers (`on:` field)
- Executing custom commands (`engine.command`)
- Configuring GitHub Actions runtime
- Defining approval gates and manual controls
- Setting up sandbox environments

Main workflows are validated against `pkg/parser/schemas/main_workflow_schema.json`.

### Included Files

Included files are **reusable components** that provide configuration and instructions without controlling workflow execution. They have restricted capabilities to maintain security boundaries:

- Cannot define triggers (no `on:` field)
- Cannot execute custom commands (no `engine.command`)
- Cannot override sandbox configuration
- Cannot define custom jobs or GitHub Actions configuration

Included files are validated against `pkg/parser/schemas/included_file_schema.json`.

> [!NOTE]
> This separation ensures that imported components cannot subvert security controls or change workflow behavior in unexpected ways.

## Property Availability Matrix

The following table shows which properties are available in each schema type:

| Property | Main Workflow | Included File | Notes |
|----------|---------------|---------------|-------|
| **Common Properties (13)** | | | |
| `description` | ✅ | ✅ | Workflow description |
| `engine` | ✅ | ✅ | AI engine configuration |
| `mcp-servers` | ✅ | ✅ | MCP server definitions |
| `metadata` | ✅ | ✅ | Custom metadata key-value pairs |
| `network` | ✅ | ✅ | Network permissions |
| `permissions` | ✅ | ✅ | GitHub Actions permissions |
| `runtimes` | ✅ | ✅ | Runtime version overrides |
| `safe-inputs` | ✅ | ✅ | Custom input tool definitions |
| `safe-outputs` | ✅ | ✅ | Safe output handlers |
| `secret-masking` | ✅ | ✅ | Secret masking configuration |
| `services` | ✅ | ✅ | Docker services |
| `steps` | ✅ | ✅ | Workflow steps |
| `tools` | ✅ | ✅ | Tool configurations |
| **Main-Only Properties (25)** | | | |
| `bots` | ✅ | ❌ | Bot user filtering |
| `cache` | ✅ | ❌ | GitHub Actions cache |
| `command` | ✅ | ❌ | Custom engine command path |
| `concurrency` | ✅ | ❌ | Concurrency controls |
| `container` | ✅ | ❌ | Container configuration |
| `env` | ✅ | ❌ | Environment variables |
| `environment` | ✅ | ❌ | GitHub environment |
| `features` | ✅ | ❌ | Feature flags |
| `github-token` | ✅ | ❌ | GitHub token configuration |
| `if` | ✅ | ❌ | Conditional execution |
| `imports` | ✅ | ❌ | Import other files |
| `jobs` | ✅ | ❌ | Custom GitHub Actions jobs |
| `labels` | ✅ | ❌ | Workflow categorization |
| `name` | ✅ | ❌ | Workflow name |
| `on` | ✅ | ❌ | Workflow triggers |
| `post-steps` | ✅ | ❌ | Post-execution steps |
| `roles` | ✅ | ❌ | Role-based access control |
| `run-name` | ✅ | ❌ | GitHub Actions run name |
| `runs-on` | ✅ | ❌ | Runner specification |
| `sandbox` | ✅ | ❌ | Sandbox configuration |
| `source` | ✅ | ❌ | Source tracking |
| `strict` | ✅ | ❌ | Strict validation mode |
| `timeout-minutes` | ✅ | ❌ | Workflow timeout |
| `timeout_minutes` | ✅ | ❌ | Workflow timeout (deprecated) |
| `tracker-id` | ✅ | ❌ | External tracker ID |
| **Included-Only Properties (2)** | | | |
| `applyTo` | ❌ | ✅ | Target workflow filter |
| `inputs` | ❌ | ✅ | Parameterized inputs |

**Summary**: 38 properties in main workflows, 15 in included files, 13 common between both.

## Design Rationale

### Why Triggers (`on:`) are Main-Only

**Semantic Model**: Triggers define the entry point of workflow execution. Allowing included files to define triggers would create ambiguity about when workflows run.

**Security**: Entry points should be explicitly defined in the repository's main workflow files where they can be reviewed and audited.

**Design Decision**: Workflows without `on:` fields are automatically recognized as shared components and skipped during compilation, preventing generation of incomplete GitHub Actions workflows.

### Why Custom Commands (`engine.command`) are Main-Only

**Security**: The `engine.command` property allows specifying arbitrary executables to run as the AI engine. This capability must be restricted to main workflows where it can be properly reviewed.

**Attack Surface**: If included files could override the engine command, a malicious import could execute arbitrary code by pointing to a compromised binary.

**Safe Alternative**: Included files can still configure engine settings like `model`, `max-turns`, and `args`, but cannot change the fundamental execution path.

### Why Sandbox Configuration is Main-Only

**Isolation Boundary**: The `sandbox` property controls fundamental security boundaries like the MCP gateway and sandbox runtime environment. These settings define the trust model for the entire workflow.

**Consistency**: All imported components execute within the sandbox defined by the main workflow, ensuring consistent security policies.

**Configuration Scope**: Sandbox settings are workflow-wide and cannot be mixed between different imported files without creating security confusion.

### Why MCP Servers are Simplified in Included Files

While both main workflows and included files support `mcp-servers`, the merge behavior differs:

- **Main workflows**: Full control over all MCP configurations
- **Included files**: Can define MCP servers, but imported definitions take precedence during merge
- **Security**: This prevents included files from hijacking existing MCP server definitions
- **Flexibility**: Shared components can still provide MCP server configurations that work across multiple workflows

See [Imports](/gh-aw/reference/imports/#mcp-servers-mcp-servers) for detailed merge semantics.

## Usage Guidance

### When to Create a Main Workflow

Create a main workflow (`.github/workflows/*.md`) when you need:

- **Entry point**: Define when automation runs using `on:` triggers
- **Full control**: Access to all configuration options
- **Custom execution**: Override engine commands or sandbox settings
- **GitHub Actions**: Custom jobs, environments, or runner configurations
- **Standalone execution**: Workflow runs independently without imports

**Example use cases**:
- Issue responder triggered by issue events
- PR reviewer triggered by pull request events
- Scheduled maintenance workflows
- Manual approval workflows with `/command` triggers

### When to Create an Included File

Create an included file (`.github/workflows/shared/*.md` or similar) when you need:

- **Reusability**: Share configuration across multiple workflows
- **Modularity**: Organize related tools and settings together
- **Version control**: Maintain consistent configurations
- **Safe imports**: Provide functionality without security risks
- **Parameterization**: Use `inputs` field for configurable shared components

**Example use cases**:
- Common tool configurations (GitHub, web-fetch, bash permissions)
- MCP server definitions used by multiple workflows
- Standard network permission sets
- Reusable setup steps or safe-output configurations

### Migration Strategies

#### From Main to Included

If you have duplicate configuration in multiple main workflows:

1. Identify common properties (tools, mcp-servers, network, etc.)
2. Extract to shared file in `.github/workflows/shared/`
3. Remove `on:`, `name`, and other main-only properties
4. Add to main workflows using `imports:` field
5. Test compilation to ensure merge behavior is correct

#### From Included to Main

If an included file needs workflow-specific capabilities:

1. Copy included file to `.github/workflows/`
2. Add required `on:` trigger
3. Add `name` property for workflow identification
4. Add any needed main-only properties (sandbox, jobs, etc.)
5. Remove `applyTo` and `inputs` if present
6. Compile and test as standalone workflow

## Examples

### Valid Main Workflow

```aw wrap
---
name: Issue Responder
on:
  issues:
    types: [opened]

engine: copilot

permissions:
  issues: write
  contents: read

tools:
  github:
    toolsets: [issues]
  bash:
    allowed: [read, list]

network:
  allowed:
    - github.com

sandbox:
  mode: mcp-gateway
---

# Issue Responder

Read the newly opened issue and provide helpful resources based on the issue content.
```

### Valid Included File

```aw wrap
---
description: Common GitHub tools and permissions

tools:
  github:
    toolsets: [issues, pull_requests]
  bash:
    allowed: [read, list]
  web-fetch: {}

network:
  allowed:
    - github.com
    - api.github.com

permissions:
  contents: read
  issues: write
---

# Shared GitHub Configuration

This file provides standard GitHub tool configurations used across multiple workflows.
```

### Common Mistakes and Fixes

#### ❌ Mistake: Adding Triggers to Included File

```yaml wrap
# shared/common.md - INVALID
---
on: issues  # ERROR: 'on' is not allowed in included files
tools:
  github:
    toolsets: [issues]
---
```

**Fix**: Remove `on:` field. Included files are triggered through main workflows that import them.

#### ❌ Mistake: Using Custom Command in Included File

```yaml wrap
# shared/custom-engine.md - INVALID
---
engine:
  id: copilot
  command: /custom/copilot  # ERROR: 'command' not allowed in included files
---
```

**Fix**: Move engine command configuration to main workflow. Included files can only configure non-security-sensitive engine properties.

#### ❌ Mistake: Expecting Imported Permissions to Merge

```yaml wrap
# shared/perms.md
---
permissions:
  contents: read
  issues: write
---

# main.md - INVALID
---
on: issues
imports:
  - shared/perms.md
# ERROR: Permissions not automatically inherited
---
```

**Fix**: Explicitly declare all required permissions in main workflow. Imported permissions are validated, not merged.

```yaml wrap
# main.md - VALID
---
on: issues
imports:
  - shared/perms.md
permissions:
  contents: read
  issues: write
---
```

## Related Documentation

- [Imports](/gh-aw/reference/imports/) - Importing and merging workflow components
- [Frontmatter](/gh-aw/reference/frontmatter/) - Complete frontmatter reference
- [Workflow Structure](/gh-aw/reference/workflow-structure/) - Basic workflow organization
- [Engines](/gh-aw/reference/engines/) - AI engine configuration
- [Network](/gh-aw/reference/network/) - Network permission configuration
- [Sandbox](/gh-aw/reference/sandbox/) - Sandbox environment details
- [Compilation Process](/gh-aw/reference/compilation-process/) - How workflows are compiled and validated
