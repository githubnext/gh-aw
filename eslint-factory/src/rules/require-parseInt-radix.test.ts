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
        `parseInt(value, 10);`,
        `Number.parseInt(value, 10);`,
        `Number["parseInt"](value, 10);`,
        `globalThis.parseInt(value, 10);`,
        `globalThis["parseInt"](value, 10);`,
        `window.parseInt(value, 10);`,
        `window["parseInt"](value, 10);`,
        `global.parseInt(value, 10);`,
        `global["parseInt"](value, 10);`,
      ],
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

  it("invalid: computed Number.parseInt access without radix is flagged", () => {
    cjsRuleTester.run("require-parseInt-radix", requireParseIntRadixRule, {
      valid: [],
      invalid: [
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
