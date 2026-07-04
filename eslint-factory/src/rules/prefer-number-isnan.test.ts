import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { preferNumberIsNanRule } from "./prefer-number-isnan";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

const esmRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "module",
  },
});

describe("prefer-number-isnan", () => {
  it("uses the correct docs URL", () => {
    expect(preferNumberIsNanRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#prefer-number-isnan");
  });

  it("valid: Number.isNaN and non-global forms are accepted", () => {
    cjsRuleTester.run("prefer-number-isnan", preferNumberIsNanRule, {
      valid: [`Number.isNaN(value);`, `Number["isNaN"](value);`, `foo.isNaN(value);`],
      invalid: [],
    });
  });

  it("valid: locally shadowed bindings are intentionally excluded", () => {
    esmRuleTester.run("prefer-number-isnan", preferNumberIsNanRule, {
      valid: [
        `function isNaN(value) { return false; } isNaN(value);`,
        `const isNaN = Number.isNaN; isNaN(value);`,
        `const globalThis = { isNaN(value) { return value; } }; globalThis.isNaN(value);`,
        `const window = { isNaN(value) { return value; } }; window["isNaN"](value);`,
        `const global = { isNaN(value) { return value; } }; global.isNaN(value);`,
      ],
      invalid: [],
    });
  });

  it("invalid: global isNaN() is flagged with a replacement suggestion", () => {
    cjsRuleTester.run("prefer-number-isnan", preferNumberIsNanRule, {
      valid: [],
      invalid: [
        {
          code: `isNaN(value);`,
          errors: [{ messageId: "preferNumberIsNaN", suggestions: [{ messageId: "replaceWithNumberIsNaN", output: `Number.isNaN(value);` }] }],
        },
      ],
    });
  });

  it("invalid: global object isNaN() access is flagged for direct and computed forms", () => {
    cjsRuleTester.run("prefer-number-isnan", preferNumberIsNanRule, {
      valid: [],
      invalid: [
        {
          code: `globalThis.isNaN(value);`,
          errors: [{ messageId: "preferNumberIsNaN", suggestions: [{ messageId: "replaceWithNumberIsNaN", output: `Number.isNaN(value);` }] }],
        },
        {
          code: `globalThis["isNaN"](value);`,
          errors: [{ messageId: "preferNumberIsNaN", suggestions: [{ messageId: "replaceWithNumberIsNaN", output: `Number.isNaN(value);` }] }],
        },
        {
          code: `window.isNaN(value);`,
          errors: [{ messageId: "preferNumberIsNaN", suggestions: [{ messageId: "replaceWithNumberIsNaN", output: `Number.isNaN(value);` }] }],
        },
        {
          code: `window["isNaN"](value);`,
          errors: [{ messageId: "preferNumberIsNaN", suggestions: [{ messageId: "replaceWithNumberIsNaN", output: `Number.isNaN(value);` }] }],
        },
        {
          code: `global.isNaN(value);`,
          errors: [{ messageId: "preferNumberIsNaN", suggestions: [{ messageId: "replaceWithNumberIsNaN", output: `Number.isNaN(value);` }] }],
        },
        {
          code: `global["isNaN"](value);`,
          errors: [{ messageId: "preferNumberIsNaN", suggestions: [{ messageId: "replaceWithNumberIsNaN", output: `Number.isNaN(value);` }] }],
        },
      ],
    });
  });
});
