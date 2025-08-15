# 🔐 Secrets Management

Agentic workflows automatically handle several types of secrets and support custom secret references.

## Automatically Provided Secrets

These secrets are automatically available in all workflows:

### `GITHUB_TOKEN`
- **Purpose**: GitHub API access for repository operations
- **Scope**: Permissions defined in workflow `permissions:` section
- **Usage**: Automatically used by GitHub tools and MCP servers

```yaml
permissions:
  issues: write      # GITHUB_TOKEN gets issue write access
  contents: read     # GITHUB_TOKEN gets content read access
```

## User-Defined Secrets

You need to define custom secrets in your repository or organization settings to enable usage of your chosen agentic processor and external services.

### AI Engine Secrets

#### `ANTHROPIC_API_KEY`
- **Purpose**: Claude engine access
- **Required for**: `engine: claude` workflows (default)
- **Setup**: Add to repository or organization secrets
- **Usage**: Automatically used by Claude engine

#### `OPENAI_API_KEY`
- **Purpose**: Codex and OpenAI-based engines
- **Required for**: `engine: codex` workflows (experimental)
- **Setup**: Add to repository or organization secrets
- **Usage**: Automatically used by Codex engine

### MCP Server Secrets

Custom secrets for MCP servers are referenced using `${secrets.SECRET_NAME}` syntax:

```yaml
tools:
  trello:
    mcp:
      type: stdio
      command: "python"
      args: ["-m", "trello_mcp"]
      env:
        TRELLO_TOKEN: "${{ secrets.TRELLO_TOKEN }}"
        TRELLO_KEY: "${{ secrets.TRELLO_KEY }}"
    allowed: ["list_boards", "create_card"]
```

### Custom Environment Variables

You can define custom environment variables in the workflow frontmatter:

```yaml
env:
  NODE_ENV: "production"
  DEBUG: "false"
  CUSTOM_CONFIG: "${{ secrets.MY_CONFIG }}"
```

## Setting Up Secrets

### Repository Secrets
```bash
# Set secrets for a specific repository
gh secret set ANTHROPIC_API_KEY -a actions --body <your-api-key>
gh secret set TRELLO_TOKEN -a actions --body <your-trello-token>
```

### Organization Secrets
```bash
# Set secrets for all repositories in an organization
gh secret set ANTHROPIC_API_KEY -a actions --org <your-org> --body <your-api-key>
```

## Security Best Practices

### Secret Access
- Secrets are only accessible to workflows with appropriate permissions
- Secrets are automatically masked in workflow logs
- Use minimal scope secrets when possible

### Secret Naming
- Use UPPER_CASE names for secrets
- Use descriptive names that indicate the service: `TRELLO_TOKEN`, `SLACK_WEBHOOK`
- Avoid exposing secret values in frontmatter or markdown content

### MCP Server Security
```yaml
# ✅ Good: Secret reference
env:
  API_KEY: "${{ secrets.MY_API_KEY }}"

# ❌ Bad: Hardcoded value
env:
  API_KEY: "sk-1234567890abcdef"
```

## Related Documentation

- [Commands](commands.md) - CLI commands for workflow management
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [Frontmatter Options](frontmatter.md) - All configuration options
- [Tools Configuration](tools.md) - GitHub and other tools setup
- [MCPs](mcps.md) - Model Context Protocol setup and configuration
- [Include Directives](include-directives.md) - Modularizing workflows with includes
