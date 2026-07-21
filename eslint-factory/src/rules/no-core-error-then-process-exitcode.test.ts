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
        // Exports between statements break adjacency at module scope
        `const helper = 1; core.error("msg"); export { helper }; process.exitCode = 1;`,
        // process.exit (covered by the sibling rule)
        `core.error("msg"); process.exit(1);`,
        // Not a simple assignment: += is not flagged
        `core.error("msg"); process.exitCode += 1;`,
      ],
      invalid: [
        {
          code: `core.error("fatal"); process.exitCode = 1;`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [] }],
        },
        {
          code: `core.error("something went wrong"); process.exitCode = 1;`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [] }],
        },
        {
          code: "core.error(`ERROR: ${msg}`); process.exitCode = 1;",
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [] }],
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
          // SwitchCase path reports the pattern without an autofix outside main()
          code: `switch (x) { case 1: core.error("fatal"); process.exitCode = 1; break; }`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [] }],
        },
        {
          // exitCode = 2 is also flagged
          code: `core.error("critical"); process.exitCode = 2;`,
          errors: [{ messageId: "noCoreErrorThenProcessExitCode", suggestions: [] }],
        },
      ],
    });
  });
});
