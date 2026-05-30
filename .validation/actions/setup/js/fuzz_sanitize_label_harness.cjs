// @ts-check
/**
 * Fuzz test harness for sanitize_label_content
 * This file is used by Go fuzz tests to test the sanitizeLabelContent function with various inputs.
 */

const { sanitizeLabelContent } = require("./sanitize_label_content.cjs");

/**
 * Test the sanitizeLabelContent function with given input
 * @param {string} text - Input text to sanitize
 * @returns {{sanitized: string, error: string | null}} Result object
 */
function testSanitizeLabelContent(text) {
  try {
    const result = sanitizeLabelContent(text);
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
      // Parse input as JSON: { text: string }
      const { text } = JSON.parse(input);
      const result = testSanitizeLabelContent(text);
      process.stdout.write(JSON.stringify(result));
      process.exit(0);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      process.stdout.write(JSON.stringify({ sanitized: "", error: errorMsg }));
      process.exit(1);
    }
  });
}

module.exports = { testSanitizeLabelContent };
