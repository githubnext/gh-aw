import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireGetExecOutputExitCodeCheckRule } from "./require-getexecoutput-exitcode-check";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-getexecoutput-exitcode-check", () => {
  it("uses the correct docs URL", () => {
    expect(requireGetExecOutputExitCodeCheckRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-getexecoutput-exitcode-check");
  });

  it("valid", () => {
    cjsRuleTester.run("require-getexecoutput-exitcode-check", requireGetExecOutputExitCodeCheckRule, {
      valid: [
        // destructuring includes exitCode
        `async function f() { const { stdout, exitCode } = await exec.getExecOutput("git", ["status"], { ignoreReturnCode: true }); if (exitCode !== 0) throw new Error("failed"); }`,
        // only exitCode destructured
        `async function f() { const { exitCode } = await execApi.getExecOutput("git", ["status"], { ignoreReturnCode: true }); }`,
        // identifier binding, .exitCode read later
        `async function f() { const result = await exec.getExecOutput("git", ["status"], { ignoreReturnCode: true }); if (result.exitCode !== 0) throw new Error("failed"); }`,
        // no ignoreReturnCode option at all — default throw-on-failure behavior applies
        `async function f() { const { stdout } = await exec.getExecOutput("git", ["status"]); }`,
        // ignoreReturnCode explicitly false — default throw-on-failure still applies
        `async function f() { const { stdout } = await exec.getExecOutput("git", ["status"], { ignoreReturnCode: false }); }`,
        // rest element could capture exitCode; don't flag
        `async function f() { const { ...rest } = await exec.getExecOutput("git", ["status"], { ignoreReturnCode: true }); console.log(rest.exitCode); }`,
        // spread in options can't be statically ruled out, so not flagged; but exitCode also directly accessed
        `async function f() { const r = (await exec.getExecOutput("git", ["status"], { ...opts, ignoreReturnCode: true })).exitCode; }`,
        // direct member access on the awaited result
        `async function f() { const code = (await exec.getExecOutput("git", ["status"], { ignoreReturnCode: true })).exitCode; }`,
      ],
      invalid: [],
    });
  });

  it("invalid: ignoreReturnCode: true but exitCode never read", () => {
    cjsRuleTester.run("require-getexecoutput-exitcode-check", requireGetExecOutputExitCodeCheckRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() { const { stdout } = await exec.getExecOutput("git", ["status"], { ignoreReturnCode: true }); }`,
          errors: [{ messageId: "missingExitCodeCheck" }],
        },
        {
          code: `async function f() { const { stdout, stderr } = await execApi.getExecOutput("git", ["bundle", "verify", "b"], { ignoreReturnCode: true, silent: true }); }`,
          errors: [{ messageId: "missingExitCodeCheck" }],
        },
        {
          code: `async function f() { const result = await exec.getExecOutput("git", ["status"], { ignoreReturnCode: true }); return result.stdout.trim(); }`,
          errors: [{ messageId: "missingExitCodeCheck" }],
        },
        {
          code: `async function f() { await exec.getExecOutput("git", ["status"], { ignoreReturnCode: true }); }`,
          errors: [{ messageId: "missingExitCodeCheck" }],
        },
        {
          code: `async function f() { return await exec.getExecOutput("git", ["status"], { ignoreReturnCode: true }); }`,
          errors: [{ messageId: "missingExitCodeCheck" }],
        },
      ],
    });
  });
});
