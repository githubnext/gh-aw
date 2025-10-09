---
engine:
  id: custom
  env:
    GITHUB_AW_AGENT_VERSION: "2.5.1"
    GITHUB_AW_AGENT_MODEL_VERSION: "gpt-4o-mini"
  steps:
    - name: Install GenAIScript
      run: npm install -g genaiscript@${GITHUB_AW_AGENT_VERSION} && genaiscript --version
      env:
        GITHUB_AW_AGENT_VERSION: ${{ env.GITHUB_AW_AGENT_VERSION }}
    
    - name: Convert prompt to GenAI format
      run: |
        cp "$GITHUB_AW_PROMPT" /tmp/gh-aw/aw-prompts/prompt.genai.md
        sed -i '1i ---' /tmp/gh-aw/aw-prompts/prompt.genai.md
        sed -i "2i model: ${GITHUB_AW_AGENT_MODEL_VERSION}" /tmp/gh-aw/aw-prompts/prompt.genai.md
        sed -i '3i ---' /tmp/gh-aw/aw-prompts/prompt.genai.md
        echo "Generated GenAI prompt file"
      env:
        GITHUB_AW_PROMPT: ${{ env.GITHUB_AW_PROMPT }}
        GITHUB_AW_AGENT_MODEL_VERSION: ${{ env.GITHUB_AW_AGENT_MODEL_VERSION }}
    
    - name: Run GenAIScript
      id: genaiscript
      run: genaiscript run /tmp/gh-aw/aw-prompts/prompt.genai.md --mcp-config $GITHUB_AW_MCP_CONFIG --out /tmp/gh-aw/genaiscript-output.md
      env:
        DEBUG: genaiscript:*
        GITHUB_AW_PROMPT: ${{ env.GITHUB_AW_PROMPT }}
        GITHUB_AW_MCP_CONFIG: ${{ env.GITHUB_AW_MCP_CONFIG }}
---

<!--
This shared configuration sets up a custom agentic engine using microsoft/genaiscript.

**Usage:**
Include this file in your workflow using frontmatter imports:

```yaml
---
imports:
  - shared/genaiscript.md
---
```

**Requirements:**
- The workflow will install genaiscript npm package using version from `GITHUB_AW_AGENT_VERSION` env var
- The original prompt file is converted to GenAI markdown format (prompt.genai.md)
- GenAIScript is executed with MCP server configuration if available
- Output is captured in the agent log file

**Note**: 
- This workflow requires internet access to install npm packages
- The genaiscript version can be customized by setting the `GITHUB_AW_AGENT_VERSION` environment variable (default: `2.5.1`)
- The AI model can be customized by setting the `GITHUB_AW_AGENT_MODEL_VERSION` environment variable (default: `gpt-4o-mini`)
- MCP server configuration is automatically passed if configured in the workflow
-->
