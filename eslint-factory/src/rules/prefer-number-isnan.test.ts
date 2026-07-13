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
        `import { isNaN } from "lodash"; isNaN(value);`,
        `const globalThis = { isNaN(value) { return value; } }; globalThis.isNaN(value);`,
        `const window = { isNaN(value) { return value; } }; window["isNaN"](value);`,
        `const global = { isNaN(value) { return value; } }; global.isNaN(value);`,
        `import { globalThis } from "./global-shim"; globalThis.isNaN(value);`,
        `import { window } from "./browser-shim"; window.isNaN(value);`,
        `import { global } from "./server-shim"; global["isNaN"](value);`,
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

  it("invalid: provably numeric arguments are autofixed", () => {
    cjsRuleTester.run("prefer-number-isnan", preferNumberIsNanRule, {
      valid: [],
      invalid: [
        {
          code: `isNaN(parseInt(value, 10));`,
          output: `Number.isNaN(parseInt(value, 10));`,
          errors: [{ messageId: "preferNumberIsNaN" }],
        },
        {
          code: `isNaN(parseFloat(value));`,
          output: `Number.isNaN(parseFloat(value));`,
          errors: [{ messageId: "preferNumberIsNaN" }],
        },
        {
          code: `isNaN(Number(value));`,
          output: `Number.isNaN(Number(value));`,
          errors: [{ messageId: "preferNumberIsNaN" }],
        },
        {
          code: `isNaN(Number.parseInt(value, 10));`,
          output: `Number.isNaN(Number.parseInt(value, 10));`,
          errors: [{ messageId: "preferNumberIsNaN" }],
        },
        {
          code: `isNaN(42);`,
          output: `Number.isNaN(42);`,
          errors: [{ messageId: "preferNumberIsNaN" }],
        },
      ],
    });
  });

  it("valid: method calls on arbitrary receivers are not treated as provably numeric", () => {
    cjsRuleTester.run("prefer-number-isnan", preferNumberIsNanRule, {
      valid: [],
      invalid: [
        // getTime/valueOf can be defined on any object; must remain suggestion-only
        {
          code: `isNaN(d.getTime());`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `Number.isNaN(d.getTime());` }] }],
        },
        {
          code: `isNaN(x.valueOf());`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `Number.isNaN(x.valueOf());` }] }],
        },
      ],
    });
  });

  it("invalid: shadowed parseInt/parseFloat/Number fall back to suggestion-only", () => {
    esmRuleTester.run("prefer-number-isnan", preferNumberIsNanRule, {
      valid: [],
      invalid: [
        {
          code: `function parseInt() { return "x"; } isNaN(parseInt(v));`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `function parseInt() { return "x"; } Number.isNaN(parseInt(v));` }] }],
        },
        {
          code: `const Number = {}; isNaN(0);`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `const Number = {}; Number.isNaN(0);` }] }],
        },
      ],
    });
  });

  it("invalid: unknown/raw arguments remain suggestion-only with coercion caveat", () => {
    cjsRuleTester.run("prefer-number-isnan", preferNumberIsNanRule, {
      valid: [],
      invalid: [
        {
          code: `isNaN(process.env.PORT);`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `Number.isNaN(process.env.PORT);` }] }],
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
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `Number.isNaN(value);` }] }],
        },
        {
          code: `globalThis["isNaN"](value);`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `Number.isNaN(value);` }] }],
        },
        {
          code: `window.isNaN(value);`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `Number.isNaN(value);` }] }],
        },
        {
          code: `window["isNaN"](value);`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `Number.isNaN(value);` }] }],
        },
        {
          code: `global.isNaN(value);`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `Number.isNaN(value);` }] }],
        },
        {
          code: `global["isNaN"](value);`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `Number.isNaN(value);` }] }],
        },
      ],
    });
  });

  it("invalid: global isNaN() is still flagged in ESM mode without a shadow", () => {
    esmRuleTester.run("prefer-number-isnan", preferNumberIsNanRule, {
      valid: [],
      invalid: [
        {
          code: `isNaN(value);`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `Number.isNaN(value);` }] }],
        },
        {
          code: `window.isNaN(value);`,
          errors: [{ messageId: "preferNumberIsNaNWithCoercionCaveat", suggestions: [{ messageId: "replaceWithNumberIsNaNWithNumberWrapReview", output: `Number.isNaN(value);` }] }],
        },
      ],
    });
  });
});
