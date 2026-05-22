// @ts-check

/**
 * Pi Provider Extension for gh-aw
 *
 * Registers Pi providers from the AWF-injected environment and calls the AWF
 * API proxy /reflect endpoint at session start to dynamically discover the
 * open LLM inference paths configured for this run. This gives operators
 * runtime visibility into which provider/model combination is active and
 * verifies that the expected gateway port is reachable before the agent starts
 * working.
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
 *   GH_AW_PI_MODEL   The original engine.model value; may be "provider/model"
 *                    or bare "model". Preferred over PI_MODEL so gh-aw can pass
 *                    model context to extensions without changing Pi CLI behavior.
 *   PI_MODEL         Legacy fallback used when GH_AW_PI_MODEL is not set.
 */

"use strict";

const { fetchAWFReflect, AWF_API_PROXY_REFLECT_URL, AWF_REFLECT_OUTPUT_PATH, AWF_REFLECT_TIMEOUT_MS, AWF_MODELS_URL_TIMEOUT_MS } = require("./awf_reflect.cjs");

// Default logger: prefixed with "[gh-aw/pi-provider]" for easy grepping.
// prettier-ignore
const DEFAULT_LOGGER = /** @type {(msg: string) => void} */ (msg => process.stderr.write(`[gh-aw/pi-provider] ${new Date().toISOString()} ${msg}\n`));

/**
 * Return the workflow-configured model string exposed to Pi extensions.
 * GH_AW_PI_MODEL takes precedence because gh-aw sets it explicitly for extensions
 * while continuing to pass the CLI model via --model. PI_MODEL remains a legacy
 * fallback for older callers.
 *
 * @returns {string}
 */
function getConfiguredModel() {
  return process.env.GH_AW_PI_MODEL || process.env.PI_MODEL || "";
}

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
 * Uses the "api-proxy" Docker service hostname so the URL reflects the actual
 * address used by Pi's models.json routing within the AWF Docker network.
 *
 * @param {string} provider - Lowercase provider prefix (e.g. "copilot", "anthropic").
 * @returns {string|null}
 */
function resolveGatewayUrl(provider) {
  const GATEWAY_PORTS = /** @type {Record<string, number>} */ {
    copilot: 10002,
    anthropic: 10001,
    openai: 10000,
    codex: 10000,
    google: 10003,
  };
  const port = GATEWAY_PORTS[provider];
  if (!port) return null;
  return `http://api-proxy:${port}`;
}

/**
 * Register a Pi provider and any aliases.
 *
 * @param {any} pi
 * @param {string[]} names
 * @param {Record<string, any>} config
 * @param {(msg: string) => void} logger
 */
function registerProviderAliases(pi, names, config, logger) {
  for (const name of names) {
    pi.registerProvider(name, config);
    logger(`registered provider=${name}`);
  }
}

/**
 * Register all supported Pi providers discovered from the environment.
 *
 * @param {any} pi
 * @param {(msg: string) => void} logger
 * @returns {number}
 */
function registerConfiguredProviders(pi, logger) {
  let registeredCount = 0;

  const copilotToken = process.env.COPILOT_GITHUB_TOKEN || process.env.GITHUB_TOKEN;
  if (copilotToken) {
    registerProviderAliases(
      pi,
      ["github-copilot", "copilot"],
      {
        apiKey: copilotToken,
        api: "openai-completions",
        ...(process.env.GITHUB_COPILOT_BASE_URL ? { baseUrl: process.env.GITHUB_COPILOT_BASE_URL } : {}),
      },
      logger
    );
    registeredCount += 2;
  }

  if (process.env.ANTHROPIC_API_KEY) {
    registerProviderAliases(
      pi,
      ["anthropic"],
      {
        apiKey: process.env.ANTHROPIC_API_KEY,
        api: "anthropic",
        ...(process.env.ANTHROPIC_BASE_URL ? { baseUrl: process.env.ANTHROPIC_BASE_URL } : {}),
      },
      logger
    );
    registeredCount += 1;
  }

  const openAIKey = process.env.CODEX_API_KEY || process.env.OPENAI_API_KEY;
  if (openAIKey) {
    registerProviderAliases(
      pi,
      ["openai", "codex"],
      {
        apiKey: openAIKey,
        api: "openai-completions",
        ...(process.env.OPENAI_BASE_URL ? { baseUrl: process.env.OPENAI_BASE_URL } : {}),
      },
      logger
    );
    registeredCount += 2;
  }

  if (registeredCount === 0) {
    logger("no provider credentials detected for Pi provider registration");
  }

  return registeredCount;
}

/**
 * Pi provider extension for gh-aw.
 *
 * Registers providers immediately, then subscribes to the `agent_start` and `agent_end`
 * Pi SDK events and calls the AWF /reflect endpoint to discover and log the open LLM
 * inference paths before the agent begins its first turn and again after it finishes.
 * The post-run fetch is the authoritative snapshot used by the step summary; the pre-run
 * fetch captures the initial proxy state for diagnostics in case the session exits
 * unexpectedly before reaching `agent_end`.
 * Both calls are best-effort: any network or parse error is logged but does not abort the
 * agent session.
 *
 * @param {any} pi - Pi ExtensionAPI instance
 * @returns {void}
 */
function piProviderExtension(pi) {
  const log = DEFAULT_LOGGER;
  registerConfiguredProviders(pi, log);

  pi.on("agent_start", async () => {
    const model = getConfiguredModel();
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

    // Fetch AWF API proxy reflection data before the agent runs to capture initial proxy state.
    // This is best-effort: failures are logged but do not affect the agent session.
    // Skip when AWF_REFLECT_ENABLED is not "1" (e.g. sandbox.agent: false — no api-proxy running).
    if (process.env.AWF_REFLECT_ENABLED === "1") {
      await fetchAWFReflect({
        reflectUrl: AWF_API_PROXY_REFLECT_URL,
        outputPath: AWF_REFLECT_OUTPUT_PATH,
        timeoutMs: AWF_REFLECT_TIMEOUT_MS,
        modelsTimeoutMs: AWF_MODELS_URL_TIMEOUT_MS,
        logger: log,
      });
    }
  });

  pi.on("agent_end", async () => {
    // Fetch AWF API proxy reflection data after the agent finishes for the post-run step summary.
    // This is best-effort: failures are logged but do not affect the agent exit code.
    // Skip when AWF_REFLECT_ENABLED is not "1" (e.g. sandbox.agent: false — no api-proxy running).
    if (process.env.AWF_REFLECT_ENABLED === "1") {
      await fetchAWFReflect({
        reflectUrl: AWF_API_PROXY_REFLECT_URL,
        outputPath: AWF_REFLECT_OUTPUT_PATH,
        timeoutMs: AWF_REFLECT_TIMEOUT_MS,
        modelsTimeoutMs: AWF_MODELS_URL_TIMEOUT_MS,
        logger: log,
      });
    }
  });
}

module.exports = piProviderExtension;
module.exports.getConfiguredModel = getConfiguredModel;
module.exports.extractProviderFromModel = extractProviderFromModel;
module.exports.resolveGatewayUrl = resolveGatewayUrl;
module.exports.registerConfiguredProviders = registerConfiguredProviders;
