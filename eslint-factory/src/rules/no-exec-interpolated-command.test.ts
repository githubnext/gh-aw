import { RuleTester } from "@typescript-eslint/rule-tester";
import { noExecInterpolatedCommandRule } from "./no-exec-interpolated-command";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: "latest",
    sourceType: "commonjs",
  },
});

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
    // Not exec.exec — unrelated call
    { code: `someOther.exec(\`git checkout \${branch}\`);` },
    // Bare exec() call — not a member expression
    { code: `exec(\`git checkout \${branch}\`);` },
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
      errors: [{ messageId: "interpolatedCommand" }],
    },
    // getExecOutput with interpolated command
    {
      code: "exec.getExecOutput(`git rev-parse --verify ${ref}`, [], opts);",
      errors: [{ messageId: "interpolatedCommand" }],
    },
    // Template with only a single interpolation (whole command dynamic)
    {
      code: "exec.exec(`git am --3way ${patchPath}`, [], opts);",
      errors: [{ messageId: "interpolatedCommand" }],
    },
  ],
});
