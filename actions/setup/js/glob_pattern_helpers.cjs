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

module.exports = {
  globPatternToRegex,
  parseGlobPatterns,
  matchesGlobPattern,
};
