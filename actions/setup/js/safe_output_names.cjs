// @ts-check

/**
 * Resolve historical diagnostic names without advertising duplicate tools.
 * @param {string} name
 * @returns {string}
 */
function normalizeSafeOutputName(name) {
  if (name === "missing_tool") return "report_missing_tool";
  if (name === "missing_data") return "report_missing_data";
  return name;
}

/**
 * @param {Record<string, any>} config
 * @returns {Record<string, any>}
 */
function normalizeSafeOutputConfig(config) {
  return Object.fromEntries(Object.entries(config).map(([key, value]) => [normalizeSafeOutputName(key.replace(/-/g, "_")), value]));
}

module.exports = { normalizeSafeOutputName, normalizeSafeOutputConfig };
