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
        // Command variable (identifier) — no definition in scope, cannot resolve, not flagged
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
        // Variable holds a static string — safe, must not be flagged
        { code: `const cmd = "git"; exec.exec(cmd, [branch]);` },
        // Variable holds a static template literal — safe
        { code: "const cmd = `git`; exec.exec(cmd, [branch]);" },
        // Reassigned variable — skipped to avoid false positives
        { code: `let cmd = "git"; cmd = "other"; exec.exec(cmd, [branch]);` },
        // Dynamic initializer with reassignment — must still be skipped
        { code: "let cmd = `git checkout ${branch}`; cmd = 'git'; exec.exec(cmd, []);" },
        // Parameter as command — skipped (def.type === "Parameter")
        { code: `(function(cmd) { exec.exec(cmd, []); })("git");` },
        // Cross-function binding is intentionally out of scope
        { code: "function outer(branch) { const cmd = `git checkout ${branch}`; function inner() { exec.exec(cmd, []); } }" },
        // Static template literal with a chained string method — still static, safe
        { code: "exec.exec(`git`.trim(), [branch]);" },
        // Fully static concatenation with a chained string method — safe
        { code: `exec.exec(("git" + " checkout").toLowerCase(), [branch]);` },
        // Static replace with static arguments — safe
        { code: `exec.exec("git-checkout".replace("-", " "), [branch]);` },
        // Static replace with a fully static replacer callback — safe
        { code: `exec.exec("git-checkout".replace("-", () => " "), [branch]);` },
        // A write-once interpolation resolving to a literal is safe
        { code: `function run() { const branch = "main"; const ref = branch; exec.exec(\`git checkout \${ref}\`, []); }` },
        // A digits-only sanitized interpolation is safe
        { code: `function run(port) { const safePort = String(port).replace(/[^0-9]/g, ""); exec.exec(\`netstat | grep :\${safePort}\`, []); }` },
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
        // Variable holds an interpolated template literal — must be flagged (indirection)
        {
          code: "const cmd = `git checkout ${branch}`; exec.exec(cmd, []);",
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "exec" } }],
        },
        // Same-function indirection should still be flagged with function-boundary resolution
        {
          code: "function run(branch) { const cmd = `git checkout ${branch}`; exec.exec(cmd, []); }",
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "exec" } }],
        },
        // Nested block in same function should still resolve and be flagged
        {
          code: "function run(branch) { const cmd = `git checkout ${branch}`; if (ok) { exec.exec(cmd, []); } }",
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "exec" } }],
        },
        // Variable holds dynamic string concatenation — must be flagged (indirection)
        {
          code: `const cmd = "git checkout " + branchName; exec.exec(cmd, []);`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "dynamic string concatenation", method: "exec" } }],
        },
        // Chained .trim() must not defeat the check
        {
          code: "exec.exec(`git checkout ${branch}`.trim(), []);",
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "exec" } }],
        },
        // Chained .toLowerCase() must not defeat the check
        {
          code: "exec.exec(`git log --author=${author}`.toLowerCase(), []);",
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "exec" } }],
        },
        // Chained string methods on a dynamic concatenation are also flagged
        {
          code: `exec.exec(("git checkout " + branchName).trim(), []);`,
          errors: [{ messageId: "interpolatedCommand", data: { kind: "dynamic string concatenation", method: "exec" } }],
        },
        // Chained string method on a variable holding a dynamic command
        {
          code: "function run(branch) { const cmd = `git checkout ${branch}`; exec.exec(cmd.trim(), []); }",
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "exec" } }],
        },
        // Dynamic replacement argument on a static receiver is also flagged
        {
          code: 'exec.exec("git checkout PLACEHOLDER".replace("PLACEHOLDER", `${branch}-x`), []);',
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "exec" } }],
        },
        // Replacer callback returning a dynamic template literal is also flagged
        {
          code: 'exec.exec("git checkout PLACEHOLDER".replace("PLACEHOLDER", () => `${branch}-x`), []);',
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "exec" } }],
        },
        // Replacer callback with a block body / return statement is also flagged
        {
          code: 'exec.exec("git checkout PLACEHOLDER".replaceAll("PLACEHOLDER", function() { return "x-" + branch; }), []);',
          errors: [{ messageId: "interpolatedCommand", data: { kind: "dynamic string concatenation", method: "exec" } }],
        },
        // Chained aliases are also flagged when they resolve to a dynamic command
        {
          code: "function run(branch) { const dynamic = `git checkout ${branch}`; const cmd = dynamic; exec.exec(cmd, []); }",
          errors: [{ messageId: "interpolatedCommand", data: { kind: "interpolated template literal", method: "exec" } }],
        },
      ],
    });
  });
});
