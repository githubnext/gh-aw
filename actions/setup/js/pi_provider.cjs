// @ts-check

/**
 * Pi Provider Extension for gh-aw
 *
 * Calls the AWF API proxy /reflect endpoint at session start to dynamically
 * discover the open LLM inference paths configured for this run.  This gives
 * operators runtime visibility into which provider/model combination is active
 * and verifies that the expected gateway port is reachable before the agent
 * starts working.
 *
 * When the model uses provider/model format (e.g. "copilot/claude-sonnet-4"),
 * the extension logs the matched endpoint so failures can be diagnosed without
 * inspecting container internals.
 *
 * The extension is automatically added to every Pi agent invocation by the
 * gh-aw compiler alongside pi_steering_extension.cjs.  No workflow frontmatter
 * configuration is required.
 *
 * Configuration (read from environment variables):
 *   PI_MODEL   The engine.model value; may be "provider/model" or bare "model".
 */

"use strict";

const { fetchAWFReflect, AWF_API_PROXY_REFLECT_URL, AWF_REFLECT_OUTPUT_PATH, AWF_REFLECT_TIMEOUT_MS, AWF_MODELS_URL_TIMEOUT_MS } = require("./awf_reflect.cjs");

// Default logger: prefixed with "[gh-aw/pi-provider]" for easy grepping.
// prettier-ignore
const DEFAULT_LOGGER = /** @type {(msg: string) => void} */ (msg => process.stderr.write(`[gh-aw/pi-provider] ${new Date().toISOString()} ${msg}\n`));

/**
 * Extract the provider prefix from a "provider/model" string.
 * Returns an empty string when no slash is present (bare model name).
 *
 * @param {string} model
 * @returns {string}
 */
function extractProviderFromModel(model) {
  if (!model) return "";
  const slashIdx = model.indexOf("/");
  if (slashIdx <= 0) return "";
  return model.slice(0, slashIdx).toLowerCase();
}

/**
 * Resolve the expected LLM gateway base URL for a given provider prefix.
 * Returns null when the provider is not one of the well-known AWF sidecar providers.
 *
 * @param {string} provider - Lowercase provider prefix (e.g. "copilot", "anthropic").
 * @returns {string|null}
 */
function resolveGatewayUrl(provider) {
  const GATEWAY_PORTS = /** @type {Record<string, number>} */ {
    copilot: 10002,
    anthropic: 10000,
    openai: 10001,
    codex: 10001,
    google: 10003,
  };
  const port = GATEWAY_PORTS[provider];
  if (!port) return null;
  return `http://host.docker.internal:${port}`;
}

/**
 * Pi provider extension for gh-aw.
 *
 * Subscribes to the `agent_start` Pi SDK event to log the active provider and
 * gateway URL.  Registers a `process.once('beforeExit')` handler to fetch the
 * AWF /reflect endpoint after the agent finishes its last turn — matching the
 * timing used by copilot_harness.cjs and claude_harness.cjs, which call
 * fetchAWFReflect after the agent subprocess exits.  Calling /reflect at
 * agent_start is too early: the api-proxy management endpoint may not yet be
 * serving the full /reflect response before the first LLM turn completes.
 *
 * The reflect fetch is best-effort: any network or parse error is logged but
 * does not abort the agent session or affect the process exit code.
 *
 * @param {any} pi - Pi ExtensionAPI instance
 * @returns {void}
 */
function piProviderExtension(pi) {
  const log = DEFAULT_LOGGER;

  pi.on("agent_start", async () => {
    const model = process.env.PI_MODEL || "";
    const provider = extractProviderFromModel(model);

    if (provider) {
      const gatewayUrl = resolveGatewayUrl(provider);
      if (gatewayUrl) {
        log(`provider=${provider} model=${model} gateway=${gatewayUrl}`);
      } else {
        log(`provider=${provider} model=${model} (no known AWF gateway port for this provider)`);
      }
    } else {
      log(`model=${model || "(not set)"} (no provider prefix — defaulting to Copilot gateway)`);
    }
  });

  // Fetch AWF API proxy reflection data after Pi finishes all turns.
  // beforeExit fires when the Node.js event loop is draining (Pi is done),
  // while the AWF container is still running — the same lifecycle point at
  // which copilot_harness.cjs and claude_harness.cjs call fetchAWFReflect.
  // Scheduling the async fetch here keeps the event loop alive until the
  // request completes (or times out), then the process exits normally.
  // Note: if Pi calls process.exit() directly, beforeExit will not fire and
  // the reflect file will simply not be written (same as the previous
  // agent_start failure path — the step summary will show no provider data).
  process.once("beforeExit", async () => {
    await fetchAWFReflect({
      reflectUrl: AWF_API_PROXY_REFLECT_URL,
      outputPath: AWF_REFLECT_OUTPUT_PATH,
      timeoutMs: AWF_REFLECT_TIMEOUT_MS,
      modelsTimeoutMs: AWF_MODELS_URL_TIMEOUT_MS,
      logger: log,
    });
  });
}

module.exports = piProviderExtension;
module.exports.extractProviderFromModel = extractProviderFromModel;
module.exports.resolveGatewayUrl = resolveGatewayUrl;
