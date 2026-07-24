import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noChildProcessInterpolatedCommandRule } from "./no-child-process-interpolated-command";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: "latest",
    sourceType: "commonjs",
  },
});

describe("no-child-process-interpolated-command", () => {
  it("flags dynamic child_process command strings for shell-evaluated methods", () => {
    ruleTester.run("no-child-process-interpolated-command", noChildProcessInterpolatedCommandRule, {
      valid: [
        { code: `const { execSync } = require("child_process"); execSync("git status");` },
        { code: `const cp = require("child_process"); cp.execSync(\`git status\`);` },
        { code: `const cp = require("child_process"); cp.spawn("git", ["status"]);` },
        { code: `const { spawn } = require("child_process"); spawn(\`git \${branch}\`, ["status"]);` },
        { code: `const { spawnSync } = require("child_process"); const cmd = \`git checkout \${branch}\`; spawnSync(cmd);` },
        { code: `const { execFileSync } = require("child_process"); execFileSync("git", ["status"], { shell: false });` },
        { code: `const { execSync } = require("child_process"); const cmd = "git status"; execSync(cmd);` },
        { code: `const { execSync } = require("child_process"); let cmd = \`git checkout \${branch}\`; cmd = "git status"; execSync(cmd);` },
        { code: `const { execSync } = require("child_process"); (function(cmd) { execSync(cmd); })("git status");` },
        { code: `exec.exec(\`git checkout \${branch}\`, []);` },
        {
          code: `import { exec } from "node:child_process"; exec("git status");`,
          languageOptions: { sourceType: "module" },
        },
        {
          code: `import { execSync } from "child_process"; import { cmd } from "./cmd"; execSync(cmd);`,
          languageOptions: { sourceType: "module" },
        },
      ],
      invalid: [
        {
          code: `const { execSync } = require("child_process"); execSync(\`git checkout \${branch}\`);`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "execSync" } }],
        },
        {
          code: `function run(x) { const { execSync } = require("child_process"); const cmd = \`git log --author=\${x}\`; execSync(cmd); }`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "execSync" } }],
        },
        {
          code: `function run(x) { const { execSync } = require("child_process"); const cmd = "git " + x; execSync(cmd); }`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "dynamic string concatenation", method: "execSync" } }],
        },
        {
          code: `const cp = require("child_process"); const run = cp.execSync; run("git checkout " + branch);`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "dynamic string concatenation", method: "execSync" } }],
        },
        {
          code: `const cp = require("node:child_process"); cp.exec(\`git checkout \${branch}\`);`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "exec" } }],
        },
        {
          code: `const { spawn } = require("child_process"); spawn(\`git checkout \${branch}\`, { shell: true });`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "spawn" } }],
        },
        {
          code: `const { spawn } = require("child_process"); const opts = { shell: true }; spawn(\`git checkout \${branch}\`, opts);`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "spawn" } }],
        },
        {
          code: `const { spawn } = require("child_process"); spawn(\`git checkout \${branch}\`, { shell: "/bin/bash" });`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "spawn" } }],
        },
        {
          code: `const { spawn } = require("child_process"); const opts = [{ shell: true }]; spawn("git checkout " + branch, ...opts);`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "dynamic string concatenation", method: "spawn" } }],
        },
        {
          code: `const { spawnSync } = require("child_process"); spawnSync("git checkout " + branch, ["--"], { shell: true });`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "dynamic string concatenation", method: "spawnSync" } }],
        },
        {
          code: `const { execFileSync } = require("child_process"); execFileSync(\`git \${branch}\`, ["status"], { shell: true });`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "execFileSync" } }],
        },
        {
          code: `const { execFile } = require("child_process"); execFile("git " + branch, { shell: true });`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "dynamic string concatenation", method: "execFile" } }],
        },
        {
          code: `import { execSync, "exec" as run } from "child_process"; execSync(\`git checkout \${branch}\`); run("git checkout " + branch);`,
          languageOptions: { sourceType: "module" },
          errors: [
            { messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "execSync" } },
            { messageId: "interpolatedCommand", data: { kind: "dynamic string concatenation", method: "exec" } },
          ],
        },
      ],
    });
  });
});
