import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noCoreErrorThenProcessExitCodeRule } from "./no-core-error-then-process-exitcode";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: "latest",
    sourceType: "module",
  },
});

describe("no-core-error-then-process-exitcode", () => {
  it("valid and invalid cases", () => {
    ruleTester.run("no-core-error-then-process-exitcode", noCoreErrorThenProcessExitCodeRule, {
      valid: [
        // core.setFailed is already the correct pattern
        `function run() { core.setFailed("msg"); return; }`,
        // process.exitCode = 0 is a clean exit
        `core.error("msg"); process.exitCode = 0;`,
        // core.error without process.exitCode assignment
        `core.error("msg");`,
        // process.exitCode = 1 without a preceding core.error
        `process.exitCode = 1;`,
        // Non-core object
        `logger.error("msg"); process.exitCode = 1;`,
        // core.warning is not core.error
        `core.warning("msg"); process.exitCode = 1;`,
        // Variable assignment — runtime value unknown
        `core.error("msg"); process.exitCode = code;`,
        // Exports between statements break the scan at module scope
        `const helper = 1; core.error("msg"); export { helper }; process.exitCode = 1;`,
        // process.exit (covered by the sibling rule)
        `core.error("msg"); process.exit(1);`,
        // Not a simple assignment: += is not flagged
        `core.error("msg"); process.exitCode += 1;`,
        // core.setFailed between error and exitCode stops scanning
        `core.error("x"); core.setFailed("y"); process.exitCode = 1;`,
        // return between error and exitCode stops scanning (inside a function)
        `function run() { core.error("x"); return; process.exitCode = 1; }`,
        // throw between error and exitCode stops scanning
        `function run() { core.error("x"); throw new Error("x"); process.exitCode = 1; }`,
        // break between error and exitCode stops scanning (inside a loop)
        `while (true) { core.error("x"); break; process.exitCode = 1; }`,
        // continue between error and exitCode stops scanning (inside a loop)
        `for (let i = 0; i < 10; i++) { core.error("x"); continue; process.exitCode = 1; }`,
        // process.exit() between error and exitCode stops scanning (dot access)
        `core.error("x"); process.exit(1); process.exitCode = 1;`,
        // process["exit"]() between error and exitCode stops scanning (computed access)
        `core.error("x"); process["exit"](1); process.exitCode = 1;`,
      ],
      invalid: [
        {
          // module top-level: autofix is safe — no caller continues after replacement
          code: `core.error("fatal"); process.exitCode = 1;`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [{ messageId: "replaceWithSetFailed", output: 'core.setFailed("fatal");\n ' }] }],
        },
        {
          code: `core.error("something went wrong"); process.exitCode = 1;`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [{ messageId: "replaceWithSetFailed", output: 'core.setFailed("something went wrong");\n ' }] }],
        },
        {
          code: "core.error(`ERROR: ${msg}`); process.exitCode = 1;",
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [{ messageId: "replaceWithSetFailed", output: "core.setFailed(`ERROR: ${msg}`);\n " }] }],
        },
        {
          // Inside a named function — no autofix suggestion because return; only exits the helper
          code: `function helper() { core.error("fatal"); process.exitCode = 1; }`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [] }],
        },
        {
          // Inside main() — autofix is safe
          code: `async function main() { core.error("fatal"); process.exitCode = 1; }`,
          errors: [
            {
              messageId: "noCoreErrorThenProcessExitCode",
              suggestions: [{ messageId: "replaceWithSetFailed", output: 'async function main() { core.setFailed("fatal"); return;\n  }' }],
            },
          ],
        },
        {
          // export async function main() should also be recognized as a safe entrypoint
          code: `export async function main() { core.error("fatal"); process.exitCode = 1; }`,
          errors: [
            {
              messageId: "noCoreErrorThenProcessExitCode",
              suggestions: [{ messageId: "replaceWithSetFailed", output: 'export async function main() { core.setFailed("fatal"); return;\n  }' }],
            },
          ],
        },
        {
          // Multiple arguments are reported but not auto-fixed because setFailed only accepts the message
          code: `async function main() { core.error("fatal", { title: "oops" }); process.exitCode = 1; }`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [] }],
        },
        {
          // SwitchCase at module top level: autofix is safe (enclosingFn === null)
          code: `switch (x) { case 1: core.error("fatal"); process.exitCode = 1; break; }`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [{ messageId: "replaceWithSetFailed", output: 'switch (x) { case 1: core.setFailed("fatal");\n  break; }' }] }],
        },
        {
          // exitCode = 2 is also flagged
          code: `core.error("critical"); process.exitCode = 2;`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [{ messageId: "replaceWithSetFailed", output: 'core.setFailed("critical");\n ' }] }],
        },
        {
          // Non-adjacent pair: intervening statement does not defeat detection; no autofix suggestion
          code: `core.error("x"); core.info("y"); process.exitCode = 1;`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [] }],
        },
        {
          // Two intervening statements
          code: `core.error("fatal"); core.info("a"); core.info("b"); process.exitCode = 1;`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [] }],
        },
        {
          // Two consecutive core.error calls before the same process.exitCode: only the first
          // core.error reports (non-adjacent, no autofix) — deduplication prevents a second
          // diagnostic and any conflicting autofix from the adjacent core.error("b").
          code: `core.error("a"); core.error("b"); process.exitCode = 1;`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [] }],
        },
      ],
    });
  });
});
