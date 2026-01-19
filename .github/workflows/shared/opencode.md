---
engine:
  id: custom
  env:
    GH_AW_AGENT_VERSION: "0.0.384"
    GH_AW_AGENT_MODEL: "gpt-5-mini"
  steps:
    - name: Install GitHub CLI and Copilot CLI extension
      run: |
        # GitHub CLI is already installed in ubuntu-latest runners
        # Install/upgrade Copilot CLI extension
        gh extension install github/copilot-cli --force || gh extension upgrade github/copilot-cli
        
        # Verify installation
        gh copilot --version
        
        # Install jq for JSON processing
        sudo apt-get update && sudo apt-get install -y jq
      env:
        GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
    
    - name: Configure Copilot CLI MCP servers
      run: |
        set -e
        
        # Create Copilot CLI config directory
        mkdir -p ~/.config/github-copilot
        
        # Check if MCP config exists
        if [ -n "$GH_AW_MCP_CONFIG" ] && [ -f "$GH_AW_MCP_CONFIG" ]; then
          echo "Found MCP configuration at: $GH_AW_MCP_CONFIG"
          
          # Copilot CLI uses the same MCP config format, so we can copy it directly
          # The config should already be in the correct format at $GH_AW_MCP_CONFIG
          cp "$GH_AW_MCP_CONFIG" ~/.config/github-copilot/mcp-config.json
          
          echo "✅ Copilot CLI MCP configuration created successfully"
          echo "Configuration contents:"
          cat ~/.config/github-copilot/mcp-config.json | jq .
        else
          echo "⚠️  No MCP config found - Copilot CLI will run without MCP tools"
        fi
      env:
        GH_AW_MCP_CONFIG: ${{ env.GH_AW_MCP_CONFIG }}
    
    - name: Run Copilot CLI
      id: opencode
      run: |
        # Read the prompt from file
        PROMPT="$(cat "$GH_AW_PROMPT")"
        
        # Run copilot CLI with the specified model
        # Use gh copilot suggest for code generation tasks
        echo "$PROMPT" | gh copilot suggest --model "${GH_AW_AGENT_MODEL}"
      env:
        GH_AW_AGENT_MODEL: ${{ env.GH_AW_AGENT_MODEL }}
        GH_AW_PROMPT: ${{ env.GH_AW_PROMPT }}
        GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
---

<!--
This shared configuration sets up a custom agentic engine using GitHub Copilot CLI (copilot-cli).

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
    GH_AW_AGENT_VERSION: "0.0.384"  # Copilot CLI version (via gh extension)
    GH_AW_AGENT_MODEL: "gpt-5"      # Use a different AI model
---
```

**Supported Models:**
Copilot CLI supports various AI models:
- **GPT-5 Series**: `gpt-5`, `gpt-5-mini`, `gpt-5.2`, `gpt-5.2-codex`
- **GPT-4 Series**: `gpt-4.1`
- **Claude Series**: `claude-sonnet-4.5`, `claude-haiku-4.5`, `claude-opus-4.5`
- **Gemini Series**: `gemini-2.5-pro`, `gemini-3-pro`, `gemini-3-flash`

**MCP Server Integration:**
Copilot CLI automatically integrates with MCP servers configured in your workflow:

1. **Automatic Configuration**: MCP servers defined in your workflow's `tools:` or `mcp-servers:` 
   sections are automatically configured for Copilot CLI
2. **Config Location**: MCP config is copied to `~/.config/github-copilot/mcp-config.json`
3. **Server Types Supported**:
   - `local` servers (stdio): Command-based MCP servers
   - `remote` servers (http): HTTP-based MCP servers
4. **Environment Variables**: All environment variables from MCP server configs are preserved

**Requirements:**
- GitHub CLI (`gh`) is pre-installed on ubuntu-latest runners
- Copilot CLI extension is installed via `gh extension install github/copilot-cli`
- `jq` is installed for JSON processing
- The prompt file is read directly in the Run step
- Output is captured in the agent log file

**Environment Variables:**
- `GH_AW_AGENT_VERSION`: Copilot CLI version (default: `0.0.384`)
- `GH_AW_AGENT_MODEL`: AI model name (default: `gpt-5-mini`)
- `GH_AW_MCP_CONFIG`: Path to MCP config JSON file (automatically set by gh-aw)
- `GITHUB_TOKEN`: Required for GitHub CLI authentication

**Note**: 
- Copilot CLI uses GitHub's authentication and the GITHUB_TOKEN secret
- The model can be customized by setting the `GH_AW_AGENT_MODEL` environment variable
- Copilot CLI supports multiple LLM providers through GitHub Copilot's model marketplace
- MCP servers are configured automatically if the workflow includes MCP tools (github, playwright, safe-outputs, etc.)
- Copilot CLI is installed as a GitHub CLI extension, so version updates happen through `gh extension upgrade`
-->
