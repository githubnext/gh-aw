// @ts-check

const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");

/**
 * Parse allowed base branch patterns from config value (array or comma-separated string)
 * @param {string[]|string|undefined} allowedBaseBranchesValue
 * @returns {Set<string>}
 */
function parseAllowedBaseBranches(allowedBaseBranchesValue) {
  const set = new Set();
  if (Array.isArray(allowedBaseBranchesValue)) {
    allowedBaseBranchesValue
      .map(branch => String(branch).trim())
      .filter(Boolean)
      .forEach(branch => set.add(branch));
  } else if (typeof allowedBaseBranchesValue === "string") {
    allowedBaseBranchesValue
      .split(",")
      .map(branch => branch.trim())
      .filter(Boolean)
      .forEach(branch => set.add(branch));
  }
  return set;
}

/**
 * Check if a base branch matches an allowed pattern.
 * Supports exact matches and "*" glob patterns (e.g. "release/*").
 * @param {string} baseBranch
 * @param {Set<string>} allowedBaseBranches
 * @returns {boolean}
 */
function isBaseBranchAllowed(baseBranch, allowedBaseBranches) {
  if (allowedBaseBranches.has(baseBranch)) {
    return true;
  }
  for (const pattern of allowedBaseBranches) {
    if (pattern === "*") {
      return true;
    }
    // Exact-match patterns were already handled above by Set lookup.
    // Skip regex compilation unless the pattern is a glob.
    if (!pattern.includes("*")) {
      continue;
    }
    const regex = globPatternToRegex(pattern, { pathMode: true, caseSensitive: true });
    if (regex.test(baseBranch)) {
      return true;
    }
  }
  return false;
}

module.exports = {
  parseAllowedBaseBranches,
  isBaseBranchAllowed,
};
