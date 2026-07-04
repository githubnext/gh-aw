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
    // CJS-only: actions/setup/js targets CommonJS; ESM counterparts tested in the shadowed-bindings block below
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
        // Dynamic computed access — identifier property reference, not string literal "isNaN"
        `globalThis[isNaN](value);`,
      ],
      invalid: [],
    });
  });

  it("valid: isNaN used as a callback reference is not a CallExpression and is not flagged", () => {
    cjsRuleTester.run("prefer-number-isnan", preferNumberIsNanRule, {
      valid: [`values.some(isNaN);`],
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
        {
          // Raw string argument (e.g. env var) — suggestion preserves argument so callers must review whether to wrap with Number(...)
          code: `isNaN(process.env.PORT);`,
          errors: [{ messageId: "preferNumberIsNaN", suggestions: [{ messageId: "replaceWithNumberIsNaN", output: `Number.isNaN(process.env.PORT);` }] }],
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
