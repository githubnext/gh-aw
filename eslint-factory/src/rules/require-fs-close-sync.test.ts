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

  it("valid: try/finally close and alias/property close forms", () => {
    cjsRuleTester.run("require-fs-close-sync", requireFsCloseSyncRule, {
      valid: [
        `function run(path) { const fd = fs.openSync(path, "w"); try { write(fd); } finally { fs.closeSync(fd); } }`,
        `function run(path) { const fd = fs.openSync(path, "w"); const handle = { fd }; fs.closeSync(handle.fd); }`,
        `function run(path) { const fd = fs.openSync(path, "w"); const alias = fd; fs.closeSync(alias); }`,
      ],
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
          code: `function run(path) {\n  let outputFd;\n  outputFd = fs.openSync(path, "w");\n  core.info("opened");\n}`,
          errors: [{ messageId: "missingCloseSync", data: { fdName: "outputFd" }, line: 3, column: 14 }],
        },
      ],
    });
  });

  it("invalid: closeSync in a nested function does not satisfy the requirement", () => {
    cjsRuleTester.run("require-fs-close-sync", requireFsCloseSyncRule, {
      valid: [],
      invalid: [
        {
          code: `function outer(path) { const fd = fs.openSync(path, "w"); function inner() { fs.closeSync(fd); } inner(); }`,
          errors: [{ messageId: "missingCloseSync", data: { fdName: "fd" } }],
        },
        {
          code: `function outer(path) { const fd = fs.openSync(path, "w"); const cleanup = () => { fs.closeSync(fd); }; cleanup(); }`,
          errors: [{ messageId: "missingCloseSync", data: { fdName: "fd" } }],
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
