// @ts-check
/**
 * Fuzz test harness for mentions filtering
 * This file is used by Go fuzz tests to test the neutralizeMentions function
 * in isolation with various inputs.
 */

const { sanitizeContent } = require("./sanitize_content.cjs");

/**
 * Test the mentions filtering with given input and allowed aliases
 * @param {string} text - Input text to sanitize
 * @param {string[]} allowedAliases - List of allowed mention aliases
 * @returns {{sanitized: string, error: string | null}} Result object
 */
function testMentionsFiltering(text, allowedAliases) {
  try {
    const result = sanitizeContent(text, { allowedAliases });
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
      // Parse input as JSON: { text: string, allowedAliases: string[] }
      const { text, allowedAliases } = JSON.parse(input);
      const result = testMentionsFiltering(text, allowedAliases || []);
      process.stdout.write(JSON.stringify(result));
      process.exit(0);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      process.stdout.write(JSON.stringify({ sanitized: "", error: errorMsg }));
      process.exit(1);
    }
  });
}

module.exports = { testMentionsFiltering };
