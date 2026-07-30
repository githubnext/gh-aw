import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireEscapedRegexpInterpolationRule } from "./require-escaped-regexp-interpolation";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-escaped-regexp-interpolation", () => {
  it("uses the correct docs URL", () => {
    expect(requireEscapedRegexpInterpolationRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-escaped-regexp-interpolation");
  });

  it("valid: non-interpolated RegExp patterns are accepted", () => {
    cjsRuleTester.run("require-escaped-regexp-interpolation", requireEscapedRegexpInterpolationRule, {
      valid: ['new RegExp("^[a-z]+$");', "new RegExp(`^[a-z]+$`);", "new RegExp(somePattern);", "new RegExp(somePattern, 'g');"],
      invalid: [],
    });
  });

  it("valid: interpolated value already passed through an escape helper is accepted", () => {
    cjsRuleTester.run("require-escaped-regexp-interpolation", requireEscapedRegexpInterpolationRule, {
      valid: [
        "new RegExp(`^${escapeRegExp(varName)}$`);",
        "new RegExp(`(^|[-_\\\\s])${escapeRegex(qualifier)}($|[-_\\\\s])`);",
        "new RegExp(`\\\\$\\\\{${utils.escapeRegExp(varName)}\\\\}`, 'g');",
        "new RegExp(`^${ESCAPED_NAME}$`);",
        "new RegExp(`^${escapedValue}$`);",
      ],
      invalid: [],
    });
  });

  it('valid: standard inline .replace(…, "\\\\$&") escape form is accepted', () => {
    cjsRuleTester.run("require-escaped-regexp-interpolation", requireEscapedRegexpInterpolationRule, {
      valid: ['new RegExp(`^${varName.replace(/[.*+?^${}()|[\\\\]\\\\\\\\]/g, "\\\\$&")}$`);', 'new RegExp(`^${qualifier.replace(/[.*+?^${}()|[\\\\]\\\\\\\\]/g, "\\\\$&")}($|[-_\\\\s])`);'],
      invalid: [],
    });
  });

  it("valid: unrelated `new` calls to other constructors are not flagged", () => {
    cjsRuleTester.run("require-escaped-regexp-interpolation", requireEscapedRegexpInterpolationRule, {
      valid: ["new Foo(`^${bar}$`);", "new Date(`${year}-01-01`);"],
      invalid: [],
    });
  });

  it("invalid: interpolated loop variable in RegExp pattern without escaping", () => {
    cjsRuleTester.run("require-escaped-regexp-interpolation", requireEscapedRegexpInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: "const pattern = new RegExp(`\\\\$\\\\{${varName}\\\\}`, 'g');",
          errors: [{ messageId: "unescapedInterpolation" }],
        },
      ],
    });
  });

  it("invalid: interpolated function parameter in RegExp pattern without escaping", () => {
    cjsRuleTester.run("require-escaped-regexp-interpolation", requireEscapedRegexpInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: "function hasQualifier(name, qualifier) { return new RegExp(`(^|[-_\\\\s])${qualifier}($|[-_\\\\s])`).test(name); }",
          errors: [{ messageId: "unescapedInterpolation" }],
        },
      ],
    });
  });

  it("invalid: multiple unescaped interpolations are each reported", () => {
    cjsRuleTester.run("require-escaped-regexp-interpolation", requireEscapedRegexpInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: "new RegExp(`${prefix}-${suffix}`);",
          errors: [{ messageId: "unescapedInterpolation" }, { messageId: "unescapedInterpolation" }],
        },
      ],
    });
  });

  it("invalid: mixed escaped and unescaped interpolations only report the unescaped one", () => {
    cjsRuleTester.run("require-escaped-regexp-interpolation", requireEscapedRegexpInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: "new RegExp(`${escapeRegExp(prefix)}-${suffix}`);",
          errors: [{ messageId: "unescapedInterpolation" }],
        },
      ],
    });
  });

  it("invalid: identifier named unescapedValue is not treated as safe", () => {
    cjsRuleTester.run("require-escaped-regexp-interpolation", requireEscapedRegexpInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: "new RegExp(`^${unescapedValue}$`);",
          errors: [{ messageId: "unescapedInterpolation" }],
        },
      ],
    });
  });

  it("invalid: escapeHtml call is not treated as a regex-escape helper", () => {
    cjsRuleTester.run("require-escaped-regexp-interpolation", requireEscapedRegexpInterpolationRule, {
      valid: [],
      invalid: [
        {
          code: "new RegExp(`^${escapeHtml(userInput)}$`);",
          errors: [{ messageId: "unescapedInterpolation" }],
        },
      ],
    });
  });
});
