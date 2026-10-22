import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noSplitFilterTrimWithoutTrimMapRule } from "./no-split-filter-trim-without-trim-map";

const ruleTester = new RuleTester({ languageOptions: { ecmaVersion: 2022, sourceType: "commonjs" } });

describe("no-split-filter-trim-without-trim-map", () => {
  it("uses the correct docs URL", () => {
    expect(noSplitFilterTrimWithoutTrimMapRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-split-filter-trim-without-trim-map");
  });

  it("validates split/filter(trim) usage", () => {
    ruleTester.run("no-split-filter-trim-without-trim-map", noSplitFilterTrimWithoutTrimMapRule, {
      valid: [
        `s.split(",").map(v => v.trim()).filter(Boolean);`,
        `s.split("\\n").filter(l => l.trim());`,
        `s.split(",").filter(v => v.length > 0);`,
        `s.filter(v => v.trim());`,
        `s.split(",").filter(v => v.trim() && other(v));`,
        `s.split(",").filter(v => !v.trim());`,
        `s.split(",").filter(v => !!!v.trim());`,
        `s.split(",").filter(v => v.trim().length === 0);`,
        `s.split(",").filter(v => v.trim() === "admin");`,
        `s.split(",").filter(v => v === v.trim());`,
        `s.split(",").filter(v => v.trim().length === min);`,
        `s.split(",").filter(v => v.trim() !== "admin");`,
        `s.split(",").filter(v => v.trim()).map(v => v.trim());`,
        `s.split(",").filter(v => v.trim()).map(v => { return v.trim(); });`,
      ],
      invalid: [
        { code: `s.split(",").filter(n => n.trim());`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
        { code: `process.env.X?.split(",").filter(p => p.trim());`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
        { code: `s.split(",").filter(p => p.trim().length > 0);`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
        { code: `s.split(",").filter(p => 0 < p.trim().length);`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
        { code: `s.split(",").filter(p => p.trim().length >= 1);`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
        { code: `s.split(",").filter(p => p.trim().length !== 0);`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
        { code: `s.split(",").filter(p => p.trim() !== "");`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
        { code: `s.split(",").filter(p => "" != p.trim());`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
        { code: `s.split(",").filter(p => !!p.trim());`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
        { code: `s.split(",").filter(function (p) { return p.trim(); });`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
        { code: `s.split(";").filter(p => { return !!p.trim(); });`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
        { code: `s.split(",").filter(p => p.trim()).map(async p => p.trim());`, errors: [{ messageId: "filterTrimKeepsWhitespace" }] },
      ],
    });
  });
});
