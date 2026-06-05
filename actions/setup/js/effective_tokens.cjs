// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { computeInferenceCostUSD } = require("./model_costs.cjs");

/**
 * Effective Tokens (ET) computation module.
 *
 * Implements the Effective Tokens specification defined in
 * docs/src/content/docs/specs/effective-tokens-specification.md.
 *
 * Effective token values are normalized from model inference cost (USD).
 * Cost is computed using the same pricing logic used by AI Credits and then
 * converted to an ET-like token unit using a dedicated USD-per-token factor.
 */

const USD_PER_EFFECTIVE_TOKEN = 0.000003;

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
 * Returns the token class weights in use.
 * Uses the built-in defaults.
 * @returns {{ input: number, cached_input: number, output: number, reasoning: number, cache_write: number }}
 */
function getTokenClassWeights() {
  return defaultTokenClassWeights();
}

/**
 * Compatibility helper retained for older callers.
 * Model multiplier data is no longer used for ET computation.
 * @param {string} _model - Model name
 * @returns {number}
 */
function getModelMultiplier(_model) {
  return 1.0;
}

/**
 * Computes the base weighted token count for a single invocation.
 *
 * Formula (base spec Section 4.3 + cache_write implementation extension):
 *   effective_input = max(I - C, 0)
 *   base = (w_in × effective_input) + (w_cache × C) + (w_out × O) + (w_reason × R) + (w_cache_write × W)
 *
 * Note: cache_write (W) with weight w_cache_write is an implementation extension;
 * the core spec formula covers I, C, O, and R only.
 *
 * @param {number} inputTokens - Raw input tokens (I), including cached input when reported by provider
 * @param {number} outputTokens - Raw output tokens (O)
 * @param {number} cacheReadTokens - Cached input tokens (C)
 * @param {number} cacheWriteTokens - Cache write tokens (W)
 * @param {number} [reasoningTokens=0] - Reasoning tokens (R)
 * @returns {number} Base weighted token count
 */
function computeBaseWeightedTokens(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens = 0) {
  const w = getTokenClassWeights();
  const input = inputTokens || 0;
  const cached = cacheReadTokens || 0;
  const effectiveInput = Math.max(input - cached, 0);

  return w.input * effectiveInput + w.cached_input * cached + w.output * (outputTokens || 0) + w.reasoning * (reasoningTokens || 0) + w.cache_write * (cacheWriteTokens || 0);
}

/**
 * Converts inference cost in USD to normalized effective tokens.
 * Uses the same model/provider pricing calculation as AI Credits.
 *
 * @param {string} model - Model name used for the invocation
 * @param {number} inputTokens - Raw input tokens (I)
 * @param {number} outputTokens - Raw output tokens (O)
 * @param {number} cacheReadTokens - Cached input tokens (C)
 * @param {number} cacheWriteTokens - Cache write tokens (W)
 * @param {number} [reasoningTokens=0] - Reasoning tokens (R)
 * @param {string} [provider=""] - Provider name
 * @returns {number} Effective token count (exact real value)
 */
function computeEffectiveTokens(model, inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens = 0, provider = "") {
  const costUSD = computeInferenceCostUSD({
    provider: provider || "",
    model,
    inputTokens,
    outputTokens,
    cacheReadTokens,
    cacheWriteTokens,
    reasoningTokens,
  });
  if (!Number.isFinite(costUSD) || costUSD <= 0) {
    return computeBaseWeightedTokens(inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens, reasoningTokens);
  }
  return costUSD / USD_PER_EFFECTIVE_TOKEN;
}

/**
 * Formats an ET number in a compact, human-readable form.
 *
 * Ranges:
 *   < 1,000        → exact integer (e.g. "900")
 *   1,000–999,999  → Xk with one decimal when non-zero (e.g. "1.2K", "450K")
 *   >= 1,000,000   → Xm with one decimal when non-zero (e.g. "1.2M", "3M")
 *
 * @param {number} n - Non-negative ET value (should be rounded before passing)
 * @returns {string} Compact string representation
 */
