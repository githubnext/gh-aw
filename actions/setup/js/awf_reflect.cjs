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
const { withRetry, sleep } = require("./error_recovery.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

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
// Delay before the first models_url probe when using GitHub OIDC auth with the local api-proxy.
// This reduces startup-race 503s while the proxy completes OIDC token exchange.
const AWF_MODELS_URL_OIDC_INITIAL_DELAY_MS_DEFAULT = 5000;
// Gemini model name prefix stripped from model IDs in the Gemini models API response.
// Example: { name: "models/gemini-1.5-pro" } → "gemini-1.5-pro"
const GEMINI_MODEL_NAME_PREFIX = "models/";
const REFLECT_PROVIDER_GITHUB = "github";
const REFLECT_PROVIDER_OPENAI = "openai";
const REFLECT_PROVIDER_ANTHROPIC = "anthropic";

/**
 * @typedef {{
 *   configured?: boolean,
 *   models_url?: string | null,
 *   port?: number | null,
 *   provider?: string,
 * }} ReflectEndpoint
 */

/**
 * @typedef {{
 *   endpoints?: ReflectEndpoint[],
 * }} ReflectData
 */

const REFLECT_PROVIDER_ALIASES = {
  // Only GitHub has multiple externally-visible aliases in reflect payloads.
  github: new Set(["github", "copilot", "github-copilot", "github_models"]),
  openai: new Set(["openai"]),
  anthropic: new Set(["anthropic"]),
};

// Default logger used by fetchAWFReflect when no logger is provided via options.
// All lines are prefixed with "[awf-reflect]" for easy grepping in combined logs.
// prettier-ignore
const DEFAULT_REFLECT_LOGGER = /** @type {(msg: string) => void} */ (msg => process.stderr.write(`[awf-reflect] ${new Date().toISOString()} ${msg}\n`));

/**
 * Normalize provider IDs used in reflect/provider resolution.
 *
 * @param {unknown} provider
 * @param {string} [fallback]
 * @returns {string}
 */
function normalizeReflectProviderName(provider, fallback = "") {
  const normalized = String(provider || "")
    .toLowerCase()
    .trim();
  return normalized || fallback;
}

/**
 * Extract model IDs from a provider API response body.
 *
 * Handles:
 *   - OpenAI / Anthropic / Copilot format: { data: [{ id: "..." }, ...] }
 *   - Gemini format: { models: [{ name: "models/gemini-1.5-pro" }, ...] }
 *
 * @param {any|null} json - Parsed API response
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
  let isInitialProbeDelayed = false;
  try {
    const modelsHost = new URL(modelsUrl).hostname.toLowerCase();
    isInitialProbeDelayed = process.env.AWF_AUTH_TYPE === "github-oidc" && modelsHost === "api-proxy";
  } catch {
    // Ignore invalid URL parsing and proceed without startup delay.
  }
  if (isInitialProbeDelayed) {
    const configuredDelay = Number.parseInt(process.env.AWF_MODELS_URL_OIDC_INITIAL_DELAY_MS || "", 10);
    const initialProbeDelay = Number.isFinite(configuredDelay) && configuredDelay >= 0 ? configuredDelay : AWF_MODELS_URL_OIDC_INITIAL_DELAY_MS_DEFAULT;
    if (initialProbeDelay > 0) {
      logger(`awf-reflect: delaying initial models probe for ${modelsUrl} by ${initialProbeDelay}ms (AWF_AUTH_TYPE=github-oidc)`);
      await sleep(initialProbeDelay);
    }
  }

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
          if (err instanceof Error && err.name === "AbortError") {
            return null; // already logged above
          }
          /** @type {any} */
          const e = err;
          const status = e?.status ?? e?.response?.status ?? null;
          if (status === 503) {
            throw e;
          }
          logger(`awf-reflect: models fetch error for ${modelsUrl}: ${getErrorMessage(err)}`);
          return null;
        } finally {
          clearTimeout(timer);
        }
      },
      retryConfig,
      `awf-reflect models fetch for ${modelsUrl}`
    );
  } catch (err) {
    /** @type {any} */
    const e = err;
    const original = e?.originalError || e;
    const status = original?.status ?? original?.response?.status ?? null;
    if (status === 503) {
      logger(`awf-reflect: models fetch returned 503 for ${modelsUrl}`);
      return null;
    }
    logger(`awf-reflect: models fetch error for ${modelsUrl}: ${getErrorMessage(err)}`);
    return null;
  }
}

