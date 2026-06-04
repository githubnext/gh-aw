// @ts-check

/**
 * AWF API proxy /reflect endpoint helpers shared by harnesses.
 *
 * Fetches the api-proxy sidecar's /reflect endpoint and persists the response to disk
 * so that the post-run step summary (awf_reflect_summary.cjs) can include provider and
 * model information without needing the containers to still be running.
 *
 * Provides model-ID parsing helpers that understand the response formats used by all
 * supported providers (OpenAI / Anthropic / Copilot and Gemini).
 *
 * Exported by: copilot_harness.cjs, claude_harness.cjs (and any future agent harnesses)
 */

"use strict";

require("./shim.cjs");

const fs = require("fs");
const path = require("path");
const { withRetry } = require("./error_recovery.cjs");

// AWF API proxy management endpoint for discovering configured LLM providers and available models.
// The api-proxy sidecar exposes /reflect on its management port (port 10000) inside the AWF
// Docker network. From the agent container, the proxy is reachable via the "api-proxy" hostname.
const AWF_API_PROXY_REFLECT_URL = "http://api-proxy:10000/reflect";
// Path inside the agent container where the reflect payload is persisted. The directory is
// co-located with other AWF firewall observability data so it is included in the agent artifact.
const AWF_REFLECT_OUTPUT_PATH = "/tmp/gh-aw/sandbox/firewall/awf-reflect.json";
// Milliseconds to wait for the /reflect endpoint before giving up.
const AWF_REFLECT_TIMEOUT_MS = 60000;
// Milliseconds to wait for each models_url fallback fetch (shorter than the main reflect timeout).
const AWF_MODELS_URL_TIMEOUT_MS = 3000;
// Maximum attempts for models_url fallback fetches when the proxy is not yet ready.
const AWF_MODELS_URL_MAX_ATTEMPTS = 5;
// Base delay between models_url fallback retries. Uses exponential backoff.
const AWF_MODELS_URL_RETRY_BASE_MS = 250;
// Cap for exponential backoff delay between retries.
const AWF_MODELS_URL_RETRY_MAX_MS = 2000;
// Gemini model name prefix stripped from model IDs in the Gemini models API response.
// Example: { name: "models/gemini-1.5-pro" } → "gemini-1.5-pro"
const GEMINI_MODEL_NAME_PREFIX = "models/";

// Default logger used by fetchAWFReflect when no logger is provided via options.
// All lines are prefixed with "[awf-reflect]" for easy grepping in combined logs.
// prettier-ignore
const DEFAULT_REFLECT_LOGGER = /** @type {(msg: string) => void} */ (msg => process.stderr.write(`[awf-reflect] ${new Date().toISOString()} ${msg}\n`));

/**
 * Extract model IDs from a provider API response body.
 *
 * Handles:
 *   - OpenAI / Anthropic / Copilot format: { data: [{ id: "..." }, ...] }
 *   - Gemini format: { models: [{ name: "models/gemini-1.5-pro" }, ...] }
 *
 * @param {object|null} json - Parsed API response
 * @returns {string[]|null} Sorted array of model IDs, or null if unavailable
 */
function extractModelIds(json) {
  if (!json || typeof json !== "object") return null;

  // OpenAI / Anthropic / Copilot format: { data: [{ id: "..." }, ...] }
  if (Array.isArray(json.data)) {
    const ids = json.data.map(m => m && (m.id || m.name)).filter(Boolean);
    return ids.length > 0 ? ids.sort() : null;
  }

  // Gemini format: { models: [{ name: "models/gemini-1.5-pro", ... }, ...] }
  if (Array.isArray(json.models)) {
    const ids = json.models
      .map(m => {
        if (!m) return null;
        const name = m.name || null;
        if (!name) return null;
        return name.startsWith(GEMINI_MODEL_NAME_PREFIX) ? name.slice(GEMINI_MODEL_NAME_PREFIX.length) : name;
      })
      .filter(Boolean);
    return ids.length > 0 ? ids.sort() : null;
  }

  return null;
}

