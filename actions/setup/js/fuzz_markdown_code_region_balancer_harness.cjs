// @ts-check
/**
 * Fuzz test harness for markdown code region balancer
 * This file is used by Go fuzz tests to test the balanceCodeRegions function with various inputs.
 */

const { balanceCodeRegions, isBalanced, countCodeRegions } = require("./markdown_code_region_balancer.cjs");

/**
 * Test the balanceCodeRegions function with given input
 * @param {string} markdown - Input markdown to balance
 * @returns {{balanced: string, isBalanced: boolean, counts: {total: number, balanced: number, unbalanced: number}, error: string | null}} Result object
 */
function testBalanceCodeRegions(markdown) {
  try {
    const balanced = balanceCodeRegions(markdown);
    const balanced_check = isBalanced(balanced);
    const counts = countCodeRegions(markdown);
    return {
      balanced,
      isBalanced: balanced_check,
      counts,
      error: null,
    };
  } catch (err) {
    return {
      balanced: "",
      isBalanced: false,
      counts: { total: 0, balanced: 0, unbalanced: 0 },
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
      // Parse input as JSON: { markdown: string }
      const { markdown } = JSON.parse(input);
      const result = testBalanceCodeRegions(markdown || "");
      process.stdout.write(JSON.stringify(result));
      process.exit(0);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      process.stdout.write(
        JSON.stringify({
          balanced: "",
          isBalanced: false,
          counts: { total: 0, balanced: 0, unbalanced: 0 },
          error: errorMsg,
        })
      );
      process.exit(1);
    }
  });
}

module.exports = {
  testBalanceCodeRegions,
};
