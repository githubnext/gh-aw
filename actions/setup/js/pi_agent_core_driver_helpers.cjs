// @ts-check

/**
 * Shared helpers for pi-agent-core drivers.
 *
 * Consumed by both the built-in driver (pi_agent_core_driver.cjs) and the
 * sample driver (.github/drivers/pi_agent_core_driver_sample_node.cjs) so
 * that JSONL formatting and provider key resolution have a single source of
 * truth.
 */

"use strict";

// ---------------------------------------------------------------------------
// JSONL emitter
// ---------------------------------------------------------------------------

/** @param {unknown} obj */
function emitJsonl(obj) {
  process.stdout.write(JSON.stringify(obj) + "\n");
}

// ---------------------------------------------------------------------------
// Built-in provider API key resolution
// ---------------------------------------------------------------------------

/**
 * Resolve an API key for the given provider name from well-known environment
 * variables.  Returns `undefined` when no matching variable is set.
 *
 * @param {string} provider
 * @returns {string|undefined}
 */
function getApiKey(provider) {
  switch (provider) {
    case "github-copilot":
    case "copilot":
      return process.env.COPILOT_GITHUB_TOKEN || process.env.GITHUB_TOKEN;
    case "anthropic":
      return process.env.ANTHROPIC_API_KEY;
    case "openai":
    case "codex":
      return process.env.CODEX_API_KEY || process.env.OPENAI_API_KEY;
    default:
      return undefined;
  }
}

module.exports = { emitJsonl, getApiKey };
