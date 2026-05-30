// @ts-check
/**
 * Fuzz test harness for isSafeExpression in runtime_import.cjs.
 *
 * Tests the security invariants of expression validation, covering:
 * - Simple allowed expressions (github.*, needs.*, steps.*, etc.)
 * - Compound expressions: AND (&&), OR (||), comparisons (==, !=, <=, >=)
 * - Ternary-style expressions: condition && 'value' || 'default'
 * - Standalone literals: strings, numbers, booleans
 * - Security boundaries: secrets.*, vars.*, runner.*, github.token
 * - Dangerous prototype-pollution property names
 *
 * Used by tests and by Go's fuzzing framework when run as main module via stdin.
 */

const { isSafeExpression } = require("./runtime_import.cjs");

/**
 * Evaluates isSafeExpression and returns a structured result.
 * Never throws — all errors are captured in the error field.
 * @param {string} expression - The expression to test (without ${{ }})
 * @returns {{ safe: boolean, error: string | null }}
 */
function testIsSafeExpression(expression) {
  try {
    const safe = isSafeExpression(expression);
    return { safe, error: null };
  } catch (err) {
    return { safe: false, error: err instanceof Error ? err.message : String(err) };
  }
}

/**
 * Returns true when the expression is known to be unsafe regardless of
 * what other sub-expressions it may contain.
 * Used by the fuzzer to assert security invariants.
 * @param {string} expression
 * @returns {boolean}
 */
function containsUnsafeRoot(expression) {
  const trimmed = expression.trim();
  // Expressions that start with a disallowed namespace are always unsafe.
  // We only check "starts with" to avoid false positives on safe sub-expressions.
  return /^secrets\./.test(trimmed) || /^runner\./.test(trimmed) || /^vars\./.test(trimmed) || trimmed === "github.token";
}

// Read input from stdin for Go-driven fuzzing
if (require.main === module) {
  let input = "";

  process.stdin.on("data", chunk => {
    input += chunk;
  });

  process.stdin.on("end", () => {
    try {
      // Expected JSON: { expression: string }
      const { expression } = JSON.parse(input);
      const result = testIsSafeExpression(expression ?? "");
      process.stdout.write(JSON.stringify(result));
      process.exit(0);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      process.stdout.write(JSON.stringify({ safe: false, error: errorMsg }));
      process.exit(1);
    }
  });
}

module.exports = { testIsSafeExpression, containsUnsafeRoot };
