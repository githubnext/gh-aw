// @ts-check

const fs = require("fs");
const path = require("path");

const { computeInferenceAIC, formatAIC } = require("./model_costs.cjs");

const TOKEN_USAGE_FILENAME = "token-usage.jsonl";

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

    let content;
    try {
      content = fs.readFileSync(filePath, "utf8");
    } catch (err) {
      throw new Error(`Failed to read file ${filePath}: ${String(err)}`, { cause: err });
    }
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
 * @param {number | string | undefined} value
 * @returns {string}
 */
function formatAICCredits(value) {
  const numericValue = Number(value || 0);
  const safeValue = Number.isFinite(numericValue) ? Math.max(0, Math.ceil(numericValue)) : 0;
  return formatAIC(safeValue);
}

module.exports = {
  findJSONLFiles,
  sumAICFromUsageJSONLFiles,
  calculateDailyAICStats,
  formatAICCredits,
};
