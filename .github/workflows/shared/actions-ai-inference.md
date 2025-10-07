---
engine:
  id: custom
  steps:
    - name: Run AI Inference
      uses: actions/ai-inference@v1
      with:
        prompt-file: ${{ env.GITHUB_AW_PROMPT }}
        model: gpt-4o-mini
        enable-github-mcp: ${{ secrets.GITHUB_MCP_TOKEN != '' }}
        github-mcp-token: ${{ secrets.GITHUB_MCP_TOKEN }}
---

<!--
This shared configuration sets up a custom agentic engine using GitHub's AI inference action.

**Note**: When using this shared configuration, ensure your workflow includes `models: read` permission.

## GitHub MCP Integration (Optional)

This configuration supports optional GitHub MCP (Model Context Protocol) integration, which provides AI models with access to GitHub tools for repository management, issue tracking, and pull request operations.

### Enabling GitHub MCP

To enable GitHub MCP, configure a `GITHUB_MCP_TOKEN` secret in your repository:

1. Go to your repository settings → Secrets and variables → Actions
2. Create a new repository secret named `GITHUB_MCP_TOKEN`
3. Use a Personal Access Token (PAT) with appropriate permissions as the secret value

**Important**: The GitHub MCP integration requires a Personal Access Token (PAT) and cannot use the built-in `GITHUB_TOKEN`.

When the `GITHUB_MCP_TOKEN` secret is configured, the AI model will automatically have access to GitHub tools. If the secret is not configured, the workflow will function normally without MCP capabilities.
-->
