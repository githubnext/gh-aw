import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireMkdirSyncTryCatchRule } from "./require-mkdirsync-try-catch";

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

describe("require-mkdirsync-try-catch", () => {
  it("valid: fs.mkdirSync inside try block passes (CommonJS)", () => {
    cjsRuleTester.run("require-mkdirsync-try-catch", requireMkdirSyncTryCatchRule, {
      valid: [
        `const fs = require("fs"); try { fs.mkdirSync(dir, { recursive: true }); } catch (e) {}`,
        `const fs = require("fs"); try { fs.mkdirSync(dir); } catch (e) {}`,
        `const fs = require("fs"); function f() { try { fs.mkdirSync(dir, { recursive: true }); } catch (e) {} }`,
        `const fs = require("fs"); try { fs["mkdirSync"](dir, { recursive: true }); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("valid: destructured mkdirSync inside try block passes", () => {
    cjsRuleTester.run("require-mkdirsync-try-catch", requireMkdirSyncTryCatchRule, {
      valid: [`const { mkdirSync } = require("fs"); try { mkdirSync(dir, { recursive: true }); } catch (e) {}`, `const { mkdirSync } = require("node:fs"); try { mkdirSync(dir); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: non-fs objects with mkdirSync name are ignored", () => {
    cjsRuleTester.run("require-mkdirsync-try-catch", requireMkdirSyncTryCatchRule, {
      valid: [`mockFs.mkdirSync(dir, { recursive: true });`, `storage.mkdirSync(dir);`, `myObj.mkdirSync(path);`, `const fs = require("mock-fs"); fs.mkdirSync(dir, { recursive: true });`],
      invalid: [],
    });
  });

  it("valid: other fs methods remain out of scope", () => {
    cjsRuleTester.run("require-mkdirsync-try-catch", requireMkdirSyncTryCatchRule, {
      valid: [`fs.existsSync(path);`, `fs.unlinkSync(path);`, `fs.statSync(path);`, `fs.readdirSync(dir);`],
      invalid: [],
    });
  });

  it("invalid: bare fs.mkdirSync is flagged (CommonJS)", () => {
    cjsRuleTester.run("require-mkdirsync-try-catch", requireMkdirSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const fs = require("fs"); fs.mkdirSync(dir, { recursive: true });`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); try {\n  fs.mkdirSync(dir, { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdirSync call.\n  throw new Error(\n    "fs.mkdirSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); fs.mkdirSync(path.join(base, "subdir"), { recursive: true });`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: `path.join(base, "subdir")` },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); try {\n  fs.mkdirSync(path.join(base, "subdir"), { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdirSync call.\n  throw new Error(\n    "fs.mkdirSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); function setup() { fs.mkdirSync(outputDir, { recursive: true }); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "outputDir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); function setup() { try {\n  fs.mkdirSync(outputDir, { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdirSync call.\n  throw new Error(\n    "fs.mkdirSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: destructured mkdirSync outside try is flagged", () => {
    cjsRuleTester.run("require-mkdirsync-try-catch", requireMkdirSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { mkdirSync } = require("fs"); mkdirSync(dir, { recursive: true });`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { mkdirSync } = require("fs"); try {\n  mkdirSync(dir, { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdirSync call.\n  throw new Error(\n    "fs.mkdirSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const { mkdirSync } = require("node:fs"); mkdirSync(dir);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { mkdirSync } = require("node:fs"); try {\n  mkdirSync(dir);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdirSync call.\n  throw new Error(\n    "fs.mkdirSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fs.mkdirSync in async function without try is flagged", () => {
    cjsRuleTester.run("require-mkdirsync-try-catch", requireMkdirSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const fs = require("fs"); async function run() { fs.mkdirSync(tmpDir, { recursive: true }); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "tmpDir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); async function run() { try {\n  fs.mkdirSync(tmpDir, { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdirSync call.\n  throw new Error(\n    "fs.mkdirSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fs.mkdirSync inside try/finally without catch is flagged", () => {
    cjsRuleTester.run("require-mkdirsync-try-catch", requireMkdirSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const fs = require("fs"); try { fs.mkdirSync(dir, { recursive: true }); } finally { cleanup(); }`,
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

  it("valid: fs.mkdirSync inside try block passes (ESM)", () => {
    esmRuleTester.run("require-mkdirsync-try-catch", requireMkdirSyncTryCatchRule, {
      valid: [`import * as fs from "fs"; try { fs.mkdirSync(dir, { recursive: true }); } catch (e) {}`],
      invalid: [],
    });
  });

  it("invalid: bare fs.mkdirSync is flagged (ESM)", () => {
    esmRuleTester.run("require-mkdirsync-try-catch", requireMkdirSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `import * as fs from "fs"; fs.mkdirSync(dir, { recursive: true });`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `import * as fs from "fs"; try {\n  fs.mkdirSync(dir, { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.mkdirSync call.\n  throw new Error(\n    "fs.mkdirSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `import { mkdirSync } from "fs"; mkdirSync(dir, { recursive: true });`,
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
