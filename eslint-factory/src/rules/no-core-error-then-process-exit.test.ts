import { RuleTester } from "@typescript-eslint/rule-tester";
import { noCoreErrorThenProcessExitRule } from "./no-core-error-then-process-exit";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: "latest",
    sourceType: "module",
  },
});

ruleTester.run("no-core-error-then-process-exit", noCoreErrorThenProcessExitRule, {
  valid: [
    // core.setFailed is already correct
    `core.setFailed("msg"); return;`,
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
  ],
  invalid: [
    {
      code: `core.error("something went wrong"); process.exit(1);`,
      errors: [{ messageId: "noCoreErrorThenProcessExit" }],
    },
    {
      code: `core.error("gateway failure: " + msg); process.exit(1);`,
      errors: [{ messageId: "noCoreErrorThenProcessExit" }],
    },
    {
      code: `core.error(\`ERROR: \${message}\`); process.exit(1);`,
      errors: [{ messageId: "noCoreErrorThenProcessExit" }],
    },
    {
      code: `function run() { core.error("oops"); process.exit(1); }`,
      errors: [{ messageId: "noCoreErrorThenProcessExit" }],
    },
    {
      // Computed property: core["error"]
      code: `core["error"]("msg"); process.exit(1);`,
      errors: [{ messageId: "noCoreErrorThenProcessExit" }],
    },
  ],
});