function formatET(n) {
  if (n < 1000) return String(n);
  if (n < 1_000_000) return `${(n / 1000).toFixed(1).replace(/\.0$/, "")}K`;
  return `${(n / 1_000_000).toFixed(1).replace(/\.0$/, "")}M`;
}

/**
 * Build a deterministic compact model identifier for footer rendering.
 * Uses well-known shortcuts for popular model families and a deterministic fallback.
 *
 * Examples:
 * - claude-sonnet-4.6 -> sonnet46
 * - gpt-5.5 -> gpt55
 * - claude-opus-4-7 -> opus47
 *
 * @param {string|undefined|null} modelName
 * @returns {string}
 */
function reduceModelNameToIdentifier(modelName) {
  const normalized = String(modelName || "")
    .trim()
    .toLowerCase();
  if (!normalized) return "";

  if (normalized === "opus" || normalized === "sonnet" || normalized === "haiku") {
    return normalized;
  }

  const VERSION_SUFFIX_PATTERN = "[-_\\s]*([0-9]+)(?:[._-]+([0-9]+))?";
  const FALLBACK_LETTER_LENGTH = 3;
  const FALLBACK_DIGIT_LENGTH = 2;
  const FALLBACK_PADDING_CHAR = "x";

  /** @type {Array<{ familyPattern: RegExp, versionPattern: RegExp, prefix: string }>} */
  const shortcuts = [
    { familyPattern: /sonnet/, versionPattern: new RegExp(`sonnet${VERSION_SUFFIX_PATTERN}`), prefix: "sonnet" },
    { familyPattern: /opus/, versionPattern: new RegExp(`opus${VERSION_SUFFIX_PATTERN}`), prefix: "opus" },
    { familyPattern: /haiku/, versionPattern: new RegExp(`haiku${VERSION_SUFFIX_PATTERN}`), prefix: "haiku" },
    { familyPattern: /gpt/, versionPattern: new RegExp(`gpt${VERSION_SUFFIX_PATTERN}`), prefix: "gpt" },
    { familyPattern: /^o[0-9](?:$|[-_])/, versionPattern: new RegExp(`o${VERSION_SUFFIX_PATTERN}`), prefix: "o" },
    { familyPattern: /gemini/, versionPattern: new RegExp(`gemini${VERSION_SUFFIX_PATTERN}`), prefix: "gem" },
  ];

  for (const { familyPattern, versionPattern, prefix } of shortcuts) {
    if (!familyPattern.test(normalized)) continue;
    const version = extractModelVersionDigits(normalized, versionPattern);
    return `${prefix}${version}${extractKnownModelTierSuffix(normalized, prefix)}`;
  }

  return buildFallbackModelIdentifier(normalized, FALLBACK_LETTER_LENGTH, FALLBACK_DIGIT_LENGTH, FALLBACK_PADDING_CHAR);
}

/**
 * Returns a compact monochrome Unicode symbol-prefixed alias for a model name.
 * Uses distinct symbols for each compact model kind so aliases remain scannable.
 *
 * Examples:
 * - claude-sonnet-4.6 -> ◉ sonnet46
 * - gpt-5.5 -> ■ gpt55
 * - gemini-2.5-pro -> ★ gem25pro
 *
 * @param {string|undefined|null} modelName
 * @returns {string}
 */
function formatModelEmojiAlias(modelName) {
  const identifier = reduceModelNameToIdentifier(modelName);
  if (!identifier) return "";

  const normalized = String(modelName || "")
    .trim()
    .toLowerCase();

  let emoji = "○";
  if (/sonnet/.test(normalized)) {
    emoji = "◉";
  } else if (/opus/.test(normalized)) {
    emoji = "◆";
  } else if (/haiku/.test(normalized)) {
    emoji = "▲";
  } else if (/^o[0-9](?:$|[-_])/.test(normalized)) {
    emoji = "●";
  } else if (/gpt|openai/.test(normalized)) {
    emoji = "■";
  } else if (/gemini|gemma|google|^gem[0-9]/.test(normalized)) {
    emoji = "★";
  }

  return `${emoji} ${identifier}`;
}

