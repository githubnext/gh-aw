---
runtimes:
  python:
    version: "3.12"
pre-agent-steps:
  - name: Preinstall Pydantic AI CLI
    run: |
      python3 -m pip install --quiet --user --disable-pip-version-check "pydantic-ai==$GH_AW_ENGINE_VERSION"
      "$HOME/.local/bin/pai" --version
engine:
  id: pydantic-ai
  version: "2.26.0"
  display-name: Pydantic AI
  description: Pydantic AI CLI (pai) running one-shot prompts with MCP tool support
  experimental: true
  mcp: true
  provider:
    name: github
  behaviors:
    secret-strategy: universal-llm-consumer
    manifest:
      files:
        - AGENTS.md
      path-prefixes:
        - .pydantic-ai/
    network:
      defaults:
        - host.docker.internal
        - github.com
        - raw.githubusercontent.com
        - api.github.com
        - objects.githubusercontent.com
        - pypi.org
        - files.pythonhosted.org
      provider-domains:
        copilot: api.githubcopilot.com
        anthropic: api.anthropic.com
        openai: api.openai.com
    execution:
      command-name: pai
      args:
        - --no-stream
      step-name: Execute Pydantic AI CLI
      model-env-var: PAI_MODEL
      model-env-provider-prefix: openai-chat
      model-env-provider-separator: ":"
      write-timestamp: true
      provider-env-mode: universal-llm-consumer
    harness-script: |
      const { spawnSync } = require("child_process");
      const { existsSync, readFileSync } = require("fs");
      const { join } = require("path");
      const { homedir } = require("os");

      const [command, ...commandArgs] = process.argv.slice(2);

      const promptFile = process.env.GH_AW_PROMPT;
      if (!promptFile) {
        throw new Error("GH_AW_PROMPT is not set");
      }
      const workspace = process.env.GITHUB_WORKSPACE;
      if (!workspace) {
        throw new Error("GITHUB_WORKSPACE is not set");
      }

      const localBin = join(homedir(), ".local", "bin");
      const env = { ...process.env, PATH: `${localBin}:${process.env.PATH || ""}` };
      delete env.COPILOT_GITHUB_TOKEN;
      // `pai` reads its credentials from the environment, so the placeholder key the
      // AWF api-proxy expects has to be substituted here, inside the sandbox: the
      // proxy treats a request whose key matches its own OPENAI_API_KEY as BYOK and
      // forwards it upstream instead of steering it to the configured provider.
      env.OPENAI_API_KEY = "awf-copilot-proxy";

      const args = [...commandArgs];
      // The proxy exposes Copilot Claude models under their dotted IDs.
      const model = env.PAI_MODEL?.replace(/^([^:]+:claude-(?:haiku|sonnet|opus)-\d+)-(\d+)$/, "$1.$2");
      if (model) {
        args.push("-m", model);
      }
      const agentSpec = join(workspace, ".pydantic-ai", "agent.json");
      if (existsSync(agentSpec)) {
        args.push("-a", agentSpec);
      }
      args.push(readFileSync(promptFile, "utf8"));

      const result = spawnSync(command, args, { cwd: workspace, encoding: "utf8", env });
      process.stdout.write(result.stdout || "");
      process.stderr.write(result.stderr || "");
      if (result.error || result.status !== 0) {
        throw new Error(`Pydantic AI execution failed: ${result.error?.message || `exit code ${result.status}`}`);
      }
    mcp:
      config-path: .pydantic-ai/agent.json
      config-adapter: |
        // Converts the MCP gateway's standard HTTP-based configuration into a
        // Pydantic AI agent spec (https://ai.pydantic.dev), which is the only way
        // the `pai` CLI can be given MCP servers: each gateway server becomes an
        // `MCP` capability entry and the spec file is passed via `pai -a <file>`.
        // An agent spec must declare a `model`, but the harness always appends
        // `-m "$PAI_MODEL"` when the workflow declares a model, which takes
        // precedence. The value below is only a valid-by-construction fallback for
        // workflows that do not declare a model.
        const fs = require("fs");
        const path = require("path");

        const requireEnvVar = name => {
          const value = process.env[name];
          if (!value) throw new Error(`${name} environment variable is required`);
          return value;
        };

        const gatewayOutputPath = requireEnvVar("MCP_GATEWAY_OUTPUT");
        const workspace = requireEnvVar("GITHUB_WORKSPACE");
        const gatewayDomain = process.env.MCP_GATEWAY_DOMAIN || "host.docker.internal";
        const gatewayPort = requireEnvVar("MCP_GATEWAY_PORT");
        const gatewayURL = `http://${gatewayDomain}:${gatewayPort}`;

        let cliServers;
        try {
          cliServers = new Set(JSON.parse(process.env.GH_AW_MCP_CLI_SERVERS || "[]"));
        } catch (error) {
          throw new Error(`Failed to parse GH_AW_MCP_CLI_SERVERS: ${error instanceof Error ? error.message : String(error)}`);
        }

        const gatewayOutput = JSON.parse(fs.readFileSync(gatewayOutputPath, "utf8"));
        const rawServers = gatewayOutput.mcpServers;
        const servers = rawServers && typeof rawServers === "object" && !Array.isArray(rawServers) ? rawServers : {};

        const capabilities = [];
        for (const [name, entry] of Object.entries(servers)) {
          if (cliServers.has(name) || !entry || typeof entry !== "object") continue;
          if (typeof entry.url !== "string") {
            console.log(`Skipping MCP server ${name}: the Pydantic AI CLI only supports HTTP MCP servers`);
            continue;
          }
          const mcp = {
            id: name,
            url: entry.url.replace(/^http:\/\/[^/]+\/mcp\//, `${gatewayURL}/mcp/`),
          };
          if (entry.headers && typeof entry.headers === "object") mcp.headers = entry.headers;
          capabilities.push({ MCP: mcp });
        }

        const configPath = path.join(workspace, ".pydantic-ai", "agent.json");
        fs.mkdirSync(path.dirname(configPath), { recursive: true });
        fs.writeFileSync(configPath, JSON.stringify({ model: "openai-chat:gpt-5", capabilities }, null, 2), { mode: 0o600 });
        fs.chmodSync(configPath, 0o600);
        console.log(`Wrote ${capabilities.length} MCP server(s) to ${configPath}`);
    log-parser: |
      function parseLog(logContent) {
        const lines = logContent.split("\n");
        const logEntries = [];
        const mcpFailures = [];
        let maxTurnsHit = false;
        const AWF_INFRA_RE = /^\[(INFO|WARN|SUCCESS|ERROR|entrypoint|health-check)\]|^ (?:Container|Network|Volume) |^Process exiting with code:/;
        let inputTokens = 0;
        let outputTokens = 0;
        let toolCallIndex = 0;
        let turnCount = 0;
        let pendingText = [];

        function flushText() {
          if (pendingText.length === 0) return;
          const text = pendingText.join("\n").trim();
          if (text) {
            logEntries.push({ type: "assistant", message: { content: [{ type: "text", text }] } });
            turnCount++;
          }
          pendingText = [];
        }

        logEntries.push({ type: "system", subtype: "init", model: null, session_id: null });

        for (const line of lines) {
          if (!line.trim()) continue;
          if (AWF_INFRA_RE.test(line)) continue;
          if (/max.?turns|maximum.*turns.*reached|turn limit/i.test(line)) maxTurnsHit = true;
          if (/MCP server .* failed|MCP.*connection.*error|Failed to connect to MCP/i.test(line)) {
            const serverMatch = line.match(/MCP server ['"]?([^\s'"]+)['"]?/i);
            mcpFailures.push(serverMatch ? serverMatch[1] : line.trim());
          }

          let parsed = null;
          try {
            if (line.trim().startsWith("{")) parsed = JSON.parse(line.trim());
          } catch (e) { /* not JSON */ }

          if (parsed) {
            if (parsed.input_tokens) inputTokens += parsed.input_tokens;
            if (parsed.output_tokens) outputTokens += parsed.output_tokens;
            const entryType = parsed.type != null ? String(parsed.type) : "log";
            const msg = parsed.msg || parsed.message || parsed.content || "";

            if (/tool[._]call|tool[._]use/i.test(entryType)) {
              flushText();
              const toolId = `pai_tool_${toolCallIndex++}`;
              const toolName = parsed.tool || parsed.name || entryType;
              logEntries.push({ type: "assistant", message: { content: [{ type: "tool_use", id: toolId, name: toolName, input: {} }] } });
              logEntries.push({ type: "user", message: { content: [{ type: "tool_result", tool_use_id: toolId, content: msg }] } });
            } else if (msg) {
              pendingText.push(msg);
            }
          } else {
            pendingText.push(line.trim());
          }
        }
        flushText();

        const usage = {};
        if (inputTokens) usage.input_tokens = inputTokens;
        if (outputTokens) usage.output_tokens = outputTokens;
        logEntries.push({ type: "result", num_turns: turnCount, usage });
        const parts = [`**Turns:** ${turnCount}`, `**Tool calls:** ${toolCallIndex}`];
        if (inputTokens || outputTokens) parts.push(`**Tokens:** ${((inputTokens ?? 0) + (outputTokens ?? 0)).toLocaleString()}`);
        if (mcpFailures.length) parts.push(`**MCP failures:** ${mcpFailures.length}`);
        if (maxTurnsHit) parts.push("**Max turns reached**");
        return { markdown: parts.join(" · "), logEntries, mcpFailures, maxTurnsHit };
      }
---

<!--
# Pydantic AI

Shared engine definition for the [Pydantic AI](https://ai.pydantic.dev) CLI
(`pai`). Import this file and set `engine: id: pydantic-ai` to use it:

```yaml
engine:
  id: pydantic-ai
model: copilot/claude-sonnet-4-5
imports:
  - shared/pydantic.md
```

`model` must use `provider/model` format. Supported providers are `copilot`,
`anthropic`, and `openai`. Requests are routed through the AWF proxy, so the
model is rewritten to the `openai-chat:<model>` form the Pydantic AI CLI
expects and passed with `-m`. Copilot Claude aliases such as
`claude-sonnet-4-5` are normalized to the dotted model IDs exposed by the
proxy, such as `claude-sonnet-4.5`.

MCP servers are rendered into a Pydantic AI agent spec at
`.pydantic-ai/agent.json` and passed with `-a`, so safe outputs flow through
the standard `safeoutputs` server automatically.

`pai` reads its credentials from the environment, so a harness script assembles
the command line and substitutes the AWF proxy placeholder API key inside the
sandbox. Setting the placeholder on the execution step instead would make the
proxy itself adopt it as a BYOK OpenAI key and forward requests upstream rather
than steering them to the configured provider.

The CLI is installed with `pip install --user pydantic-ai==<engine version>`
before the agent runs.
-->
