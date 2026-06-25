#!/usr/bin/env node
// @ts-check
"use strict";

/**
 * Pi Agent Core Driver Sample
 *
 * This is a minimal sample of a pi-agent-core driver that can be customised
 * and placed in .github/drivers/ in your repository.  Set it as the engine
 * driver in your workflow frontmatter:
 *
 *   engine:
 *     id: pi
 *     driver: .github/drivers/pi_agent_core_driver_sample_node.cjs
 *
 * The driver uses @earendil-works/pi-agent-core to run a pi agent session
 * directly from Node.js and writes a JSONL log that gh-aw's parser understands.
 *
 * For the full implementation see:
 *   actions/setup/js/pi_agent_core_driver.cjs
 *
 * See also:
 *   https://github.com/earendil-works/pi/blob/main/packages/agent/README.md
 */

const { execSync } = require("child_process");
const fs = require("fs");
const crypto = require("crypto");

// ---------------------------------------------------------------------------
// Minimal JSONL emitter
// ---------------------------------------------------------------------------

/** @param {unknown} obj */
function emitJsonl(obj) {
  process.stdout.write(JSON.stringify(obj) + "\n");
}

// ---------------------------------------------------------------------------
// API key resolution (customise as needed)
// ---------------------------------------------------------------------------

/** @param {string} provider */
function getApiKey(provider) {
  switch (provider) {
    case "github-copilot":
    case "copilot":
      return process.env.COPILOT_GITHUB_TOKEN || process.env.GITHUB_TOKEN;
    case "anthropic":
      return process.env.ANTHROPIC_API_KEY;
    case "openai":
      return process.env.CODEX_API_KEY || process.env.OPENAI_API_KEY;
    default:
      return undefined;
  }
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

async function main() {
  // Install required packages before dynamic imports.
  execSync(
    "npm install --ignore-scripts --no-save @earendil-works/pi-agent-core @earendil-works/pi-ai",
    { stdio: "inherit", cwd: process.env.GITHUB_WORKSPACE || process.cwd() }
  );

  const promptFile = process.env.GH_AW_PROMPT;
  if (!promptFile) throw new Error("GH_AW_PROMPT is not set");
  const prompt = fs.readFileSync(promptFile, "utf8");

  // Choose a model in "provider/model" format (default: GitHub Copilot).
  const modelStr = process.env.GH_AW_PI_MODEL || process.env.PI_MODEL || "copilot/claude-sonnet-4-20250514";
  const slashIdx = modelStr.indexOf("/");
  const providerPrefix = slashIdx > 0 ? modelStr.slice(0, slashIdx).toLowerCase() : "copilot";
  const modelId = slashIdx > 0 ? modelStr.slice(slashIdx + 1) : modelStr;

  // Resolve provider and api type.
  let provider = "github-copilot";
  let api = "openai-completions";
  let baseUrl = "https://api.githubcopilot.com";
  if (providerPrefix === "anthropic") {
    provider = "anthropic";
    api = "anthropic-messages";
    baseUrl = "https://api.anthropic.com";
  }

  /** @type {Record<string, unknown>} */
  const model = {
    id: modelId,
    name: modelId,
    api,
    provider,
    baseUrl,
    reasoning: false,
    input: ["text"],
    cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 },
    contextWindow: 200000,
    maxTokens: 8192,
  };

  // ESM modules must be loaded with dynamic import() in a CJS file.
  const { Agent } = await import("@earendil-works/pi-agent-core");
  await import("@earendil-works/pi-ai"); // registers built-in API providers

  const sessionId = crypto.randomUUID();
  const startMs = Date.now();
  let inputTokens = 0;
  let outputTokens = 0;
  let turns = 0;

  const agent = new Agent({
    initialState: { systemPrompt: "", model },
    getApiKey,
  });

  agent.subscribe(event => {
    switch (event.type) {
      case "agent_start":
        emitJsonl({ type: "init", model: modelId, session_id: sessionId });
        break;
      case "message_update":
        if (event.assistantMessageEvent?.type === "text_delta") {
          emitJsonl({ type: "assistant", content: event.assistantMessageEvent.delta, delta: true });
        }
        break;
      case "tool_execution_start":
        emitJsonl({ type: "tool_use", tool_name: event.toolName, tool_id: event.toolCallId, parameters: event.args ?? {} });
        break;
      case "tool_execution_end": {
        const out = typeof event.result === "string" ? event.result : event.result != null ? JSON.stringify(event.result) : "";
        emitJsonl({ type: "tool_result", tool_id: event.toolCallId, status: event.isError ? "error" : "success", output: out });
        break;
      }
      case "turn_end":
        turns++;
        if (event.message?.usage) {
          inputTokens += event.message.usage.input ?? 0;
          outputTokens += event.message.usage.output ?? 0;
        }
        break;
      case "agent_end":
        emitJsonl({ type: "result", stats: { input_tokens: inputTokens, output_tokens: outputTokens, duration_ms: Date.now() - startMs, turns } });
        break;
    }
  });

  await agent.prompt(prompt);
  await agent.waitForIdle();
}

if (require.main === module) {
  main().catch(err => {
    process.stderr.write(`[pi-agent-core-driver-sample] ${err instanceof Error ? err.stack : String(err)}\n`);
    process.exit(1);
  });
}
