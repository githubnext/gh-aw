---
description: Azure DevOps MCP server for agentic workflows.
import-schema:
  organization:
    type: string
    required: true
    description: Azure DevOps organization name (the subdomain in https://dev.azure.com/<organization>)

network:
  allowed:
    - "*.dev.azure.com"
    - "*.visualstudio.com"
    - "*.microsoftonline.com"

mcp-servers:
  azure-devops:
    url: "https://mcp.dev.azure.com/${{ github.aw.import-inputs.organization }}"
    headers:
      Authorization: "${{ secrets.ADO_MCP_AUTH_TOKEN }}"
    allowed:
      - "*"
---

<!--

## Azure DevOps MCP Server

This shared configuration provides the Azure DevOps MCP Server, exposing work items,
repositories, pipelines, and other Azure DevOps resources to the agent.

### Authentication

The server authenticates using a bearer token passed in the `Authorization` header.
Store the token as a repository secret named `ADO_MCP_AUTH_TOKEN`.

Obtain a token using one of:

- **Personal Access Token (PAT)**: Create a PAT at `https://dev.azure.com/<org>/_usersSettings/tokens`
  with the required scopes, then add it as the `ADO_MCP_AUTH_TOKEN` secret.

- **OIDC federated token** (recommended for GitHub Actions): Exchange the GitHub Actions OIDC
  token for an Azure DevOps access token using the Azure DevOps resource audience
  (`499b84ac-1321-427f-aa17-267ca6975798`), then write the result to `$GITHUB_ENV` as
  `ADO_MCP_AUTH_TOKEN` in a `pre-steps` block. Import `shared/azure-auth.md` alongside this
  component if the agent also needs the Azure CLI.

### Setup

1. Obtain an Azure DevOps access token (PAT or OIDC-derived) for your organization.

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

### Available Tools

Tool availability depends on the permissions granted to the access token and the
features enabled in your Azure DevOps organization. Common tools include work item
read and write operations, repository and pull request access, and pipeline operations.

Use `allowed: ["*"]` (the default) to expose all tools, or restrict to specific
tool names once you have confirmed which tools your organization's MCP endpoint exposes.

-->
