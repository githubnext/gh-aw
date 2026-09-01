import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireUnlinkSyncRmdirSyncTryCatchRule } from "./require-unlinksync-rmdirsync-try-catch";

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

describe("require-unlinksync-rmdirsync-try-catch", () => {
  it("valid: fs.unlinkSync inside try block passes (CommonJS)", () => {
    cjsRuleTester.run("require-unlinksync-rmdirsync-try-catch", requireUnlinkSyncRmdirSyncTryCatchRule, {
      valid: [
        `const fs = require("fs"); try { fs.unlinkSync(testFile); } catch (e) {}`,
        `const fs = require("fs"); try { fs.rmdirSync(tempDir); } catch (e) {}`,
        `const fs = require("fs"); function f() { try { fs.unlinkSync(testFile); } catch (e) {} }`,
        `const fs = require("fs"); try { fs["unlinkSync"](testFile); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("valid: destructured unlinkSync/rmdirSync inside try block passes", () => {
    cjsRuleTester.run("require-unlinksync-rmdirsync-try-catch", requireUnlinkSyncRmdirSyncTryCatchRule, {
      valid: [`const { unlinkSync } = require("fs"); try { unlinkSync(testFile); } catch (e) {}`, `const { rmdirSync } = require("node:fs"); try { rmdirSync(dir); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: non-fs receiver names are ignored", () => {
    cjsRuleTester.run("require-unlinksync-rmdirsync-try-catch", requireUnlinkSyncRmdirSyncTryCatchRule, {
      valid: [`mockFs.unlinkSync(testFile);`, `storage.rmdirSync(dir);`, `const fs = require("mock-fs"); fs.unlinkSync(testFile);`],
      invalid: [],
    });
  });

  it("valid: other fs methods remain out of scope", () => {
    cjsRuleTester.run("require-unlinksync-rmdirsync-try-catch", requireUnlinkSyncRmdirSyncTryCatchRule, {
      valid: [`fs.existsSync(path);`, `fs.rmSync(path);`, `fs.statSync(path);`, `fs.readdirSync(dir);`],
      invalid: [],
    });
  });

  it("invalid: bare fs.unlinkSync and fs.rmdirSync are flagged (CommonJS)", () => {
    cjsRuleTester.run("require-unlinksync-rmdirsync-try-catch", requireUnlinkSyncRmdirSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `fs.unlinkSync(testFile);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "unlinkSync", arg: "testFile" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  fs.unlinkSync(testFile);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.unlinkSync call.\n  throw new Error(\n    "fs.unlinkSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `fs.rmdirSync(tempDir);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "rmdirSync", arg: "tempDir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  fs.rmdirSync(tempDir);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.rmdirSync call.\n  throw new Error(\n    "fs.rmdirSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const fs = require("fs"); fs.unlinkSync(testFile); fs.rmdirSync(tempDir);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "unlinkSync", arg: "testFile" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); try {\n  fs.unlinkSync(testFile);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.unlinkSync call.\n  throw new Error(\n    "fs.unlinkSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} fs.rmdirSync(tempDir);`,
                },
              ],
            },
            {
              messageId: "requireTryCatch",
              data: { method: "rmdirSync", arg: "tempDir" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const fs = require("fs"); fs.unlinkSync(testFile); try {\n  fs.rmdirSync(tempDir);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.rmdirSync call.\n  throw new Error(\n    "fs.rmdirSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: destructured unlinkSync outside try is flagged", () => {
    cjsRuleTester.run("require-unlinksync-rmdirsync-try-catch", requireUnlinkSyncRmdirSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { unlinkSync } = require("fs"); unlinkSync(testFile);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "unlinkSync", arg: "testFile" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `const { unlinkSync } = require("fs"); try {\n  unlinkSync(testFile);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.unlinkSync call.\n  throw new Error(\n    "fs.unlinkSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fs.unlinkSync inside try/finally without catch is flagged", () => {
    cjsRuleTester.run("require-unlinksync-rmdirsync-try-catch", requireUnlinkSyncRmdirSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const fs = require("fs"); try { fs.unlinkSync(testFile); } finally { cleanup(); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "unlinkSync", arg: "testFile" },
              suggestions: 1,
            },
          ],
        },
      ],
    });
  });

  it("valid: fs.unlinkSync inside try block passes (ESM)", () => {
    esmRuleTester.run("require-unlinksync-rmdirsync-try-catch", requireUnlinkSyncRmdirSyncTryCatchRule, {
      valid: [`import * as fs from "fs"; try { fs.unlinkSync(testFile); } catch (e) {}`],
      invalid: [],
    });
  });

  it("invalid: bare fs.unlinkSync is flagged (ESM)", () => {
    esmRuleTester.run("require-unlinksync-rmdirsync-try-catch", requireUnlinkSyncRmdirSyncTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `import * as fs from "fs"; fs.unlinkSync(testFile);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { method: "unlinkSync", arg: "testFile" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `import * as fs from "fs"; try {\n  fs.unlinkSync(testFile);\n} catch (err) {\n  // TODO: handle filesystem failure for this fs.unlinkSync call.\n  throw new Error(\n    "fs.unlinkSync failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
