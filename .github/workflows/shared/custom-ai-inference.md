---
engine:
  id: custom
  steps:
    - name: Run AI Inference
      uses: actions/ai-inference@v1
      with:
        prompt: ${{ env.GITHUB_AW_PROMPT }}
        model: gpt-4o-mini
---

This shared configuration sets up a custom agentic engine using GitHub's AI inference action.

**Note**: When using this shared configuration, ensure your workflow includes `models: read` permission.
