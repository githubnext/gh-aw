import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireExecFileSyncTryCatchRule } from "./require-execfilesync-try-catch";

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

describe("require-execfilesync-try-catch", () => {
  it("valid: execFileSync inside try block passes (CommonJS, destructured)", () => {
    cjsRuleTester.run("require-execfilesync-try-catch", requireExecFileSyncTryCatchRule, {
      valid: [
        `const { execFileSync } = require("child_process"); try { execFileSync("git", ["status"]); } catch (e) {}`,
        `const { execFileSync } = require("node:child_process"); try { execFileSync("git", ["status"]); } catch (e) {}`,
        `const { execFileSync } = require("child_process"); function f() { try { execFileSync("git", ["status"]); } catch (e) {} }`,
      ],
      invalid: [],
    });
  });

  it("valid: execFileSync inside try block passes (CommonJS, namespace)", () => {
    cjsRuleTester.run("require-execfilesync-try-catch", requireExecFileSyncTryCatchRule, {
      valid: [`const cp = require("child_process"); try { cp.execFileSync("git", ["status"]); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: execFileSync inside try block passes (ES module)", () => {
    esmRuleTester.run("require-execfilesync-try-catch", requireExecFileSyncTryCatchRule, {
      valid: [`import { execFileSync } from "child_process"; try { execFileSync("git", ["status"]); } catch (e) {}`, `import { execFileSync } from "node:child_process"; try { execFileSync("git", ["status"]); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: execFileSync from non-child_process module is ignored", () => {
    cjsRuleTester.run("require-execfilesync-try-catch", requireExecFileSyncTryCatchRule, {
      valid: [
        // execFileSync from an unrelated module — should not be flagged
        `const { execFileSync } = require("some-other-lib"); execFileSync("git", ["status"]);`,
        // bare execFileSync without any require — should not be flagged
        `execFileSync("git", ["status"]);`,
        // member call on unrelated object
        `mockChild.execFileSync("git", ["status"]);`,
      ],
      invalid: [],
    });
  });

  it("invalid: execFileSync without try/catch (CommonJS, destructured)", () => {
    cjsRuleTester.run("require-execfilesync-try-catch", requireExecFileSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { execFileSync } = require("child_process"); execFileSync("git", ["status"]);`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { execFileSync } = require("child_process"); try {\n  execFileSync("git", ["status"]);\n} catch (err) {\n  // TODO: handle execFileSync failure (non-zero exit / signal termination).\n  throw new Error(\n    "execFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: execFileSync without try/catch (CommonJS, namespace)", () => {
    cjsRuleTester.run("require-execfilesync-try-catch", requireExecFileSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const cp = require("child_process"); cp.execFileSync("git", ["status"]);`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const cp = require("child_process"); try {\n  cp.execFileSync("git", ["status"]);\n} catch (err) {\n  // TODO: handle execFileSync failure (non-zero exit / signal termination).\n  throw new Error(\n    "execFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const cp = require("node:child_process"); cp.execFileSync("git", ["status"]);`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const cp = require("node:child_process"); try {\n  cp.execFileSync("git", ["status"]);\n} catch (err) {\n  // TODO: handle execFileSync failure (non-zero exit / signal termination).\n  throw new Error(\n    "execFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: execFileSync without try/catch (ES module)", () => {
    esmRuleTester.run("require-execfilesync-try-catch", requireExecFileSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `import { execFileSync } from "child_process"; execFileSync("git", ["status"]);`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `import { execFileSync } from "child_process"; try {\n  execFileSync("git", ["status"]);\n} catch (err) {\n  // TODO: handle execFileSync failure (non-zero exit / signal termination).\n  throw new Error(\n    "execFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: destructured alias of execFileSync without try/catch is flagged", () => {
    cjsRuleTester.run("require-execfilesync-try-catch", requireExecFileSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { execFileSync: run } = require("child_process"); run("git", ["status"]);`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { execFileSync: run } = require("child_process"); try {\n  run("git", ["status"]);\n} catch (err) {\n  // TODO: handle execFileSync failure (non-zero exit / signal termination).\n  throw new Error(\n    "execFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: member-expression alias of execFileSync without try/catch is flagged", () => {
    cjsRuleTester.run("require-execfilesync-try-catch", requireExecFileSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const cp = require("child_process"); const run = cp.execFileSync; run("git", ["status"]);`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const cp = require("child_process"); const run = cp.execFileSync; try {\n  run("git", ["status"]);\n} catch (err) {\n  // TODO: handle execFileSync failure (non-zero exit / signal termination).\n  throw new Error(\n    "execFileSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