/**
 * Enrich a reflect response by fetching models for configured endpoints where
 * the api-proxy's startup fetch left models as null.
 *
 * This is a best-effort fallback: failures are logged but do not throw.
 *
 * @param {any} reflectData - Parsed /reflect response (mutated in-place)
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
    /** @type {any} */
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
 * Returns true when the model name matches well-known Anthropic naming patterns:
 * "claude-*" prefix, or "-opus", "-haiku", or "-sonnet" as a segment or suffix.
 *
 * @param {string} model - Lower-cased, trimmed model name.
 * @returns {boolean}
 */
function isAnthropicModelName(model) {
  return model.startsWith("claude-") || model.includes("-opus-") || model.endsWith("-opus") || model.includes("-haiku-") || model.endsWith("-haiku") || model.includes("-sonnet-") || model.endsWith("-sonnet");
}

/**
 * Returns true when the model name matches well-known OpenAI naming patterns:
 * "gpt-*" prefix, or o1/o3/o4 reasoning models.
 *
 * @param {string} model - Lower-cased, trimmed model name.
 * @returns {boolean}
 */
function isOpenAIModelName(model) {
  return model.startsWith("gpt-") || /^o[134][-.]/.test(model) || model === "o1" || model === "o3" || model === "o4";
}

/**
 * Look up a model entry in the models.json catalog, case-insensitively.
 *
 * @param {any} modelsJson
 * @param {string} modelName
 * @param {string | null | undefined} [providerName]
 * @returns {any | null}
 */
function getCatalogModelEntry(modelsJson, modelName, providerName) {
  const model = String(modelName || "")
    .toLowerCase()
    .trim();
  const provider = String(providerName || "")
    .toLowerCase()
    .trim();
  if (!model || modelsJson == null || typeof modelsJson !== "object" || Array.isArray(modelsJson)) {
    return null;
  }
  const providers = modelsJson.providers;
  if (!providers || typeof providers !== "object" || Array.isArray(providers)) {
    return null;
  }
  const providerEntries = provider
    ? Object.entries(providers).filter(
        ([name]) =>
          String(name || "")
            .toLowerCase()
            .trim() === provider
      )
    : Object.entries(providers);
  for (const [, providerData] of providerEntries) {
    const models = providerData && typeof providerData === "object" ? providerData.models : null;
    if (!models || typeof models !== "object" || Array.isArray(models)) continue;
    for (const [catalogModel, catalogEntry] of Object.entries(models)) {
      if (
        String(catalogModel || "")
          .toLowerCase()
          .trim() === model
      ) {
        return catalogEntry && typeof catalogEntry === "object" && !Array.isArray(catalogEntry) ? catalogEntry : null;
      }
    }
  }
  return null;
}

/**
 * Infer the Copilot SDK provider type for a given endpoint provider name and model name.
 *
 * The SDK's `ProviderConfig.type` field determines which API format the SDK uses when
 * communicating with the `baseUrl` endpoint:
 *   - "anthropic" — Anthropic Messages API (required for direct Anthropic API endpoints)
 *   - "azure"     — Azure OpenAI API
 *   - "openai"    — OpenAI-compatible API (default; used for Copilot, OpenAI, Gemini, etc.)
 *
 * Resolution order:
 *   1. Endpoint provider name mapping (most authoritative for BYOK endpoints).
 *   2. Model catalog lookup via the `modelsJson` catalog (explicit `provider_type` field).
 *   3. Well-known model name heuristics (e.g. "claude-*" → "anthropic", "gpt-*" → "openai").
 *   4. Default: "openai".
 *
 * @param {string} endpointProvider - The `provider` field from the AWF reflect endpoint entry.
 * @param {string} modelName - The resolved model name to use for heuristic fallback.
 * @param {any} catalogEntryOrModelsJson - Matching models.json catalog entry or full catalog (optional).
 * @returns {"openai" | "azure" | "anthropic"}
 */
