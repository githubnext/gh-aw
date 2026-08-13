import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireSyncExecTimeoutRule } from "./require-sync-exec-timeout";

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

describe("require-sync-exec-timeout", () => {
  it("valid: timeout option present (CommonJS, destructured)", () => {
    cjsRuleTester.run("require-sync-exec-timeout", requireSyncExecTimeoutRule, {
      valid: [
        `const { execSync } = require("child_process"); execSync("git status", { timeout: 5000 });`,
        `const { execFileSync } = require("child_process"); execFileSync("git", ["status"], { timeout: 5000 });`,
        `const { execFileSync } = require("child_process"); execFileSync("git", { timeout: 5000 });`,
        `const { spawnSync } = require("child_process"); spawnSync("git", ["status"], { timeout: 5000, encoding: "utf8" });`,
        `const { spawnSync } = require("child_process"); spawnSync("git", { timeout: 5000, encoding: "utf8" });`,
        `const { execSync } = require("node:child_process"); execSync("git status", { timeout: 5000 });`,
      ],
      invalid: [],
    });
  });

  it("valid: timeout option present (namespace binding)", () => {
    cjsRuleTester.run("require-sync-exec-timeout", requireSyncExecTimeoutRule, {
      valid: [`const cp = require("child_process"); cp.execSync("git status", { timeout: 5000 });`, `const cp = require("child_process"); cp.spawnSync("git", ["status"], { timeout: 5000 });`],
      invalid: [],
    });
  });

  it("valid: options passed via identifier or spread are not statically inspectable", () => {
    cjsRuleTester.run("require-sync-exec-timeout", requireSyncExecTimeoutRule, {
      valid: [`const { execSync } = require("child_process"); const opts = { timeout: 5000 }; execSync("git status", opts);`, `const { execSync } = require("child_process"); const base = {}; execSync("git status", { ...base });`],
      invalid: [],
    });
  });

  it("valid: calls from non-child_process modules or unbound identifiers are ignored", () => {
    cjsRuleTester.run("require-sync-exec-timeout", requireSyncExecTimeoutRule, {
      valid: [`const { execSync } = require("some-other-lib"); execSync("ls");`, `execSync("ls");`],
      invalid: [],
    });
  });

  it("invalid: execSync without timeout option (CommonJS, destructured)", () => {
    cjsRuleTester.run("require-sync-exec-timeout", requireSyncExecTimeoutRule, {
      valid: [],
      invalid: [
        {
          code: `const { execSync } = require("child_process"); execSync("git status");`,
          errors: [{ messageId: "requireTimeout" }],
        },
        {
          code: `const { execSync } = require("child_process"); execSync("git status", { encoding: "utf8" });`,
          errors: [{ messageId: "requireTimeout" }],
        },
        {
          code: `const { execSync } = require("child_process"); execSync("git status", { timeout: undefined });`,
          errors: [{ messageId: "requireTimeout" }],
        },
      ],
    });
  });

  it("invalid: execFileSync and spawnSync without timeout option", () => {
    cjsRuleTester.run("require-sync-exec-timeout", requireSyncExecTimeoutRule, {
      valid: [],
      invalid: [
        {
          code: `const { execFileSync } = require("child_process"); execFileSync("git", ["status"]);`,
          errors: [{ messageId: "requireTimeout" }],
        },
        {
          code: `const { spawnSync } = require("child_process"); spawnSync("git", ["status"], { encoding: "utf8" });`,
          errors: [{ messageId: "requireTimeout" }],
        },
        {
          code: `const { spawnSync } = require("child_process"); spawnSync("git", { encoding: "utf8" });`,
          errors: [{ messageId: "requireTimeout" }],
        },
        {
          code: `const cp = require("child_process"); cp.execFileSync("git", ["status"]);`,
          errors: [{ messageId: "requireTimeout" }],
        },
        {
          code: `const { execFileSync } = require("child_process"); execFileSync("git", { encoding: "utf8" });`,
          errors: [{ messageId: "requireTimeout" }],
        },
      ],
    });
  });

  it("invalid: execSync without timeout option (ES module)", () => {
    esmRuleTester.run("require-sync-exec-timeout", requireSyncExecTimeoutRule, {
      valid: [],
      invalid: [
        {
          code: `import { execSync } from "child_process"; execSync("git status");`,
          errors: [{ messageId: "requireTimeout" }],
        },
      ],
    });
  });
});
