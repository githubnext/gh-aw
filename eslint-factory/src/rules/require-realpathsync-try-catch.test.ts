import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireRealpathSyncTryCatchRule } from "./require-realpathsync-try-catch";

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

describe("require-realpathsync-try-catch", () => {
  it("valid: fs.realpathSync inside try block passes (CommonJS)", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [
        `const fs = require("fs"); try { fs.realpathSync(dir); } catch (e) {}`,
        `const fs = require("fs"); function f() { try { fs.realpathSync(dir); } catch (e) {} }`,
        `const fs = require("fs"); try { fs["realpathSync"](dir); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("valid: destructured realpathSync inside try block passes", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [`const { realpathSync } = require("fs"); try { realpathSync(dir); } catch (e) {}`, `const { realpathSync } = require("node:fs"); try { realpathSync(dir); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: non-fs receiver names with realpathSync are ignored", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [`mockFs.realpathSync(dir);`, `storage.realpathSync(dir);`, `myObj.realpathSync(dir);`, `const fs = require("mock-fs"); fs.realpathSync(dir);`],
      invalid: [],
    });
  });

  it("valid: other fs methods remain out of scope", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [`fs.existsSync(path);`, `fs.mkdirSync(dir);`, `fs.statSync(path);`, `fs.readdirSync(dir);`],
      invalid: [],
    });
  });

  it("invalid: bare fs.realpathSync is flagged (CommonJS)", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `fs.realpathSync(promptsDir);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "promptsDir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  fs.realpathSync(promptsDir);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); const root = fs.realpathSync(promptsDir);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "promptsDir" },
            },
          ],
        },
        {
          code: `const fs = require("fs"); let resolvedRoot; resolvedRoot = fs.realpathSync(root);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "root" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); let resolvedRoot; try {\n  resolvedRoot = fs.realpathSync(root);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); function resolve() { fs.realpathSync(dir); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); function resolve() { try {\n  fs.realpathSync(dir);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: destructured realpathSync outside try is flagged", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { realpathSync } = require("fs"); realpathSync(dir);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { realpathSync } = require("fs"); try {\n  realpathSync(dir);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fs.realpathSync in async function without try is flagged", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const fs = require("fs"); async function run() { let resolved; resolved = fs.realpathSync(dir); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); async function run() { let resolved; try {\n  resolved = fs.realpathSync(dir);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fs.realpathSync inside try/finally without catch is flagged", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const fs = require("fs"); try { fs.realpathSync(dir); } finally { cleanup(); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: 1,
            },
          ],
        },
      ],
    });
  });

  it("valid: fs.realpathSync inside try block passes (ESM)", () => {
    esmRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [`import * as fs from "fs"; try { fs.realpathSync(dir); } catch (e) {}`],
      invalid: [],
    });
  });

  it("invalid: bare fs.realpathSync is flagged (ESM)", () => {
    esmRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `import * as fs from "fs"; fs.realpathSync(dir);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `import * as fs from "fs"; try {\n  fs.realpathSync(dir);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `import { realpathSync } from "fs"; realpathSync(dir);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: 1,
            },
          ],
        },
      ],
    });
  });
});
