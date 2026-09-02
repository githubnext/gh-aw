import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireMatchResultNullCheckRule } from "./require-match-result-null-check";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-match-result-null-check", () => {
  it("uses the correct docs URL", () => {
    expect(requireMatchResultNullCheckRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-match-result-null-check");
  });

  it("valid: guarded with if-statement before indexed access", () => {
    ruleTester.run("require-match-result-null-check", requireMatchResultNullCheckRule, {
      valid: [
        `const m = text.match(/(\\d+)/); if (m) { use(m[1]); }`,
        `const m = text.match(/(\\d+)/); if (!m) { return null; } use(m[1]);`,
        `const m = text.match(/(\\d+)/); if (m === null) { return null; } use(m[1]);`,
      ],
      invalid: [],
    });
  });

  it("valid: optional chaining access", () => {
    ruleTester.run("require-match-result-null-check", requireMatchResultNullCheckRule, {
      valid: [`const m = text.match(/(\\d+)/); const value = m?.[1];`, `const m = text.match(/(\\d+)/); const value = m?.groups;`],
      invalid: [],
    });
  });

  it("valid: logical guard before access", () => {
    ruleTester.run("require-match-result-null-check", requireMatchResultNullCheckRule, {
      valid: [`const m = text.match(/(\\d+)/); const value = m && m[1];`],
      invalid: [],
    });
  });

  it("valid: optional-chained if-test guard before bare access (check_rate_limit.cjs pattern)", () => {
    ruleTester.run("require-match-result-null-check", requireMatchResultNullCheckRule, {
      valid: [`const m = text.match(/(\\d+)/); if (m?.[1]) { use(m[1]); }`, `const m = text.match(/(?<year>\\d+)/); if (m?.groups) { use(m.groups); }`],
      invalid: [],
    });
  });

  it("valid: ternary test guard before access", () => {
    ruleTester.run("require-match-result-null-check", requireMatchResultNullCheckRule, {
      valid: [`const m = text.match(/(\\d+)/); const value = m ? m[1] : null;`],
      invalid: [],
    });
  });

  it("valid: inline match() call directly optional-chained (no intermediate variable)", () => {
    ruleTester.run("require-match-result-null-check", requireMatchResultNullCheckRule, {
      valid: [`const value = text.match(/(\\d+)/)?.[1];`],
      invalid: [],
    });
  });

  it("invalid: indexed access without any guard", () => {
    ruleTester.run("require-match-result-null-check", requireMatchResultNullCheckRule, {
      valid: [],
      invalid: [
        {
          code: `const m = text.match(/(\\d+)/); use(m[1]);`,
          errors: [{ messageId: "requireNullCheck", data: { name: "m", access: "[1]", accessNoDot: "[1]" } }],
        },
      ],
    });
  });

  it("invalid: property access (.groups) without any guard", () => {
    ruleTester.run("require-match-result-null-check", requireMatchResultNullCheckRule, {
      valid: [],
      invalid: [
        {
          code: `const m = text.match(/(?<year>\\d+)/); use(m.groups);`,
          errors: [{ messageId: "requireNullCheck", data: { name: "m", access: ".groups", accessNoDot: "groups" } }],
        },
      ],
    });
  });

  it("invalid: only one report per variable even with multiple unguarded accesses", () => {
    ruleTester.run("require-match-result-null-check", requireMatchResultNullCheckRule, {
      valid: [],
      invalid: [
        {
          code: `const m = text.match(/(\\d+)/); use(m[1]); use(m[2]);`,
          errors: [{ messageId: "requireNullCheck", data: { name: "m", access: "[1]", accessNoDot: "[1]" } }],
        },
      ],
    });
  });
});
