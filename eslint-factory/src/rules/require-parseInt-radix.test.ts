import { RuleTester } from "eslint";
import { describe, it } from "vitest";
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
          errors: [{ messageId: "requireRadix" }],
        },
        {
          code: `parseInt(str.trim());`,
          errors: [{ messageId: "requireRadix" }],
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
          errors: [{ messageId: "requireRadix" }],
        },
        {
          code: `Number["parseInt"](value);`,
          errors: [{ messageId: "requireRadix" }],
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
          errors: [{ messageId: "requireRadix" }],
        },
        {
          code: `globalThis["parseInt"](value);`,
          errors: [{ messageId: "requireRadix" }],
        },
        {
          code: `window.parseInt(value);`,
          errors: [{ messageId: "requireRadix" }],
        },
        {
          code: `window["parseInt"](value);`,
          errors: [{ messageId: "requireRadix" }],
        },
        {
          code: `global.parseInt(value);`,
          errors: [{ messageId: "requireRadix" }],
        },
        {
          code: `global["parseInt"](value);`,
          errors: [{ messageId: "requireRadix" }],
        },
      ],
    });
  });
});
