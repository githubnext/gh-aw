import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireFsCloseSyncRule } from "./require-fs-close-sync";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-fs-close-sync", () => {
  it("valid: closed descriptors for declarator, assignment, and module scope", () => {
    cjsRuleTester.run("require-fs-close-sync", requireFsCloseSyncRule, {
      valid: [
        `function run(path) { const fd = fs.openSync(path, "w"); fs.closeSync(fd); }`,
        `function run(path) { let fd; fd = fs.openSync(path, "w"); if (fd >= 0) fs.closeSync(fd); }`,
        `const fd = fs.openSync(path, "w"); fs.closeSync(fd);`,
      ],
      invalid: [],
    });
  });

  it("valid: out-of-scope forms (destructuring and inline openSync argument) are ignored", () => {
    cjsRuleTester.run("require-fs-close-sync", requireFsCloseSyncRule, {
      valid: [`function run(path) { const { fd } = record; const opened = fs.openSync(path, "w"); fs.closeSync(opened); fs.closeSync(fd); }`, `function run(path) { consume(fs.openSync(path, "w")); }`],
      invalid: [],
    });
  });

  it("invalid: unclosed descriptors are reported", () => {
    cjsRuleTester.run("require-fs-close-sync", requireFsCloseSyncRule, {
      valid: [],
      invalid: [
        {
          code: `function run(path) { const fd = fs.openSync(path, "w"); return fd; }`,
          errors: [{ messageId: "missingCloseSync", data: { fdName: "fd" } }],
        },
        {
          code: `function run(path) { let outputFd; outputFd = fs.openSync(path, "w"); core.info("opened"); }`,
          errors: [{ messageId: "missingCloseSync", data: { fdName: "outputFd" } }],
        },
      ],
    });
  });

  it("invalid: closeSync in another function does not satisfy the requirement", () => {
    cjsRuleTester.run("require-fs-close-sync", requireFsCloseSyncRule, {
      valid: [],
      invalid: [
        {
          code: `function openIt(path) { let fd; fd = fs.openSync(path, "w"); }\nfunction closeIt(fd) { fs.closeSync(fd); }`,
          errors: [{ messageId: "missingCloseSync", data: { fdName: "fd" } }],
        },
      ],
    });
  });
});