/**
 * Escapes HTML-sensitive characters for safe embedding in HTML fragments.
 *
 * @param {string} value
 * @returns {string}
 */
function escapeHtml(value) {
  return String(value).replace(/[&<>"']/g, ch => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[ch] ?? ch);
}

/**
 * Formats a compact alias legend for the models shown in the ET details table.
 *
 * @param {string[]} models
 * @returns {string}
 */
function formatModelEmojiAliasLegend(models) {
  const seen = new Set();
  const entries = [];

  for (const model of models || []) {
    const normalized = String(model || "").trim();
    if (!normalized || seen.has(normalized)) continue;
    seen.add(normalized);
    const alias = formatModelEmojiAlias(normalized);
    if (!alias) continue;
    entries.push(`${alias}=${escapeHtml(normalized)}`);
  }

  return entries.join(" · ");
}

/**
 * Preserve useful tier qualifiers so the compact identifier reflects important
 * distinctions (for example gpt-5.4-mini vs gpt-5.4, gemini-2.5-pro vs gemini-2.5).
 *
 * @param {string} normalizedModelName
 * @param {string} _familyPrefix
 * @returns {string}
 */
function extractKnownModelTierSuffix(normalizedModelName, _familyPrefix) {
  if (hasDelimitedModelQualifier(normalizedModelName, "mini")) return "mini";
  if (hasDelimitedModelQualifier(normalizedModelName, "nano")) return "nano";
  if (hasDelimitedModelQualifier(normalizedModelName, "codex")) return "codex";
  if (hasDelimitedModelQualifier(normalizedModelName, "pro")) return "pro";
  return "";
}

/**
 * @param {string} normalizedModelName
 * @param {string} qualifier
 * @returns {boolean}
 */
function hasDelimitedModelQualifier(normalizedModelName, qualifier) {
  return new RegExp(`(^|[-_\\s])${qualifier}($|[-_\\s])`).test(normalizedModelName);
}

/**
 * @param {string} normalizedModelName
 * @param {RegExp} familyVersionPattern
 * @returns {string}
 */
function extractModelVersionDigits(normalizedModelName, familyVersionPattern) {
  const familyMatch = normalizedModelName.match(familyVersionPattern);
  if (familyMatch) {
    return normalizeVersionDigits(familyMatch[1], familyMatch[2]);
  }

  const firstNumericMatch = normalizedModelName.match(/([0-9]+)(?:[._-]+([0-9]+))?/);
  if (firstNumericMatch) {
    return normalizeVersionDigits(firstNumericMatch[1], firstNumericMatch[2]);
  }

  return "00";
}

/**
 * @param {string|undefined} major
 * @param {string|undefined} minor
 * @returns {string}
 */
function normalizeVersionDigits(major, minor) {
  const majorDigit = getFirstDigit(major);
  // Treat any 3+ digit minor segment as a build/date-like stamp (e.g. 100, 20250514),
  // not a semantic minor version, so identifiers stay stable (gpt-5-2025-08-07 -> gpt50).
  const minorIsDateLike = minor && /^\d{3,}$/.test(minor);
  const minorDigit = getFirstDigit(minor, Boolean(minorIsDateLike));
  return `${majorDigit}${minorDigit}`;
}

/**
 * @param {string|undefined} value
 * @param {boolean} [treatAsMissing=false]
 * @returns {string}
 */
function getFirstDigit(value, treatAsMissing = false) {
  if (!value || treatAsMissing) return "0";
  const digitMatch = value.match(/\d/);
  return digitMatch ? digitMatch[0] : "0";
}

/**
 * @param {string} normalizedModelName
 * @param {number} fallbackLetterLength
 * @param {number} fallbackDigitLength
 * @param {string} fallbackPaddingChar
 * @returns {string}
 */
function buildFallbackModelIdentifier(normalizedModelName, fallbackLetterLength, fallbackDigitLength, fallbackPaddingChar) {
  const compact = normalizedModelName.replace(/[^a-z0-9]+/g, "");
  if (!compact) return "";

  // Pad with "x" to keep a fixed family slot for short/unknown model names.
  const letterPart = compact.replace(/[0-9]/g, "").slice(0, fallbackLetterLength).padEnd(fallbackLetterLength, fallbackPaddingChar);
  const digitPart = compact
    .replace(/[^0-9]/g, "")
    .slice(0, fallbackDigitLength)
    .padEnd(fallbackDigitLength, "0");
  return `${letterPart}${digitPart}`.slice(0, 5);
}

/**
 * Resets the cached multipliers (for testing purposes).
 * @internal
 */
function _resetCache() {
  // No-op: retained for test compatibility.
}

/**
 * Resolve the actual model name to use in footer rendering.
 *
 * Prefers `primary_model` from agent_usage.json (the actual model name recorded
 * by the firewall proxy during the run) over `GH_AW_ENGINE_MODEL` (which may be
 * a user-supplied alias such as "agent" that hasn't been resolved to a real name).
 *
 * Falls back to `GH_AW_ENGINE_MODEL` when agent_usage.json is absent, unreadable,
 * or does not contain a `primary_model` field (e.g. single-model runs before this
 * field was introduced, or runs without token-usage.jsonl data).
 *
 * @returns {string}
 */
function resolveActualModelName() {
  const usage = readAgentUsage();
  if (usage && typeof usage.primary_model === "string" && usage.primary_model) {
    return usage.primary_model;
  }
  return process.env.GH_AW_ENGINE_MODEL || "";
}

/**
 * Read effective tokens from the GH_AW_EFFECTIVE_TOKENS environment variable and return
 * a pre-formatted suffix string suitable for appending to footer text.
 * Returns "" when the variable is absent or the parsed value is not a positive integer.
 * @returns {string} Suffix string, e.g. " · 12.5K" or ""
 */
function getEffectiveTokensSuffix() {
  const raw = process.env.GH_AW_EFFECTIVE_TOKENS ?? "";
  const parsed = parseInt(raw, 10);

  if (!isNaN(parsed) && parsed > 0) {
    const reducedModel = reduceModelNameToIdentifier(resolveActualModelName());
    const modelPrefix = reducedModel ? `${reducedModel} ` : "";
    return ` · ${modelPrefix}${formatET(parsed)}`;
  }
  return "";
}

const AGENT_USAGE_PATH = "/tmp/gh-aw/agent_usage.json";

/**
 * Read the aggregated token usage written by parse_token_usage.cjs.
 * Returns null when the file is absent or unparseable.
 * @returns {{input_tokens?: number, output_tokens?: number, cache_read_tokens?: number, cache_write_tokens?: number, effective_tokens?: number, primary_model?: string} | null}
 */
function readAgentUsage() {
  try {
    if (!fs.existsSync(AGENT_USAGE_PATH)) return null;
    const content = fs.readFileSync(AGENT_USAGE_PATH, "utf8");
    if (!content.trim()) return null;
    const parsed = JSON.parse(content);
    if (!parsed || typeof parsed !== "object") return null;
    return parsed;
  } catch {
    return null;
  }
}

/**
 * Build a collapsible `<details>` table showing the data used to compute ET.
 *
 * Three tiers of detail, in order of preference:
 *  1. `tokenUsageMarkdown` — full per-model breakdown produced by `generateTokenUsageSummary`
 *     (passed in by the caller after reading/parsing token-usage.jsonl)
 *  2. Aggregated data from `agent_usage.json` — weighted computation table with model multiplier
 *  3. Weights-only table — when no token count data is available
 *
 * @param {string} effectiveTokens - Total effective token count (string)
 * @param {{ markdown?: string | null, modelNames?: string[] } | null} [tokenUsageDetails] - Pre-rendered per-model table data
 * @returns {string} Markdown/HTML `<details>` block
 */
function buildETComputationTable(effectiveTokens, tokenUsageDetails = null) {
  const w = getTokenClassWeights();
  const formula = `inference_cost_usd ÷ ${USD_PER_EFFECTIVE_TOKEN}`;
  const tokenUsageMarkdown = tokenUsageDetails?.markdown || null;
  const modelAliasLegend = formatModelEmojiAliasLegend(tokenUsageDetails?.modelNames || []);

  const lines = [];
  lines.push("<details>");
  lines.push("<summary>ET computation details</summary>");
  lines.push("");

  if (tokenUsageMarkdown) {
    // Full per-model breakdown — includes ET per model, token class counts,
    // duration, request count, and ET weight disclosure.
    lines.push(tokenUsageMarkdown.trimEnd());
  } else {
    const usage = readAgentUsage();
    if (usage) {
      const inputTokens = usage.input_tokens || 0;
      const cachedInputTokens = usage.cache_read_tokens || 0;
      const effectiveInputTokens = Math.max(inputTokens - cachedInputTokens, 0);
      const inputWeighted = w.input * effectiveInputTokens;
      const cachedWeighted = w.cached_input * cachedInputTokens;
      const outputWeighted = w.output * (usage.output_tokens || 0);
      const cacheWriteWeighted = w.cache_write * (usage.cache_write_tokens || 0);
      // Reasoning tokens are not tracked in agent_usage.json (they are captured per-model in
      // token-usage.jsonl but not aggregated into the summary file), so they are omitted here.
      // The step summary's Token Usage table includes reasoning per model when available.
      const baseWeighted = inputWeighted + cachedWeighted + outputWeighted + cacheWriteWeighted;

      lines.push("| Token class | Count | Weight | Weighted tokens |");
      lines.push("|-------------|------:|------:|---------------:|");
      lines.push(`| Input (minus cached) | ${effectiveInputTokens.toLocaleString()} | ×${w.input} | ${Math.round(inputWeighted).toLocaleString()} |`);
      lines.push(`| Cached input | ${cachedInputTokens.toLocaleString()} | ×${w.cached_input} | ${Math.round(cachedWeighted).toLocaleString()} |`);
      lines.push(`| Output | ${(usage.output_tokens || 0).toLocaleString()} | ×${w.output} | ${Math.round(outputWeighted).toLocaleString()} |`);
      lines.push(`| Cache write | ${(usage.cache_write_tokens || 0).toLocaleString()} | ×${w.cache_write} | ${Math.round(cacheWriteWeighted).toLocaleString()} |`);
      lines.push(`| **Base weighted** | | | **${Math.round(baseWeighted).toLocaleString()}** |`);

      const etVal = Number.parseInt(effectiveTokens || "", 10);
      if (Number.isInteger(etVal) && etVal > 0) {
        lines.push(`| **Effective tokens** | | | **${etVal.toLocaleString()}** |`);
      }
    } else {
      lines.push("| Token class | Weight |");
      lines.push("|-------------|-------:|");
      lines.push(`| Input | ×${w.input} |`);
      lines.push(`| Cached input | ×${w.cached_input} |`);
      lines.push(`| Output | ×${w.output} |`);
      lines.push(`| Reasoning | ×${w.reasoning} |`);
      lines.push(`| Cache write | ×${w.cache_write} |`);
      lines.push("");
      lines.push("_Token counts unavailable — see step summary for per-model breakdown._");
    }
  }

  lines.push("");
  if (modelAliasLegend) {
    lines.push(`<sub>Model aliases: ${modelAliasLegend}</sub>`);
  }
  lines.push(`<sub>ET formula: ${formula} · ET weights: input=${w.input} · cached_input=${w.cached_input} · output=${w.output} · reasoning=${w.reasoning} · cache_write=${w.cache_write}</sub>`);
  lines.push("");
  lines.push("</details>");
  lines.push("");

  return lines.join("\n");
}

module.exports = {
  defaultTokenClassWeights,
  getTokenClassWeights,
  getModelMultiplier,
  computeBaseWeightedTokens,
  computeEffectiveTokens,
  formatET,
  formatModelEmojiAlias,
  formatModelEmojiAliasLegend,
  reduceModelNameToIdentifier,
  resolveActualModelName,
  getEffectiveTokensSuffix,
  AGENT_USAGE_PATH,
  readAgentUsage,
  buildETComputationTable,
  _resetCache,
};
