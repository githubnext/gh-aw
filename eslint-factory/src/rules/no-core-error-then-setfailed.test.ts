import { RuleTester } from "@typescript-eslint/rule-tester";
import { noCoreErrorThenSetFailedRule } from "./no-core-error-then-setfailed";

const ruleTester = new RuleTester();

ruleTester.run("no-core-error-then-setfailed", noCoreErrorThenSetFailedRule, {
  valid: [
    // Only core.setFailed — no preceding core.error
    {
      code: `core.setFailed("something went wrong");`,
    },
    // Only core.error — no following setFailed
    {
      code: `core.error("something went wrong");`,
    },
    // core.error followed by something other than setFailed
    {
      code: `core.error("msg"); return;`,
    },
    // core.warning followed by core.setFailed is allowed (different method)
    {
      code: `core.warning("msg"); core.setFailed("msg");`,
    },
    // core.error without adjacent setFailed (setFailed is non-adjacent)
    {
      code: `core.error("msg"); doSomething(); core.setFailed("msg");`,
    },
  ],
  invalid: [
    // Adjacent core.error then core.setFailed — direct
    {
      code: `core.error("msg"); core.setFailed("msg");`,
      errors: [{ messageId: "noCoreErrorThenSetFailed" }],
    },
    // Inside a catch block
    {
      code: `
        try {
          doSomething();
        } catch (err) {
          core.error(\`Failed: \${err.message}\`);
          core.setFailed(\`ERR: Failed: \${err.message}\`);
        }
      `,
      errors: [{ messageId: "noCoreErrorThenSetFailed" }],
    },
    // With an alias (c = core)
    {
      code: `const c = core; c.error("msg"); c.setFailed("msg");`,
      errors: [{ messageId: "noCoreErrorThenSetFailed" }],
    },
    // Computed property access
    {
      code: `core["error"]("msg"); core["setFailed"]("msg");`,
      errors: [{ messageId: "noCoreErrorThenSetFailed" }],
    },
    // Has suggestion to remove core.error call
    {
      code: `core.error("msg"); core.setFailed("msg");`,
      errors: [
        {
          messageId: "noCoreErrorThenSetFailed",
          suggestions: [
            {
              messageId: "removeErrorCall",
              output: ` core.setFailed("msg");`,
            },
          ],
        },
      ],
    },
  ],
});
