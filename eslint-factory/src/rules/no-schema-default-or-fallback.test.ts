import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noSchemaDefaultOrFallbackRule } from "./no-schema-default-or-fallback";

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

describe("no-schema-default-or-fallback", () => {
  it("valid: nullish coalescing on .default is not flagged", () => {
    cjsRuleTester.run("no-schema-default-or-fallback", noSchemaDefaultOrFallbackRule, {
      valid: [`const v = inputSchema.default ?? undefined;`, `const v = schema.default ?? 1;`, `const v = tool.input.default ?? "";`],
      invalid: [],
    });
  });

  it("valid: || on unrelated properties is not flagged", () => {
    cjsRuleTester.run("no-schema-default-or-fallback", noSchemaDefaultOrFallbackRule, {
      valid: [`const v = config.max || 10;`, `const v = options.timeout || 60;`, `const v = value || "";`, `const v = obj.defaults || {};`],
      invalid: [],
    });
  });

  it("invalid: .default || fallback is flagged for numeric literal fallback", () => {
    cjsRuleTester.run("no-schema-default-or-fallback", noSchemaDefaultOrFallbackRule, {
      valid: [],
      invalid: [
        {
          code: `const v = inputSchema.default || 0;`,
          errors: [{ messageId: "useNullishCoalescing" }],
        },
      ],
    });
  });

  it("invalid: .default || fallback is flagged for string/boolean/identifier fallbacks", () => {
    cjsRuleTester.run("no-schema-default-or-fallback", noSchemaDefaultOrFallbackRule, {
      valid: [],
      invalid: [
        {
          code: `const v = schema.default || "";`,
          errors: [{ messageId: "useNullishCoalescing" }],
        },
        {
          code: `const v = schema.default || false;`,
          errors: [{ messageId: "useNullishCoalescing" }],
        },
        {
          code: `const v = tool.inputSchema.default || fallbackValue;`,
          errors: [{ messageId: "useNullishCoalescing" }],
        },
      ],
    });
  });

  it("invalid: nested member expression object is flagged (ESM)", () => {
    esmRuleTester.run("no-schema-default-or-fallback", noSchemaDefaultOrFallbackRule, {
      valid: [],
      invalid: [
        {
          code: `const v = props[name].default || undefined;`,
          errors: [{ messageId: "useNullishCoalescing" }],
        },
      ],
    });
  });
});
