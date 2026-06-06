// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");

const DEFAULT_MODELS_PATH = path.join(__dirname, "models.json");

/** @type {Array<{id: string, provider: string, model: string, pricing: Record<string, number>}> | null | undefined} */
let _catalog = undefined;

function getModelsPath() {
  const override = process.env.GH_AW_MODELS_JSON_PATH;
  return override && override.trim() ? override : DEFAULT_MODELS_PATH;
}

/**
 * @param {unknown} value
 * @returns {number}
 */
function parsePrice(value) {
  if (typeof value === "number" && Number.isFinite(value)) return value;
  if (typeof value === "string" && value.trim()) {
    const parsed = Number.parseFloat(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return 0;
}

/**
 * @returns {Array<{id: string, provider: string, model: string, pricing: Record<string, number>}>}
 */
function loadCatalog() {
  if (_catalog !== undefined) {
    return _catalog || [];
  }

  try {
    const raw = fs.readFileSync(getModelsPath(), "utf8");
    const parsed = JSON.parse(raw);
    const providers = parsed?.providers && typeof parsed.providers === "object" ? parsed.providers : {};
    _catalog = Object.entries(providers).flatMap(([providerName, providerData]) => {
      const provider = normalizeProvider(providerName);
      if (!provider) return [];
      const models = providerData?.models && typeof providerData.models === "object" ? providerData.models : {};
      return Object.entries(models).map(([modelName, modelData]) => {
        const model = normalizeModel(modelName);
        const id = `${provider}/${model}`;
        /** @type {Record<string, number>} */
        const pricing = {};
        for (const [key, value] of Object.entries(modelData?.cost || {})) {
          pricing[key] = parsePrice(value);
        }
        return { id, provider, model, pricing };
      });
    });
  } catch {
    _catalog = null;
  }

  return _catalog || [];
}

/**
 * @param {string} provider
 * @returns {string}
 */
function normalizeProvider(provider) {
  const normalized = String(provider || "")
    .trim()
    .toLowerCase();
  return normalized === "github" ? "github-copilot" : normalized;
}

/**
 * @param {string} model
 * @returns {string}
 */
function normalizeModel(model) {
  return String(model || "")
    .trim()
    .toLowerCase();
}

/**
 * @param {string} value
 * @returns {string}
 */
function normalizeComparableID(value) {
  return normalizeModel(value).replace(/[._]+/g, "-");
}

/**
 * @param {string} provider
 * @returns {boolean}
 */
function providerIncludesCacheReadsInInput(provider) {
  switch (normalizeProvider(provider)) {
    case "":
    case "anthropic":
    case "openai":
    case "azure-openai":
    case "azure_openai":
      return true;
    default:
      return false;
  }
}

/**
 * @param {string} provider
 * @param {string} model
 * @returns {Record<string, number> | null}
 */
function findModelPricing(provider, model) {
  const catalog = loadCatalog();
  const normalizedProvider = normalizeProvider(provider);
  const normalizedModel = normalizeModel(model);
  const comparableModel = normalizeComparableID(model);
  if (!normalizedModel) return null;

  const fullID = normalizedModel.includes("/") ? normalizedModel : normalizedProvider ? `${normalizedProvider}/${normalizedModel}` : "";
  const comparableFullID = normalizeComparableID(fullID);

  for (const entry of catalog) {
    if ((fullID && entry.id === fullID) || (comparableFullID && normalizeComparableID(entry.id) === comparableFullID)) {
      return entry.pricing;
    }
  }

  /** @type {Record<string, number> | null} */
  let bestProviderScoped = null;
  let bestProviderScopedLen = -1;
  /** @type {Record<string, number> | null} */
  let bestGeneric = null;
  let bestGenericLen = -1;

  for (const entry of catalog) {
    const comparableEntryModel = normalizeComparableID(entry.model);
    if (entry.model === normalizedModel || comparableEntryModel === comparableModel) {
      if (normalizedProvider && entry.provider === normalizedProvider) {
        return entry.pricing;
      }
      if (!bestGeneric) {
        bestGeneric = entry.pricing;
      }
      continue;
    }

    if (normalizedModel.startsWith(entry.model) || comparableModel.startsWith(comparableEntryModel)) {
      if (normalizedProvider && entry.provider === normalizedProvider && entry.model.length > bestProviderScopedLen) {
        bestProviderScoped = entry.pricing;
        bestProviderScopedLen = entry.model.length;
      }
      if (entry.model.length > bestGenericLen) {
        bestGeneric = entry.pricing;
        bestGenericLen = entry.model.length;
      }
    }
  }

  return bestProviderScoped || bestGeneric;
}

/**
 * @param {number} usd
 * @returns {number}
 */
function usdToAIC(usd) {
  return usd / 0.01;
}

/**
 * @param {Object} params
 * @param {string} params.provider
 * @param {string} params.model
 * @param {number} params.inputTokens
 * @param {number} params.outputTokens
 * @param {number} params.cacheReadTokens
 * @param {number} params.cacheWriteTokens
 * @param {number} [params.reasoningTokens]
 * @returns {number}
 */
function computeInferenceCostUSD({ provider, model, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens = 0 }) {
  const pricing = findModelPricing(provider, model);
  if (!pricing) return 0;

  const input = inputTokens || 0;
  const output = outputTokens || 0;
  const cacheRead = cacheReadTokens || 0;
  const cacheWrite = cacheWriteTokens || 0;
  const reasoning = reasoningTokens || 0;
  const effectiveInput = providerIncludesCacheReadsInInput(provider) ? Math.max(input - cacheRead, 0) : input;

  const promptPrice = pricing.input || 0;
  const completionPrice = pricing.output || 0;
  const cacheReadPrice = pricing.cache_read || promptPrice;
  const cacheWritePrice = pricing.cache_write || promptPrice;
  const reasoningPrice = pricing.reasoning || completionPrice;

  return effectiveInput * promptPrice + output * completionPrice + cacheRead * cacheReadPrice + cacheWrite * cacheWritePrice + reasoning * reasoningPrice;
}

/**
 * @param {Object} params
 * @param {string} params.provider
 * @param {string} params.model
 * @param {number} params.inputTokens
 * @param {number} params.outputTokens
 * @param {number} params.cacheReadTokens
 * @param {number} params.cacheWriteTokens
 * @param {number} [params.reasoningTokens]
 * @returns {number}
 */
function computeInferenceAIC(params) {
  return usdToAIC(computeInferenceCostUSD(params));
}

/**
 * @param {number} value
 * @returns {string}
 */
function formatAIC(value) {
  if (!Number.isFinite(value) || value <= 0) return "";
  if (value >= 1000) {
    /** @type {[number, string][]} */
    const units = [
      [1_000_000, "M"],
      [1_000, "K"],
    ];
    for (const [threshold, suffix] of units) {
      if (value >= threshold) {
        return `${(value / threshold).toFixed(1).replace(/\.0$/, "")}${suffix}`;
      }
    }
  }
  if (value >= 10) return value.toFixed(1).replace(/\.0$/, "");
  if (value >= 1) return value.toFixed(2).replace(/0$/, "").replace(/\.0$/, "");
  return value.toFixed(3).replace(/0+$/, "").replace(/\.$/, "");
}

function _resetModelCostsCache() {
  _catalog = undefined;
}

module.exports = {
  computeInferenceAIC,
  computeInferenceCostUSD,
  findModelPricing,
  formatAIC,
  getModelsPath,
  loadCatalog,
  usdToAIC,
  _resetModelCostsCache,
};
