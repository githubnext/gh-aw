import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { preferActionsExecOverChildProcessRule } from "./prefer-actions-exec-over-child-process";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: "latest",
    sourceType: "commonjs",
  },
});

const esmRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: "latest",
    sourceType: "module",
  },
});

describe("prefer-actions-exec-over-child-process", () => {
  it("flags child_process output-capturing calls (CommonJS, destructured)", () => {
    cjsRuleTester.run("prefer-actions-exec-over-child-process", preferActionsExecOverChildProcessRule, {
      valid: [
        // @actions/exec is already used
        { code: `async function f() { await exec.getExecOutput("git", ["status"]); }` },
        { code: `async function f() { await exec.exec("git", ["status"]); }` },
        // spawn / spawnSync are out of scope (no @actions/exec equivalent)
        { code: `const { spawn } = require("child_process"); spawn("node", ["server.js"]);` },
        { code: `const { spawnSync } = require("child_process"); spawnSync("git", ["status"]);` },
        { code: `const cp = require("child_process"); cp.spawn("node", ["server.js"]);` },
        { code: `const cp = require("child_process"); cp.spawnSync("git", ["status"]);` },
        // Same method name from an unrelated module — should not be flagged
        { code: `const { execSync } = require("some-other-lib"); execSync("git status");` },
        { code: `const { exec } = require("./local-exec-helper.cjs"); exec("git status");` },
        // Bare identifier without any require — should not be flagged
        { code: `execSync("git status");` },
      ],
      invalid: [
        {
          code: `const { execSync } = require("child_process"); execSync("git status");`,
          errors: [{ messageId: "preferActionsExec", data: { method: "execSync" } }],
        },
        {
          code: `const { exec } = require("child_process"); exec("git status");`,
          errors: [{ messageId: "preferActionsExec", data: { method: "exec" } }],
        },
        {
          code: `const { execFile } = require("child_process"); execFile("git", ["status"]);`,
          errors: [{ messageId: "preferActionsExec", data: { method: "execFile" } }],
        },
        {
          code: `const { execFileSync } = require("child_process"); execFileSync("git", ["status"]);`,
          errors: [{ messageId: "preferActionsExec", data: { method: "execFileSync" } }],
        },
        {
          code: `const cp = require("child_process"); cp.execSync("git status");`,
          errors: [{ messageId: "preferActionsExec", data: { method: "execSync" } }],
        },
        {
          code: `const cp = require("node:child_process"); cp.execFileSync("git", ["status"]);`,
          errors: [{ messageId: "preferActionsExec", data: { method: "execFileSync" } }],
        },
        {
          code: `require("child_process").execSync("git status");`,
          errors: [{ messageId: "preferActionsExec", data: { method: "execSync" } }],
        },
        {
          code: `const run = require("child_process").execSync; run("git status");`,
          errors: [{ messageId: "preferActionsExec", data: { method: "execSync" } }],
        },
        {
          code: `function f() { const { execSync } = require("child_process"); execSync("git status"); }`,
          errors: [{ messageId: "preferActionsExec", data: { method: "execSync" } }],
        },
      ],
    });
  });

  it("flags child_process output-capturing calls (ES module)", () => {
    esmRuleTester.run("prefer-actions-exec-over-child-process", preferActionsExecOverChildProcessRule, {
      valid: [{ code: `import { spawn } from "child_process"; spawn("node", ["server.js"]);` }, { code: `import { spawnSync } from "node:child_process"; spawnSync("git", ["status"]);` }],
      invalid: [
        {
          code: `import { execSync } from "child_process"; execSync("git status");`,
          errors: [{ messageId: "preferActionsExec", data: { method: "execSync" } }],
        },
        {
          code: `import { execFileSync } from "node:child_process"; execFileSync("git", ["status"]);`,
          errors: [{ messageId: "preferActionsExec", data: { method: "execFileSync" } }],
        },
      ],
    });
  });
});
