// @ts-check

const fs = require("fs");
const path = require("path");

const { computeEffectiveTokens, formatET } = require("./effective_tokens.cjs");
const { computeInferenceAIC, formatAIC } = require("./model_costs.cjs");

const TOKEN_USAGE_FILENAME = "token-usage.jsonl";
const TOKEN_USAGE_RELATIVE_PATH = path.join("api-proxy-logs", TOKEN_USAGE_FILENAME);

/**
 * @param {string} root
 * @returns {string}
 */
function findTokenUsageFile(root) {
  const direct = path.join(root, TOKEN_USAGE_RELATIVE_PATH);
  if (fs.existsSync(direct)) {
    return direct;
  }

  /** @type {string[]} */
  const queue = [root];
  for (let index = 0; index < queue.length; index++) {
    const current = queue[index];
    if (!current) continue;
    /** @type {fs.Dirent[]} */
    let entries = [];
    try {
      entries = fs.readdirSync(current, { withFileTypes: true });
    } catch {
      continue;
    }
    for (const entry of entries) {
      const fullPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        queue.push(fullPath);
        continue;
      }
      if (entry.isFile() && entry.name === TOKEN_USAGE_FILENAME) {
        return fullPath;
      }
    }
  }
  return "";
}

/**
 * @param {string} root
 * @returns {string[]}
 */
function findJSONLFiles(root) {
  /** @type {string[]} */
  const files = [];
  if (!root || !fs.existsSync(root)) {
    return files;
  }

  /** @type {string[]} */
  const queue = [root];
  for (let index = 0; index < queue.length; index++) {
    const current = queue[index];
    if (!current) continue;
    /** @type {fs.Dirent[]} */
    let entries = [];
    try {
      entries = fs.readdirSync(current, { withFileTypes: true });
    } catch {
      continue;
    }
    for (const entry of entries) {
      const fullPath = path.join(current, entry.name);
      if (entry.isDirectory()) {
        queue.push(fullPath);
        continue;
      }
      if (entry.isFile() && entry.name.endsWith(".jsonl")) {
        files.push(fullPath);
      }
    }
  }
  return files;
}

/**
 * @param {string} filePath
 * @returns {number}
 */
function sumEffectiveTokensFromTokenUsageFile(filePath) {
  if (!filePath || !fs.existsSync(filePath)) {
    return 0;
  }

  const content = fs.readFileSync(filePath, "utf8");
  if (!content.trim()) {
    return 0;
  }

  let total = 0;
  for (const rawLine of content.split("\n")) {
    const line = rawLine.trim();
    if (!line || !line.startsWith("{")) {
      continue;
    }

    try {
      const parsed = JSON.parse(line);
      const explicit = Number(parsed?.effective_tokens);
      if (Number.isFinite(explicit) && explicit > 0) {
        total += Math.round(explicit);
        continue;
      }

      const computed = computeEffectiveTokens(
        String(parsed?.model || ""),
        Number(parsed?.input_tokens || 0),
        Number(parsed?.output_tokens || 0),
        Number(parsed?.cache_read_tokens || 0),
        Number(parsed?.cache_write_tokens || 0),
        Number(parsed?.reasoning_tokens || 0)
      );
      if (Number.isFinite(computed) && computed > 0) {
        total += Math.round(computed);
      }
    } catch {
      // Ignore malformed lines.
    }
  }

  return total;
}

/**
 * @param {string} filePath
 * @returns {number}
 */
function sumAICFromTokenUsageFile(filePath) {
  if (!filePath || !fs.existsSync(filePath)) {
    return 0;
  }

  const content = fs.readFileSync(filePath, "utf8");
  if (!content.trim()) {
    return 0;
  }

  let total = 0;
  for (const rawLine of content.split("\n")) {
    const line = rawLine.trim();
    if (!line || !line.startsWith("{")) {
      continue;
    }

    try {
      const parsed = JSON.parse(line);
      const explicit = Number(parsed?.aic);
      if (Number.isFinite(explicit) && explicit > 0) {
        total += explicit;
        continue;
      }
      const computed = computeInferenceAIC({
        provider: String(parsed?.provider || ""),
        model: String(parsed?.model || ""),
        inputTokens: Number(parsed?.input_tokens || 0),
        outputTokens: Number(parsed?.output_tokens || 0),
        cacheReadTokens: Number(parsed?.cache_read_tokens || 0),
        cacheWriteTokens: Number(parsed?.cache_write_tokens || 0),
        reasoningTokens: Number(parsed?.reasoning_tokens || 0),
      });
      if (Number.isFinite(computed) && computed > 0) {
        total += computed;
      }
    } catch {
      // Ignore malformed lines.
    }
  }

  return total;
}

/**
 * @param {Array<string>} filePaths
 * @returns {number}
 */
