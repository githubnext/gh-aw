// @ts-check
/**
 * Fuzz test harness for removeXmlComments in sanitize_content_core.cjs
 * This file is used by Go fuzz tests to validate that the depth-tracking
 * comment scanner handles arbitrary inputs safely.
 */

const { removeXmlComments } = require("./sanitize_content_core.cjs");

/**
 * Test the removeXmlComments function with given input
 * @param {string} text - Input text to process
 * @returns {{result: string, error: string | null}} Result object
 */
function testRemoveXmlComments(text) {
  try {
    const result = removeXmlComments(text);
    return { result, error: null };
  } catch (err) {
    return {
      result: "",
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
      const result = testRemoveXmlComments(text);
      process.stdout.write(JSON.stringify(result));
      process.exit(0);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      process.stdout.write(JSON.stringify({ result: "", error: errorMsg }));
      process.exit(1);
    }
  });
}

module.exports = { testRemoveXmlComments };
