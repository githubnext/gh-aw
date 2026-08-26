import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noJsonStringifyEqualityRule } from "./no-json-stringify-equality";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("no-json-stringify-equality", () => {
  it("uses the correct docs URL", () => {
    expect(noJsonStringifyEqualityRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-json-stringify-equality");
  });

  it("flags JSON.stringify comparisons and accepts safe alternatives", () => {
    ruleTester.run("no-json-stringify-equality", noJsonStringifyEqualityRule, {
      valid: [
        // Deep-equality comparisons that do not rely on stringify at all.
        `deepEqual(normalizedLeft, normalizedRight);`,
        // Comparing a stringified value against a plain string literal is fine
        // (no key-order ambiguity: only one side is derived from an object).
        `JSON.stringify(value) === "{}"`,
        `"{}" === JSON.stringify(value)`,
        // Non-equality binary operators on stringify results are out of scope.
        `JSON.stringify(a).length > JSON.stringify(b).length`,
        // Comparing stringify results against a non-stringify expression.
        `JSON.stringify(a) === cachedSerialized`,
      ],
      invalid: [
        {
          code: `JSON.stringify(normalizedLeft) === JSON.stringify(normalizedRight);`,
          errors: [{ messageId: "jsonStringifyEquality", data: { operator: "===" } }],
        },
        {
          code: `JSON.stringify(a) !== JSON.stringify(b);`,
          errors: [{ messageId: "jsonStringifyEquality", data: { operator: "!==" } }],
        },
        {
          code: `if (JSON.stringify(left) == JSON.stringify(right)) { doSomething(); }`,
          errors: [{ messageId: "jsonStringifyEquality", data: { operator: "==" } }],
        },
        {
          code: `const changed = JSON.stringify(prevState) != JSON.stringify(nextState);`,
          errors: [{ messageId: "jsonStringifyEquality", data: { operator: "!=" } }],
        },
      ],
    });
  });
});
