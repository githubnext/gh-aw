// Uses eslint's RuleTester rather than @typescript-eslint/rule-tester, matching the
// convention of all other rule tests in this package. The rule uses @typescript-eslint/utils
// internally but the standard eslint RuleTester is sufficient for all test scenarios here.
import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noCoreErrorThenSetFailedRule } from "./no-core-error-then-setfailed";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: "latest",
    sourceType: "module",
  },
});

describe("no-core-error-then-setfailed", () => {
  it("valid and invalid cases", () => {
    ruleTester.run("no-core-error-then-setfailed", noCoreErrorThenSetFailedRule, {
      valid: [
        // Only core.setFailed — no preceding core.error
        `core.setFailed("something went wrong");`,
        // Only core.error — no following setFailed
        `core.error("something went wrong");`,
        // core.error followed by something other than setFailed
        `function f() { core.error("msg"); return; }`,
        // core.warning followed by core.setFailed is allowed (different method)
        `core.warning("msg"); core.setFailed("msg");`,
        // core.error without adjacent setFailed (setFailed is non-adjacent)
        `core.error("msg"); doSomething(); core.setFailed("msg");`,
        // Different messages — core.error provides extra context not repeated by setFailed
        `core.error("upload failed: " + filename); core.setFailed("action failed");`,
        // Different template literals — messages differ in prefix
        {
          code: `
            try {
              doSomething();
            } catch (err) {
              core.error(\`Failed: \${err.message}\`);
              core.setFailed(\`ERR: Failed: \${err.message}\`);
            }
          `,
        },
        // core.error with annotation properties — carries extra diagnostic context
        `core.error("msg", { title: "Upload error" }); core.setFailed("msg");`,
        // Different core objects (cross-alias false-positive guard):
        // c1 and c2 are different objects even if both are in CORE_ALIASES
        `const c1 = core; const c2 = coreObj; c1.error("msg"); c2.setFailed("msg");`,
        // Non-core alias is not flagged
        `const c = notCore; c.error("msg"); c.setFailed("msg");`,
      ],
      invalid: [
        // Adjacent core.error then core.setFailed with same literal — has suggestion
        {
          code: `core.error("msg"); core.setFailed("msg");`,
          errors: [{ messageId: "noCoreErrorThenSetFailed", suggestions: [{ messageId: "removeErrorCall", output: ` core.setFailed("msg");` }] }],
        },
        // With an alias (const c = core) and matching messages — has suggestion
        {
          code: `const c = core; c.error("msg"); c.setFailed("msg");`,
          errors: [{ messageId: "noCoreErrorThenSetFailed", suggestions: [{ messageId: "removeErrorCall", output: `const c = core;  c.setFailed("msg");` }] }],
        },
        // Computed property access with matching messages — has suggestion
        {
          code: `core["error"]("msg"); core["setFailed"]("msg");`,
          errors: [{ messageId: "noCoreErrorThenSetFailed", suggestions: [{ messageId: "removeErrorCall", output: ` core["setFailed"]("msg");` }] }],
        },
        // Same template literal in both calls — side-effect-free (identifier inside), has suggestion
        {
          code: `core.error(\`error: \${msg}\`); core.setFailed(\`error: \${msg}\`);`,
          errors: [
            {
              messageId: "noCoreErrorThenSetFailed",
              suggestions: [{ messageId: "removeErrorCall", output: ` core.setFailed(\`error: \${msg}\`);` }],
            },
          ],
        },
        // Same call-expression argument — NOT side-effect-free: report but no suggestion
        {
          code: `core.error(nextMessage()); core.setFailed(nextMessage());`,
          errors: [{ messageId: "noCoreErrorThenSetFailed", suggestions: [] }],
        },
        // Inside a block with matching messages — has suggestion
        {
          code: `function run() { core.error("fatal"); core.setFailed("fatal"); }`,
          errors: [
            {
              messageId: "noCoreErrorThenSetFailed",
              suggestions: [{ messageId: "removeErrorCall", output: `function run() {  core.setFailed("fatal"); }` }],
            },
          ],
        },
      ],
    });
  });
});
