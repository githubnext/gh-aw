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
        // setFailed has a return inside the if-block; outer doMore() is not reached via setFailed path
        `function f() { if (x) { core.setFailed("bad"); return; } doMore(); }`,
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
          code: `function f() { core.setFailed("bad"); doMore(); keepGoing(); }`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: [{ messageId: "addReturn", output: `function f() { core.setFailed("bad"); return; doMore(); keepGoing(); }` }] }],
        },
        {
          code: `function f() { if (x) { core.setFailed("bad"); doMore(); keepGoing(); } }`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: [{ messageId: "addReturn", output: `function f() { if (x) { core.setFailed("bad"); return; doMore(); keepGoing(); } }` }] }],
        },
        {
          code: `function f() {
  if (x) {
    core.setFailed("bad"); // keep with setFailed
    doMore();
    keepGoing();
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
    keepGoing();
  }
}`,
                },
              ],
            },
          ],
        },
        {
          code: `function f() {
  try {
    ok();
  } catch (e) {
    core.setFailed("bad");
    core.setOutput("locked", "false");
  }
}`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: undefined }],
        },
        {
          code: `function f() {
  if (x) {
    core.setFailed("bad"); // keep with setFailed
  }
  doMore();
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
  }
  doMore();
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

  it("invalid: core.setFailed last in nested block with outer continuation (Gap 1)", () => {
    ruleTester.run("require-return-after-core-setfailed", requireReturnAfterCoreSetFailedRule, {
      valid: [],
      invalid: [
        // setFailed last in if-block; outer block continues — must be flagged
        {
          code: `function f() { if (!ok) { core.setFailed("msg"); } doMore(); }`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: [{ messageId: "addReturn", output: `function f() { if (!ok) { core.setFailed("msg"); return; } doMore(); }` }] }],
        },
        {
          code: `function f() { while (next()) { if (bad()) { core.setFailed("x"); } } }`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: [{ messageId: "addReturn", output: `function f() { while (next()) { if (bad()) { core.setFailed("x"); return; } } }` }] }],
        },
        {
          code: `function f() { do { if (bad()) { core.setFailed("x"); } } while (next()); }`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: [{ messageId: "addReturn", output: `function f() { do { if (bad()) { core.setFailed("x"); return; } } while (next()); }` }] }],
        },
        {
          code: `function f() { for (;;) { if (bad()) { core.setFailed("x"); } } }`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: [{ messageId: "addReturn", output: `function f() { for (;;) { if (bad()) { core.setFailed("x"); return; } } }` }] }],
        },
        {
          code: `function f(x) { switch (x) { case 1: if (bad) { core.setFailed("x"); } case 2: doMore(); } }`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: [{ messageId: "addReturn", output: `function f(x) { switch (x) { case 1: if (bad) { core.setFailed("x"); return; } case 2: doMore(); } }` }] }],
        },
        {
          code: `function f() {
  if (x) {
    core.setFailed("bad");
  }
  doMore();
}`,
          errors: [
            {
              messageId: "missingReturnAfterSetFailed",
              suggestions: [
                {
                  messageId: "addReturn",
                  output: `function f() {
  if (x) {
    core.setFailed("bad");
    return;
  }
  doMore();
}`,
                },
              ],
            },
          ],
        },
        // else-block continuation — if branch still falls through to doMore
        {
          code: `function f() { if (x) { core.setFailed("bad"); } else { return; } doMore(); }`,
          errors: [{ messageId: "missingReturnAfterSetFailed", suggestions: [{ messageId: "addReturn", output: `function f() { if (x) { core.setFailed("bad"); return; } else { return; } doMore(); }` }] }],
        },
      ],
    });
  });

  it("valid: continue after setFailed is accepted — known limitation: does not stop post-loop execution", () => {
    ruleTester.run("require-return-after-core-setfailed", requireReturnAfterCoreSetFailedRule, {
      valid: [
        // continue ends the current iteration; the loop and any post-loop code
        // still run in a failed state. This is a known, documented limitation:
        // the rule accepts break/continue to cover the common loop-guard pattern.
        `for (const x of items) { if (bad(x)) { core.setFailed(err); continue; } process(x); }`,
        `for (const x of items) { core.setFailed(err); continue; }`,
      ],
      invalid: [],
    });
  });
});
