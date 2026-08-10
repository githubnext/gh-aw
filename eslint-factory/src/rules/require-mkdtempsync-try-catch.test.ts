import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireMkdtempSyncTryCatchRule } from "./require-mkdtempsync-try-catch";

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

describe("require-mkdtempsync-try-catch", () => {
  it("valid: fs.mkdtempSync inside try block passes (CommonJS)", () => {
    cjsRuleTester.run("require-mkdtempsync-try-catch", requireMkdtempSyncTryCatchRule, {
      valid: [
        `const fs = require("fs"); try { fs.mkdtempSync(path.join(os.tmpdir(), "prefix-")); } catch (e) {}`,
        `const fs = require("fs"); function f() { try { fs.mkdtempSync(prefix); } catch (e) {} }`,
        `const fs = require("fs"); try { fs["mkdtempSync"](prefix); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("valid: destructured mkdtempSync inside try block passes", () => {
    cjsRuleTester.run("require-mkdtempsync-try-catch", requireMkdtempSyncTryCatchRule, {
      valid: [`const { mkdtempSync } = require("fs"); try { mkdtempSync(prefix); } catch (e) {}`, `const { mkdtempSync } = require("node:fs"); try { mkdtempSync(prefix); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: non-fs receiver names with mkdtempSync are ignored", () => {
    cjsRuleTester.run("require-mkdtempsync-try-catch", requireMkdtempSyncTryCatchRule, {
      valid: [`mockFs.mkdtempSync(prefix);`, `storage.mkdtempSync(prefix);`, `myObj.mkdtempSync(prefix);`, `const fs = require("mock-fs"); fs.mkdtempSync(prefix);`],
      invalid: [],
    });
  });

  it("valid: other fs methods remain out of scope", () => {
    cjsRuleTester.run("require-mkdtempsync-try-catch", requireMkdtempSyncTryCatchRule, {
      valid: [`fs.existsSync(path);`, `fs.mkdirSync(dir);`, `fs.statSync(path);`, `fs.readdirSync(dir);`],
      invalid: [],
    });
  });

  it("invalid: bare fs.mkdtempSync is flagged (CommonJS)", () => {
    cjsRuleTester.run("require-mkdtempsync-try-catch", requireMkdtempSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-"));`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: `path.join(os.tmpdir(), "gh-aw-")` },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-"));\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdtempSync call.\n  throw new Error(\n    "fs.mkdtempSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); const tmpDir = fs.mkdtempSync(prefix);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "prefix" },
            },
          ],
        },
        {
          code: `const fs = require("fs"); let tmpDir; tmpDir = fs.mkdtempSync(prefix);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "prefix" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); let tmpDir; try {\n  tmpDir = fs.mkdtempSync(prefix);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdtempSync call.\n  throw new Error(\n    "fs.mkdtempSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); function setup() { fs.mkdtempSync(prefix); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "prefix" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); function setup() { try {\n  fs.mkdtempSync(prefix);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdtempSync call.\n  throw new Error(\n    "fs.mkdtempSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: destructured mkdtempSync outside try is flagged", () => {
    cjsRuleTester.run("require-mkdtempsync-try-catch", requireMkdtempSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { mkdtempSync } = require("fs"); mkdtempSync(prefix);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "prefix" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { mkdtempSync } = require("fs"); try {\n  mkdtempSync(prefix);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdtempSync call.\n  throw new Error(\n    "fs.mkdtempSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fs.mkdtempSync in async function without try is flagged", () => {
    cjsRuleTester.run("require-mkdtempsync-try-catch", requireMkdtempSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const fs = require("fs"); async function run() { let tempDir; tempDir = fs.mkdtempSync(prefix); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "prefix" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); async function run() { let tempDir; try {\n  tempDir = fs.mkdtempSync(prefix);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdtempSync call.\n  throw new Error(\n    "fs.mkdtempSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fs.mkdtempSync inside try/finally without catch is flagged", () => {
    cjsRuleTester.run("require-mkdtempsync-try-catch", requireMkdtempSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const fs = require("fs"); try { fs.mkdtempSync(prefix); } finally { cleanup(); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "prefix" },
              suggestions: 1,
            },
          ],
        },
      ],
    });
  });

  it("valid: fs.mkdtempSync inside try block passes (ESM)", () => {
    esmRuleTester.run("require-mkdtempsync-try-catch", requireMkdtempSyncTryCatchRule, {
      valid: [`import * as fs from "fs"; try { fs.mkdtempSync(prefix); } catch (e) {}`],
      invalid: [],
    });
  });

  it("invalid: bare fs.mkdtempSync is flagged (ESM)", () => {
    esmRuleTester.run("require-mkdtempsync-try-catch", requireMkdtempSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `import * as fs from "fs"; fs.mkdtempSync(prefix);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "prefix" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `import * as fs from "fs"; try {\n  fs.mkdtempSync(prefix);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdtempSync call.\n  throw new Error(\n    "fs.mkdtempSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `import { mkdtempSync } from "fs"; mkdtempSync(prefix);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "prefix" },
              suggestions: 1,
            },
          ],
        },
      ],
    });
  });
});
