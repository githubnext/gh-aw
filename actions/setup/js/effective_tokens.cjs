// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Effective Tokens (ET) computation module.
 *
 * Implements the Effective Tokens specification defined in
 * docs/src/content/docs/reference/effective-tokens-specification.md.
 *
 * Formula:
 *   base_weighted_tokens = (w_in × I) + (w_cache × C) + (w_out × O) + (w_reason × R) + (w_cache_write × W)
 *   effective_tokens     = m × base_weighted_tokens
 *
 * Token class default weights (from spec Section 4.2):
 *   Input          (I):  w_in         = 1.0
 *   Cached Input   (C):  w_cache      = 0.1
 *   Output         (O):  w_out        = 4.0
 *   Reasoning      (R):  w_reason     = 4.0
 *   Cache Write    (W):  w_cache_write = 1.0 (implementation extension)
 *
 * The per-model multiplier (m) is loaded from the GH_AW_MODEL_MULTIPLIERS
 * environment variable, which contains the JSON content of model_multipliers.json.
 * Falls back to m=1.0 (reference baseline) for unknown models.
 */

/** @type {{ token_class_weights: { input: number, cached_input: number, output: number, reasoning: number, cache_write: number }, multipliers: Record<string, number> } | null} */
let _parsedMultipliers = null;

/**
 * Default token class weights from the ET specification (Section 4.2).
 * @returns {{ input: number, cached_input: number, output: number, reasoning: number, cache_write: number }}
 */
function defaultTokenClassWeights() {
  return {
    input: 1.0,
    cached_input: 0.1,
    output: 4.0,
    reasoning: 4.0,
    cache_write: 1.0,
  };
}

/**
 * Loads and parses the model multipliers from the GH_AW_MODEL_MULTIPLIERS env var.
 * Caches the result after first parse. Returns null if not available or invalid.
 * @returns {{ token_class_weights: { input: number, cached_input: number, output: number, reasoning: number, cache_write: number }, multipliers: Record<string, number> } | null}
 */
function getMultipliersData() {
  if (_parsedMultipliers !== null) {
    return _parsedMultipliers;
  }

  const raw = process.env.GH_AW_MODEL_MULTIPLIERS;
  if (!raw || !raw.trim()) {
    return null;
  }

  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object") {
      return null;
    }
    const weights = { ...defaultTokenClassWeights(), ...(parsed.token_class_weights || {}) };
    // Ensure zero-valued weights fall back to defaults
    const defaults = defaultTokenClassWeights();
    for (const key of Object.keys(defaults)) {
      if (!weights[key]) {
        weights[key] = defaults[key];
      }
    }
    const multipliers = /** @type {Record<string, number>} */ {};
    for (const [model, mult] of Object.entries(parsed.multipliers || {})) {
      multipliers[model.toLowerCase()] = Number(mult);
    }
    _parsedMultipliers = { token_class_weights: weights, multipliers };
    return _parsedMultipliers;
  } catch {
    return null;
  }
}

/**
 * Returns the token class weights in use.
 * Uses values from GH_AW_MODEL_MULTIPLIERS if available, otherwise defaults.
 * @returns {{ input: number, cached_input: number, output: number, reasoning: number, cache_write: number }}
 */
function getTokenClassWeights() {
  const data = getMultipliersData();
  return data ? data.token_class_weights : defaultTokenClassWeights();
}

/**
 * Returns the per-model cost multiplier for the given model name.
 *
 * Lookup order:
 * 1. Exact case-insensitive match in GH_AW_MODEL_MULTIPLIERS
 * 2. Longest prefix match (e.g. "claude-sonnet-4.6-preview" → "claude-sonnet-4.6")
 * 3. Default: 1.0 (unknown model treated as reference baseline)
 *
 * @param {string} model - Model name
 * @returns {number} Copilot multiplier for the model
 */
function getModelMultiplier(model) {
  const data = getMultipliersData();
  if (!data || !model) {
    return 1.0;
  }

  const key = model.toLowerCase().trim();
  if (!key) {
    return 1.0;
  }

  const { multipliers } = data;

  // Exact match
  if (key in multipliers) {
    return multipliers[key];
  }

  // Longest prefix match
  let best = "";
  let bestMult = 1.0;
  for (const [name, mult] of Object.entries(multipliers)) {
    if (key.startsWith(name) && name.length > best.length) {
      best = name;
      bestMult = mult;
    }
  }

  return bestMult;
}

/**
 * Computes the base weighted token count for a single invocation.
 *
 * Formula (ET specification Section 4.3):
 *   base = (w_in × I) + (w_cache × C) + (w_out × O) + (w_reason × R) + (w_cache_write × W)
 *
 * @param {number} inputTokens - Raw input tokens (I)
 * @param {number} outputTokens - Raw output tokens (O)
 * @param {number} cacheReadTokens - Cached input tokens (C)
 * @param {number} cacheWriteTokens - Cache write tokens (W)
 * @param {number} [reasoningTokens=0] - Reasoning tokens (R)
 * @returns {number} Base weighted token count
 */
function computeBaseWeightedTokens(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens = 0) {
  const w = getTokenClassWeights();
  return w.input * (inputTokens || 0) + w.cached_input * (cacheReadTokens || 0) + w.output * (outputTokens || 0) + w.reasoning * (reasoningTokens || 0) + w.cache_write * (cacheWriteTokens || 0);
}

/**
 * Computes the effective token count for a single model invocation.
 *
 * Formula (ET specification Section 4.4):
 *   effective_tokens = m × base_weighted_tokens
 *
 * Result is rounded to the nearest integer.
 *
 * @param {string} model - Model name used for the invocation
 * @param {number} inputTokens - Raw input tokens (I)
 * @param {number} outputTokens - Raw output tokens (O)
 * @param {number} cacheReadTokens - Cached input tokens (C)
 * @param {number} cacheWriteTokens - Cache write tokens (W)
 * @param {number} [reasoningTokens=0] - Reasoning tokens (R)
 * @returns {number} Effective token count (rounded integer)
 */
function computeEffectiveTokens(model, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens = 0) {
  const base = computeBaseWeightedTokens(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens);
  if (base === 0) {
    return 0;
  }
  const m = getModelMultiplier(model);
  return Math.round(base * m);
}

/**
 * Resets the cached multipliers (for testing purposes).
 * @internal
 */
function _resetCache() {
  _parsedMultipliers = null;
}

module.exports = {
  defaultTokenClassWeights,
  getTokenClassWeights,
  getModelMultiplier,
  computeBaseWeightedTokens,
  computeEffectiveTokens,
  _resetCache,
};
