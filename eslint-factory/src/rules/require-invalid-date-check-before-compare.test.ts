import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireInvalidDateCheckBeforeCompareRule } from "./require-invalid-date-check-before-compare";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-invalid-date-check-before-compare", () => {
  it("uses the correct docs URL", () => {
    expect(requireInvalidDateCheckBeforeCompareRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-invalid-date-check-before-compare");
  });

  it("invalid: new Date(run.created_at) compared without NaN check (check_rate_limit.cjs pattern)", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `const runCreatedAt = new Date(run.created_at); if (runCreatedAt < thresholdTime) { hasMore = false; }`,
          errors: [{ messageId: "requireInvalidDateCheck", data: { subject: "'runCreatedAt'", operator: "<", getTimeTarget: "runCreatedAt" } }],
        },
      ],
    });
  });

  it("invalid: two new Date() values compared directly (check_runs_helpers.cjs pattern)", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `if (new Date(run.started_at ?? 0) > new Date(existing.started_at ?? 0)) { latestByName.set(run.name, run); }`,
          errors: [
            { messageId: "requireInvalidDateCheck", data: { subject: "An inline `new Date(...)` expression", operator: ">", getTimeTarget: "it" } },
            { messageId: "requireInvalidDateCheck", data: { subject: "An inline `new Date(...)` expression", operator: ">", getTimeTarget: "it" } },
          ],
        },
      ],
    });
  });

  it("valid: validated with Number.isNaN(d.getTime()) before comparison", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [
        `const d = new Date(input); if (Number.isNaN(d.getTime())) { throw new Error("bad date"); } if (d < threshold) { doIt(); }`,
        `const d = new Date(input); if (!Number.isNaN(d.getTime()) && d > threshold) { doIt(); }`,
        `const d = new Date(input); if (isNaN(d.getTime())) { return; } if (d >= threshold) { doIt(); }`,
      ],
      invalid: [],
    });
  });

  it("valid: new Date() with no args or Date.now()-derived args are always finite", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [`const now = new Date(); if (now > threshold) { doIt(); }`, `const cutoff = new Date(Date.now() - windowMs); if (cutoff < other) { doIt(); }`],
      invalid: [],
    });
  });

  it("valid: date variable used without relational comparison", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [`const d = new Date(input); core.info(d.toISOString());`, `const d = new Date(input); if (d.getTime() === other) { doIt(); }`],
      invalid: [],
    });
  });
});