/**
 * Fetch model IDs from a single models_url endpoint via HTTP GET.
 * Used as a fallback when the api-proxy's startup model-fetch returned null.
 * The api-proxy injects the correct auth headers when forwarding the request.
 *
 * @param {string} modelsUrl - URL of the models endpoint on the api-proxy
 * @param {number} timeoutMs - Request timeout in milliseconds
 * @param {(msg: string) => void} logger
 * @returns {Promise<string[]|null>}
 */
async function fetchModelsFromUrl(modelsUrl, timeoutMs, logger) {
  let attemptCounter = 0;
  const retryConfig = {
    maxRetries: AWF_MODELS_URL_MAX_ATTEMPTS - 1,
    // withRetry multiplies delay before the next attempt, so divide by 2 here
    // to preserve the intended first backoff of AWF_MODELS_URL_RETRY_BASE_MS.
    initialDelayMs: Math.ceil(AWF_MODELS_URL_RETRY_BASE_MS / 2),
    maxDelayMs: AWF_MODELS_URL_RETRY_MAX_MS,
    backoffMultiplier: 2,
    jitterMs: 0,
    shouldRetry: error => {
      const original = error?.originalError || error;
      const status = original?.status ?? original?.response?.status ?? null;
      const shouldRetry = status === 503;
      if (shouldRetry && attemptCounter < AWF_MODELS_URL_MAX_ATTEMPTS) {
        logger(`awf-reflect: models fetch returned 503 for ${modelsUrl}; retrying (attempt ${attemptCounter + 1}/${AWF_MODELS_URL_MAX_ATTEMPTS})`);
      }
      return shouldRetry;
    },
  };

  try {
    return await withRetry(
      async () => {
        attemptCounter += 1;
        const ac = new AbortController();
        const timer = setTimeout(() => {
          logger(`awf-reflect: models fetch timed out for ${modelsUrl}`);
          ac.abort();
        }, timeoutMs);
        try {
          const res = await fetch(modelsUrl, { signal: ac.signal });
          if (!res.ok) {
            if (res.status === 503) {
              const err = Object.assign(new Error(`models fetch returned 503 for ${modelsUrl}`), { status: 503 });
              throw err;
            }
            logger(`awf-reflect: models fetch returned ${res.status} for ${modelsUrl}`);
            return null;
          }
          const json = await res.json();
          const models = extractModelIds(json);
          if (models) {
            logger(`awf-reflect: fetched ${models.length} model(s) from ${modelsUrl}`);
          }
          return models;
        } catch (err) {
          const e = /** @type {Error} */ err;
          if (e.name === "AbortError") {
            return null; // already logged above
          }
          const status = e?.status ?? e?.response?.status ?? null;
          if (status === 503) {
            throw e;
          }
          logger(`awf-reflect: models fetch error for ${modelsUrl}: ${e.message}`);
          return null;
        } finally {
          clearTimeout(timer);
        }
      },
      retryConfig,
      `awf-reflect models fetch for ${modelsUrl}`
    );
  } catch (err) {
    const e = /** @type {Error} */ err;
    const original = e?.originalError || e;
    const status = original?.status ?? original?.response?.status ?? null;
    if (status === 503) {
      logger(`awf-reflect: models fetch returned 503 for ${modelsUrl}`);
      return null;
    }
    logger(`awf-reflect: models fetch error for ${modelsUrl}: ${e.message}`);
    return null;
  }
}

/**
 * Enrich a reflect response by fetching models for configured endpoints where
 * the api-proxy's startup fetch left models as null.
 *
 * This is a best-effort fallback: failures are logged but do not throw.
 *
 * @param {object} reflectData - Parsed /reflect response (mutated in-place)
 * @param {number} timeoutMs - Per-request timeout for models_url fetches
 * @param {(msg: string) => void} logger
 * @returns {Promise<void>}
 */
