/**
 * Utility functions for converting Go regex patterns to JavaScript-compatible patterns
 */

/**
 * Converts a Go regex pattern to a JavaScript-compatible pattern
 * @param {string} goPattern - The Go regex pattern (may include (?i) for case-insensitive)
 * @returns {{pattern: string, flags: string}} - Object with converted pattern and flags
 */
function convertGoPatternToJS(goPattern) {
  let pattern = goPattern;
  let flags = "g"; // Default to global matching

  // Convert (?i) inline case-insensitive flag to JavaScript 'i' flag
  if (pattern.startsWith("(?i)")) {
    pattern = pattern.substring(4); // Remove (?i) prefix
    flags = "gi"; // Add case-insensitive flag
  }

  return { pattern, flags };
}

/**
 * Creates a JavaScript RegExp from a Go regex pattern
 * @param {string} goPattern - The Go regex pattern
 * @returns {RegExp} - JavaScript RegExp object
 */
function createJSRegexFromGoPattern(goPattern) {
  const { pattern, flags } = convertGoPatternToJS(goPattern);
  return new RegExp(pattern, flags);
}

/**
 * Tests if a Go regex pattern is compatible with JavaScript
 * @param {string} goPattern - The Go regex pattern to test
 * @returns {{compatible: boolean, error?: string, convertedPattern?: string, flags?: string}} - Compatibility result
 */
function testGoPatternCompatibility(goPattern) {
  try {
    const { pattern, flags } = convertGoPatternToJS(goPattern);
    new RegExp(pattern, flags); // Test if it compiles
    return {
      compatible: true,
      convertedPattern: pattern,
      flags
    };
  } catch (error) {
    return {
      compatible: false,
      error: error instanceof Error ? error.message : String(error)
    };
  }
}

module.exports = {
  convertGoPatternToJS,
  createJSRegexFromGoPattern,
  testGoPatternCompatibility
};