function inferProviderTypeForModel(endpointProvider, modelName, catalogEntryOrModelsJson) {
  // 1. Endpoint provider name mapping.
  const ep = String(endpointProvider || "")
    .toLowerCase()
    .trim();
  if (ep === "anthropic") return "anthropic";
  if (ep === "azure" || ep === "azure-openai" || ep === "azure_openai") return "azure";
  if (ep === "openai") return "openai";
  // GitHub Copilot provider is a multi-model proxy that always uses OpenAI wire protocol.
  if (ep === "copilot" || ep === "github-copilot") return "openai";
  // For unknown providers, fall through to model-based lookup.

  const model = String(modelName || "")
    .toLowerCase()
    .trim();
  const catalogEntry =
    catalogEntryOrModelsJson && typeof catalogEntryOrModelsJson === "object" && !Array.isArray(catalogEntryOrModelsJson) && "providers" in catalogEntryOrModelsJson
      ? getCatalogModelEntry(catalogEntryOrModelsJson, model)
      : catalogEntryOrModelsJson;

  // 2. Model catalog lookup.
  if (model) {
    const pt = catalogEntry && typeof catalogEntry.provider_type === "string" ? catalogEntry.provider_type.trim() : "";
    if (pt === "anthropic" || pt === "azure" || pt === "openai") return /** @type {"openai" | "azure" | "anthropic"} */ pt;
  }

  // 3. Well-known model name heuristics.
  if (model) {
    if (isAnthropicModelName(model)) return "anthropic";
    if (isOpenAIModelName(model)) return "openai";
  }

  // 4. Default.
  return "openai";
}

/**
 * Infer the SDK wire API for a model.
 *
 * Resolution order:
 *   1. For Anthropic provider types: undefined (wireApi ignored by SDK).
 *   2. `models.json` explicit `wire_api`/`wireApi`.
 *   3. Heuristic default for OpenAI/Azure-compatible models: "completions".
 *
 * @param {"openai" | "azure" | "anthropic"} providerType
 * @param {string} modelName
 * @param {any} catalogEntryOrModelsJson
 * @returns {"completions" | "responses" | undefined}
 */
function inferWireApiForModel(providerType, modelName, catalogEntryOrModelsJson) {
  if (providerType === "anthropic") {
    return undefined;
  }
  const model = String(modelName || "").trim();
  if (!model) return undefined;
  const catalogEntry =
    catalogEntryOrModelsJson && typeof catalogEntryOrModelsJson === "object" && !Array.isArray(catalogEntryOrModelsJson) && "providers" in catalogEntryOrModelsJson
      ? getCatalogModelEntry(catalogEntryOrModelsJson, model)
      : catalogEntryOrModelsJson;
  // Keep the camelCase fallback for defensive compatibility with injected catalog
  // objects that bypass the normalized models.json pipeline.
  const rawWireApi = typeof catalogEntry?.wire_api === "string" ? catalogEntry.wire_api : typeof catalogEntry?.wireApi === "string" ? catalogEntry.wireApi : "";
  const normalizedWireApi = String(rawWireApi || "")
    .toLowerCase()
    .trim();
  if (normalizedWireApi === "responses" || normalizedWireApi === "completions") {
    return /** @type {"responses" | "completions"} */ normalizedWireApi;
  }
  return "completions";
}

/**
 * Derive a base URL string from an endpoint object.
 * Prefers the origin of `models_url`; falls back to `http://api-proxy:<port>`.
 * Returns an empty string when neither is available.
 *
 * @param {{ models_url?: string | null, port?: number | null }} endpoint
 * @returns {string}
 */
function endpointBaseUrl(endpoint) {
  if (typeof endpoint.models_url === "string" && endpoint.models_url) {
    try {
      return new URL(endpoint.models_url).origin;
    } catch {
      // fall through to port-based construction
    }
  }
  if (endpoint.port != null) {
    return `http://api-proxy:${String(endpoint.port)}`;
  }
  return "";
}

