import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noExecInterpolatedCommandRule } from "./no-exec-interpolated-command";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: "latest",
    sourceType: "commonjs",
  },
});

describe("no-exec-interpolated-command", () => {
  it("accepts static command forms and flags dynamic ones", () => {
    ruleTester.run("no-exec-interpolated-command", noExecInterpolatedCommandRule, {
      valid: [
        // Static string — no interpolation, safe
        { code: `exec.exec("git", ["checkout", branch]);` },
        // Static template literal — no expressions, safe
        { code: "exec.exec(`git`, [`checkout`, branch]);" },
        // getExecOutput with static command
        { code: `exec.getExecOutput("git", ["rev-parse", "--abbrev-ref", "HEAD"], opts);` },
        // Command variable (identifier) — not a string literal, out of scope
        { code: `exec.exec(myCommand, [arg1]);` },
        // Single-word static template literal — no interpolation
        { code: "exec.exec(`git`, [branch]);" },
        // Fully static concatenation built from literal leaves
        { code: `exec.exec("tool --retries " + 3, [], opts);` },
        // Fully static string concatenation remains allowed
        { code: `exec.exec("git" + " checkout", [branch]);` },
        // Not exec.exec — unrelated call
        { code: `someOther.exec(\`git checkout \${branch}\`);` },
        // Alias object name is intentionally out of scope
        { code: `execAlias.exec(\`git checkout \${branch}\`);` },
        // Bare exec() call — not a member expression
        { code: `exec(\`git checkout \${branch}\`);` },
        // Spread first argument is intentionally out of scope
        { code: `exec.exec(...args);` },
      ],
      invalid: [
        // Template literal with interpolation as command
        {
          code: "exec.exec(`git checkout ${branch}`, [], opts);",
          errors: [{ messageId: "interpolatedCommand" }],
        },
        // Template literal with multiple interpolations
        {
          code: "exec.exec(`git checkout -B ${branchName} ${baseRef}`, [], opts);",
          errors: [{ messageId: "interpolatedCommand" }],
        },
        // Dynamic string concatenation
        {
          code: `exec.exec("git checkout " + branchName, [], opts);`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "dynamic string concatenation", method: "exec" } }],
        },
        // Multi-segment dynamic string concatenation
        {
          code: `exec.exec("git checkout " + branchName + " " + ref, [], opts);`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "dynamic string concatenation", method: "exec" } }],
        },
        // getExecOutput with interpolated command
        {
          code: "exec.getExecOutput(`git rev-parse --verify ${ref}`, [], opts);",
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "getExecOutput" } }],
        },
        // Template with only a single interpolation (whole command dynamic)
        {
          code: "exec.exec(`git am --3way ${patchPath}`, [], opts);",
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "exec" } }],
        },
      ],
    });
  });
});
