// Uses eslint's RuleTester rather than @typescript-eslint/rule-tester, matching the
// convention of all other rule tests in this package. The rule uses @typescript-eslint/utils
// internally but the standard eslint RuleTester is sufficient for all test scenarios here.
import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noCoreErrorThenProcessExitRule } from "./no-core-error-then-process-exit";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: "latest",
    sourceType: "module",
  },
});

describe("no-core-error-then-process-exit", () => {
  it("valid and invalid cases", () => {
    ruleTester.run("no-core-error-then-process-exit", noCoreErrorThenProcessExitRule, {
      valid: [
        // core.setFailed is already the correct pattern (inside a function)
        `function run() { core.setFailed("msg"); return; }`,
        // process.exit(0) is fine
        `core.error("msg"); process.exit(0);`,
        // core.error without process.exit is fine
        `core.error("msg");`,
        // process.exit(1) without core.error before it is fine
        `process.exit(1);`,
        // Non-core object
        `logger.error("msg"); process.exit(1);`,
        // core.warning is not core.error
        `core.warning("msg"); process.exit(1);`,
        // Variable argument — runtime value cannot be proven non-zero
        `core.error("msg"); process.exit(code);`,
        // Function call argument — runtime value unknown
        `core.error("msg"); process.exit(getExitCode());`,
        // String literal argument — not a numeric literal
        `core.error("msg"); process.exit("1");`,
      ],
      invalid: [
        {
          // The trailing space in each output is the whitespace between the two original
          // statements that is not part of either ExpressionStatement node's range. The
          // suggestion fixer removes the process.exit() node but leaves the inter-statement
          // whitespace intact, which is expected ESLint suggestion behavior.
          code: `core.error("something went wrong"); process.exit(1);`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [{ messageId: "replaceWithSetFailed", output: 'core.setFailed("something went wrong");\n ' }] }],
        },
        {
          code: `core.error("gateway failure: " + msg); process.exit(1);`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [{ messageId: "replaceWithSetFailed", output: 'core.setFailed("gateway failure: " + msg);\n ' }] }],
        },
        {
          code: `core.error(\`ERROR: \${message}\`); process.exit(1);`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [{ messageId: "replaceWithSetFailed", output: "core.setFailed(`ERROR: ${message}`);\n " }] }],
        },
        {
          code: `function run() { core.error("oops"); process.exit(1); }`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [{ messageId: "replaceWithSetFailed", output: 'function run() { core.setFailed("oops"); return;\n  }' }] }],
        },
        {
          // Computed property: core["error"]
          code: `core["error"]("msg"); process.exit(1);`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [{ messageId: "replaceWithSetFailed", output: 'core.setFailed("msg");\n ' }] }],
        },
      ],
    });
  });
});