/**
 * Resolve a configured provider endpoint from AWF /reflect data.
 *
 * @param {{
 *   provider?: string,
 *   reflectData: ReflectData | null | undefined,
 *   logger?: (msg: string) => void,
 * }} options
 * @returns {{ provider: string, endpointProvider: string, port: number|null, baseUrl: string } | null}
 */
function resolveProviderEndpointFromReflect(options) {
  const logger = (options && options.logger) || DEFAULT_REFLECT_LOGGER;
  const provider = normalizeReflectProviderName(options?.provider, "openai");
  const reflectData = options?.reflectData;
  /** @type {{ endpoints?: unknown } | null} */
  const reflectRecord = reflectData && typeof reflectData === "object" ? reflectData : null;
  const endpointCandidates = Array.isArray(reflectRecord?.endpoints) ? reflectRecord.endpoints : [];
  const endpoints = endpointCandidates.filter(/** @param {any} ep */ ep => ep && ep.configured === true);
  if (endpoints.length === 0) {
    logger(`awf-reflect: no configured endpoints available while resolving provider=${provider}`);
    return null;
  }

  /** @param {string} endpointProvider */
  const endpointProviderMatches = endpointProvider => {
    // Keep aliases aligned with pkg/workflow/llm_provider.go (llmProviderAliases).
    // If alias handling changes, update both places in the same PR.
    const normalized = normalizeReflectProviderName(endpointProvider);
    if (!normalized) return false;
    if (provider === REFLECT_PROVIDER_GITHUB) {
      return REFLECT_PROVIDER_ALIASES.github.has(normalized);
    }
    if (provider === REFLECT_PROVIDER_OPENAI) {
      return REFLECT_PROVIDER_ALIASES.openai.has(normalized);
    }
    if (provider === REFLECT_PROVIDER_ANTHROPIC) {
      return REFLECT_PROVIDER_ALIASES.anthropic.has(normalized);
    }
    return normalized === provider;
  };

  const matched = endpoints.find(ep => typeof ep?.provider === "string" && endpointProviderMatches(ep.provider)) || endpoints[0];
  const baseUrl = endpointBaseUrl(matched);
  if (!baseUrl) {
    logger(`awf-reflect: matched provider=${provider} but could not derive baseUrl`);
    return null;
  }
  const endpointProvider = String(matched.provider || "unknown");
  const parsedPort = matched.port == null ? null : Number(matched.port);
  const port = Number.isFinite(parsedPort) ? parsedPort : null;
  logger(`awf-reflect: provider=${provider} mapped to endpoint provider=${endpointProvider} baseUrl=${baseUrl}`);
  return { provider, endpointProvider, port, baseUrl };
}

/**
 * Resolve multi-provider BYOK configuration from AWF /reflect data.
 *
 * Returns `null` when no configured endpoints are present or the data is
 * unavailable.
 *
 * Each endpoint becomes a `NamedProviderConfig` (using the endpoint's `provider`
 * field as the stable name) and every model advertised by that endpoint becomes a
 * `ProviderModelConfig` tuple `{ id, provider }` referencing it.  Callers can
 * derive provider-qualified selection ids as `"<providerName>/<modelId>"` if needed.
 *
 * The primary model is the first model that matches `options.model` (if set),
 * otherwise the first model across all providers.
 *
 * @param {{
 *   model?: string,
 *   reflectData: ReflectData | null | undefined,
 *   modelsJson?: object | null,
 *   logger?: (msg: string) => void,
 * }} [options]
 * @returns {{
 *   model: string,
 *   providers: Array<{ name: string, type: "openai" | "azure" | "anthropic", baseUrl: string, wireApi?: "completions" | "responses" }>,
 *   models: Array<{ id: string, provider: string }>,
 * } | null}
 */
