---
engine:
  id: custom
  env:
    GH_AW_AGENT_VERSION: "0.15.13"
    GH_AW_AGENT_MODEL: "anthropic/claude-sonnet-3.5"
  steps:
    - name: Install OpenCode and jq
      run: |
        npm install -g "opencode-ai@${GH_AW_AGENT_VERSION}"
        sudo apt-get update && sudo apt-get install -y jq
      env:
        GH_AW_AGENT_VERSION: ${{ env.GH_AW_AGENT_VERSION }}
    
    - name: Configure OpenCode MCP servers
      run: |
        set -e
        
        # Create OpenCode config directory
        mkdir -p ~/.config/opencode
        
        # Check if MCP config exists
        if [ -n "$GH_AW_MCP_CONFIG" ] && [ -f "$GH_AW_MCP_CONFIG" ]; then
          echo "Found MCP configuration at: $GH_AW_MCP_CONFIG"
          
          # Create base OpenCode config with proper schema
          echo "{\"\\$schema\": \"https://opencode.sh/schema.json\", \"mcp\": {}}" > ~/.config/opencode/opencode.json
          
          # Transform Copilot-style MCP config to OpenCode format
          jq -r '.mcpServers | to_entries[] | 
            if .value.type == "local" or (.value.command and .value.args) then
              {
                key: .key,
                value: {
                  type: "local",
                  command: (
                    if .value.command then
                      [.value.command] + (.value.args // [])
                    else
                      []
                    end
                  ),
                  enabled: true,
                  environment: (.value.env // {})
                }
              }
            elif .value.type == "http" then
              {
                key: .key,
                value: {
                  type: "remote",
                  url: .value.url,
                  enabled: true,
                  headers: (.value.headers // {})
                }
              }
            else empty end' "$GH_AW_MCP_CONFIG" | \
            jq -s 'reduce .[] as $item ({}; .[$item.key] = $item.value)' > /tmp/mcp-servers.json
          
          # Merge MCP servers into config
          jq --slurpfile servers /tmp/mcp-servers.json '.mcp = $servers[0]' \
            ~/.config/opencode/opencode.json > /tmp/opencode-final.json
          mv /tmp/opencode-final.json ~/.config/opencode/opencode.json
          
          echo "✅ OpenCode MCP configuration created successfully"
          echo "Configuration contents:"
          cat ~/.config/opencode/opencode.json | jq .
        else
          echo "⚠️  No MCP config found - OpenCode will run without MCP tools"
        fi
      env:
        GH_AW_MCP_CONFIG: ${{ env.GH_AW_MCP_CONFIG }}
    
    - name: Run OpenCode
      id: opencode
      run: |
        opencode run "$(cat "$GH_AW_PROMPT")" --model "${GH_AW_AGENT_MODEL}" --print-logs
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
    GH_AW_AGENT_VERSION: "0.15.13"  # Use a different OpenCode version
    GH_AW_AGENT_MODEL: "openai/gpt-4"  # Use a different AI model
---
```

**MCP Server Integration:**
OpenCode automatically integrates with MCP servers configured in your workflow:

1. **Automatic Configuration**: MCP servers defined in your workflow's `tools:` or `mcp-servers:` 
   sections are automatically configured for OpenCode
2. **Format Transformation**: The Copilot-style MCP config at `$GH_AW_MCP_CONFIG` is transformed 
   to OpenCode's format at `~/.config/opencode/opencode.json`
3. **Server Types Supported**:
   - `local` servers (stdio): Command and args are merged into a single command array
   - `remote` servers (http): URL and headers are preserved for HTTP-based MCP servers
4. **Environment Variables**: All environment variables from MCP server configs are preserved

**Requirements:**
- The workflow will install opencode-ai npm package using version from `GH_AW_AGENT_VERSION` env var
- `jq` is installed for JSON transformation between MCP config formats
- The prompt file is read directly in the Run OpenCode step using command substitution
- OpenCode is executed in non-interactive mode with logs printed to stderr
- Output is captured in the agent log file

**Environment Variables:**
- `GH_AW_AGENT_VERSION`: OpenCode version (default: `0.15.13`)
- `GH_AW_AGENT_MODEL`: AI model in `provider/model` format (default: `anthropic/claude-sonnet-3.5`)
- `GH_AW_MCP_CONFIG`: Path to MCP config JSON file (automatically set by gh-aw)
- `ANTHROPIC_API_KEY`: Required if using Anthropic models
- `OPENAI_API_KEY`: Required if using OpenAI models

**Note**: 
- This workflow requires internet access to install npm packages
- The opencode version can be customized by setting the `GH_AW_AGENT_VERSION` environment variable
- The AI model can be customized by setting the `GH_AW_AGENT_MODEL` environment variable
- OpenCode is provider-agnostic and supports multiple LLM providers
- MCP servers are configured automatically if the workflow includes MCP tools (github, playwright, safe-outputs, etc.)
-->
