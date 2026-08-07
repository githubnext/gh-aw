---
engine:
  id: opencode
  version: "1.2.14"
  display-name: OpenCode
  description: OpenCode CLI with headless mode and multi-provider LLM support
  runtime-id: opencode
  experimental: true
  provider:
    name: github
  behaviors:
    secret-strategy: universal-llm-consumer
    capabilities:
      max-turns: true
    manifest:
      files:
        - opencode.jsonc
        - AGENTS.md
      path-prefixes:
        - .opencode/
    network:
      defaults:
        - host.docker.internal
        - github.com
        - raw.githubusercontent.com
        - registry.npmjs.org
        - opencode.ai
        - models.dev
      provider-domains:
        copilot: api.githubcopilot.com
        anthropic: api.anthropic.com
        openai: api.openai.com
        google: generativelanguage.googleapis.com
        groq: api.groq.com
        mistral: api.mistral.ai
        deepseek: api.deepseek.com
        xai: api.x.ai
    installation:
      package-manager: npm
      package-name: opencode-ai
      step-name: Install OpenCode
      binary-name: opencode
      include-node-setup: true
      cooldown: true
      verify-command: opencode --version
      verify-step-name: Verify OpenCode CLI installation
      docs-url: https://opencode.ai/docs
    config-file:
      path: opencode.jsonc
      step-name: Write OpenCode Config
      content: |-
        {
          "agent": {
            "build": {
              "permission": {
                "bash": "allow",
                "edit": "allow",
                "read": "allow",
                "glob": "allow",
                "grep": "allow",
                "webfetch": "allow",
                "websearch": "allow",
                "external_directory": "allow"
              }
            }
          },
          "autoupdate": false,
          "disabled_providers": ["opencode", "openai"],
          "provider": {
            "awf-proxy": {
              "api": "http://172.30.0.30:10002",
              "options": {
                "apiKey": "awf-copilot-proxy"
              },
              "models": {
                "claude-sonnet-4.5": {}
              }
            }
          }
        }
      merge-strategy: json-merge
    execution:
      command-name: opencode
      args:
        - run
        - --print-logs
        - --log-level
        - DEBUG
      step-name: Execute OpenCode CLI
      model-env-var: OPENCODE_MODEL
      model-env-provider-prefix: awf-proxy
      mcp-config-env-var: GH_AW_MCP_CONFIG
      write-timestamp: true
      provider-env-mode: universal-llm-consumer
      env:
        XDG_DATA_HOME: /tmp/opencode-data
    mcp:
      config-path: opencode.jsonc
    log-parser: |
      function parseLog(logContent) {
        const lines = logContent.split("\n");
        const entries = [];
        const mcpFailures = [];
        let maxTurnsHit = false;
        const AWF_INFRA_RE = /^\[(INFO|WARN|SUCCESS|ERROR|entrypoint|health-check)\]|^ (?:Container|Network|Volume) |^Process exiting with code:/;
        let inputTokens = 0;
        let outputTokens = 0;
        let toolCalls = 0;
        let errors = 0;

        for (const line of lines) {
          if (!line.trim()) continue;
          if (AWF_INFRA_RE.test(line)) continue;
          if (/max.?turns|maximum.*turns.*reached|turn limit/i.test(line)) maxTurnsHit = true;
          if (/MCP server .* failed|MCP.*connection.*error|Failed to connect to MCP/i.test(line)) {
            const serverMatch = line.match(/MCP server ['"]?([^\s'"]+)['"]?/i);
            mcpFailures.push(serverMatch ? serverMatch[1] : line.trim());
          }

          // Try to parse JSON log lines (OpenCode --print-logs output)
          let parsed = null;
          try {
            if (line.trim().startsWith("{")) parsed = JSON.parse(line.trim());
          } catch (e) { /* not JSON */ }

          if (parsed) {
            const entry = {
              type: parsed.type || parsed.msg || "log",
              level: parsed.level || "INFO",
              service: parsed.service || "",
              message: parsed.msg || parsed.message || "",
              raw: line
            };
            if (/tool[._]call|tool[._]use/i.test(entry.type)) toolCalls++;
            if (/error/i.test(entry.level)) errors++;
            if (parsed.input_tokens) inputTokens += parsed.input_tokens;
            if (parsed.output_tokens) outputTokens += parsed.output_tokens;
            entries.push(entry);
          } else {
            entries.push({ type: "text", message: line.trim(), raw: line });
          }
        }

        const jsonEntries = entries.filter(e => e.type !== "text");
        const textEntries = entries.filter(e => e.type === "text");

        let md = "### OpenCode Agent Log Summary\n\n";
        md += `| Metric | Value |\n|--------|-------|\n`;
        md += `| Total lines | ${lines.filter(l => l.trim()).length} |\n`;
        md += `| Structured (JSON) entries | ${jsonEntries.length} |\n`;
        md += `| Text entries | ${textEntries.length} |\n`;
        md += `| Tool calls | ${toolCalls} |\n`;
        md += `| Errors | ${errors} |\n`;
        md += `| MCP failures | ${mcpFailures.length} |\n`;
        md += `| Max turns hit | ${maxTurnsHit} |\n`;
        if (inputTokens || outputTokens) {
          md += `| Input tokens | ${inputTokens} |\n`;
          md += `| Output tokens | ${outputTokens} |\n`;
        }
        md += "\n";

        if (jsonEntries.length > 0) {
          md += "<details><summary>Structured Log Entries (first 30)</summary>\n\n";
          for (const entry of jsonEntries.slice(0, 30)) {
            const level = entry.level === "ERROR" ? "❌" : entry.level === "WARN" ? "⚠️" : "ℹ️";
            const msg = entry.message.length > 200 ? entry.message.slice(0, 200) + "..." : entry.message;
            md += `${level} \`[${entry.service}]\` **${entry.type}**: ${msg}\n\n`;
          }
          md += "</details>\n";
        }

        if (textEntries.length > 0 && jsonEntries.length === 0) {
          md += "<details><summary>Log Output</summary>\n\n```\n";
          const preview = textEntries.slice(0, 100).map(e => e.message).join("\n");
          md += preview.length > 3000 ? preview.slice(0, 3000) + "\n..." : preview;
          md += "\n```\n</details>\n";
        }

        return { markdown: md, logEntries: entries, mcpFailures, maxTurnsHit };
      }
---

<!--
# OpenCode CLI

Shared engine definition for the [OpenCode](https://opencode.ai) multi-provider AI
coding agent (BYOK). Import this file and set `engine: opencode` to use it:

```yaml
engine:
  id: opencode
model: copilot/claude-sonnet-4.5
imports:
  - shared/opencode.md
```

`model` must use `provider/model` format. Supported providers are `copilot`,
`anthropic`, `openai`, and `codex`.
-->