function resolveMultiProviderFromReflect(options) {
  const configuredModel = typeof options?.model === "string" ? options.model.trim() : "";
  const logger = (options && options.logger) || DEFAULT_REFLECT_LOGGER;

  const reflectData = options?.reflectData;
  if (reflectData == null) {
    logger("sdk-mode(multi): no reflect data provided; cannot resolve multi-provider config");
    return null;
  }

  /** @type {any} */
  const rd = reflectData;
  const endpoints = Array.isArray(rd?.endpoints) ? rd.endpoints.filter(ep => ep && ep.configured === true) : [];

  if (endpoints.length === 0) {
    logger(`sdk-mode(multi): no configured endpoints in awf-reflect data; cannot build multi-provider config`);
    return null;
  }

  /** @type {Array<{ name: string, type: "openai" | "azure" | "anthropic", baseUrl: string, wireApi?: "completions" | "responses" }>} */
  const providers = [];
  /** @type {Array<{ id: string, provider: string }>} */
  const models = [];

  // Track used provider names to avoid duplicates when multiple endpoints share the same
  // provider label (e.g. two "copilot" entries at different ports).
  /** @type {Map<string, number>} */
  const providerNameCount = new Map();

  for (const endpoint of endpoints) {
    const baseUrl = endpointBaseUrl(endpoint);
    if (!baseUrl) {
      logger(`sdk-mode(multi): skipping endpoint with no resolvable baseUrl (provider=${String(endpoint.provider || "unknown")})`);
      continue;
    }

    const rawProviderName = String(endpoint.provider || "").trim();
    if (!rawProviderName) {
      logger("sdk-mode(multi): skipping endpoint with no provider name");
      continue;
    }

    // Ensure unique provider names by appending a suffix when the same name appears twice.
    const existing = providerNameCount.get(rawProviderName) ?? 0;
    providerNameCount.set(rawProviderName, existing + 1);
    const providerName = existing === 0 ? rawProviderName : `${rawProviderName}-${existing}`;

    const endpointModels = Array.isArray(endpoint.models) ? endpoint.models.filter(m => typeof m === "string" && m.trim().length > 0) : [];

    // Infer provider type and wire API using the configured model if available,
    // otherwise fall back to the first model.
    // For multi-model providers (e.g. Copilot), different models may have different wire APIs,
    // so we prefer the configured model to ensure the correct wireApi is selected.
    const firstModel = endpointModels.length > 0 ? endpointModels[0] : "";
    const modelForInference = configuredModel && endpointModels.includes(configuredModel) ? configuredModel : firstModel;
    const catalogProviderName = rawProviderName.toLowerCase() === "copilot" ? "github-copilot" : rawProviderName;
    const catalogEntry = modelForInference ? getCatalogModelEntry(options?.modelsJson ?? null, modelForInference, catalogProviderName) : null;
    const providerType = inferProviderTypeForModel(rawProviderName, modelForInference, catalogEntry);
    const wireApi = inferWireApiForModel(providerType, modelForInference, catalogEntry);

    logger(
      `sdk-mode(multi): resolved provider="${providerName}" (raw="${rawProviderName}") type="${providerType}" wireApi="${wireApi || "(none)"}" ` +
        `inferredFrom="${modelForInference}" modelCount=${endpointModels.length} baseUrl="${baseUrl}"`
    );

    providers.push({
      name: providerName,
      type: providerType,
      baseUrl,
      ...(wireApi ? { wireApi } : {}),
    });

    for (const modelId of endpointModels) {
      models.push({ id: modelId, provider: providerName });
    }
  }

  if (providers.length === 0) {
    logger("sdk-mode(multi): no providers resolved from awf-reflect data; cannot build multi-provider config");
    return null;
  }

  // Determine the primary model: prefer the configured model if it appears in the model list;
  // otherwise fall back to the first model across all providers.
  let primaryModel = "";
  if (configuredModel) {
    const match = models.find(m => m.id === configuredModel);
    if (match) primaryModel = match.id;
  }
  if (!primaryModel && models.length > 0) {
    primaryModel = models[0].id;
  }

  if (!primaryModel) {
    logger("sdk-mode(multi): no models found in awf-reflect endpoints; cannot build multi-provider config");
    return null;
  }

  logger(`sdk-mode(multi): resolved ${providers.length} providers, ${models.length} models (primary model: ${primaryModel})`);
  return { model: primaryModel, providers, models };
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
    getCatalogModelEntry,
    inferProviderTypeForModel,
    inferWireApiForModel,
    normalizeReflectProviderName,
    resolveProviderEndpointFromReflect,
    resolveMultiProviderFromReflect,
  };
}
