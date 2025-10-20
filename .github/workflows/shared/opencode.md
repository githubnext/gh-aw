---
engine:
  id: custom
  env:
    GH_AW_AGENT_VERSION: "0.1.0"
    GH_AW_AGENT_MODEL: "anthropic/claude-3-5-sonnet-20241022"
  steps:
    - name: Install OpenCode
      run: npm install -g opencode-ai@${GH_AW_AGENT_VERSION}
      env:
        GH_AW_AGENT_VERSION: ${{ env.GH_AW_AGENT_VERSION }}
    
    - name: Run OpenCode
      id: opencode
      run: |
        opencode run "$(cat "$GH_AW_PROMPT")" --model "${GH_AW_AGENT_MODEL}" --no-tui
      env:
        GH_AW_AGENT_MODEL: ${{ env.GH_AW_AGENT_MODEL }}
        GH_AW_PROMPT: ${{ env.GH_AW_PROMPT }}
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
---

<!--
This shared configuration sets up a custom agentic engine using sst/opencode.

**Usage:**
Include this file in your workflow using frontmatter imports:

```yaml
---
imports:
  - shared/opencode.md
---
```

**Customizing Configuration:**
You can override the default environment variables by setting them in your workflow:

```yaml
---
imports:
  - shared/opencode.md
engine:
  env:
    GH_AW_AGENT_VERSION: "0.2.0"  # Use a different OpenCode version
    GH_AW_AGENT_MODEL: "openai/gpt-4"  # Use a different AI model
---
```

**Requirements:**
- The workflow will install opencode-ai npm package using version from `GH_AW_AGENT_VERSION` env var
- The prompt file is read directly in the Run OpenCode step using command substitution
- OpenCode is executed in non-TUI mode with the specified model
- Output is captured in the agent log file

**Environment Variables:**
- `GH_AW_AGENT_VERSION`: OpenCode version (default: `0.1.0`)
- `GH_AW_AGENT_MODEL`: AI model in `provider/model` format (default: `anthropic/claude-3-5-sonnet-20241022`)
- `ANTHROPIC_API_KEY`: Required if using Anthropic models
- `OPENAI_API_KEY`: Required if using OpenAI models

**Note**: 
- This workflow requires internet access to install npm packages
- The opencode version can be customized by setting the `GH_AW_AGENT_VERSION` environment variable
- The AI model can be customized by setting the `GH_AW_AGENT_MODEL` environment variable
- OpenCode is provider-agnostic and supports multiple LLM providers
-->
