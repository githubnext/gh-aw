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
-->
