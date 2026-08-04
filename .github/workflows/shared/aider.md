---
runtimes:
  python:
    version: "3.12"
engine:
  id: aider
  display-name: Aider
  description: Aider AI pair programming CLI running in scripting (non-interactive) mode
  experimental: true
  provider:
    name: github
  behaviors:
    secret-strategy: universal-llm-consumer
    manifest:
      files:
        - .aider.conf.yml
        - CONVENTIONS.md
      path-prefixes:
        - .aider/
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
    mcp:
      unsupported: true
    execution:
      command-name: aider
      args:
        - --yes-always
        - --no-auto-commits
        - --no-check-update
        - --no-show-release-notes
        - --no-detect-urls
        - --no-pretty
        - --no-stream
        - --no-fancy-input
        - --analytics-disable
      step-name: Execute Aider CLI
      model-env-var: AIDER_MODEL
      model-env-provider-prefix: openai
      provider-env-mode: universal-llm-consumer
      write-timestamp: true
      env:
        OPENAI_API_BASE: http://172.30.0.30:10002
        AIDER_GIT: "false"
        AIDER_CHECK_UPDATE: "false"
        AIDER_ANALYTICS_DISABLE: "true"
    harness-script: |
      const { spawnSync } = require("child_process");
      const { join } = require("path");
      const { homedir } = require("os");

      const AIDER_VERSION = "0.86.2";
      const [command, ...commandArgs] = process.argv.slice(2);

      const fail = (result, action) => {
        if (result.error || result.status !== 0) {
          throw new Error(`${action} failed`);
        }
      };

      const localBin = join(homedir(), ".local", "bin");
      const env = { ...process.env, PATH: `${localBin}:${process.env.PATH || ""}` };

      fail(
        spawnSync(
          "python3",
          ["-m", "pip", "install", "--quiet", "--user", "--disable-pip-version-check", `aider-chat==${AIDER_VERSION}`],
          { stdio: "inherit", env }
        ),
        "Aider installation"
      );

      const promptFile = process.env.GH_AW_PROMPT;
      if (!promptFile) {
        throw new Error("GH_AW_PROMPT is not set");
      }

      fail(
        spawnSync(command, [...commandArgs, "--message-file", promptFile], { stdio: "inherit", env }),
        "Aider execution"
      );
---

<!--
# Aider CLI

Shared engine definition for [Aider](https://github.com/Aider-AI/aider), the
open-source AI pair programming CLI ([docs](https://aider.chat/docs/)).
Import this file and set `engine: id: aider` to use it:

```yaml
engine:
  id: aider
model: copilot/claude-sonnet-4.5
imports:
  - shared/aider.md
```

`model` must use `provider/model` format. Supported providers are `copilot`,
`anthropic`, and `openai`. Requests are routed through the AWF proxy, so the
model name is rewritten to Aider's `openai/<model>` LiteLLM form and served
from `OPENAI_API_BASE`.

Aider runs in scripting mode: the generated prompt file is passed with
`--message-file` and all confirmations are auto-accepted (`--yes-always`).
Aider has no MCP client support, so the engine definition declares
`behaviors.mcp.unsupported: true`. Workflows that configure MCP-backed tools
(such as `github:` or `playwright:`) with this engine fail to compile. Use
`bash:` tools, file edits, and safe outputs instead.
-->
