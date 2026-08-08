---
engine:
  id: lightpanda
  display-name: Lightpanda
  description: Lightpanda headless browser agent for web navigation and data extraction
  experimental: true
  mcp: false
  provider:
    name: github
  behaviors:
    secret-strategy: universal-llm-consumer
    manifest:
      files:
        - .lp-agent.zon
    network:
      defaults:
        - host.docker.internal
        - github.com
        - raw.githubusercontent.com
        - release-assets.githubusercontent.com
        - objects.githubusercontent.com
        - api.github.com
      provider-domains:
        copilot: api.githubcopilot.com
        anthropic: api.anthropic.com
        openai: api.openai.com
    execution:
      command-name: lightpanda
      step-name: Execute Lightpanda Agent
      model-env-var: LIGHTPANDA_MODEL
      provider-env-mode: universal-llm-consumer
      write-timestamp: true
      env:
        LIGHTPANDA_DISABLE_TELEMETRY: "true"
        LIGHTPANDA_DISABLE_CORE_DUMP: "true"
    harness-script: |
      const { spawnSync } = require("child_process");
      const { chmodSync, existsSync, mkdtempSync, readFileSync, rmSync } = require("fs");
      const { tmpdir } = require("os");
      const { join } = require("path");
      const { fetchAWFReflect, resolveProviderEndpointFromReflect } = require("./awf_reflect.cjs");

      const [command] = process.argv.slice(2);
      const log = message => process.stderr.write(`[lightpanda-harness] ${message}\n`);
      const fail = (result, action) => {
        if (result.error) throw result.error;
        if (result.status !== 0) throw new Error(`${action} failed with exit code ${result.status ?? "unknown"}`);
      };

      const installDir = mkdtempSync(join(tmpdir(), "lightpanda-"));
      const binaryPath = join(installDir, command || "lightpanda");
      const releaseURL = "https://github.com/lightpanda-io/browser/releases/download/nightly/lightpanda-x86_64-linux";

      const main = async () => {
        try {
          log("downloading lightpanda nightly binary...");
          fail(
            spawnSync("curl", ["--fail", "--location", "--silent", "--show-error", "--output", binaryPath, releaseURL], { stdio: "inherit" }),
            "lightpanda download"
          );
          chmodSync(binaryPath, 0o755);

          const selectedModel = process.env.LIGHTPANDA_MODEL;
          if (!selectedModel) throw new Error("LIGHTPANDA_MODEL is required");
          const model = selectedModel.includes("/") ? selectedModel.slice(selectedModel.indexOf("/") + 1) : selectedModel;

          const provider = process.env.GH_AW_LLM_PROVIDER;
          if (!provider) throw new Error("GH_AW_LLM_PROVIDER is required");

          if (process.env.AWF_REFLECT_ENABLED !== "1") {
            throw new Error("Lightpanda requires AWF endpoint discovery (AWF_REFLECT_ENABLED=1)");
          }

          const reflectResult = await fetchAWFReflect({ logger: log });
          if (!reflectResult.ok || !reflectResult.reflectData) {
            throw new Error(`Unable to discover LLM endpoint from /reflect: ${reflectResult.reason || "empty response"}`);
          }
          const endpoint = resolveProviderEndpointFromReflect({
            provider,
            reflectData: reflectResult.reflectData,
            logger: log,
          });
          if (!endpoint || !endpoint.baseUrl) {
            throw new Error(`No configured /reflect endpoint found for provider ${provider}`);
          }
          let baseUrl = endpoint.baseUrl;
          const reflectedEndpoint = reflectResult.reflectData.endpoints?.find(
            entry => entry?.configured === true && entry.provider === endpoint.endpointProvider
          );
          if (typeof reflectedEndpoint?.models_url === "string") {
            const modelsURL = new URL(reflectedEndpoint.models_url);
            const basePath = modelsURL.pathname.replace(/\/models\/?$/i, "");
            baseUrl = `${modelsURL.origin}${basePath}`;
          }
          log(`configured lightpanda endpoint for provider=${provider}: ${baseUrl}`);

          const env = { ...process.env };
          env.LIGHTPANDA_DISABLE_TELEMETRY = "true";
          env.LIGHTPANDA_DISABLE_CORE_DUMP = "true";
          if (!env.OPENAI_API_KEY) env.OPENAI_API_KEY = "awf-copilot-proxy";

          const promptFile = process.env.GH_AW_PROMPT;
          if (!promptFile) throw new Error("GH_AW_PROMPT is required");
          const prompt = readFileSync(promptFile, "utf8").trim();

          log(`running lightpanda agent with model=${model}`);
          fail(
            spawnSync(
              binaryPath,
              ["agent", "--provider", "openai", "--model", model, "--base-url", baseUrl, "--task", prompt, "--verbosity", "low"],
              { stdio: "inherit", env }
            ),
            "lightpanda agent execution"
          );
        } finally {
          if (existsSync(installDir)) rmSync(installDir, { recursive: true, force: true });
        }
      };

      main().catch(error => {
        log(error instanceof Error ? error.message : String(error));
        process.exitCode = 1;
      });
---

<!--
# Lightpanda Browser Agent

Shared engine definition for the [Lightpanda](https://github.com/lightpanda-io/browser)
headless browser agent. Import this file and set `engine.id: lightpanda` to use it.

```yaml
engine:
  id: lightpanda
model: copilot/claude-sonnet-4-5
imports:
  - shared/lightpanda.md
```

`model` must use `provider/model` format. The engine uses `--provider openai` with
the AWF proxy endpoint discovered via `/reflect`. The model's provider prefix is
stripped and passed as `--model <model>` to the Lightpanda CLI.

The nightly binary is downloaded from GitHub Releases at runtime, so no pre-install
step is required. Telemetry and core dumps are disabled automatically.

Lightpanda has no MCP client. The engine is specialized for web browsing, navigation,
and data extraction tasks using its built-in browser primitives.
-->
