import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noJsonStringifyEqualityRule } from "./no-json-stringify-equality";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

const messageFor = (operator: string) =>
  `Comparing JSON.stringify(...) results with '${operator}' is unreliable: two deeply-equal objects with different key insertion order produce different strings, causing false negatives. Use a structural deep-equality check (e.g. a recursive deepEqual helper) instead.`;

describe("no-json-stringify-equality", () => {
  it("uses the correct docs URL", () => {
    expect(noJsonStringifyEqualityRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-json-stringify-equality");
  });

  it("invalid: comparing two JSON.stringify() results is flagged for every equality operator", () => {
    cjsRuleTester.run("no-json-stringify-equality", noJsonStringifyEqualityRule, {
      valid: [],
      invalid: [
        {
          code: `const equal = JSON.stringify(left) === JSON.stringify(right);`,
          errors: [{ message: messageFor("===") }],
        },
        {
          code: `const changed = JSON.stringify(before) !== JSON.stringify(after);`,
          errors: [{ message: messageFor("!==") }],
        },
        {
          code: `const equal = JSON.stringify(left) == JSON.stringify(right);`,
          errors: [{ message: messageFor("==") }],
        },
        {
          code: `const changed = JSON.stringify(before) != JSON.stringify(after);`,
          errors: [{ message: messageFor("!=") }],
        },
      ],
    });
  });

  it("invalid: flags the helper form found in evaluate_outcomes.cjs", () => {
    cjsRuleTester.run("no-json-stringify-equality", noJsonStringifyEqualityRule, {
      valid: [],
      invalid: [
        {
          code: `function stateValuesEqual(key, left, right) {
  const normalizedLeft = normalizeStateValue(key, left);
  const normalizedRight = normalizeStateValue(key, right);
  return JSON.stringify(normalizedLeft) === JSON.stringify(normalizedRight);
}`,
          errors: [{ message: messageFor("===") }],
        },
      ],
    });
  });

  it("valid: a single JSON.stringify() operand is not flagged", () => {
    cjsRuleTester.run("no-json-stringify-equality", noJsonStringifyEqualityRule, {
      valid: [`const isEmpty = JSON.stringify(value) === "{}";`, `const equal = serialized === JSON.stringify(value);`, `const equal = JSON.stringify(value) === serialized;`],
      invalid: [],
    });
  });

  it("valid: non-equality operators are not flagged", () => {
    cjsRuleTester.run("no-json-stringify-equality", noJsonStringifyEqualityRule, {
      valid: [`const longer = JSON.stringify(left) < JSON.stringify(right);`, `const joined = JSON.stringify(left) + JSON.stringify(right);`],
      invalid: [],
    });
  });

  it("valid: other JSON methods and non-JSON calls are not flagged", () => {
    cjsRuleTester.run("no-json-stringify-equality", noJsonStringifyEqualityRule, {
      valid: [`const equal = JSON.parse(left) === JSON.parse(right);`, `const equal = serialize(left) === serialize(right);`],
      invalid: [],
    });
  });

  it("valid: a shadowed JSON binding is ignored", () => {
    cjsRuleTester.run("no-json-stringify-equality", noJsonStringifyEqualityRule, {
      valid: [
        `function compare(left, right) {
  const JSON = { stringify: value => String(value) };
  return JSON.stringify(left) === JSON.stringify(right);
}`,
      ],
      invalid: [],
    });
  });
});
