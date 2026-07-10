import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireReturnAfterCoreSetFailedRule } from "./require-return-after-core-setfailed";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-return-after-core-setfailed", () => {
  it("uses the correct docs URL", () => {
    expect(requireReturnAfterCoreSetFailedRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-return-after-core-setfailed");
  });

  it("valid: core.setFailed followed by return", () => {
    ruleTester.run("require-return-after-core-setfailed", requireReturnAfterCoreSetFailedRule, {
      valid: [
        `function f() { core.setFailed("bad"); return; }`,
        `function f() { core.setFailed("bad"); return null; }`,
        `function f() { if (x) { core.setFailed("bad"); return; } }`,
        `function f() { core.setFailed("bad"); throw new Error("bad"); }`,
        `function f() { core.setFailed("bad"); process.exit(1); }`,
        `function f() { for (;;) { core.setFailed("bad"); break; } }`,
        `switch (x) { case "a": core.setFailed("bad"); break; }`,
        // setFailed is the last statement in the block — no next statement to check
        `function f() { core.setFailed("bad"); }`,
        `function f() { if (x) { core.setFailed("bad"); } }`,
      ],
      invalid: [],
    });
  });

  it("valid: non-core.setFailed calls are ignored", () => {
    ruleTester.run("require-return-after-core-setfailed", requireReturnAfterCoreSetFailedRule, {
      valid: [`function f() { other.setFailed("bad"); doMore(); }`, `function f() { core.setOutput("x", 1); doMore(); }`, `function f() { setFailed("bad"); doMore(); }`],
      invalid: [],
    });
  });

  it("invalid: core.setFailed followed by non-control-transfer statement", () => {
    ruleTester.run("require-return-after-core-setfailed", requireReturnAfterCoreSetFailedRule, {
      valid: [],
      invalid: [
        {
          code: `function f() { core.setFailed("bad"); doMore(); }`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: [{ messageId: "addReturn", output: `function f() { core.setFailed("bad"); return; doMore(); }` }] }],
        },
        {
          code: `function f() { if (x) { core.setFailed("bad"); doMore(); } }`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: [{ messageId: "addReturn", output: `function f() { if (x) { core.setFailed("bad"); return; doMore(); } }` }] }],
        },
        {
          code: `function f() {
  if (x) {
    core.setFailed("bad"); // keep with setFailed
    doMore();
  }
}`,
          errors: [
            {
              messageId: "missingReturnAfterSetFailed",
              suggestions: [
                {
                  messageId: "addReturn",
                  output: `function f() {
  if (x) {
    core.setFailed("bad"); // keep with setFailed
    return;
    doMore();
  }
}`,
                },
              ],
            },
          ],
        },
        {
          code: `switch (x) { case "a": core.setFailed("bad"); doMore(); break; }`,
          errors: [{ messageId: "missingReturnAfterSetFailed" }],
        },
        {
          code: `core.setFailed("bad");
doMore();`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: undefined }],
        },
      ],
    });
  });

  it("valid: core.setFailed last in if-block is not flagged (outer block continues)", () => {
    ruleTester.run("require-return-after-core-setfailed", requireReturnAfterCoreSetFailedRule, {
      valid: [
        // setFailed is the last statement in the if-block; no sibling in the same block follows it
        `function f() { if (!ok) { core.setFailed("msg"); } doMore(); }`,
      ],
      invalid: [],
    });
  });
});
