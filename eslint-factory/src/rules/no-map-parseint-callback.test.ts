import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noMapParseIntCallbackRule } from "./no-map-parseint-callback";

const ruleTester = new RuleTester({ languageOptions: { ecmaVersion: 2022, sourceType: "commonjs" } });

describe("no-map-parseint-callback", () => {
  it("uses the correct docs URL", () => {
    expect(noMapParseIntCallbackRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-map-parseint-callback");
  });

  it("validates map(parseInt) usage", () => {
    ruleTester.run("no-map-parseint-callback", noMapParseIntCallbackRule, {
      valid: [`parts.map(Number);`, `parts.map(v => parseInt(v, 10));`, `parts.map(parseFloat);`, `function f(parseInt) { return a.map(parseInt); }`, `a.forEach(parseInt);`],
      invalid: [
        { code: `parts.map(parseInt);`, errors: [{ messageId: "noMapParseInt" }] },
        { code: `parts.map(Number.parseInt);`, errors: [{ messageId: "noMapParseInt" }] },
        { code: `s.split(".").map(parseInt, null);`, errors: [{ messageId: "noMapParseInt" }] },
      ],
    });
  });
});
