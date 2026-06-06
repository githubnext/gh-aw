// @ts-check
/**
 * Fuzz test harness for bash_command_parser.cjs
 *
 * Tests security and correctness invariants of the bash pipeline parser:
 *   - splitOnPipelineOperators: splitting on &&, ||, |, ;
 *   - extractCommandName: extracts first executable word from a segment
 *   - extractCommandNamesFromPipeline: end-to-end pipeline parsing
 *
 * Security invariants:
 *   - The parser never throws on arbitrary input (robustness)
 *   - An empty or unparseable command always yields an empty array (safe default)
 *   - Operators inside quoted strings are never treated as separators
 *   - The result is always a (possibly empty) array of strings
 *
 * Used by:
 *   - fuzz_bash_command_parser_harness.test.cjs: property-based tests in vitest
 *   - Go fuzzer: reads JSON from stdin when run as main module
 */

"use strict";

const { splitOnPipelineOperators, extractCommandName, extractCommandNamesFromPipeline } = require("./bash_command_parser.cjs");

/**
 * Test splitOnPipelineOperators and return a structured result.
 * Never throws — all errors are captured in the error field.
 *
 * @param {string} commandText
 * @returns {{ segments: string[], error: string | null }}
 */
function testSplitOnPipelineOperators(commandText) {
  try {
    const segments = splitOnPipelineOperators(commandText);
    return { segments, error: null };
  } catch (err) {
    return { segments: [], error: err instanceof Error ? err.message : String(err) };
  }
}

/**
 * Test extractCommandName and return a structured result.
 * Never throws.
 *
 * @param {string} segment
 * @returns {{ name: string | null, error: string | null }}
 */
function testExtractCommandName(segment) {
  try {
    const name = extractCommandName(segment);
    return { name, error: null };
  } catch (err) {
    return { name: null, error: err instanceof Error ? err.message : String(err) };
  }
}

/**
 * Test extractCommandNamesFromPipeline and return a structured result.
 * Never throws.
 *
 * @param {string} commandText
 * @returns {{ names: string[], error: string | null }}
 */
function testExtractCommandNamesFromPipeline(commandText) {
  try {
    const names = extractCommandNamesFromPipeline(commandText);
    return { names, error: null };
  } catch (err) {
    return { names: [], error: err instanceof Error ? err.message : String(err) };
  }
}

/**
 * Check the security invariant: a command containing only quoted pipeline operators
 * must NOT be split into multiple segments.
 *
 * @param {string} operator - e.g. "&&", "||", "|", ";"
 * @returns {boolean} true when the invariant holds
 */
function quotedOperatorIsNotSplit(operator) {
  const singleQuoted = `echo '${operator}'`;
  const doubleQuoted = `echo "${operator}"`;

  const singleResult = testSplitOnPipelineOperators(singleQuoted);
  const doubleResult = testSplitOnPipelineOperators(doubleQuoted);

  return singleResult.error === null && singleResult.segments.length === 1 && doubleResult.error === null && doubleResult.segments.length === 1;
}

/**
 * Check the no-throw invariant for a given input.
 * Returns true when no error is thrown and result arrays are valid.
 *
 * @param {string} input
 * @returns {boolean}
 */
function noThrowInvariant(input) {
  const split = testSplitOnPipelineOperators(input);
  const name = testExtractCommandName(input);
  const names = testExtractCommandNamesFromPipeline(input);

  return split.error === null && Array.isArray(split.segments) && name.error === null && names.error === null && Array.isArray(names.names);
}

/**
 * Check the safe-default invariant: empty / whitespace-only input yields empty arrays.
 *
 * @param {string} input - Should be empty or whitespace-only
 * @returns {boolean}
 */
function emptyInputYieldsEmptyArrays(input) {
  const split = testSplitOnPipelineOperators(input);
  const names = testExtractCommandNamesFromPipeline(input);
  return split.segments.length === 0 && names.names.length === 0;
}

// ─────────────────────────────────────────────────────────────────────────────
// Standalone entry point for Go-driven fuzzing
// ─────────────────────────────────────────────────────────────────────────────

if (require.main === module) {
  let input = "";

  process.stdin.on("data", chunk => {
    input += chunk;
  });

  process.stdin.on("end", () => {
    try {
      // Expected JSON: { commandText: string, mode?: "split" | "name" | "pipeline" }
      const { commandText, mode } = JSON.parse(input);
      const text = commandText ?? "";

      let result;
      switch (mode) {
        case "split":
          result = testSplitOnPipelineOperators(text);
          break;
        case "name":
          result = testExtractCommandName(text);
          break;
        default:
          result = testExtractCommandNamesFromPipeline(text);
      }

      process.stdout.write(JSON.stringify(result));
      process.exit(0);
    } catch (err) {
      const errorMsg = err instanceof Error ? err.message : String(err);
      process.stdout.write(JSON.stringify({ error: errorMsg }));
      process.exit(1);
    }
  });
}

module.exports = {
  testSplitOnPipelineOperators,
  testExtractCommandName,
  testExtractCommandNamesFromPipeline,
  quotedOperatorIsNotSplit,
  noThrowInvariant,
  emptyInputYieldsEmptyArrays,
};