function sumAICFromUsageJSONLFiles(filePaths) {
  if (!Array.isArray(filePaths) || filePaths.length === 0) {
    return 0;
  }

  /**
   * @param {unknown} usage
   * @returns {Record<string, unknown> | null}
   */
  function normalizeUsageRecord(usage) {
    if (usage && typeof usage === "object" && !Array.isArray(usage)) {
      // prettier-ignore
      const record = /** @type {Record<string, unknown>} */ (usage);
      return record;
    }
    return null;
  }

  /**
   * @param {unknown} value
   * @returns {number | null}
   */
  function toFiniteNumber(value) {
    if (typeof value === "string" && !value.trim()) {
      return null;
    }
    const num = Number(value);
    return Number.isFinite(num) ? num : null;
  }

  /**
   * @param {Record<string, unknown> | null} usage
   * @param {Record<string, unknown>} parsed
   * @param {string} snakeCase
   * @param {string} camelCase
   * @returns {number}
   */
  function getNumericField(usage, parsed, snakeCase, camelCase) {
    const candidates = [usage?.[snakeCase], usage?.[camelCase], parsed[snakeCase], parsed[camelCase]];
    for (const candidate of candidates) {
      const num = toFiniteNumber(candidate);
      if (num !== null) {
        return num;
      }
    }
    return 0;
  }

  /**
   * @param {Record<string, unknown> | null} usage
   * @param {Record<string, unknown>} parsed
   * @param {string[]} keys
   * @returns {number}
   */
  function getNumericAliasField(usage, parsed, keys) {
    for (const key of keys) {
      const candidates = [usage?.[key], parsed[key]];
      for (const candidate of candidates) {
        const num = toFiniteNumber(candidate);
        if (num !== null) {
          return num;
        }
      }
    }
    return 0;
  }

  /**
   * @param {Record<string, unknown> | null} usage
   * @param {Record<string, unknown>} parsed
   * @param {string} snakeCase
   * @param {string} camelCase
   * @returns {string}
   */
  function getStringField(usage, parsed, snakeCase, camelCase) {
    const candidates = [usage?.[snakeCase], usage?.[camelCase], parsed[snakeCase], parsed[camelCase]];
    for (const candidate of candidates) {
      if (typeof candidate === "string" && candidate.trim()) {
        return candidate;
      }
    }
    return "";
  }

  let total = 0;
  for (const filePath of filePaths) {
    if (!filePath || !fs.existsSync(filePath)) {
      continue;
    }

    const content = fs.readFileSync(filePath, "utf8");
    if (!content.trim()) {
      continue;
    }

    for (const rawLine of content.split("\n")) {
      const line = rawLine.trim();
      if (!line || !line.startsWith("{")) {
        continue;
      }

      try {
        const parsed = JSON.parse(line);
        if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
          continue;
        }

        const usage = normalizeUsageRecord(parsed.usage);
        const explicitAICredits = getNumericAliasField(usage, parsed, ["ai_credits", "aiCredits"]);
        if (explicitAICredits > 0) {
          total += explicitAICredits;
          continue;
        }
        const explicitAIC = getNumericAliasField(usage, parsed, ["aic"]);
        if (explicitAIC > 0) {
          total += explicitAIC;
          continue;
        }

        const computed = computeInferenceAIC({
          provider: getStringField(usage, parsed, "provider", "provider"),
          model: getStringField(usage, parsed, "model", "model"),
          inputTokens: getNumericField(usage, parsed, "input_tokens", "inputTokens"),
          outputTokens: getNumericField(usage, parsed, "output_tokens", "outputTokens"),
          cacheReadTokens: getNumericField(usage, parsed, "cache_read_tokens", "cacheReadTokens"),
          cacheWriteTokens: getNumericField(usage, parsed, "cache_write_tokens", "cacheWriteTokens"),
          reasoningTokens: getNumericField(usage, parsed, "reasoning_tokens", "reasoningTokens"),
        });
        if (Number.isFinite(computed) && computed > 0) {
          total += computed;
        }
      } catch {
        // Ignore malformed lines.
      }
    }
  }

  return total;
}

/**
 * @param {Array<{effective_tokens:number}>} runs
 * @returns {{count:number,total:number,average:number,min:number,max:number,stddev:number}}
 */
function calculateDailyEffectiveWorkflowStats(runs) {
  const values = runs.map(run => Number(run?.effective_tokens || 0)).filter(value => Number.isFinite(value) && value > 0);
  if (values.length === 0) {
    return { count: 0, total: 0, average: 0, min: 0, max: 0, stddev: 0 };
  }

  const total = values.reduce((sum, value) => sum + value, 0);
  const average = total / values.length;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const variance = values.length > 1 ? values.reduce((sum, value) => sum + (value - average) ** 2, 0) / (values.length - 1) : 0;

  return {
    count: values.length,
    total,
    average,
    min,
    max,
    stddev: Math.sqrt(variance),
  };
}

/**
 * @param {Array<{aic:number}>} runs
 * @returns {{count:number,total:number,average:number,min:number,max:number,stddev:number}}
 */
function calculateDailyAICStats(runs) {
  const values = runs.map(run => Number(run?.aic || 0)).filter(value => Number.isFinite(value) && value > 0);
  if (values.length === 0) {
    return { count: 0, total: 0, average: 0, min: 0, max: 0, stddev: 0 };
  }

  const total = values.reduce((sum, value) => sum + value, 0);
  const average = total / values.length;
  const min = Math.min(...values);
  const max = Math.max(...values);
  const variance = values.length > 1 ? values.reduce((sum, value) => sum + (value - average) ** 2, 0) / (values.length - 1) : 0;

  return {
    count: values.length,
    total,
    average,
    min,
    max,
    stddev: Math.sqrt(variance),
  };
}

/**
 * @param {number | undefined} value
 * @returns {string}
 */
function formatEffectiveTokens(value) {
  const safeValue = Number.isFinite(value) ? Math.max(0, Math.round(value || 0)) : 0;
  return formatET(safeValue);
}

/**
 * @param {number | string | undefined} value
 * @returns {string}
 */
function formatAICCredits(value) {
  const numericValue = Number(value || 0);
  const safeValue = Number.isFinite(numericValue) ? Math.max(0, Math.ceil(numericValue)) : 0;
  return formatAIC(safeValue);
}

module.exports = {
  findTokenUsageFile,
  findJSONLFiles,
  sumEffectiveTokensFromTokenUsageFile,
  sumAICFromTokenUsageFile,
  sumAICFromUsageJSONLFiles,
  calculateDailyEffectiveWorkflowStats,
  calculateDailyAICStats,
  formatEffectiveTokens,
  formatAICCredits,
};
