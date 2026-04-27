// @ts-check

/**
 * @param {string} value
 * @returns {boolean}
 */
function isGitHubExpression(value) {
  const trimmed = value.trim();
  return /^\$\{\{[\s\S]*\}\}$/.test(trimmed);
}

/**
 * @param {string} extValue
 * @returns {string}
 */
function normalizeAllowedExtension(extValue) {
  const trimmed = extValue.trim();
  if (!trimmed) {
    return "";
  }
  if (isGitHubExpression(trimmed)) {
    return trimmed;
  }
  const normalized = trimmed.toLowerCase();
  return normalized.startsWith(".") ? normalized : `.${normalized}`;
}

/**
 * @param {string | undefined} envValue
 * @returns {{rawValues: string[], normalizedValues: string[], hasUnresolvedExpression: boolean} | null}
 */
function parseAllowedExtensionsEnv(envValue) {
  if (!envValue) {
    return null;
  }

  const rawValues = envValue.split(",").map(extValue => extValue.trim());
  return {
    rawValues,
    normalizedValues: rawValues.map(normalizeAllowedExtension).filter(Boolean),
    hasUnresolvedExpression: rawValues.some(isGitHubExpression),
  };
}

module.exports = {
  isGitHubExpression,
  normalizeAllowedExtension,
  parseAllowedExtensionsEnv,
};
