---
title: SpecOps
description: Maintain and propagate W3C-style specifications using agentic workflows
---

SpecOps is a pattern for maintaining formal specifications using agentic workflows. It leverages the [`w3c-specification-writer` agent](https://github.com/github/gh-aw/blob/main/.github/agents/w3c-specification-writer.agent.md) to create W3C-style specifications with RFC 2119 keywords (MUST, SHALL, SHOULD, MAY) and automatically propagates changes to consuming implementations across repositories.

## How SpecOps Works

1. **Update specification** — Trigger a workflow with the `w3c-specification-writer` agent to edit the spec document (RFC 2119 keywords, version bump, change log).
2. **Review changes** — Approve the specification pull request.
3. **Propagate automatically** — On merge, workflows detect updates and create PRs in consuming repositories (like [gh-aw-mcpg](https://github.com/github/gh-aw-mcpg)) to maintain compliance.
4. **Verify compliance** — Test generation workflows update compliance test suites against the new requirements.

## Update Specifications

Create a workflow to update specifications using the [`w3c-specification-writer` agent](https://github.com/github/gh-aw/blob/main/.github/agents/w3c-specification-writer.agent.md):

```yaml
---
name: Update MCP Gateway Spec
on:
  workflow_dispatch:
    inputs:
      change_description:
        description: 'What needs to change in the spec?'
        required: true
        type: string

engine: copilot
strict: true

safe-outputs:
  create-pull-request:
    title-prefix: "[spec] "
    labels: [documentation, specification]

tools:
  edit:
  bash:
---

# Specification Update Workflow

Update the MCP Gateway specification using the w3c-specification-writer agent.

**Change Request**: ${{ inputs.change_description }}

## Your Task

1. Review the current specification at `docs/src/content/docs/reference/mcp-gateway.md`

2. Apply the requested changes following W3C conventions:
   - Use RFC 2119 keywords (MUST, SHALL, SHOULD, MAY)
   - Update version number (major/minor/patch)
   - Add entry to Change Log section
   - Update Status of This Document if needed

3. Ensure changes maintain clear conformance requirements, testable specifications, and complete examples

4. Create a pull request with the updated specification
```

## Propagate Changes

After specification updates merge, automatically propagate changes to consuming repositories:

```yaml
---
name: Propagate Spec Changes
on:
  push:
    branches:
      - main
    paths:
      - 'docs/src/content/docs/reference/mcp-gateway.md'

engine: copilot
strict: true

safe-outputs:
  create-pull-request:
    title-prefix: "[spec-update] "
    labels: [dependencies, specification]

tools:
  github:
    toolsets: [repos, pull_requests]
  edit:
  bash:
---

# Specification Propagation Workflow

The MCP Gateway specification has been updated. Propagate changes to consuming repositories.

## Consuming Repositories

- **gh-aw-mcpg**: Update implementation compliance, schemas, and tests
- **gh-aw**: Update MCP gateway validation and documentation

## Your Task

1. Read the latest specification version and change log
2. Identify breaking changes and new requirements
3. For each consuming repository:
   - Update implementation to match spec
   - Run tests to verify compliance
   - Create pull request with changes
4. Create tracking issue linking all PRs
```

## Specification Structure

W3C-style specifications require: Abstract, Status, Introduction, Conformance, numbered technical sections with RFC 2119 keywords, Compliance testing, References, and a Change log.

**Example RFC 2119 usage**:
```markdown
## 3. Gateway Configuration

The gateway MUST validate all configuration fields before startup.
The gateway SHOULD log validation errors with field names.
The gateway MAY cache validated configurations.
```

See the [`w3c-specification-writer` agent](https://github.com/github/gh-aw/blob/main/.github/agents/w3c-specification-writer.agent.md) for a complete template and guidelines.

## Semantic Versioning

| Bump | When |
|------|------|
| **Major (X.0.0)** | Breaking changes |
| **Minor (0.Y.0)** | New features, backward-compatible |
| **Patch (0.0.Z)** | Bug fixes, clarifications |

The [MCP Gateway Specification](/gh-aw/reference/mcp-gateway/) is a live example — maintained by the `layout-spec-maintainer` workflow and implemented in [gh-aw-mcpg](https://github.com/github/gh-aw-mcpg).

## Related Patterns

- **[MultiRepoOps](/gh-aw/patterns/multi-repo-ops/)** — Cross-repository coordination
