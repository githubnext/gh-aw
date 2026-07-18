// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { TMP_GH_AW_PATH } = require("./constants.cjs");

const DEFAULT_MODELS_JSON_PATH = path.join(__dirname, "models.json");
const MERGED_MODELS_JSON_PATH = `${TMP_GH_AW_PATH}/models.json`;

/**
 * @param {unknown} value
 * @returns {value is Record<string, unknown>}
 */
function isPlainObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

/**
 * Deep-merge a models overlay on top of a base models.json document.
 *
 * Both documents share the structure:
 *   { "providers": { "<provider>": { "models": { "<model>": { "cost": { ... } } } } } }
 *
 * Merge rules:
 *   - providers present only in the base are kept unchanged.
 *   - providers present only in the overlay are added to the result.
 *   - when a provider exists in both, its models maps are merged: overlay models
 *     win over base models (override or fill gaps at the individual model level).
 *   - other top-level keys (if any) are passed through from base; overlay keys
 *     take precedence.
 *
 * @param {Record<string, unknown>} base  - parsed models.json content
 * @param {Record<string, unknown>} overlay - parsed frontmatter `models` content
 * @returns {Record<string, unknown>} merged document
 */
function mergeModelCosts(base, overlay) {
  const result = { ...base };

  const baseProviders = isPlainObject(base.providers) ? /** @type {Record<string, unknown>} */ base.providers : {};
  const overlayProviders = isPlainObject(overlay.providers) ? /** @type {Record<string, unknown>} */ overlay.providers : {};

  if (Object.keys(overlayProviders).length === 0) {
    return result;
  }

  /** @type {Record<string, unknown>} */
  const mergedProviders = { ...baseProviders };

  for (const [providerName, overlayProvider] of Object.entries(overlayProviders)) {
    if (!isPlainObject(overlayProvider)) {
      // Scalar or array: take overlay value as-is.
      mergedProviders[providerName] = overlayProvider;
      continue;
    }

    const baseProvider = isPlainObject(baseProviders[providerName]) ? /** @type {Record<string, unknown>} */ baseProviders[providerName] : {};

    const baseModels = isPlainObject(baseProvider.models) ? /** @type {Record<string, unknown>} */ baseProvider.models : {};
    const overlayModels = isPlainObject(overlayProvider.models) ? /** @type {Record<string, unknown>} */ overlayProvider.models : {};

    mergedProviders[providerName] = {
      ...baseProvider,
      ...overlayProvider,
      models: { ...baseModels, ...overlayModels },
    };
  }

  result.providers = mergedProviders;
  return result;
}

/**
 * Read the base models.json and merge any frontmatter `models` overlay on top, then
 * write the combined catalog to /tmp/gh-aw/models.json so the agent job can
 * use it via GH_AW_MODELS_JSON_PATH.
 *
 * @param {typeof import('@actions/core')} core
 */
function writeMergedModelsJSON(core) {
  const baseModelsPath = process.env.GH_AW_MODELS_JSON_SRC_PATH || DEFAULT_MODELS_JSON_PATH;

  /** @type {Record<string, unknown>} */
  let base = {};
  if (fs.existsSync(baseModelsPath)) {
    try {
      const parsed = JSON.parse(fs.readFileSync(baseModelsPath, "utf8"));
      if (isPlainObject(parsed)) {
        base = parsed;
      } else {
        core.warning(`models.json is not a JSON object at ${baseModelsPath}`);
      }
    } catch (error) {
      core.warning(`Failed to parse models.json at ${baseModelsPath}: ${String(error)}`);
    }
  } else {
    core.warning(`models.json not found at ${baseModelsPath}`);
  }

  // Parse optional frontmatter `models` overlay from env var (serialized by the compiler).
  const modelCostsEnv = process.env.GH_AW_INFO_MODEL_COSTS;
  /** @type {Record<string, unknown>} */
  let overlay = {};
  if (modelCostsEnv) {
    try {
      const parsed = JSON.parse(modelCostsEnv);
      if (isPlainObject(parsed)) {
        overlay = parsed;
      } else {
        core.warning("GH_AW_INFO_MODEL_COSTS must be a JSON object, ignoring");
      }
    } catch {
      core.warning(`Failed to parse GH_AW_INFO_MODEL_COSTS: ${modelCostsEnv}`);
    }
  }

  const merged = Object.keys(overlay).length > 0 ? mergeModelCosts(base, overlay) : base;

  try {
    fs.writeFileSync(MERGED_MODELS_JSON_PATH, JSON.stringify(merged), "utf8");
  } catch (err) {
    throw new Error(`Failed to write file ${MERGED_MODELS_JSON_PATH}: ${String(err)}`, { cause: err });
  }
  core.info(`Generated merged models.json at: ${MERGED_MODELS_JSON_PATH}`);
}

module.exports = {
  isPlainObject,
  mergeModelCosts,
  writeMergedModelsJSON,
  DEFAULT_MODELS_JSON_PATH,
  MERGED_MODELS_JSON_PATH,
};
