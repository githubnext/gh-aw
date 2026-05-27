// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { TMP_GH_AW_PATH } = require("./constants.cjs");

const DEFAULT_MODEL_MULTIPLIERS_PATH = `${TMP_GH_AW_PATH}/model_multipliers.json`;

/**
 * @param {unknown} value
 * @returns {value is Record<string, unknown>}
 */
function isPlainObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

/**
 * @param {Record<string, unknown>} rawMultipliers
 * @returns {Record<string, number>}
 */
function normalizeMultipliers(rawMultipliers) {
  /** @type {Record<string, number>} */
  const normalized = {};
  for (const [key, value] of Object.entries(rawMultipliers)) {
    if (typeof value === "number" && Number.isFinite(value) && value > 0) {
      normalized[key] = value;
    }
  }
  return normalized;
}

/**
 * @param {unknown} rawModels
 * @returns {Array<{alias: string, targets: string[]}>}
 */
function normalizeAliasRows(rawModels) {
  if (!isPlainObject(rawModels)) {
    return [];
  }

  /** @type {Array<{alias: string, targets: string[]}>} */
  const rows = [];
  for (const [alias, rawTargets] of Object.entries(rawModels)) {
    if (!Array.isArray(rawTargets)) {
      continue;
    }
    const trimmedTargets = rawTargets.map(target => (typeof target === "string" ? target.trim() : ""));
    const targets = trimmedTargets.filter(target => target !== "");
    if (targets.length === 0) {
      continue;
    }
    rows.push({ alias, targets });
  }

  rows.sort((a, b) => {
    // Sort the default alias ("") last for readability in step-summary tables.
    // We substitute it with U+ffff only for comparison; it is a noncharacter and
    // reliably sorts after typical printable alias strings in localeCompare.
    const left = a.alias || "\uffff";
    const right = b.alias || "\uffff";
    return left.localeCompare(right);
  });
  return rows;
}

/**
 * @param {string} value
 * @returns {string}
 */
function escapeTableValue(value) {
  return value.replace(/\|/g, "\\|");
}

/**
 * @param {Array<{alias: string, targets: string[]}>} aliasRows
 * @returns {string}
 */
function renderModelAliasSummary(aliasRows) {
  const lines = [];
  lines.push("<details>");
  lines.push(`<summary>AWF model aliases (${aliasRows.length})</summary>`);
  lines.push("");
  lines.push("| Alias | Resolution order |");
  lines.push("|-------|------------------|");
  for (const row of aliasRows) {
    const aliasLabel = row.alias === "" ? "(default)" : `\`${escapeTableValue(row.alias)}\``;
    const resolutionOrder = row.targets.map(target => `\`${escapeTableValue(target)}\``).join(" → ");
    lines.push(`| ${aliasLabel} | ${resolutionOrder} |`);
  }
  lines.push("");
  lines.push("</details>");
  lines.push("");
  return lines.join("\n");
}

/**
 * @param {Array<{alias: string, targets: string[]}>} aliasRows
 * @param {(message: string) => void} warn
 */
function writeAliasSummary(aliasRows, warn) {
  if (aliasRows.length === 0) {
    return;
  }

  const summaryPath = process.env.GITHUB_STEP_SUMMARY;
  if (!summaryPath) {
    return;
  }

  try {
    fs.appendFileSync(summaryPath, renderModelAliasSummary(aliasRows), "utf8");
  } catch (error) {
    warn(`warning: failed to write AWF model alias summary: ${String(error)}`);
  }
}

/**
 * @param {object} options
 * @param {string} options.configPath
 * @param {string} options.multipliersPath
 * @param {(message: string) => void} options.warn
 */
function mergeModelMultipliers({ configPath, multipliersPath, warn }) {
  if (!fs.existsSync(configPath) || !fs.existsSync(multipliersPath)) {
    return;
  }

  /** @type {Record<string, unknown> | null} */
  let multipliersDoc = null;
  try {
    const parsed = JSON.parse(fs.readFileSync(multipliersPath, "utf8"));
    if (isPlainObject(parsed)) {
      multipliersDoc = parsed;
    }
  } catch (error) {
    warn(`warning: failed to parse model multipliers file: ${String(error)}`);
    return;
  }
  if (!multipliersDoc || !isPlainObject(multipliersDoc.multipliers)) {
    return;
  }

  const normalized = normalizeMultipliers(multipliersDoc.multipliers);

  /** @type {Record<string, unknown> | null} */
  let configDoc = null;
  try {
    const parsed = JSON.parse(fs.readFileSync(configPath, "utf8"));
    if (isPlainObject(parsed)) {
      configDoc = parsed;
    }
  } catch (error) {
    warn(`warning: failed to parse awf-config.json before model multiplier merge: ${String(error)}`);
    return;
  }
  if (!configDoc) {
    return;
  }

  const apiProxy = isPlainObject(configDoc.apiProxy) ? configDoc.apiProxy : {};
  const aliasRows = normalizeAliasRows(apiProxy.models);
  if (Object.keys(normalized).length > 0) {
    apiProxy.modelMultipliers = normalized;
  } else {
    delete apiProxy.modelMultipliers;
  }
  configDoc.apiProxy = apiProxy;

  fs.writeFileSync(configPath, JSON.stringify(configDoc), "utf8");
  writeAliasSummary(aliasRows, warn);
}

/**
 * @param {object} [options]
 * @param {string} [options.runnerTemp]
 * @param {string} [options.configPath]
 * @param {string} [options.multipliersPath]
 * @param {(message: string) => void} [options.warn]
 */
function main(options = {}) {
  const runnerTemp = options.runnerTemp ?? process.env.RUNNER_TEMP;
  if (!runnerTemp) {
    throw new Error("RUNNER_TEMP is required");
  }

  const configPath = options.configPath ?? path.join(runnerTemp, "gh-aw", "awf-config.json");
  const multipliersPath = options.multipliersPath ?? process.env.GH_AW_MODEL_MULTIPLIERS_PATH ?? DEFAULT_MODEL_MULTIPLIERS_PATH;
  const warn = options.warn ?? (message => process.stderr.write(`${message}\n`));

  mergeModelMultipliers({ configPath, multipliersPath, warn });
}

if (require.main === module) {
  try {
    main();
  } catch (error) {
    process.stderr.write(`${error instanceof Error ? error.message : String(error)}\n`);
    process.exit(1);
  }
}

module.exports = {
  DEFAULT_MODEL_MULTIPLIERS_PATH,
  isPlainObject,
  normalizeMultipliers,
  normalizeAliasRows,
  renderModelAliasSummary,
  writeAliasSummary,
  mergeModelMultipliers,
  main,
};
