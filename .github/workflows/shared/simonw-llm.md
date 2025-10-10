---
engine:
  id: custom
  steps:
    - name: Install simonw/llm CLI
      run: |
        pip install llm llm-github-models llm-tools-mcp
        llm --version
        
        # Show logs database path for debugging
        echo "LLM logs database: $(llm logs path)"
      env:
        GITHUB_AW_MCP_CONFIG: ${{ env.GITHUB_AW_MCP_CONFIG }}
        GITHUB_AW_PROMPT: ${{ env.GITHUB_AW_PROMPT }}
        GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}
    
    - name: Configure llm with GitHub Models and MCP
      run: |
        # GitHub Models uses GITHUB_TOKEN by default, no key setup needed
        echo "✓ GitHub Models configured (using GITHUB_TOKEN)"
        
        # Configure MCP tools if MCP config is available
        if [ -n "$GITHUB_AW_MCP_CONFIG" ] && [ -f "$GITHUB_AW_MCP_CONFIG" ]; then
          # Create llm-tools-mcp config directory
          mkdir -p ~/.llm-tools-mcp
          mkdir -p ~/.llm-tools-mcp/logs
          
          # Copy MCP configuration to the expected location for llm-tools-mcp
          cp "$GITHUB_AW_MCP_CONFIG" ~/.llm-tools-mcp/mcp.json
          
          echo "✓ MCP configuration installed at ~/.llm-tools-mcp/mcp.json"
          echo "📋 MCP configuration:"
          cat ~/.llm-tools-mcp/mcp.json
          echo ""
          
          # List available tools for debugging
          echo "🔧 Listing available MCP tools:"
          llm tools list || echo "⚠ Failed to list tools"
        else
          echo "ℹ No MCP configuration available"
        fi
      env:
        GITHUB_AW_MCP_CONFIG: ${{ env.GITHUB_AW_MCP_CONFIG }}
        GITHUB_AW_PROMPT: ${{ env.GITHUB_AW_PROMPT }}
        GITHUB_AW_SAFE_OUTPUTS: ${{ env.GITHUB_AW_SAFE_OUTPUTS }}
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
        LLM_TOOLS_MCP_FULL_ERRORS: "1"
    
    - name: Run llm CLI with prompt
      id: llm_execution
      run: |
        set -o pipefail
        
        # Use GitHub Models
        MODEL="github/gpt-4o-mini"
        
        echo "Using model: $MODEL"
        
        # Run llm with the prompt from the file
        # Use -T MCP to enable MCP tools if configured
        # Additional flags for debugging:
        #   --td: Show full details of tool executions
        #   -u: Show token usage
        if [ -f ~/.llm-tools-mcp/mcp.json ]; then
          echo "🚀 Running with MCP tools enabled (debug mode)"
          llm -m "$MODEL" -T MCP --td -u "$(cat $GITHUB_AW_PROMPT)" 2>&1 | tee /tmp/gh-aw/llm-output.txt
        else
          echo "Running without MCP tools"
          llm -m "$MODEL" -u "$(cat $GITHUB_AW_PROMPT)" 2>&1 | tee /tmp/gh-aw/llm-output.txt
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
        LLM_TOOLS_MCP_FULL_ERRORS: "1"
    
    - name: Upload MCP server logs
      if: always()
      uses: actions/upload-artifact@v4
      with:
        name: llm-mcp-logs
        path: ~/.llm-tools-mcp/logs/
        if-no-files-found: ignore
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

**Debugging and Logging:**
This configuration includes maximum debug tracing for troubleshooting MCP server issues:

1. **Environment Variables:**
   - `LLM_TOOLS_MCP_FULL_ERRORS=1` - Enables full error stack traces for MCP connection failures
   - Set on both "Configure llm" and "Run llm CLI" steps

2. **CLI Flags:**
   - `--td` (--tools-debug) - Shows full details of tool executions
   - `-u` (--usage) - Shows token usage information

3. **MCP Server Logs:**
   - MCP server connection logs are written to `~/.llm-tools-mcp/logs/`
   - These logs are uploaded as workflow artifacts (artifact name: `llm-mcp-logs`)
   - Each MCP server session creates a timestamped log file

4. **Diagnostic Output:**
   - MCP configuration is printed to stdout during setup
   - Available MCP tools are listed with `llm tools list`
   - LLM logs database path is displayed

5. **Log Locations:**
   - MCP server logs: `~/.llm-tools-mcp/logs/` (uploaded as artifacts)
   - LLM conversation logs: View with `llm logs path` command
   - Workflow output: `/tmp/gh-aw/llm-output.txt`

**Troubleshooting MCP Issues:**
If MCP servers fail to load:
1. Check the workflow run artifacts for `llm-mcp-logs`
2. Review the "Configure llm with GitHub Models and MCP" step output for configuration details
3. Check the "Run llm CLI" step output for tool execution details
4. Look for error messages with full stack traces (enabled by LLM_TOOLS_MCP_FULL_ERRORS)

**Note**: 
- This workflow requires internet access to install Python packages
- The llm CLI stores conversations in a local SQLite database
- Output is automatically captured for safe-outputs processing
- You can customize the model by modifying the MODEL variable in the run step
- GitHub Models provides free access to 30+ AI models
-->
