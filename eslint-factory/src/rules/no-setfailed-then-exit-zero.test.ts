import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noSetFailedThenExitZeroRule } from "./no-setfailed-then-exit-zero";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: "latest",
    sourceType: "commonjs",
  },
});

describe("no-setfailed-then-exit-zero", () => {
  it("uses the correct docs URL", () => {
    expect(noSetFailedThenExitZeroRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-setfailed-then-exit-zero");
  });

  it("valid and invalid cases", () => {
    ruleTester.run("no-setfailed-then-exit-zero", noSetFailedThenExitZeroRule, {
      valid: [
        // process.exit(0) without a preceding core.setFailed is fine
        `process.exit(0);`,
        // core.setFailed followed by return is the correct pattern
        `function f() { core.setFailed("bad"); return; }`,
        // core.setFailed followed by process.exit(1) is fine — non-zero exit preserves failure
        `core.setFailed("bad"); process.exit(1);`,
        // core.setFailed followed by process.exit(2) is fine
        `function f() { core.setFailed("bad"); process.exit(2); }`,
        // core.setFailed with no next statement is fine
        `function f() { core.setFailed("bad"); }`,
        // Non-core object is not flagged
        `other.setFailed("bad"); process.exit(0);`,
        // core.error (not setFailed) is not flagged by this rule
        `core.error("bad"); process.exit(0);`,
        // return between setFailed and process.exit(0) stops scanning
        `function f() { core.setFailed("bad"); return; process.exit(0); }`,
        // throw between setFailed and process.exit(0) stops scanning
        `function f() { core.setFailed("bad"); throw new Error("x"); process.exit(0); }`,
        // process.exit(variable) — runtime value unknown, not flagged
        `core.setFailed("bad"); process.exit(code);`,
        // process.exit with non-zero string literal — not matched
        `core.setFailed("bad"); process.exit("0");`,
        // Computed alias not matching core
        `const c = other; function f() { c.setFailed("bad"); process.exit(0); }`,
        // Destructured from non-core — not flagged
        `const { setFailed } = other; function f() { setFailed("bad"); process.exit(0); }`,
        // Destructured setFailed with return is the correct pattern
        `const { setFailed } = core; function f() { setFailed("bad"); return; }`,
      ],
      invalid: [
        // Adjacent: core.setFailed immediately followed by process.exit(0)
        {
          code: `core.setFailed("bad"); process.exit(0);`,
          errors: [{ messageId: "noSetFailedThenExitZero", suggestions: [{ messageId: "replaceWithReturn", output: `core.setFailed("bad"); return;` }] }],
        },
        // Inside a function
        {
          code: `function f() { core.setFailed("bad"); process.exit(0); }`,
          errors: [{ messageId: "noSetFailedThenExitZero", suggestions: [{ messageId: "replaceWithReturn", output: `function f() { core.setFailed("bad"); return; }` }] }],
        },
        // process.exit() with no arguments (defaults to 0)
        {
          code: `function f() { core.setFailed("bad"); process.exit(); }`,
          errors: [{ messageId: "noSetFailedThenExitZero", suggestions: [{ messageId: "replaceWithReturn", output: `function f() { core.setFailed("bad"); return; }` }] }],
        },
        // Computed property: core["setFailed"]
        {
          code: `core["setFailed"]("bad"); process.exit(0);`,
          errors: [{ messageId: "noSetFailedThenExitZero", suggestions: [{ messageId: "replaceWithReturn", output: `core["setFailed"]("bad"); return;` }] }],
        },
        // Non-adjacent: intervening log statement does not stop detection
        {
          code: `core.setFailed("bad"); core.info("msg"); process.exit(0);`,
          errors: [{ messageId: "noSetFailedThenExitZero", suggestions: [] }],
        },
        // Aliased core object
        {
          code: `const c = core; function f() { c.setFailed("bad"); process.exit(0); }`,
          errors: [{ messageId: "noSetFailedThenExitZero", suggestions: [{ messageId: "replaceWithReturn", output: `const c = core; function f() { c.setFailed("bad"); return; }` }] }],
        },
        // Inside an if block
        {
          code: `function f() { if (bad) { core.setFailed("x"); process.exit(0); } }`,
          errors: [{ messageId: "noSetFailedThenExitZero", suggestions: [{ messageId: "replaceWithReturn", output: `function f() { if (bad) { core.setFailed("x"); return; } }` }] }],
        },
        // switch-case
        {
          code: `switch (x) { case 1: core.setFailed("bad"); process.exit(0); break; }`,
          errors: [{ messageId: "noSetFailedThenExitZero", suggestions: [{ messageId: "replaceWithReturn", output: `switch (x) { case 1: core.setFailed("bad"); return; break; }` }] }],
        },
        // Destructured setFailed from core
        {
          code: `const { setFailed } = core; function f() { setFailed("bad"); process.exit(0); }`,
          errors: [{ messageId: "noSetFailedThenExitZero", suggestions: [{ messageId: "replaceWithReturn", output: `const { setFailed } = core; function f() { setFailed("bad"); return; }` }] }],
        },
        // Renamed destructured binding: const { setFailed: sf } = core
        {
          code: `const { setFailed: sf } = core; function f() { sf("bad"); process.exit(0); }`,
          errors: [{ messageId: "noSetFailedThenExitZero", suggestions: [{ messageId: "replaceWithReturn", output: `const { setFailed: sf } = core; function f() { sf("bad"); return; }` }] }],
        },
      ],
    });
  });
});
