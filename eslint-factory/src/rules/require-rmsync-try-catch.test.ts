import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireRmSyncTryCatchRule } from "./require-rmsync-try-catch";

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

describe("require-rmsync-try-catch", () => {
  it("valid: fs.rmSync inside try block passes (CommonJS)", () => {
    cjsRuleTester.run("require-rmsync-try-catch", requireRmSyncTryCatchRule, {
      valid: [
        `const fs = require("fs"); try { fs.rmSync(dir, { recursive: true }); } catch (e) {}`,
        `const fs = require("fs"); try { fs.rmSync(dir); } catch (e) {}`,
        `const fs = require("fs"); function f() { try { fs.rmSync(dir, { recursive: true }); } catch (e) {} }`,
        `const fs = require("fs"); try { fs["rmSync"](dir, { recursive: true }); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("valid: destructured rmSync inside try block passes", () => {
    cjsRuleTester.run("require-rmsync-try-catch", requireRmSyncTryCatchRule, {
      valid: [`const { rmSync } = require("fs"); try { rmSync(dir, { recursive: true }); } catch (e) {}`, `const { rmSync } = require("node:fs"); try { rmSync(dir); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: non-fs receiver names with rmSync are ignored", () => {
    cjsRuleTester.run("require-rmsync-try-catch", requireRmSyncTryCatchRule, {
      valid: [`mockFs.rmSync(dir, { recursive: true });`, `storage.rmSync(dir);`, `myObj.rmSync(path);`, `const fs = require("mock-fs"); fs.rmSync(dir, { recursive: true });`],
      invalid: [],
    });
  });

  it("valid: other fs methods remain out of scope", () => {
    cjsRuleTester.run("require-rmsync-try-catch", requireRmSyncTryCatchRule, {
      valid: [`fs.existsSync(path);`, `fs.unlinkSync(path);`, `fs.statSync(path);`, `fs.readdirSync(dir);`],
      invalid: [],
    });
  });

  it("invalid: bare fs.rmSync is flagged (CommonJS)", () => {
    cjsRuleTester.run("require-rmsync-try-catch", requireRmSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `fs.rmSync(dir, { recursive: true });`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  fs.rmSync(dir, { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.rmSync call.\n  throw new Error(\n    "fs.rmSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); fs.rmSync(dir, { recursive: true });`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); try {\n  fs.rmSync(dir, { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.rmSync call.\n  throw new Error(\n    "fs.rmSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); fs.rmSync(path.join(base, "subdir"), { recursive: true });`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: `path.join(base, "subdir")` },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); try {\n  fs.rmSync(path.join(base, "subdir"), { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.rmSync call.\n  throw new Error(\n    "fs.rmSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); function setup() { fs.rmSync(outputDir, { recursive: true }); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "outputDir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); function setup() { try {\n  fs.rmSync(outputDir, { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.rmSync call.\n  throw new Error(\n    "fs.rmSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: destructured rmSync outside try is flagged", () => {
    cjsRuleTester.run("require-rmsync-try-catch", requireRmSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { rmSync } = require("fs"); rmSync(dir, { recursive: true });`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { rmSync } = require("fs"); try {\n  rmSync(dir, { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.rmSync call.\n  throw new Error(\n    "fs.rmSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const { rmSync } = require("node:fs"); rmSync(dir);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { rmSync } = require("node:fs"); try {\n  rmSync(dir);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.rmSync call.\n  throw new Error(\n    "fs.rmSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fs.rmSync in async function without try is flagged", () => {
    cjsRuleTester.run("require-rmsync-try-catch", requireRmSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const fs = require("fs"); async function run() { fs.rmSync(tmpDir, { recursive: true }); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "tmpDir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); async function run() { try {\n  fs.rmSync(tmpDir, { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.rmSync call.\n  throw new Error(\n    "fs.rmSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fs.rmSync inside try/finally without catch is flagged", () => {
    cjsRuleTester.run("require-rmsync-try-catch", requireRmSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const fs = require("fs"); try { fs.rmSync(dir, { recursive: true }); } finally { cleanup(); }`,
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

  it("valid: fs.rmSync inside try block passes (ESM)", () => {
    esmRuleTester.run("require-rmsync-try-catch", requireRmSyncTryCatchRule, {
      valid: [`import * as fs from "fs"; try { fs.rmSync(dir, { recursive: true }); } catch (e) {}`],
      invalid: [],
    });
  });

  it("invalid: bare fs.rmSync is flagged (ESM)", () => {
    esmRuleTester.run("require-rmsync-try-catch", requireRmSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `import * as fs from "fs"; fs.rmSync(dir, { recursive: true });`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "dir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `import * as fs from "fs"; try {\n  fs.rmSync(dir, { recursive: true });\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.rmSync call.\n  throw new Error(\n    "fs.rmSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `import { rmSync } from "fs"; rmSync(dir, { recursive: true });`,
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
