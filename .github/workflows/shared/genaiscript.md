---
engine:
  id: custom
  env:
    GITHUB_AW_AGENT_VERSION: "1.140.0"
  steps:
    - name: Install GenAIScript
      run: npm install -g genaiscript@${GITHUB_AW_AGENT_VERSION} && genaiscript --version
      env:
        GITHUB_AW_AGENT_VERSION: ${{ env.GITHUB_AW_AGENT_VERSION }}
    
    - name: Convert prompt to GenAI format
      run: |
        if [ ! -f "$GITHUB_AW_PROMPT" ]; then
          echo "Error: Prompt file not found at $GITHUB_AW_PROMPT"
          exit 1
        fi
        echo '---' > /tmp/aw-prompts/prompt.genai.md
        echo 'model: gpt-4o-mini' >> /tmp/aw-prompts/prompt.genai.md
        echo '---' >> /tmp/aw-prompts/prompt.genai.md
        cat "$GITHUB_AW_PROMPT" >> /tmp/aw-prompts/prompt.genai.md
        echo "Generated GenAI prompt file"
      env:
        GITHUB_AW_PROMPT: ${{ env.GITHUB_AW_PROMPT }}
    
    - name: Run GenAIScript
      id: genaiscript
      run: |
        MCP_ARG=""
        if [ -f "$GITHUB_AW_MCP_CONFIG" ]; then
          echo "Using MCP configuration from $GITHUB_AW_MCP_CONFIG"
          MCP_ARG="--mcp-config $GITHUB_AW_MCP_CONFIG"
        fi
        genaiscript run /tmp/aw-prompts/prompt.genai.md $MCP_ARG --output /tmp/genaiscript-output.md || echo "GenAIScript completed"
        if [ -f /tmp/genaiscript-output.md ]; then
          cat /tmp/genaiscript-output.md > /tmp/agent-log.txt
          echo "GenAIScript execution completed"
        else
          echo "GenAIScript execution completed (no output)" > /tmp/agent-log.txt
        fi
      env:
        GITHUB_AW_PROMPT: ${{ env.GITHUB_AW_PROMPT }}
        GITHUB_AW_MCP_CONFIG: ${{ env.GITHUB_AW_MCP_CONFIG }}
---

<!--
This shared configuration sets up a custom agentic engine using microsoft/genaiscript.

**Usage:**
Include this file in your workflow to use genaiscript as the engine:

In your workflow file, use the import directive with this shared config.

**Requirements:**
- The workflow will install genaiscript npm package using version from `GITHUB_AW_AGENT_VERSION` env var
- The original prompt file is converted to GenAI markdown format (prompt.genai.md)
- GenAIScript is executed with MCP server configuration if available
- Output is captured in the agent log file

**Note**: 
- This workflow requires internet access to install npm packages
- The genaiscript version can be customized by setting the `GITHUB_AW_AGENT_VERSION` environment variable
- MCP server configuration is automatically passed if configured in the workflow
-->
