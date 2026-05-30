// @ts-check
/**
 * Fuzz test harness for sanitize_output (sanitizeContent with selective mention filtering)
 * This file is used by Go fuzz tests to test the sanitizeContent function with various inputs.
 */

const { sanitizeContent } = require("./sanitize_content.cjs");

/**
 * Test the sanitizeContent function with given input
 * @param {string} text - Input text to sanitize
 * @param {string[]} allowedAliases - List of allowed mention aliases
 * @param {number} maxLength - Maximum length of content
 * @returns {{sanitized: string, error: string | null}} Result object
 */
function testSanitizeOutput(text, allowedAliases, maxLength) {
  try {
    const result = sanitizeContent(text, { allowedAliases, maxLength });
    return { sanitized: result, error: null };
  } catch (err) {
    return {
      sanitized: "",
      error: err instanceof Error ? err.message : String(err),
    };
  }
}

// Read input from stdin for fuzzing
if (require.main === module) {
  let input = "";

  process.stdin.on("data", chunk => {
    input += chunk;
  });

  process.stdin.on("end", () => {
    try {
      // Parse input as JSON: { text: string, allowedAliases: string[], maxLength: number }
      const { text, allowedAliases, maxLength } = JSON.parse(input);
      const result = testSanitizeOutput(text, allowedAliases || [], maxLength);
      process.stdout.write(JSON.stringify(result));
      process.exit(0);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      process.stdout.write(JSON.stringify({ sanitized: "", error: errorMsg }));
      process.exit(1);
    }
  });
}

module.exports = { testSanitizeOutput };
