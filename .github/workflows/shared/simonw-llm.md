---
engine:
  id: custom
  steps:
    - name: Install simonw/llm CLI
      run: |
        pip install llm llm-github-models
        llm --version
      env:
        GITHUB_AW_MCP_CONFIG: ${{ env.GITHUB_AW_MCP_CONFIG }}
    
    - name: Configure llm with API key
      run: |
        if [ -n "$OPENAI_API_KEY" ]; then
          echo "$OPENAI_API_KEY" | llm keys set openai
          echo "✓ OpenAI API key configured"
        elif [ -n "$ANTHROPIC_API_KEY" ]; then
          llm install llm-claude
          echo "$ANTHROPIC_API_KEY" | llm keys set claude
          echo "✓ Anthropic API key configured"
        elif [ -n "$GITHUB_TOKEN" ]; then
          # GitHub Models uses GITHUB_TOKEN by default, no key setup needed
          echo "✓ GitHub Models configured (using GITHUB_TOKEN)"
        else
          echo "⚠ Warning: No API key found. Please set OPENAI_API_KEY, ANTHROPIC_API_KEY, or ensure GITHUB_TOKEN is available"
          exit 1
        fi
      env:
        OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        GITHUB_AW_MCP_CONFIG: ${{ env.GITHUB_AW_MCP_CONFIG }}
    
    - name: Run llm CLI with prompt
      id: llm_execution
      run: |
        set -o pipefail
        
        # Determine which model to use based on available API keys
        if [ -n "$OPENAI_API_KEY" ]; then
          MODEL="gpt-4o-mini"
        elif [ -n "$ANTHROPIC_API_KEY" ]; then
          MODEL="claude-3-5-sonnet-20241022"
        elif [ -n "$GITHUB_TOKEN" ]; then
          MODEL="github/gpt-4o-mini"
        else
          echo "No API key configured"
          exit 1
        fi
        
        echo "Using model: $MODEL"
        
        # Run llm with the prompt from the file
        # Note: MCP configuration is available via GITHUB_AW_MCP_CONFIG if needed
        llm -m "$MODEL" "$(cat $GITHUB_AW_PROMPT)" 2>&1 | tee /tmp/gh-aw/llm-output.txt
        
        # Store output for safe-outputs processing if configured
        if [ -n "$GITHUB_AW_SAFE_OUTPUTS" ]; then
          cp /tmp/gh-aw/llm-output.txt "$GITHUB_AW_SAFE_OUTPUTS"
        fi
      env:
        GITHUB_AW_PROMPT: ${{ env.GITHUB_AW_PROMPT }}
        GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}
        GITHUB_AW_MCP_CONFIG: ${{ env.GITHUB_AW_MCP_CONFIG }}
        OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
        ANTHROPIC_API_KEY: ${{ secrets.ANTHROPIC_API_KEY }}
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

<!--
This shared configuration sets up a custom agentic engine using simonw/llm CLI.

**Usage:**
Include this file in your workflow using frontmatter imports:

```yaml
---
imports:
  - shared/simonw-llm.md
---
```

**Requirements:**
- The workflow requires `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, or `GITHUB_TOKEN` secret to be configured
- The llm CLI will be installed via pip along with the llm-github-models plugin
- If using Anthropic, the llm-claude plugin will be automatically installed
- GitHub Models uses the built-in GITHUB_TOKEN (no additional setup required)

**Model Selection:**
- With OpenAI API key: Uses `gpt-4o-mini` by default
- With Anthropic API key: Uses `claude-3-5-sonnet-20241022` by default
- With GitHub Token: Uses `github/gpt-4o-mini` by default (free via GitHub Models)

**API Key Setup:**
1. Go to your repository settings → Secrets and variables → Actions
2. Create a secret named one of:
   - `OPENAI_API_KEY` (for OpenAI models)
   - `ANTHROPIC_API_KEY` (for Anthropic Claude models)
   - Use the built-in `GITHUB_TOKEN` for GitHub Models (no setup needed, free tier available)
3. Set the value to your API key (not needed for GitHub Models)

**Note**: 
- This workflow requires internet access to install Python packages
- The llm CLI stores conversations in a local SQLite database
- Output is automatically captured for safe-outputs processing
- You can customize the model by modifying the MODEL variable in the run step
- MCP server configuration is available via GITHUB_AW_MCP_CONFIG environment variable for future compatibility
-->
