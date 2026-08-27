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
        `const fs = require("fs"); try { fs.realpathSync(candidate); } catch (e) {}`,
        `const fs = require("fs"); function f() { try { fs.realpathSync(candidate); } catch (e) {} }`,
        `const fs = require("fs"); try { fs["realpathSync"](candidate); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("valid: destructured realpathSync inside try block passes", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [`const { realpathSync } = require("fs"); try { realpathSync(candidate); } catch (e) {}`, `const { realpathSync } = require("node:fs"); try { realpathSync(candidate); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: non-fs receiver names with realpathSync are ignored", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [`mockFs.realpathSync(candidate);`, `storage.realpathSync(candidate);`, `myObj.realpathSync(candidate);`, `const fs = require("mock-fs"); fs.realpathSync(candidate);`],
      invalid: [],
    });
  });

  it("valid: other fs methods remain out of scope", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [`fs.existsSync(path);`, `fs.mkdirSync(dir);`, `fs.statSync(path);`, `fs.readdirSync(dir);`, `fs.mkdtempSync(prefix);`],
      invalid: [],
    });
  });

  it("invalid: bare fs.realpathSync is flagged (CommonJS)", () => {
    cjsRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `fs.realpathSync(path.join(root, filename));`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: `path.join(root, filename)` },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  fs.realpathSync(path.join(root, filename));\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); const resolved = fs.realpathSync(candidate);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "candidate" },
            },
          ],
        },
        {
          code: `const fs = require("fs"); let resolved; resolved = fs.realpathSync(candidate);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "candidate" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); let resolved; try {\n  resolved = fs.realpathSync(candidate);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); function setup() { fs.realpathSync(candidate); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "candidate" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); function setup() { try {\n  fs.realpathSync(candidate);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
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
          code: `const { realpathSync } = require("fs"); realpathSync(candidate);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "candidate" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { realpathSync } = require("fs"); try {\n  realpathSync(candidate);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
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
          code: `const fs = require("fs"); async function run() { let resolved; resolved = fs.realpathSync(candidate); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "candidate" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); async function run() { let resolved; try {\n  resolved = fs.realpathSync(candidate);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
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
          code: `const fs = require("fs"); try { fs.realpathSync(candidate); } finally { cleanup(); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "candidate" },
              suggestions: 1,
            },
          ],
        },
      ],
    });
  });

  it("valid: fs.realpathSync inside try block passes (ESM)", () => {
    esmRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [`import * as fs from "fs"; try { fs.realpathSync(candidate); } catch (e) {}`],
      invalid: [],
    });
  });

  it("invalid: bare fs.realpathSync is flagged (ESM)", () => {
    esmRuleTester.run("require-realpathsync-try-catch", requireRealpathSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `import * as fs from "fs"; fs.realpathSync(candidate);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "candidate" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `import * as fs from "fs"; try {\n  fs.realpathSync(candidate);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.realpathSync call.\n  throw new Error(\n    "fs.realpathSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `import { realpathSync } from "fs"; realpathSync(candidate);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "candidate" },
              suggestions: 1,
            },
          ],
        },
      ],
    });
  });
});
