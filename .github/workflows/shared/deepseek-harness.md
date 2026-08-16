---
engine:
  id: deepseek-harness
  version: "0.1.0-rc.6"
  display-name: DeepSeek Harness
  description: DeepSeek Harness (dsh) with headless execution and multi-provider LLM support
  experimental: true
  mcp: false
  provider:
    name: github
  behaviors:
    secret-strategy: universal-llm-consumer
    manifest:
      files:
        - AGENTS.md
      path-prefixes:
        - .dsh/
    network:
      defaults:
        - host.docker.internal
        - github.com
        - raw.githubusercontent.com
        - api.github.com
        - objects.githubusercontent.com
        - registry.npmjs.org
      provider-domains:
        copilot: api.githubcopilot.com
        anthropic: api.anthropic.com
        openai: api.openai.com
    installation:
      package-manager: npm
      package-name: "@deepseek-ai/dsh"
      step-name: Install DeepSeek Harness
      binary-name: dsh
      include-node-setup: true
      post-install-scripts: true
      cooldown: true
      verify-command: dsh --version
      verify-step-name: Verify DeepSeek Harness installation
      docs-url: https://github.com/deepseek-ai/deepseek-harness
    execution:
      command-name: dsh
      args:
        - --profile
        - headless
      step-name: Execute DeepSeek Harness
      model-env-var: DSH_MODEL
      provider-env-mode: universal-llm-consumer
      write-timestamp: true
      env:
        DSH_PERMISSION_MODE: danger-full-access
        DSH_TELEMETRY_DISABLED: "1"
        DSH_TOOLS_MODE: native
        NO_COLOR: "1"
    harness-script: |
      const { mkdirSync, readFileSync, writeFileSync } = require("fs");
      const { join } = require("path");
      const { spawnSync } = require("child_process");
      const { fetchAWFReflect, resolveProviderEndpointFromReflect } = require("./awf_reflect.cjs");

      const [command, ...commandArgs] = process.argv.slice(2);
      const log = message => process.stderr.write(`[deepseek-harness] ${message}\n`);
      const fail = (result, action) => {
        if (result.error) throw result.error;
        if (result.status !== 0) {
          throw new Error(`${action} failed with exit code ${result.status ?? "unknown"}`);
        }
      };

      const main = async () => {
        const workspace = process.env.GITHUB_WORKSPACE;
        if (!workspace) throw new Error("GITHUB_WORKSPACE is required");

        const selectedModel = process.env.DSH_MODEL;
        if (!selectedModel || !selectedModel.includes("/")) {
          throw new Error("DSH_MODEL must use provider/model format");
        }
        const model = selectedModel.slice(selectedModel.indexOf("/") + 1);
        if (!model) throw new Error("DSH_MODEL must include a model name");

        const provider = process.env.GH_AW_LLM_PROVIDER;
        if (!provider) throw new Error("GH_AW_LLM_PROVIDER is required");

        let baseURL = process.env.OPENAI_BASE_URL;
        if (process.env.AWF_REFLECT_ENABLED === "1") {
          const result = await fetchAWFReflect({ logger: log });
          if (!result.ok || !result.reflectData) {
            throw new Error(`Unable to discover the DeepSeek Harness LLM endpoint from /reflect: ${result.reason || "empty response"}`);
          }
          const endpoint = resolveProviderEndpointFromReflect({
            provider,
            reflectData: result.reflectData,
            logger: log,
          });
          if (!endpoint?.baseUrl) {
            throw new Error(`No configured /reflect endpoint found for provider ${provider}`);
          }
          baseURL = endpoint.baseUrl;
        }
        if (!baseURL) {
          throw new Error("DeepSeek Harness requires AWF endpoint discovery or OPENAI_BASE_URL");
        }

        const dshHome = join(workspace, ".dsh");
        mkdirSync(dshHome, { recursive: true, mode: 0o700 });
        const settings = {
          "agent-default-model": {
            provider: "awf-proxy",
            model,
          },
          "llm-pi-ai": {
            providers: {
              "awf-proxy": {
                displayName: "GitHub Agentic Workflows",
                apiKeyEnv: "OPENAI_API_KEY",
                api: "openai-completions",
                baseURL,
                models: [{ id: model, name: model }],
              },
            },
          },
        };
        const settingsPath = join(dshHome, "settings.yaml");
        writeFileSync(settingsPath, JSON.stringify(settings, null, 2), { mode: 0o600 });

        const promptPath = process.env.GH_AW_PROMPT;
        if (!promptPath) throw new Error("GH_AW_PROMPT is required");
        const prompt = readFileSync(promptPath, "utf8");
        const env = { ...process.env, DSH_HOME: dshHome };
        log(`configured provider=${provider} model=${model}`);
        fail(
          spawnSync(command, [...commandArgs, prompt], {
            cwd: workspace,
            env,
            stdio: "inherit",
          }),
          "DeepSeek Harness execution"
        );
      };

      main().catch(error => {
        log(error instanceof Error ? error.message : String(error));
        process.exitCode = 1;
      });
---

<!--
# DeepSeek Harness

Shared engine definition for [DeepSeek Harness](https://github.com/deepseek-ai/deepseek-harness),
the open-source `dsh` coding agent. Import this file and set
`engine.id: deepseek-harness` with a `provider/model` model selection:

```yaml
engine:
  id: deepseek-harness
model: copilot/claude-sonnet-4.5
imports:
  - shared/deepseek-harness.md
```

The integration pins the developer-preview `@deepseek-ai/dsh` package and runs
its one-shot `headless` profile. Provider credentials and the selected endpoint
are routed through the AWF proxy and written to an ephemeral `.dsh` home.
Telemetry is disabled. Native MCP configuration is intentionally disabled for
this initial integration; gh-aw exposes configured tools through its CLI proxy.
-->