async function enrichReflectModels(reflectData, timeoutMs, logger) {
  const endpoints = Array.isArray(reflectData.endpoints) ? reflectData.endpoints : [];
  const fetches = endpoints
    .filter(ep => ep && ep.configured && ep.models === null && ep.models_url)
    .map(async ep => {
      const models = await fetchModelsFromUrl(ep.models_url, timeoutMs, logger);
      if (models) {
        ep.models = models;
      }
    });
  if (fetches.length > 0) {
    await Promise.allSettled(fetches);
  }
}

/**
 * Fetch the AWF API proxy /reflect endpoint and persist the response to disk.
 *
 * The /reflect endpoint is exposed by the api-proxy sidecar on each started provider port.
 * The active provider's gateway port should be used rather than a hardcoded port, since
 * port 10000 (the OpenAI sidecar) is only started when OpenAI credentials are configured.
 * This information is saved to AWF_REFLECT_OUTPUT_PATH so the post-run GitHub Actions step
 * (awf_reflect_summary.cjs) can include it in the step summary without requiring the
 * containers to still be running.
 *
 * When the api-proxy's startup model-fetch produced null models for a configured provider
 * (e.g. due to a transient upstream error), the function makes a best-effort fallback fetch
 * directly to each endpoint's models_url. The api-proxy injects the correct auth headers
 * when forwarding these requests, giving us a second chance at getting model data.
 *
 * The function is best-effort: any network or parse error is logged but does not abort
 * the agent run.
 *
 * @param {{
 *   reflectUrl?: string,
 *   outputPath?: string,
 *   timeoutMs?: number,
 *   modelsTimeoutMs?: number,
 *   logger?: (msg: string) => void,
 *   writeFileSync?: (path: string, data: string, options: object) => void,
 * }=} options
 * @returns {Promise<{
 *   ok: boolean,
 *   reflectUrl: string,
 *   outputPath: string,
 *   bytesWritten?: number,
 *   reflectData?: object,
 *   reason?: "unexpected_status"|"timeout"|"request_failed",
 *   status?: number,
 *   error?: string,
 * }>}
 */
async function fetchAWFReflect(options) {
  const reflectUrl = (options && options.reflectUrl) || AWF_API_PROXY_REFLECT_URL;
  const outputPath = (options && options.outputPath) || AWF_REFLECT_OUTPUT_PATH;
  const timeoutMs = options && options.timeoutMs != null ? options.timeoutMs : AWF_REFLECT_TIMEOUT_MS;
  const modelsTimeoutMs = options && options.modelsTimeoutMs != null ? options.modelsTimeoutMs : AWF_MODELS_URL_TIMEOUT_MS;
  const logger = (options && options.logger) || DEFAULT_REFLECT_LOGGER;
  const writeFile = (options && options.writeFileSync) || fs.writeFileSync;

  logger(`awf-reflect: fetching ${reflectUrl} (timeout=${timeoutMs}ms)`);

  const ac = new AbortController();
  let timedOut = false;
  const timer = setTimeout(() => {
    timedOut = true;
    logger(`awf-reflect: request timed out after ${timeoutMs}ms`);
    ac.abort();
  }, timeoutMs);

  try {
    const res = await fetch(reflectUrl, { signal: ac.signal });
    if (!res.ok) {
      logger(`awf-reflect: unexpected status ${res.status}, skipping`);
      return {
        ok: false,
        reflectUrl,
        outputPath,
        reason: "unexpected_status",
        status: res.status,
      };
    }
    const reflectData = await res.json();
    // Attempt to fill in null models for configured providers by fetching directly
    // from each endpoint's models_url. The api-proxy injects auth headers when
    // forwarding these requests, so this succeeds without needing the raw API keys.
    await enrichReflectModels(reflectData, modelsTimeoutMs, logger);
    const enrichedBody = JSON.stringify(reflectData);
    fs.mkdirSync(path.dirname(outputPath), { recursive: true });
    writeFile(outputPath, enrichedBody, { encoding: "utf8" });
    logger(`awf-reflect: saved ${enrichedBody.length}B to ${outputPath}`);
    return {
      ok: true,
      reflectUrl,
      outputPath,
      bytesWritten: enrichedBody.length,
      reflectData,
    };
  } catch (err) {
    const e = /** @type {Error} */ err;
    if (e.name === "AbortError") {
      return {
        ok: false,
        reflectUrl,
        outputPath,
        reason: "timeout",
        error: timedOut ? `request timed out after ${timeoutMs}ms` : e.message,
      };
    }
    logger(`awf-reflect: request failed: ${e.message}`);
    return {
      ok: false,
      reflectUrl,
      outputPath,
      reason: "request_failed",
      error: e.message,
    };
  } finally {
    clearTimeout(timer);
  }
}

