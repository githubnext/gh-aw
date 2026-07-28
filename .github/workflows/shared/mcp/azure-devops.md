---
description: Azure DevOps MCP server for agentic workflows.
import-schema:
  organization:
    type: string
    required: true
    description: Azure DevOps organization name only (the subdomain in https://dev.azure.com/<organization>, not a full URL)

network:
  allowed:
    - "*.dev.azure.com"
    - "*.visualstudio.com"
    - "*.microsoftonline.com"
    - app.vssps.visualstudio.com

mcp-servers:
  azure-devops:
    url: "https://mcp.dev.azure.com/${{ github.aw.import-inputs.organization }}"
    headers:
      Authorization: "****** secrets.ADO_MCP_AUTH_TOKEN }}"
    allowed:
      - "*"
---

<!--

## Azure DevOps MCP Server

> **Experimental:** Azure DevOps MCP support is still experimental. Interfaces,
> defaults, and required configuration may change in future releases.

This shared configuration provides the Azure DevOps MCP Server, exposing work items,
repositories, pipelines, and other Azure DevOps resources to the agent.

### Authentication

The server authenticates using a bearer token passed in the `Authorization` header.
Store the raw token as a repository secret named `ADO_MCP_AUTH_TOKEN`; this
component prefixes it with `Bearer`.

Obtain a token using one of:

- **OIDC federated token** (recommended for GitHub Actions): Exchange the GitHub Actions OIDC
  token for an Azure DevOps access token using the Azure DevOps resource audience
  (`499b84ac-1321-427f-aa17-267ca6975798`), then write the result to `$GITHUB_ENV` as
  `ADO_MCP_AUTH_TOKEN` in a `pre-steps` block. Import `shared/azure-auth.md` alongside this
  component if the agent also needs the Azure CLI.

- **Microsoft Entra access token** from your own identity platform flow, saved as
  `ADO_MCP_AUTH_TOKEN`.

### Setup

1. Obtain an Azure DevOps bearer token (for example OIDC-derived) for your organization.

2. Add the token as a repository secret named `ADO_MCP_AUTH_TOKEN`.

3. Import this component in your workflow:

   ```yaml
   ---
   imports:
     - uses: shared/mcp/azure-devops.md
       with:
         organization: my-org
   ---
   ```

### Network Access

This component adds the following domains to the network allow-list:
- `*.dev.azure.com`
- `*.visualstudio.com`
- `*.microsoftonline.com`
- `app.vssps.visualstudio.com`

### Available Tools

Tool availability depends on the permissions granted to the access token and the
features enabled in your Azure DevOps organization. Common tools include work item
read and write operations, repository and pull request access, and pipeline operations.

Use `allowed: ["*"]` (the default) to expose all tools, or restrict to specific
tool names once you have confirmed which tools your organization's MCP endpoint exposes.

-->
