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
        // Acceptance criterion: setFailed between error and process.exit stops scanning
        `core.error("x"); core.setFailed("x"); process.exit(1);`,
        // return between error and process.exit stops scanning
        `function run() { core.error("x"); return; process.exit(1); }`,
        // throw between error and process.exit stops scanning
        `core.error("x"); throw new Error("x"); process.exit(1);`,
        // break between error and process.exit stops scanning
        `switch(x) { case 1: core.error("x"); break; process.exit(1); }`,
        // process.exit(0) between error and a later nonzero exit stops scanning (clean exit is unreachable)
        `core.error("x"); process.exit(0); process.exit(1);`,
        // process.exit(variable) between error and a later nonzero exit stops scanning (variable terminates process)
        `core.error("x"); process.exit(code); process.exit(1);`,
      ],
      invalid: [
        {
          // module top-level: no enclosing function (enclosingFn === null), autofix is safe because
          // there is no caller that could continue after the replacement statement.
          // The trailing space in the output is inter-statement whitespace left by the fixer.
          code: `core.error("fatal"); process.exit(1);`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [{ messageId: "replaceWithSetFailed", output: 'core.setFailed("fatal");\n ' }] }],
        },
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
          // pair inside a non-main function: no autofix because `return` only exits the helper,
          // not the process. The `run` name is not the entrypoint.
          code: `function run() { core.error("oops"); process.exit(1); }`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [] }],
        },
        {
          // pair inside a value-returning helper: no autofix — `return` would make the helper
          // return `undefined` instead of aborting the process (acceptance criterion a).
          code: `function requireEnvVar(name) { const value = process.env[name]; if (!value) { core.error(\`ERROR: \${name} required\`); process.exit(1); } return value; }`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [] }],
        },
        {
          // nested function main() inside another function: must NOT get autofix -- `return` only
          // exits the inner `main`, so the outer helper continues (module-scope restriction).
          code: `function setup() { function main() { core.error("fatal"); process.exit(1); } main(); }`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [] }],
        },
        {
          // nested const main = () => {} inside another function: must NOT get autofix.
          code: `function setup() { const main = async () => { core.error("fatal"); process.exit(1); }; main(); }`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [] }],
        },
        {
          // pair inside async function main() entrypoint: autofix retained (acceptance criterion c).
          code: `async function main() { core.error("fatal"); process.exit(1); }`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [{ messageId: "replaceWithSetFailed", output: 'async function main() { core.setFailed("fatal"); return;\n  }' }] }],
        },
        {
          // pair inside const main = async () => {} entrypoint: autofix retained.
          code: `const main = async () => { core.error("fatal"); process.exit(1); }`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [{ messageId: "replaceWithSetFailed", output: 'const main = async () => { core.setFailed("fatal"); return;\n  }' }] }],
        },
        {
          // Computed property: core["error"]
          code: `core["error"]("msg"); process.exit(1);`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [{ messageId: "replaceWithSetFailed", output: 'core.setFailed("msg");\n ' }] }],
        },
        // ── Non-adjacent pairs (intervening statements) ─────────────────────────
        {
          // Acceptance criterion: intervening log statement does not defeat detection
          code: `core.error("x"); core.info("y"); process.exit(1);`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [] }],
        },
        {
          // Intervening summary call does not defeat detection
          code: `core.error("fatal"); core.summary.addRaw("fatal"); process.exit(1);`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [] }],
        },
        {
          // Two intervening statements
          code: `core.error("fatal"); core.info("a"); core.info("b"); process.exit(1);`,
          errors: [{ messageId: "noCoreErrorThenProcessExit", suggestions: [] }],
        },
      ],
    });
  });
});
