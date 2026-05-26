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
    if (typeof value === "number" && Number.isFinite(value)) {
      normalized[key] = value;
    }
  }
  return normalized;
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
  if (Object.keys(normalized).length > 0) {
    apiProxy.modelMultipliers = normalized;
  } else {
    delete apiProxy.modelMultipliers;
  }
  configDoc.apiProxy = apiProxy;

  fs.writeFileSync(configPath, JSON.stringify(configDoc), "utf8");
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
  mergeModelMultipliers,
  main,
};
