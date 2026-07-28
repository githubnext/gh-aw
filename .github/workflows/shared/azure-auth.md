---
description: Azure CLI OIDC re-authentication for agentic workflows.
import-schema:
  azure-client-id:
    type: string
    required: true
    description: Azure App (client) ID for OIDC authentication
  azure-tenant-id:
    type: string
    required: true
    description: Azure tenant ID

permissions:
  id-token: write

env:
  AZURE_CONFIG_DIR: /tmp/gh-aw/agent/.azure

network:
  allowed:
    - login.microsoftonline.com
    - management.azure.com

pre-steps:
  - name: Fetch Azure OIDC token
    id: azure-oidc
    run: |
      OIDC_TOKEN=$(curl -sS --max-time 30 \
        -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \
        "${ACTIONS_ID_TOKEN_REQUEST_URL}&audience=api://AzureADTokenExchange" \
        | jq -r '.value')
      if [ -z "$OIDC_TOKEN" ] || [ "$OIDC_TOKEN" = "null" ]; then
        echo "::error::Failed to obtain Azure OIDC token — ensure id-token: write permission is granted"
        exit 1
      fi
      echo "::add-mask::$OIDC_TOKEN"
      TOKEN_DIR="${RUNNER_TEMP:-/tmp}/gh-aw/azure"
      mkdir -p "$TOKEN_DIR"
      TOKEN_FILE=$(mktemp "$TOKEN_DIR/oidc-token.XXXXXX")
      chmod 600 "$TOKEN_FILE"
      printf '%s' "$OIDC_TOKEN" > "$TOKEN_FILE"
      echo "GH_AW_AZURE_OIDC_TOKEN_FILE=$TOKEN_FILE" >> "$GITHUB_ENV"

pre-agent-steps:
  - name: Re-authenticate Azure CLI with OIDC
    env:
      GH_AW_AZURE_CLIENT_ID: ${{ github.aw.import-inputs.azure-client-id }}
      GH_AW_AZURE_TENANT_ID: ${{ github.aw.import-inputs.azure-tenant-id }}
    run: |
      if [ -z "$GH_AW_AZURE_OIDC_TOKEN_FILE" ] || [ ! -f "$GH_AW_AZURE_OIDC_TOKEN_FILE" ]; then
        echo "::error::Azure OIDC token file not found — the Fetch Azure OIDC token step may have failed"
        exit 1
      fi
      mkdir -p "$AZURE_CONFIG_DIR"
      chmod 700 "$AZURE_CONFIG_DIR"
      cleanup() {
        rm -f "$GH_AW_AZURE_OIDC_TOKEN_FILE"
      }
      trap cleanup EXIT
      az login --service-principal \
        --username "$GH_AW_AZURE_CLIENT_ID" \
        --tenant "$GH_AW_AZURE_TENANT_ID" \
        --federated-token "$(cat "$GH_AW_AZURE_OIDC_TOKEN_FILE")" \
        --output none \
        --only-show-errors
      az account show --output table
---

<!--

## Azure CLI OIDC Re-Authentication

This shared component resolves the process-boundary authentication gap where an Azure CLI
login performed during the GitHub Actions runner setup is not available to the later agent
sandbox process.

### How It Works

1. **pre-steps**: Requests a short-lived OIDC token from the GitHub Actions token endpoint and
   writes it to a restricted file at `/tmp/gh-aw/azure/oidc-token.txt`.

2. **pre-agent-steps**: Reads the token and runs `az login --federated-token` in a writable
   `AZURE_CONFIG_DIR` (`/tmp/gh-aw/agent/.azure`). This ensures the agent sandbox inherits a
   valid Azure CLI session from a directory it can read.

3. The OIDC token file is deleted immediately after use.

### Required Permissions

The importing workflow must declare `id-token: write`:

```yaml
permissions:
  contents: read
  id-token: write
```

### Setup

1. Register a federated identity credential on your Azure app registration:
   - **Issuer**: `https://token.actions.githubusercontent.com`
   - **Subject**: matches the GitHub Actions OIDC subject for your repository and trigger
     (e.g. `repo:<org>/<repo>:ref:refs/heads/main` or `environment:<name>`)
   - **Audience**: `api://AzureADTokenExchange`

2. Add `shared/azure-auth.md` to your workflow's imports:

```yaml
---
permissions:
  contents: read
  id-token: write
imports:
  - uses: shared/azure-auth.md
    with:
      azure-client-id: "00000000-0000-0000-0000-000000000000"  # your app client ID
      azure-tenant-id: "00000000-0000-0000-0000-000000000000"  # your tenant ID
---
```

The `azure-client-id` and `azure-tenant-id` values are Azure application registration IDs,
not credentials. They can be hardcoded or read from GitHub Actions repository variables.

### Network Domains

This component adds `login.microsoftonline.com` and `management.azure.com` to the network
allow-list. Your workflow may need additional Azure service domains depending on what resources
the agent accesses.

### Usage with Azure MCP (`@azure/mcp`)

When the agent uses the `@azure/mcp` npm package via a command-based MCP server, the package
resolves credentials through `DefaultAzureCredential`. After this component authenticates the
Azure CLI, the `AzureCliCredential` provider in the chain will succeed.

Example workflow fragment:

```yaml
---
permissions:
  contents: read
  id-token: write
imports:
  - uses: shared/azure-auth.md
    with:
      azure-client-id: "00000000-0000-0000-0000-000000000000"  # your app client ID
      azure-tenant-id: "00000000-0000-0000-0000-000000000000"  # your tenant ID
mcp-servers:
  azure:
    command: npx
    args: ["-y", "@azure/mcp@latest", "server", "start", "--read-only"]
    allowed:
      - "subscription_list"
      - "subscription_get"
      - "group_list"
      - "group_get"
      - "resource_list"
      - "resource_get"
network:
  allowed:
    - defaults
    - "*.azure.com"
    - "*.microsoft.com"
    - "*.microsoftonline.com"
---
```

### Usage with Azure DevOps CLI

The Azure DevOps CLI (`az devops`) uses the same `AZURE_CONFIG_DIR` session, so it also
becomes available to the agent after this component runs.

-->
