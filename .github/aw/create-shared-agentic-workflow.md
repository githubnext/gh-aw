---
name: create-shared-agentic-workflow
description: Create shared agentic workflow components that wrap MCP servers using secure, reusable patterns.
disable-model-invocation: true
---

# Shared Agentic Workflow Designer

Create reusable shared workflow components under `.github/workflows/shared/`.

## Load These References First

- [github-agentic-workflows.md](github-agentic-workflows.md)
- [workflow-constraints.md](workflow-constraints.md)
- [shared-safe-jobs.md](shared-safe-jobs.md)
- [safe-outputs.md](safe-outputs.md)

Load these only when relevant:

- [campaign.md](campaign.md)
- [experiments.md](experiments.md)

## Core Rules

- prefer `container:`-based MCP servers
- pin versions when practical
- allow only read-only tools by default
- move writes to built-in safe outputs or custom safe-output jobs
- keep documentation in XML comments in the markdown body, not in frontmatter comments

## Ask First

Start by asking what MCP server or shared component the user wants to integrate.

Then gather:

- the server name
- the documentation URL or repository
- required secrets
- any expected write operations

## File Shape

Shared components are markdown files with frontmatter and an optional markdown body.

```yaml
---
mcp-servers:
  server-name:
    container: "registry/image"
    version: "tag"
    env:
      API_KEY: "${{ secrets.API_KEY }}"
    allowed:
      - read_tool
---
<!-- short documentation -->
```

## MCP Design Flow

1. research the server from the provided docs
2. prefer an official container image when available
3. identify required args, env vars, and mounts
4. create `.github/workflows/shared/<service>-mcp.md`
5. list required secrets clearly for the user
6. inspect available tools with `gh aw mcp inspect`
7. allow only the read-only tools by default
8. document excluded write tools and route them to safe outputs when needed

## Secret Guidance

When secrets are required, explicitly list:

- the secret name
- what it is used for
- where the user must configure it in GitHub Actions

## Tool Allowlist Guidance

- use `allowed:` with a specific list whenever possible
- exclude write tools from shared read-oriented components
- if writes are needed, describe the companion safe-output pattern rather than broadening MCP permissions

## Custom Write Behavior

When a component needs post-agent mutation logic:

- create a safe-output job
- follow [shared-safe-jobs.md](shared-safe-jobs.md)
- keep the schema explicit and typed

## Validation Loop

Use this loop until the component is valid:

```bash
gh aw compile <workflow-name> --strict
gh aw mcp inspect <workflow-name> --server <server-name> -v
```

Iterate on:

- image name and version
- env vars and secrets
- Docker args and mounts
- allowed tools

## Guidelines

- keep one shared file focused on one MCP server or one reusable concern
- prefer containers over raw commands for production use
- keep write access out of the shared component unless it is explicitly implemented as a safe-output job
- keep the generated instructions concise
