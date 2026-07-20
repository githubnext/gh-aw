import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireExecSyncTryCatchRule } from "./require-execsync-try-catch";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

const esmRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "module",
  },
});

describe("require-execsync-try-catch", () => {
  it("valid: execSync inside try block passes (CommonJS, destructured)", () => {
    cjsRuleTester.run("require-execsync-try-catch", requireExecSyncTryCatchRule, {
      valid: [
        `const { execSync } = require("child_process"); try { execSync("ls"); } catch (e) {}`,
        `const { execSync } = require("node:child_process"); try { execSync("ls"); } catch (e) {}`,
        `const { execSync } = require("child_process"); function f() { try { execSync("ls"); } catch (e) {} }`,
      ],
      invalid: [],
    });
  });

  it("valid: execSync inside try block passes (CommonJS, namespace)", () => {
    cjsRuleTester.run("require-execsync-try-catch", requireExecSyncTryCatchRule, {
      valid: [`const cp = require("child_process"); try { cp.execSync("ls"); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: execSync inside try block passes (ES module)", () => {
    esmRuleTester.run("require-execsync-try-catch", requireExecSyncTryCatchRule, {
      valid: [`import { execSync } from "child_process"; try { execSync("ls"); } catch (e) {}`, `import { execSync } from "node:child_process"; try { execSync("ls"); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: execSync from non-child_process module is ignored", () => {
    cjsRuleTester.run("require-execsync-try-catch", requireExecSyncTryCatchRule, {
      valid: [
        // execSync from an unrelated module — should not be flagged
        `const { execSync } = require("some-other-lib"); execSync("ls");`,
        // bare execSync without any require — should not be flagged
        `execSync("ls");`,
        // member call on unrelated object
        `mockChild.execSync("ls");`,
      ],
      invalid: [],
    });
  });

  it("invalid: execSync without try/catch (CommonJS, destructured)", () => {
    cjsRuleTester.run("require-execsync-try-catch", requireExecSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { execSync } = require("child_process"); execSync("ls");`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { execSync } = require("child_process"); try {\n  execSync("ls");\n} catch (err) {\n  // TODO: handle execSync failure (non-zero exit / signal termination).\n  throw new Error(\n    "execSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: execSync without try/catch (CommonJS, namespace)", () => {
    cjsRuleTester.run("require-execsync-try-catch", requireExecSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const cp = require("child_process"); cp.execSync("ls");`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const cp = require("child_process"); try {\n  cp.execSync("ls");\n} catch (err) {\n  // TODO: handle execSync failure (non-zero exit / signal termination).\n  throw new Error(\n    "execSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: execSync without try/catch (ES module)", () => {
    esmRuleTester.run("require-execsync-try-catch", requireExecSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `import { execSync } from "child_process"; execSync("ls");`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `import { execSync } from "child_process"; try {\n  execSync("ls");\n} catch (err) {\n  // TODO: handle execSync failure (non-zero exit / signal termination).\n  throw new Error(\n    "execSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: aliased execSync without try/catch is flagged", () => {
    cjsRuleTester.run("require-execsync-try-catch", requireExecSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { execSync: run } = require("child_process"); run("ls");`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { execSync: run } = require("child_process"); try {\n  run("ls");\n} catch (err) {\n  // TODO: handle execSync failure (non-zero exit / signal termination).\n  throw new Error(\n    "execSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
