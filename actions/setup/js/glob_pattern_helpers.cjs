// @ts-check

/**
 * Convert a glob pattern to a RegExp
 * @param {string} pattern - Glob pattern (e.g., "*.json", "metrics/**", "data/**\/*.csv")
 * @returns {RegExp} - Regular expression that matches the pattern
 *
 * Supports:
 * - * matches any characters except /
 * - ** matches any characters including /
 * - . is escaped to match literal dots
 * - \ is escaped properly
 *
 * @example
 * const regex = globPatternToRegex("*.json");
 * regex.test("file.json"); // true
 * regex.test("file.txt"); // false
 *
 * @example
 * const regex = globPatternToRegex("metrics/**");
 * regex.test("metrics/data.json"); // true
 * regex.test("metrics/daily/data.json"); // true
 */
function globPatternToRegex(pattern) {
  // Convert glob pattern to regex that supports directory wildcards
  // ** matches any path segment (including /)
  // * matches any characters except /
  let regexPattern = pattern
    .replace(/\\/g, "\\\\") // Escape backslashes
    .replace(/\./g, "\\.") // Escape dots
    .replace(/\*\*/g, "<!DOUBLESTAR>") // Temporarily replace **
    .replace(/\*/g, "[^/]*") // Single * matches non-slash chars
    .replace(/<!DOUBLESTAR>/g, ".*"); // ** matches everything including /
  return new RegExp(`^${regexPattern}$`);
}

/**
 * Parse a space-separated list of glob patterns into RegExp objects
 * @param {string} fileGlobFilter - Space-separated glob patterns (e.g., "*.json *.jsonl *.csv *.md")
 * @returns {RegExp[]} - Array of regular expressions
 *
 * @example
 * const patterns = parseGlobPatterns("*.json *.jsonl");
 * patterns[0].test("file.json"); // true
 * patterns[1].test("file.jsonl"); // true
 */
function parseGlobPatterns(fileGlobFilter) {
  return fileGlobFilter.trim().split(/\s+/).filter(Boolean).map(globPatternToRegex);
}

/**
 * Check if a file path matches any of the provided glob patterns
 * @param {string} filePath - File path to test (e.g., "data/file.json")
 * @param {string} fileGlobFilter - Space-separated glob patterns
 * @returns {boolean} - True if the file matches at least one pattern
 *
 * @example
 * matchesGlobPattern("file.json", "*.json *.jsonl"); // true
 * matchesGlobPattern("file.txt", "*.json *.jsonl"); // false
 */
function matchesGlobPattern(filePath, fileGlobFilter) {
  const patterns = parseGlobPatterns(fileGlobFilter);
  return patterns.some(pattern => pattern.test(filePath));
}

/**
 * Convert a simple glob pattern to a RegExp (for non-path matching)
 * @param {string} pattern - Glob pattern (e.g., "copilot", "*[bot]")
 * @param {boolean} caseSensitive - Whether matching should be case-sensitive (default: false)
 * @returns {RegExp} - Regular expression that matches the pattern
 *
 * Supports:
 * - * matches any characters (not limited to non-slash like globPatternToRegex)
 * - Escapes special regex characters except *
 * - Case-insensitive by default
 *
 * @example
 * const regex = simpleGlobToRegex("*[bot]");
 * regex.test("dependabot[bot]"); // true
 * regex.test("github-actions[bot]"); // true
 *
 * @example
 * const regex = simpleGlobToRegex("copilot");
 * regex.test("copilot"); // true
 * regex.test("Copilot"); // true (case-insensitive)
 */
function simpleGlobToRegex(pattern, caseSensitive = false) {
  // Escape special regex characters except *
  const regexPattern = pattern
    .replace(/[.+?^${}()|[\]\\]/g, "\\$&") // Escape special regex chars
    .replace(/\*/g, ".*"); // Convert * to .*

  return new RegExp(`^${regexPattern}$`, caseSensitive ? "" : "i");
}

/**
 * Check if a string matches a simple glob pattern
 * @param {string} str - String to test (e.g., "copilot", "dependabot[bot]")
 * @param {string} pattern - Glob pattern (e.g., "copilot", "*[bot]")
 * @param {boolean} caseSensitive - Whether matching should be case-sensitive (default: false)
 * @returns {boolean} - True if the string matches the pattern
 *
 * @example
 * matchesSimpleGlob("dependabot[bot]", "*[bot]"); // true
 * matchesSimpleGlob("copilot", "copilot"); // true
 * matchesSimpleGlob("Copilot", "copilot"); // true (case-insensitive by default)
 * matchesSimpleGlob("alice", "*[bot]"); // false
 */
function matchesSimpleGlob(str, pattern, caseSensitive = false) {
  if (!str || !pattern) {
    return false;
  }

  // Exact match check (case-insensitive by default)
  if (!caseSensitive && str.toLowerCase() === pattern.toLowerCase()) {
    return true;
  } else if (caseSensitive && str === pattern) {
    return true;
  }

  const regex = simpleGlobToRegex(pattern, caseSensitive);
  return regex.test(str);
}

module.exports = {
  globPatternToRegex,
  parseGlobPatterns,
  matchesGlobPattern,
  simpleGlobToRegex,
  matchesSimpleGlob,
};
