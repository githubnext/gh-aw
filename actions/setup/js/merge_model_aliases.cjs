// @ts-check

const fs = require("fs");

/**
 * Merge a base alias map with workflow-provided delta aliases.
 * @param {Record<string, string[]>} baseAliases
 * @param {Record<string, string[]>} deltaAliases
 * @returns {Record<string, string[]>}
 */
function mergeAliases(baseAliases, deltaAliases) {
  const merged = { ...baseAliases };
  for (const [key, value] of Object.entries(deltaAliases)) {
    if (Array.isArray(value)) {
      merged[key] = value;
    }
  }
  return merged;
}

/**
 * Parse builtin aliases JSON file.
 * @param {string} baseAliasesPath
 * @returns {Record<string, string[]>}
 */
function loadBaseAliases(baseAliasesPath) {
  const content = fs.readFileSync(baseAliasesPath, "utf8");
  const parsed = JSON.parse(content);
  if (!parsed || typeof parsed !== "object" || !parsed.aliases || typeof parsed.aliases !== "object") {
    throw new Error(`invalid model aliases file: ${baseAliasesPath}`);
  }
  return parsed.aliases;
}

/**
 * Merge model aliases in an AWF config file.
 * @param {string} configPath
 * @param {string} baseAliasesPath
 * @returns {boolean}
 */
function mergeModelAliasesInConfig(configPath, baseAliasesPath) {
  const configRaw = fs.readFileSync(configPath, "utf8");
  const config = JSON.parse(configRaw);
  const deltaAliases = config?.apiProxy?.models;

  if (!deltaAliases || typeof deltaAliases !== "object") {
    return false;
  }

  const baseAliases = loadBaseAliases(baseAliasesPath);
  config.apiProxy.models = mergeAliases(baseAliases, deltaAliases);
  fs.writeFileSync(configPath, `${JSON.stringify(config)}\n`, "utf8");
  return true;
}

if (require.main === module) {
  const [, , configPath, baseAliasesPath] = process.argv;
  if (!configPath || !baseAliasesPath) {
    console.error("usage: node merge_model_aliases.cjs <awf-config-path> <model-aliases-json-path>");
    process.exit(1);
  }
  try {
    mergeModelAliasesInConfig(configPath, baseAliasesPath);
  } catch (error) {
    console.error(`failed to merge model aliases: ${error.message}`);
    process.exit(1);
  }
}

module.exports = {
  mergeAliases,
  mergeModelAliasesInConfig,
  loadBaseAliases,
};