/**
 * Resolve Copilot SDK BYOK custom provider configuration from AWF /reflect data.
 * Chooses a configured endpoint and maps it to an OpenAI-compatible provider base URL.
 * Returns null when no suitable endpoint is found (e.g. no reflect data, or endpoints not
 * configured).
 *
 * Requires live reflect data passed directly via `reflectData`.
 *
 * @param {{
 *   model?: string,
 *   reflectData: object | null | undefined,
 *   logger?: (msg: string) => void,
 * }} [options]
 * @returns {{ model: string, provider: { type: "openai", baseUrl: string } } | null}
 */
function resolveCopilotSDKCustomProviderFromReflect(options) {
  const configuredModel = typeof options?.model === "string" ? options.model.trim() : "";
  const logger = (options && options.logger) || DEFAULT_REFLECT_LOGGER;

  const reflectData = options?.reflectData;
  if (reflectData == null) {
    logger("sdk-mode: no reflect data provided; cannot resolve custom provider");
    return null;
  }

  const endpoints = Array.isArray(reflectData?.endpoints) ? reflectData.endpoints.filter(ep => ep && ep.configured === true) : [];
  if (endpoints.length === 0) {
    logger("sdk-mode: no configured endpoints in awf-reflect data; cannot resolve custom provider");
    return null;
  }

  const endpoint = (configuredModel ? endpoints.find(ep => Array.isArray(ep.models) && ep.models.includes(configuredModel)) : null) || endpoints.find(ep => String(ep.provider || "").toLowerCase() === "copilot") || endpoints[0];

  let baseUrl = "";
  if (typeof endpoint?.models_url === "string" && endpoint.models_url) {
    try {
      baseUrl = new URL(endpoint.models_url).origin;
    } catch {
      // ignore malformed URL and fall back to port-based construction below
    }
  }
  if (!baseUrl && endpoint?.port != null) {
    baseUrl = `http://api-proxy:${String(endpoint.port)}`;
  }
  if (!baseUrl) {
    logger("sdk-mode: unable to derive provider baseUrl from awf-reflect endpoint data; cannot resolve custom provider");
    return null;
  }

  let model = configuredModel;
  if (!model && Array.isArray(endpoint?.models)) {
    const firstModel = endpoint.models.find(m => typeof m === "string" && m.trim().length > 0);
    model = typeof firstModel === "string" ? firstModel.trim() : "";
  }
  if (!model) {
    logger("sdk-mode: unable to derive model for custom provider from awf-reflect; cannot resolve custom provider");
    return null;
  }

  logger(`sdk-mode: custom provider resolved from awf-reflect (provider=${String(endpoint.provider || "unknown")} baseUrl=${baseUrl} model=${model})`);
  return {
    model,
    provider: { type: "openai", baseUrl },
  };
}

if (typeof module !== "undefined" && module.exports) {
  module.exports = {
    AWF_API_PROXY_REFLECT_URL,
    AWF_REFLECT_OUTPUT_PATH,
    AWF_REFLECT_TIMEOUT_MS,
    AWF_MODELS_URL_TIMEOUT_MS,
    AWF_MODELS_URL_MAX_ATTEMPTS,
    AWF_MODELS_URL_RETRY_BASE_MS,
    AWF_MODELS_URL_RETRY_MAX_MS,
    GEMINI_MODEL_NAME_PREFIX,
    enrichReflectModels,
    extractModelIds,
    fetchAWFReflect,
    fetchModelsFromUrl,
    resolveCopilotSDKCustomProviderFromReflect,
  };
}
