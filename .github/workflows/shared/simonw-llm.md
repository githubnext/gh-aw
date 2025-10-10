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
    
    - name: Configure llm with GitHub Models and MCP
      run: |
        # GitHub Models uses GITHUB_TOKEN by default, no key setup needed
        echo "✓ GitHub Models configured (using GITHUB_TOKEN)"
        
        # Configure MCP tools if MCP config is available
        if [ -n "$GITHUB_AW_MCP_CONFIG" ] && [ -f "$GITHUB_AW_MCP_CONFIG" ]; then
          # Create llm-tools-mcp config directory
          mkdir -p ~/.llm-tools-mcp
          
          # Copy MCP configuration to the expected location for llm-tools-mcp
          cp "$GITHUB_AW_MCP_CONFIG" ~/.llm-tools-mcp/mcp.json
          
          echo "✓ MCP configuration installed at ~/.llm-tools-mcp/mcp.json"
        else
          echo "ℹ No MCP configuration available"
        fi
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
        # Use -T MCP to enable MCP tools if configured
        if [ -f ~/.llm-tools-mcp/mcp.json ]; then
          echo "Running with MCP tools enabled"
          llm -m "$MODEL" -T MCP "$(cat $GITHUB_AW_PROMPT)" 2>&1 | tee /tmp/gh-aw/llm-output.txt
        else
          echo "Running without MCP tools"
          llm -m "$MODEL" "$(cat $GITHUB_AW_PROMPT)" 2>&1 | tee /tmp/gh-aw/llm-output.txt
        fi
        
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
- MCP configuration from GITHUB_AW_MCP_CONFIG is copied to `~/.llm-tools-mcp/mcp.json`
- Tools from MCP servers are automatically enabled via the `-T MCP` flag when MCP config is present
- MCP config file must be in the format expected by llm-tools-mcp (see https://github.com/VirtusLab/llm-tools-mcp)

**Note**: 
- This workflow requires internet access to install Python packages
- The llm CLI stores conversations in a local SQLite database
- Output is automatically captured for safe-outputs processing
- You can customize the model by modifying the MODEL variable in the run step
- GitHub Models provides free access to 30+ AI models
-->
