import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireFsIoTryCatchRule } from "./require-fs-io-try-catch";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-fs-io-try-catch", () => {
  it("valid: fs.statSync inside try block passes", () => {
    cjsRuleTester.run("require-fs-io-try-catch", requireFsIoTryCatchRule, {
      valid: [
        `try { const s = fs.statSync(path); } catch (e) {}`,
        `try { const entries = fs.readdirSync(dir); } catch (e) {}`,
        `try { fs.copyFileSync(src, dest); } catch (e) {}`,
        `try { fs.unlinkSync(path); } catch (e) {}`,
        `try { fs.renameSync(oldPath, newPath); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("valid: other fs methods not in scope are ignored", () => {
    cjsRuleTester.run("require-fs-io-try-catch", requireFsIoTryCatchRule, {
      valid: [`fs.existsSync(path);`, `fs.readFileSync(path, "utf8");`, `fs.writeFileSync(path, data);`, `mockFs.statSync(path);`, `storage.readdirSync(dir);`],
      invalid: [],
    });
  });

  it("invalid: fs.statSync outside try/catch is flagged", () => {
    cjsRuleTester.run("require-fs-io-try-catch", requireFsIoTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `fs.statSync(path);`,
          errors: [{ messageId: "requireTryCatch", data: { method: "statSync", arg: "path" } }],
        },
      ],
    });
  });

  it("invalid: fs.readdirSync outside try/catch is flagged", () => {
    cjsRuleTester.run("require-fs-io-try-catch", requireFsIoTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const entries = fs.readdirSync(dir);`,
          errors: [{ messageId: "requireTryCatch", data: { method: "readdirSync", arg: "dir" } }],
        },
      ],
    });
  });

  it("invalid: fs.copyFileSync outside try/catch is flagged", () => {
    cjsRuleTester.run("require-fs-io-try-catch", requireFsIoTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `fs.copyFileSync(src, dest);`,
          errors: [{ messageId: "requireTryCatch", data: { method: "copyFileSync", arg: "src" } }],
        },
      ],
    });
  });

  it("invalid: fs.unlinkSync outside try/catch is flagged", () => {
    cjsRuleTester.run("require-fs-io-try-catch", requireFsIoTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `fs.unlinkSync(outputFile);`,
          errors: [{ messageId: "requireTryCatch", data: { method: "unlinkSync", arg: "outputFile" } }],
        },
      ],
    });
  });

  it("invalid: fs.renameSync outside try/catch is flagged", () => {
    cjsRuleTester.run("require-fs-io-try-catch", requireFsIoTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `fs.renameSync(tmpPath, finalPath);`,
          errors: [{ messageId: "requireTryCatch", data: { method: "renameSync", arg: "tmpPath" } }],
        },
      ],
    });
  });

  it("valid: node:fs destructured inside try block passes", () => {
    cjsRuleTester.run("require-fs-io-try-catch", requireFsIoTryCatchRule, {
      valid: [`const { statSync } = require("node:fs"); try { statSync(path); } catch (e) {}`, `const { readdirSync } = require("fs"); try { readdirSync(dir); } catch (e) {}`],
      invalid: [],
    });
  });

  it("invalid: destructured fs methods outside try/catch are flagged", () => {
    cjsRuleTester.run("require-fs-io-try-catch", requireFsIoTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const { statSync } = require("node:fs"); statSync(path);`,
          errors: [{ messageId: "requireTryCatch" }],
        },
      ],
    });
  });
});
