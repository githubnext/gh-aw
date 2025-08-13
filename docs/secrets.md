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
You need to define custom secrets in your repository or organization settings to enable usage of your chosen agentic processor. These secrets will be referenced in generated workflows.

### `ANTHROPIC_API_KEY`
- **Purpose**: Claude engine access
- **Required for**: `engine: claude` workflows
- **Setup**: Add to repository or organization secrets

### `OPENAI_API_KEY`
- **Purpose**: Codex and OpenAI-based engines
- **Required for**: `engine: codex` workflows
- **Setup**: Add to repository or organization secrets

### `GEMINI_API_KEY`
- **Purpose**: Gemini engine access
- **Required for**: `engine: gemini` workflows
- **Setup**: Add to repository or organization secrets

## Related Documentation

- [Commands](commands.md) - CLI commands for workflow management
- [Workflow Structure](workflow-structure.md) - Directory layout and organization
- [Frontmatter Options](frontmatter.md) - All configuration options
- [Tools Configuration](tools.md) - GitHub and other tools setup
- [MCPs](mcps.md) - Model Context Protocol setup and configuration
- [Include Directives](include-directives.md) - Modularizing workflows with includes
