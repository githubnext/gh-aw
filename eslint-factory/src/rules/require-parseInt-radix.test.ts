import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireParseIntRadixRule } from "./require-parseInt-radix";

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

describe("require-parseInt-radix", () => {
  it("uses the correct docs URL", () => {
    expect(requireParseIntRadixRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-parseInt-radix");
  });

  it("valid: explicit radix is accepted for direct and computed access", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [
        `parseInt(str, 10);`,
        `parseInt(str, 16);`,
        `Number.parseInt(str, 10);`,
        `Number["parseInt"](str, 10);`,
        `globalThis.parseInt(str, 10);`,
        `globalThis["parseInt"](str, 10);`,
        `window.parseInt(str, 10);`,
        `window["parseInt"](str, 10);`,
        `global.parseInt(str, 10);`,
        `global["parseInt"](str, 10);`,
        // Non-literal identifier radix is accepted (cannot be statically verified but is not a known-bad literal)
        `parseInt(str, base);`,
        `Number.parseInt(str, radix);`,
      ],
      invalid: [],
    });
  });

  it("valid: non-parseInt calls are not flagged", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [`foo.parseInt(x);`, `parseFloat(x);`],
      invalid: [],
    });
  });

  it("valid: aliased and destructured bindings remain out of scope", () => {
    esmRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [
        `const p = parseInt; p(value);`,
        `const { parseInt } = Number; parseInt(value);`,
        `const globalThis = { parseInt(value) { return value; } }; globalThis.parseInt(value);`,
        `const window = { parseInt(value) { return value; } }; window["parseInt"](value);`,
        `const global = { parseInt(value) { return value; } }; global.parseInt(value);`,
        `function f(undefined) { parseInt(str, undefined); }`,
        `const NaN = 10; parseInt(str, NaN);`,
      ],
      invalid: [],
    });
  });

  it("invalid: global parseInt without radix is flagged", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [],
      invalid: [
        {
          code: `parseInt(str);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `parseInt(str, 10);` }] }],
        },
        {
          code: `parseInt(str.trim());`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `parseInt(str.trim(), 10);` }] }],
        },
      ],
    });
  });

  it("invalid: Number.parseInt without radix is flagged (direct and computed access)", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [],
      invalid: [
        {
          code: `Number.parseInt(str);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `Number.parseInt(str, 10);` }] }],
        },
        {
          code: `Number["parseInt"](value);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `Number["parseInt"](value, 10);` }] }],
        },
      ],
    });
  });

  it("invalid: global-object parseInt access without radix is flagged", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [],
      invalid: [
        {
          code: `globalThis.parseInt(value);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `globalThis.parseInt(value, 10);` }] }],
        },
        {
          code: `globalThis["parseInt"](value);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `globalThis["parseInt"](value, 10);` }] }],
        },
        {
          code: `window.parseInt(value);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `window.parseInt(value, 10);` }] }],
        },
        {
          code: `window["parseInt"](value);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `window["parseInt"](value, 10);` }] }],
        },
        {
          code: `global.parseInt(value);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `global.parseInt(value, 10);` }] }],
        },
        {
          code: `global["parseInt"](value);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `global["parseInt"](value, 10);` }] }],
        },
      ],
    });
  });

  it("suggestion: inserts ', 10' for single-argument parseInt calls", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [],
      invalid: [
        {
          code: `parseInt(str);`,
          errors: [
            {
              messageId: "requireRadix",
              suggestions: [{ messageId: "addRadix10", output: `parseInt(str, 10);` }],
            },
          ],
        },
        {
          code: `Number.parseInt(value);`,
          errors: [
            {
              messageId: "requireRadix",
              suggestions: [{ messageId: "addRadix10", output: `Number.parseInt(value, 10);` }],
            },
          ],
        },
        {
          code: `globalThis["parseInt"](value);`,
          errors: [
            {
              messageId: "requireRadix",
              suggestions: [{ messageId: "addRadix10", output: `globalThis["parseInt"](value, 10);` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: radix 0 is rejected (spec-equivalent to no radix)", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [],
      invalid: [
        {
          code: `parseInt(str, 0);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `parseInt(str, 10);` }] }],
        },
        {
          code: `Number.parseInt(str, 0);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `Number.parseInt(str, 10);` }] }],
        },
        {
          code: `globalThis.parseInt(str, 0);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `globalThis.parseInt(str, 10);` }] }],
        },
        {
          code: `window["parseInt"](str, 0);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `window["parseInt"](str, 10);` }] }],
        },
        {
          code: `global.parseInt(str, 0);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `global.parseInt(str, 10);` }] }],
        },
      ],
    });
  });

  it("invalid: radix undefined is rejected (equivalent to no radix)", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [],
      invalid: [
        {
          code: `parseInt(str, undefined);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `parseInt(str, 10);` }] }],
        },
        {
          code: `Number.parseInt(str, undefined);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `Number.parseInt(str, 10);` }] }],
        },
        {
          code: `globalThis.parseInt(str, undefined);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `globalThis.parseInt(str, 10);` }] }],
        },
        {
          code: `window["parseInt"](str, undefined);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `window["parseInt"](str, 10);` }] }],
        },
        {
          code: `global.parseInt(str, undefined);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `global.parseInt(str, 10);` }] }],
        },
      ],
    });
  });

  it("invalid: radix null is rejected (equivalent to no radix)", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [],
      invalid: [
        {
          code: `parseInt(str, null);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `parseInt(str, 10);` }] }],
        },
        {
          code: `Number.parseInt(str, null);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `Number.parseInt(str, 10);` }] }],
        },
        {
          code: `globalThis.parseInt(str, null);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `globalThis.parseInt(str, 10);` }] }],
        },
        {
          code: `window["parseInt"](str, null);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `window["parseInt"](str, 10);` }] }],
        },
        {
          code: `global.parseInt(str, null);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `global.parseInt(str, 10);` }] }],
        },
      ],
    });
  });

  it("invalid: global NaN radix is rejected (equivalent to no radix)", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [],
      invalid: [
        {
          code: `parseInt(str, NaN);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `parseInt(str, 10);` }] }],
        },
        {
          code: `Number.parseInt(str, NaN);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `Number.parseInt(str, 10);` }] }],
        },
        {
          code: `globalThis.parseInt(str, NaN);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `globalThis.parseInt(str, 10);` }] }],
        },
        {
          code: `window["parseInt"](str, NaN);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `window["parseInt"](str, 10);` }] }],
        },
        {
          code: `global.parseInt(str, NaN);`,
          errors: [{ messageId: "requireRadix", suggestions: [{ messageId: "addRadix10", output: `global.parseInt(str, 10);` }] }],
        },
      ],
    });
  });

  it("no broken fix for spread-element first argument", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [],
      invalid: [
        {
          // Spread as the only argument: no suggestion should be offered
          code: `parseInt(...args);`,
          errors: [{ messageId: "requireRadix", suggestions: [] }],
        },
        {
          code: `Number.parseInt(...args);`,
          errors: [{ messageId: "requireRadix", suggestions: [] }],
        },
        {
          code: `globalThis.parseInt(...args);`,
          errors: [{ messageId: "requireRadix", suggestions: [] }],
        },
      ],
    });
  });
});
