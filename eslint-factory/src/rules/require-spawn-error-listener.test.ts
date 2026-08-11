import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireSpawnErrorListenerRule } from "./require-spawn-error-listener";

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

describe("require-spawn-error-listener", () => {
  it("valid: spawn() result with an 'error' listener passes (CommonJS, destructured)", () => {
    cjsRuleTester.run("require-spawn-error-listener", requireSpawnErrorListenerRule, {
      valid: [
        `const { spawn } = require("child_process"); const child = spawn("ls", []); child.on("error", err => { console.log(err); });`,
        `const { spawn } = require("node:child_process"); const child = spawn("ls", []); child.once("error", err => { console.log(err); });`,
        `const childProcess = require("child_process"); const child = childProcess.spawn("ls", []); child.on("error", err => {});`,
        `const child_process = require("child_process"); const child = child_process.spawn("ls", []); child.on("error", err => {});`,
        `const cp = require("child_process"); const child = cp.spawn("ls", []); child.on("error", err => {});`,
        `function run() { const { spawn } = require("child_process"); const child = spawn("ls", []); child.stdout.on("data", () => {}); child.on("error", err => {}); }`,
        `function spawn() { return { stdout: { on() {} } }; } const child = spawn("ls", []); child.stdout.on("data", () => {});`,
      ],
      invalid: [],
    });
  });

  it("invalid: spawn() result missing an 'error' listener is reported (CommonJS)", () => {
    cjsRuleTester.run("require-spawn-error-listener", requireSpawnErrorListenerRule, {
      valid: [],
      invalid: [
        {
          code: `const { spawn } = require("child_process"); const child = spawn("ls", []); child.stdout.on("data", () => {});`,
          errors: [{ messageId: "missingErrorListener" }],
        },
        {
          code: `const { spawn } = require("child_process"); const child = spawn("ls", []);`,
          errors: [{ messageId: "missingErrorListener" }],
        },
        {
          code: `const childProcess = require("child_process"); const child = childProcess.spawn("ls", []); child.on("exit", () => {});`,
          errors: [{ messageId: "missingErrorListener" }],
        },
        {
          code: `const cp = require("child_process"); const child = cp.spawn("ls", []); child.on("exit", () => {});`,
          errors: [{ messageId: "missingErrorListener" }],
        },
      ],
    });
  });

  it("valid: spawn() with an 'error' listener passes (ESM)", () => {
    esmRuleTester.run("require-spawn-error-listener", requireSpawnErrorListenerRule, {
      valid: [`import { spawn } from "child_process"; const child = spawn("ls", []); child.on("error", err => {});`, `import { spawn } from "cross-spawn"; const child = spawn("ls", []); child.on("exit", () => {});`],
      invalid: [
        {
          code: `import { spawn } from "child_process"; const child = spawn("ls", []);`,
          errors: [{ messageId: "missingErrorListener" }],
        },
      ],
    });
  });

  it("does not flag spawnSync or aliased spawnImpl calls", () => {
    cjsRuleTester.run("require-spawn-error-listener", requireSpawnErrorListenerRule, {
      valid: [`const { spawnSync } = require("child_process"); const result = spawnSync("ls", []);`, `const spawnImpl = spawn; const child = spawnImpl("ls", []);`],
      invalid: [],
    });
  });

  it("does not flag assignment expressions or inline chains (documented scope limit)", () => {
    cjsRuleTester.run("require-spawn-error-listener", requireSpawnErrorListenerRule, {
      valid: [`const { spawn } = require("child_process"); let child; child = spawn("ls", []);`, `const { spawn } = require("child_process"); spawn("ls", []).on("exit", () => {});`],
      invalid: [],
    });
  });

  it("treats nested callback listeners on the same child variable as valid", () => {
    cjsRuleTester.run("require-spawn-error-listener", requireSpawnErrorListenerRule, {
      valid: [`const { spawn } = require("child_process"); const child = spawn("ls", []); setTimeout(() => { child.on("error", () => {}); }, 0);`],
      invalid: [],
    });
  });

  it("reports an error listener attached only in a conditional branch", () => {
    cjsRuleTester.run("require-spawn-error-listener", requireSpawnErrorListenerRule, {
      valid: [`const { spawn } = require("child_process"); if (verbose) { const child = spawn("ls", []); child.on("error", err => { console.log(err); }); }`],
      invalid: [
        {
          code: `const { spawn } = require("child_process"); const child = spawn("ls", []); if (verbose) { child.on("error", err => { console.log(err); }); }`,
          errors: [{ messageId: "missingErrorListener" }],
        },
      ],
    });
  });
});
