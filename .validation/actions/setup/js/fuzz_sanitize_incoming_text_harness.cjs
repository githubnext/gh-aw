// @ts-check
/**
 * Fuzz test harness for sanitize_incoming_text (core sanitization without mention filtering)
 * This file is used by Go fuzz tests to test the sanitizeIncomingText function with various inputs.
 */

const { sanitizeIncomingText } = require("./sanitize_incoming_text.cjs");

/**
 * Test the sanitizeIncomingText function with given input
 * @param {string} text - Input text to sanitize
 * @param {number} maxLength - Maximum length of content
 * @returns {{sanitized: string, error: string | null}} Result object
 */
function testSanitizeIncomingText(text, maxLength) {
  try {
    const result = sanitizeIncomingText(text, maxLength);
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
      // Parse input as JSON: { text: string, maxLength: number }
      const { text, maxLength } = JSON.parse(input);
      const result = testSanitizeIncomingText(text, maxLength);
      process.stdout.write(JSON.stringify(result));
      process.exit(0);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      process.stdout.write(JSON.stringify({ sanitized: "", error: errorMsg }));
      process.exit(1);
    }
  });
}

module.exports = { testSanitizeIncomingText };
