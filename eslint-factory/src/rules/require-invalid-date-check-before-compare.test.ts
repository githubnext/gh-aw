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
          errors: [{ messageId: "requireInvalidDateCheck", data: { subject: "Both operands of this comparison", operator: ">", getTimeTarget: "each value" } }],
        },
      ],
    });
  });

  it("invalid: only one side of the comparison is an unvalidated inline new Date()", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `if (new Date(run.started_at) > threshold) { doIt(); }`,
          errors: [{ messageId: "requireInvalidDateCheck", data: { subject: "An inline `new Date(...)` expression", operator: ">", getTimeTarget: "it" } }],
        },
      ],
    });
  });

  it("invalid: Date.now() arithmetic with a non-literal operand is not guaranteed finite", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `const cutoff = new Date(Date.now() - windowMs); if (cutoff < other) { doIt(); }`,
          errors: [{ messageId: "requireInvalidDateCheck", data: { subject: "'cutoff'", operator: "<", getTimeTarget: "cutoff" } }],
        },
      ],
    });
  });

  it("invalid: same variable name declared in a different function scope is not treated as validated", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `function a() { const d = new Date(x); if (Number.isNaN(d.getTime())) return; if (d > t) {} } function b() { const d = new Date(y); if (d > t) {} }`,
          errors: [{ messageId: "requireInvalidDateCheck", data: { subject: "'d'", operator: ">", getTimeTarget: "d" } }],
        },
      ],
    });
  });

  it("invalid: guard written after the comparison does not protect it", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `const d = new Date(input); if (d > threshold) { doIt(); } if (Number.isNaN(d.getTime())) { return; }`,
          errors: [{ messageId: "requireInvalidDateCheck", data: { subject: "'d'", operator: ">", getTimeTarget: "d" } }],
        },
      ],
    });
  });

  it("invalid: guard nested in an unrelated conditional branch does not protect the comparison", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `const d = new Date(input); if (someUnrelatedFlag) { if (Number.isNaN(d.getTime())) { return; } } if (d > threshold) { doIt(); }`,
          errors: [{ messageId: "requireInvalidDateCheck", data: { subject: "'d'", operator: ">", getTimeTarget: "d" } }],
        },
        {
          code: `const d = new Date(input); for (const item of items) { if (Number.isNaN(d.getTime())) { return; } } if (d > threshold) { doIt(); }`,
          errors: [{ messageId: "requireInvalidDateCheck", data: { subject: "'d'", operator: ">", getTimeTarget: "d" } }],
        },
      ],
    });
  });

  it("invalid: a non-exiting invalid-date guard does not protect the comparison", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `const d = new Date(input); if (Number.isNaN(d.getTime())) { core.warning("bad date"); } if (d > threshold) { doIt(); }`,
          errors: [{ messageId: "requireInvalidDateCheck", data: { subject: "'d'", operator: ">", getTimeTarget: "d" } }],
        },
      ],
    });
  });

  it("valid: validated with Number.isNaN(d.getTime()) before comparison", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [
        `const d = new Date(input); if (Number.isNaN(d.getTime())) { throw new Error("bad date"); } if (d < threshold) { doIt(); }`,
        `const d = new Date(input); if (Number.isNaN(d.getTime())) { return; } if (d >= threshold) { doIt(); }`,
        `for (const item of items) { const d = new Date(input); if (Number.isNaN(d.getTime())) { continue; } if (d > item.at) { doIt(); } }`,
        `for (const item of items) { const d = new Date(input); if (Number.isNaN(d.getTime())) { break; } if (d > item.at) { doIt(); } }`,
        `const d = new Date(input); if (!Number.isNaN(d.getTime()) && d > threshold) { doIt(); }`,
        `const d = new Date(input); if (isNaN(d.getTime()) || d > threshold) { doIt(); }`,
        `const d = new Date(input); if (Number.isNaN(d.getTime())) { throw new Error("bad date"); } else if (d > threshold) { doIt(); }`,
        `const d = new Date(input); if (Number.isNaN(d.getTime())) { return; } for (const item of items) { if (d > item.at) { doIt(); } }`,
      ],
      invalid: [],
    });
  });

  it("valid: new Date() with no args or the exact Date.now() call are always finite", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [`const now = new Date(); if (now > threshold) { doIt(); }`, `const cutoff = new Date(Date.now()); if (cutoff < other) { doIt(); }`],
      invalid: [],
    });
  });

  it("valid: same variable name validated independently in each function scope", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [`function a() { const d = new Date(x); if (Number.isNaN(d.getTime())) return; if (d > t) {} } function b() { const d = new Date(y); if (Number.isNaN(d.getTime())) return; if (d > t) {} }`],
      invalid: [],
    });
  });

  it("valid: date variable used without relational comparison", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [`const d = new Date(input); core.info(d.toISOString());`, `const d = new Date(input); if (d.getTime() === other) { doIt(); }`],
      invalid: [],
    });
  });

  it("invalid: d.getTime() compared relationally without a NaN check", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `const d = new Date(input); if (d.getTime() < threshold) { doIt(); }`,
          errors: [{ messageId: "requireInvalidDateCheck", data: { subject: "'d'", operator: "<", getTimeTarget: "d" } }],
        },
        {
          code: `if (new Date(run.started_at).getTime() > threshold) { doIt(); }`,
          errors: [{ messageId: "requireInvalidDateCheck", data: { subject: "An inline `new Date(...)` expression", operator: ">", getTimeTarget: "it" } }],
        },
      ],
    });
  });

  it("valid: d.getTime() comparison guarded by Number.isNaN(d.getTime())", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [
        `const d = new Date(input); if (Number.isNaN(d.getTime())) return; if (d.getTime() < threshold) { doIt(); }`,
        `const expirationDate = getExpiration(); if (expirationDate && expirationDate.getTime() <= Date.now()) { doIt(); }`,
      ],
      invalid: [],
    });
  });

  it("invalid: Date.parse(x) compared without a Number.isFinite/isNaN guard (check_daily_aic_workflow_guardrail.cjs pattern)", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `const createdAtMs = Date.parse(run.created_at || ""); if (createdAtMs < cutoffMs) { hasMore = false; }`,
          errors: [{ messageId: "requireInvalidDateCheckParse", data: { subject: "'createdAtMs'", operator: "<", parseTarget: "createdAtMs" } }],
        },
      ],
    });
  });

  it("invalid: inline Date.parse(...) compared without a guard", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `if (Date.parse(run.started_at) > threshold) { doIt(); }`,
          errors: [{ messageId: "requireInvalidDateCheckParse", data: { subject: "An inline `Date.parse(...)` expression", operator: ">", parseTarget: "it" } }],
        },
      ],
    });
  });

  it("invalid: two Date.parse(...) values compared directly", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `const a = Date.parse(timestamp); const b = Date.parse(threshold); return a >= b;`,
          errors: [{ messageId: "requireInvalidDateCheckParse", data: { subject: "Both operands of this comparison", operator: ">=", parseTarget: "each value" } }],
        },
      ],
    });
  });

  it("valid: Date.parse(x) validated with Number.isFinite(name) before comparison (handle_agent_failure.cjs pattern)", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [`const createdMs = Date.parse(createdAt); return Number.isFinite(createdMs) && createdMs >= windowStartMs;`],
      invalid: [],
    });
  });

  it("valid: Date.parse(x) validated with an exiting !Number.isFinite(name) guard (evaluate_outcomes.cjs pattern)", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [`const a = Date.parse(timestamp); if (!Number.isFinite(a)) return false; return a >= threshold;`],
      invalid: [],
    });
  });

  it("valid: Date.parse(x) validated with the global isFinite(name) before comparison", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [`const a = Date.parse(timestamp); if (isFinite(a) && a >= threshold) { doIt(); }`],
      invalid: [],
    });
  });

  it("valid: Date.parse(x) validated with !Number.isNaN(name) before comparison", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [`const a = Date.parse(timestamp); if (!Number.isNaN(a) && a >= threshold) { doIt(); }`],
      invalid: [],
    });
  });

  it("invalid: guard written after the Date.parse comparison does not protect it", () => {
    cjsRuleTester.run("require-invalid-date-check-before-compare", requireInvalidDateCheckBeforeCompareRule, {
      valid: [],
      invalid: [
        {
          code: `const a = Date.parse(timestamp); if (a > threshold) { doIt(); } if (!Number.isFinite(a)) { return false; }`,
          errors: [{ messageId: "requireInvalidDateCheckParse", data: { subject: "'a'", operator: ">", parseTarget: "a" } }],
        },
      ],
    });
  });
});
