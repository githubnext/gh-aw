---
engine:
  id: adal
  version: "1.7.0"
  display-name: AdaL
  description: AdaL CLI running in authenticated headless mode
  experimental: true
  mcp: false
  provider:
    name: adal
  auth:
    - role: session
      secret: ADAL_AUTH_TOKEN
  behaviors:
    supported-env-var-keys:
      - ADAL_AUTH_TOKEN
    manifest:
      files:
        - AGENTS.md
      path-prefixes:
        - .adal/
    network:
      defaults:
        - host.docker.internal
        - github.com
        - raw.githubusercontent.com
        - registry.npmjs.org
        - adal.sylph.ai
        - api.adal.sylph.ai
        - d35qg8ac0yw4p7.cloudfront.net
      provider-domains:
        adal: adal.sylph.ai
    installation:
      package-manager: npm
      package-name: "@sylphai/adal-cli"
      step-name: Install AdaL CLI
      binary-name: adal
      include-node-setup: true
      post-install-scripts: true
      cooldown: true
      verify-command: adal --version
      verify-step-name: Verify AdaL CLI installation
      docs-url: https://docs.sylph.ai/features/headless-mode
    execution:
      command-name: adal
      args:
        - --yolo
        - --output
        - stream-json
      step-name: Execute AdaL CLI
      model-env-var: ADAL_MODEL
      write-timestamp: true
      env:
        ADAL_TELEMETRY: "false"
        NO_COLOR: "1"
    harness-script: |
      const { spawnSync } = require("child_process");
      const { readFileSync } = require("fs");

      const [command, ...commandArgs] = process.argv.slice(2);
      const log = message => process.stderr.write(`[adal-harness] ${message}\n`);
      const fail = (result, action) => {
        if (result.error) throw result.error;
        if (result.signal) throw new Error(`${action} terminated by signal ${result.signal}`);
        if (result.status !== 0) throw new Error(`${action} failed with exit code ${result.status ?? "unknown"}`);
      };

      try {
        if (!process.env.ADAL_AUTH_TOKEN) {
          throw new Error("ADAL_AUTH_TOKEN is required");
        }
        const selectedModel = process.env.ADAL_MODEL;
        if (!selectedModel?.startsWith("adal/")) {
          throw new Error("ADAL_MODEL must use adal/model format");
        }
        const model = selectedModel.slice("adal/".length);
        if (!model) {
          throw new Error("ADAL_MODEL must include a model name");
        }
        const promptPath = process.env.GH_AW_PROMPT;
        if (!promptPath) {
          throw new Error("GH_AW_PROMPT is required");
        }
        const prompt = readFileSync(promptPath, "utf8");
        fail(
          spawnSync(command, [...commandArgs, "--model", model, "--query", prompt], {
            cwd: process.env.GITHUB_WORKSPACE,
            env: process.env,
            stdio: "inherit",
          }),
          "AdaL execution"
        );
      } catch (error) {
        log(error instanceof Error ? error.message : String(error));
        process.exitCode = 1;
      }
    log-parser: |
      function parseLog(logContent) {
        const lines = logContent.split("\n");
        const logEntries = [];
        const mcpFailures = [];
        const pendingTools = new Map();
        let maxTurnsHit = false;
        let toolCallIndex = 0;
        let turnCount = 0;
        let model = null;
        let sessionId = null;

        logEntries.push({ type: "system", subtype: "init", model, session_id: sessionId });

        for (const line of lines) {
          const trimmed = line.trim();
          if (!trimmed || /^\[(INFO|WARN|SUCCESS|ERROR|entrypoint|health-check|adal-harness)\]/.test(trimmed)) continue;
          if (/max.?turns|maximum.*turns.*reached|turn limit/i.test(trimmed)) maxTurnsHit = true;
          if (/MCP server .* failed|MCP.*connection.*error|Failed to connect to MCP/i.test(trimmed)) {
            const serverMatch = trimmed.match(/MCP server ['"]?([^\s'"]+)['"]?/i);
            mcpFailures.push(serverMatch ? serverMatch[1] : trimmed);
          }

          let event;
          try {
            event = JSON.parse(trimmed);
          } catch {
            continue;
          }

          if (event.type === "tool_call") {
            const id = `adal_tool_${toolCallIndex++}`;
            const name = typeof event.name === "string" && event.name ? event.name : "unknown_tool";
            const queue = pendingTools.get(name) || [];
            queue.push(id);
            pendingTools.set(name, queue);
            logEntries.push({
              type: "assistant",
              message: { content: [{ type: "tool_use", id, name, input: event.args || {} }] },
            });
          } else if (event.type === "tool_result") {
            const name = typeof event.name === "string" && event.name ? event.name : "unknown_tool";
            const queue = pendingTools.get(name) || [];
            const id = queue.shift() || `adal_tool_${toolCallIndex++}`;
            if (queue.length) pendingTools.set(name, queue);
            else pendingTools.delete(name);
            logEntries.push({
              type: "user",
              message: {
                content: [{
                  type: "tool_result",
                  tool_use_id: id,
                  content: event.status ? String(event.status) : "",
                }],
              },
            });
          } else if (event.type === "answer" && typeof event.content === "string") {
            logEntries.push({
              type: "assistant",
              message: { content: [{ type: "text", text: event.content }] },
            });
            turnCount++;
          } else if (event.type === "error" && typeof event.message === "string") {
            logEntries.push({
              type: "assistant",
              message: { content: [{ type: "text", text: event.message }] },
            });
          } else if (event.type === "complete") {
            model = typeof event.model === "string" ? event.model : model;
            sessionId = typeof event.session_id === "string" ? event.session_id : sessionId;
          }
        }

        logEntries[0].model = model;
        logEntries[0].session_id = sessionId;
        logEntries.push({ type: "result", num_turns: turnCount, usage: {} });
        const parts = [`**Turns:** ${turnCount}`, `**Tool calls:** ${toolCallIndex}`];
        if (mcpFailures.length) parts.push(`**MCP failures:** ${mcpFailures.length}`);
        if (maxTurnsHit) parts.push("**Max turns reached**");
        return { markdown: parts.join(" · "), logEntries, mcpFailures, maxTurnsHit };
      }
---

<!--
# AdaL CLI

Shared engine definition for [AdaL](https://github.com/SylphAI-Inc/adal-cli).
Import this file, set `engine.id: adal`, and select a model using the
`adal/model` format:

```yaml
engine:
  id: adal
model: adal/gpt-5.6-terra
imports:
  - shared/adal.md
```

Configure the `ADAL_AUTH_TOKEN` GitHub Actions secret with an AdaL access token.
The engine runs AdaL in headless YOLO mode and strips the `adal/` prefix before
passing the model registry key to the CLI. AdaL's native MCP configuration is
not exposed in headless mode, so gh-aw provides MCP-backed tools through its
CLI proxy.
-->
