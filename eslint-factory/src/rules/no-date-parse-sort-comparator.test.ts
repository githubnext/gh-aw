import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noDateParseSortComparatorRule } from "./no-date-parse-sort-comparator";

const ruleTester = new RuleTester({ languageOptions: { ecmaVersion: 2022, sourceType: "commonjs" } });

describe("no-date-parse-sort-comparator", () => {
  it("uses the correct docs URL", () => {
    expect(noDateParseSortComparatorRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-date-parse-sort-comparator");
  });

  it("valid and invalid cases", () => {
    ruleTester.run("no-date-parse-sort-comparator", noDateParseSortComparatorRule, {
      valid: [
        `items.sort((a, b) => (Date.parse(b.t) || 0) - (Date.parse(a.t) || 0));`,
        `items.sort((a, b) => a.ms - b.ms);`,
        `const ms = Date.parse(x) - Date.now();`,
        `items.sort();`,
        `items.sort(compare);`,
        `items.sort((a, b) => { const x = Date.parse(a.t) - 1; return a.n - b.n; });`,
      ],
      invalid: [
        { code: `items.sort((a, b) => Date.parse(b.t) - Date.parse(a.t));`, errors: [{ messageId: "noDateParseSubtraction" }] },
        { code: `items.toSorted(function (a, b) { return Date.parse(a.t) - b.ms; });`, errors: [{ messageId: "noDateParseSubtraction" }] },
        { code: `items.sort((a, b) => { return a.ms - Date.parse(b.t); });`, errors: [{ messageId: "noDateParseSubtraction" }] },
      ],
    });
  });
});
