---
engine:
  id: custom
  steps:
    - name: Run AI Inference
      uses: actions/ai-inference@v1
      with:
        prompt-file: ${{ env.GITHUB_AW_PROMPT }}
        model: gpt-4o-mini
        enable-github-mcp: ${{ secrets.github_mcp_secret != '' }}
        github-mcp-token: ${{ secrets.github_mcp_secret }}
---

<!--
This shared configuration sets up a custom agentic engine using GitHub's AI inference action.

**Note**: When using this shared configuration, ensure your workflow includes `models: read` permission.
-->
