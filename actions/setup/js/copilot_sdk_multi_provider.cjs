// @ts-check

/**
 * Copilot SDK Multi-Provider Helpers
 *
 * Shared parsing and validation utilities for the GH_AW_COPILOT_SDK_MULTI_PROVIDER_JSON
 * environment variable.  Consumed by both the production SDK driver
 * (copilot_sdk_driver.cjs) and any custom driver (e.g. the Node sample driver
 * under .github/drivers/) so that the validation logic stays in one place.
 */

"use strict";

/**
 * @param {any} p
 * @returns {boolean}
 */
function isValidProviderConfig(p) {
  return p && typeof p.name === "string" && typeof p.type === "string" && typeof p.baseUrl === "string";
}

/**
 * @param {any} m
 * @returns {boolean}
 */
function isValidModelConfig(m) {
  return m && typeof m.id === "string" && typeof m.provider === "string";
}

/**
 * Parse the GH_AW_COPILOT_SDK_MULTI_PROVIDER_JSON env var.
 *
 * Returns `null` when the env var is unset or contains invalid JSON.
 * On success returns `{ model, providers, models }` where the shapes match the
 * Copilot SDK `NamedProviderConfig` / `ProviderModelConfig` types.
 *
 * @param {string | undefined} value
 * @returns {{
 *   model: string,
 *   providers: import("@github/copilot-sdk").NamedProviderConfig[],
 *   models: import("@github/copilot-sdk").ProviderModelConfig[],
 * } | null}
 */
function parseMultiProviderJson(value) {
  if (!value) return null;
  try {
    const parsed = JSON.parse(value);
    if (!parsed || typeof parsed !== "object") return null;
    if (!Array.isArray(parsed.providers) || parsed.providers.length < 1) return null;
    if (!Array.isArray(parsed.models) || parsed.models.length < 1) return null;
    // Validate minimal shape: providers must have name/type/baseUrl, models must have id/provider
    if (!parsed.providers.every(isValidProviderConfig)) return null;
    if (!parsed.models.every(isValidModelConfig)) return null;
    const model = typeof parsed.model === "string" ? parsed.model.trim() : "";
    return { model, providers: parsed.providers, models: parsed.models };
  } catch {
    return null;
  }
}

module.exports = {
  isValidProviderConfig,
  isValidModelConfig,
  parseMultiProviderJson,
};
