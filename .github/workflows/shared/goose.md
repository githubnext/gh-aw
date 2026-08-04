---
engine:
  id: goose
  display-name: Goose
  description: Goose CLI with headless execution and MCP support
  experimental: true
  provider:
    name: github
  behaviors:
    secret-strategy: universal-llm-consumer
    capabilities:
      max-turns: true
    manifest:
      files:
        - .goosehints
      path-prefixes:
        - .goose/
    network:
      defaults:
        - host.docker.internal
        - github.com
        - raw.githubusercontent.com
        - api.github.com
        - objects.githubusercontent.com
      provider-domains:
        copilot: api.githubcopilot.com
        anthropic: api.anthropic.com
        openai: api.openai.com
        google: generativelanguage.googleapis.com
    execution:
      command-name: goose
      step-name: Execute Goose CLI
      model-env-var: GOOSE_MODEL
      mcp-config-env-var: GH_AW_MCP_CONFIG
      provider-env-mode: universal-llm-consumer
      env:
        GOOSE_PROVIDER: openai
        GOOSE_PROVIDER__TYPE: openai
        GOOSE_PROVIDER__HOST: http://172.30.0.30:10002
        GOOSE_PROVIDER__API_KEY: awf-copilot-proxy
        GOOSE_MODE: auto
        GOOSE_DISABLE_SESSION_NAMING: "true"
    mcp:
      config-path: .goose/mcp.json
    harness-script: |
      const { createHash } = require("crypto");
      const { existsSync, mkdtempSync, readFileSync, rmSync } = require("fs");
      const { tmpdir } = require("os");
      const { join } = require("path");
      const { spawnSync } = require("child_process");

      const [command, ...commandArgs] = process.argv.slice(2);
      const installDir = mkdtempSync(join(tmpdir(), "goose-"));
      const archive = join(installDir, "goose.tar.gz");
      const releaseURL = "https://github.com/aaif-goose/goose/releases/download/v1.45.0/goose-x86_64-unknown-linux-gnu.tar.gz";
      const checksum = "e0db638ac437ca0a60b0c1622f45322608d228d1a285214c3bf48fd9763346a5";
      const fail = (result, action) => {
        if (result.error || result.status !== 0) {
          throw new Error(`${action} failed`);
        }
      };
      const quote = (value) => `'${String(value).replace(/'/g, "'\\''")}'`;

      try {
        fail(spawnSync("curl", ["--fail", "--location", "--silent", "--show-error", "--output", archive, releaseURL], { stdio: "inherit" }), "Goose download");
        if (createHash("sha256").update(readFileSync(archive)).digest("hex") !== checksum) {
          throw new Error("Goose download checksum did not match");
        }
        fail(spawnSync("tar", ["-xzf", archive, "-C", installDir], { stdio: "inherit" }), "Goose extraction");

        const config = JSON.parse(readFileSync(process.env.GH_AW_MCP_CONFIG, "utf8"));
        const extensions = Object.values(config.mcpServers || {}).flatMap((server) => {
          if (!server.command) return [];
          const env = Object.entries(server.env || {}).map(([key, value]) => `${key}=${quote(value)}`);
          return ["--with-extension", [...env, quote(server.command), ...(server.args || []).map(quote)].join(" ")];
        });
        const prompt = readFileSync(process.env.GH_AW_PROMPT, "utf8");
        fail(spawnSync(join(installDir, command), [...commandArgs, "run", "--no-session", ...extensions, "-t", prompt], { stdio: "inherit", env: process.env }), "Goose execution");
      } finally {
        if (existsSync(installDir)) rmSync(installDir, { recursive: true, force: true });
      }
---

<!--
# Goose CLI

Shared engine definition for the [Goose](https://github.com/aaif-goose/goose)
open-source AI agent. Import this file and set `engine: id: goose` to use it.
-->
