import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noDuplicateConstantValuesRule } from "./no-duplicate-constant-values";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("no-duplicate-constant-values", () => {
  it("uses the correct docs URL", () => {
    expect(noDuplicateConstantValuesRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-duplicate-constant-values");
  });

  it("accepts unique and dynamic constant values", () => {
    ruleTester.run("no-duplicate-constant-values", noDuplicateConstantValuesRule, {
      valid: [
        `const FIRST = "first"; const SECOND = "second";`,
        `const FIRST = makeValue(); const SECOND = makeValue();`,
        `const FIRST = { value: 1 }; const SECOND = { value: 1 };`,
        `let first = "same"; let second = "same";`,
        `const { first, second } = value;`,
        `function first() { const VALUE = "same"; } function second() { const VALUE = "same"; }`,
      ],
      invalid: [],
    });
  });

  it("reports duplicate strings, numbers, templates, and regular expressions", () => {
    ruleTester.run("no-duplicate-constant-values", noDuplicateConstantValuesRule, {
      valid: [],
      invalid: [
        {
          code: `const FIRST = "same"; const SECOND = "same";`,
          errors: [{ messageId: "duplicateConstantValue", data: { name: "SECOND", originalName: "FIRST", value: `"same"` } }],
        },
        {
          code: 'const FIRST = `same`; const SECOND = "same";',
          errors: [{ messageId: "duplicateConstantValue", data: { name: "SECOND", originalName: "FIRST", value: `"same"` } }],
        },
        {
          code: `const FIRST = -42; const SECOND = -42; const THIRD = 42;`,
          errors: [{ messageId: "duplicateConstantValue", data: { name: "SECOND", originalName: "FIRST", value: "-42" } }],
        },
        {
          code: `const FIRST = /value/gi; const SECOND = /value/gi;`,
          errors: [{ messageId: "duplicateConstantValue", data: { name: "SECOND", originalName: "FIRST", value: "/value/gi" } }],
        },
      ],
    });
  });

  it("reports every duplicate after the first declaration", () => {
    ruleTester.run("no-duplicate-constant-values", noDuplicateConstantValuesRule, {
      valid: [],
      invalid: [
        {
          code: `const FIRST = "same"; const SECOND = "same"; const THIRD = "same";`,
          errors: [
            { messageId: "duplicateConstantValue", data: { name: "SECOND", originalName: "FIRST", value: `"same"` } },
            { messageId: "duplicateConstantValue", data: { name: "THIRD", originalName: "FIRST", value: `"same"` } },
          ],
        },
      ],
    });
  });
});
