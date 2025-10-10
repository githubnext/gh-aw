---
engine:
  id: custom
  steps:
    - name: Install simonw/llm CLI
      run: |
        pip install llm llm-github-models llm-tools-mcp
        llm --version
      env:
        GITHUB_AW_MCP_CONFIG: ${{ env.GITHUB_AW_MCP_CONFIG }}
    
    - name: Configure llm with GitHub Models
      run: |
        # GitHub Models uses GITHUB_TOKEN by default, no key setup needed
        echo "✓ GitHub Models configured (using GITHUB_TOKEN)"
      env:
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        GITHUB_AW_MCP_CONFIG: ${{ env.GITHUB_AW_MCP_CONFIG }}
    
    - name: Run llm CLI with prompt
      id: llm_execution
      run: |
        set -o pipefail
        
        # Use GitHub Models
        MODEL="github/gpt-4o-mini"
        
        echo "Using model: $MODEL"
        
        # Run llm with the prompt from the file
        # MCP tools are available via llm-tools-mcp plugin
        llm -m "$MODEL" "$(cat $GITHUB_AW_PROMPT)" 2>&1 | tee /tmp/gh-aw/llm-output.txt
        
        # Store output for safe-outputs processing if configured
        if [ -n "$GITHUB_AW_SAFE_OUTPUTS" ]; then
          cp /tmp/gh-aw/llm-output.txt "$GITHUB_AW_SAFE_OUTPUTS"
        fi
      env:
        GITHUB_AW_PROMPT: ${{ env.GITHUB_AW_PROMPT }}
        GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}
        GITHUB_AW_MCP_CONFIG: ${{ env.GITHUB_AW_MCP_CONFIG }}
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

<!--
This shared configuration sets up a custom agentic engine using simonw/llm CLI with GitHub Models.

**Usage:**
Include this file in your workflow using frontmatter imports:

```yaml
---
imports:
  - shared/simonw-llm.md
---
```

**Requirements:**
- The workflow uses GitHub Models via the built-in GITHUB_TOKEN (no additional setup required)
- The llm CLI will be installed via pip along with:
  - llm-github-models: GitHub Models integration (free tier)
  - llm-tools-mcp: MCP server support for tool access

**Model:**
- Uses `github/gpt-4o-mini` by default (free via GitHub Models)

**MCP Tools:**
- The llm-tools-mcp plugin enables MCP server integration
- MCP configuration is available via GITHUB_AW_MCP_CONFIG environment variable
- Tools from MCP servers can be accessed using the `-T MCP` flag

**Note**: 
- This workflow requires internet access to install Python packages
- The llm CLI stores conversations in a local SQLite database
- Output is automatically captured for safe-outputs processing
- You can customize the model by modifying the MODEL variable in the run step
- GitHub Models provides free access to 30+ AI models
-->
