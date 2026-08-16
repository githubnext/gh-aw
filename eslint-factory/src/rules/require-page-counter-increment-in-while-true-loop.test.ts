import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requirePageCounterIncrementInWhileTrueLoopRule } from "./require-page-counter-increment-in-while-true-loop";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-page-counter-increment-in-while-true-loop", () => {
  it("uses the correct docs URL", () => {
    expect(requirePageCounterIncrementInWhileTrueLoopRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-page-counter-increment-in-while-true-loop");
  });

  it("requires used page counters in terminating while true loops to advance", () => {
    ruleTester.run("require-page-counter-increment-in-while-true-loop", requirePageCounterIncrementInWhileTrueLoopRule, {
      valid: [
        `let page = 1; while (true) { api({ page }); if (done) break; page++; }`,
        `let page = 1; while (true) { api({ page }); if (done) break; page += 1; }`,
        `let page = 1; while (true) { api({ page }); if (done) break; page = page + 1; }`,
        `let page = 1; while (true) { api({ page }); }`,
        `let page = 1; while (true) { api({ currentPage: 1 }); break; }`,
        `let page = 1; while (true) { const page = 1; api({ page: 1 }); break; }`,
      ],
      invalid: [
        {
          code: `let page = 1; while (true) { api({ page }); if (done) break; }`,
          errors: [{ messageId: "requirePageCounterIncrement", data: { name: "page" } }],
        },
        {
          code: `let page = 1; while (true) { api({ page }); if (done) { break; } }`,
          errors: [{ messageId: "requirePageCounterIncrement", data: { name: "page" } }],
        },
        {
          code: `let page = 1; while (true) { api({ page }); if (done) break; page = 1; }`,
          errors: [{ messageId: "requirePageCounterIncrement", data: { name: "page" } }],
        },
        {
          code: `let page = 1; while (true) { api({ page }); if (done) break; page -= 1; }`,
          errors: [{ messageId: "requirePageCounterIncrement", data: { name: "page" } }],
        },
        {
          code: `let page = 1; while (true) { api({ page }); if (done) break; page += 0; }`,
          errors: [{ messageId: "requirePageCounterIncrement", data: { name: "page" } }],
        },
        {
          code: `let page = 1; while (true) { api({ page }); if (done) break; page += -1; }`,
          errors: [{ messageId: "requirePageCounterIncrement", data: { name: "page" } }],
        },
        {
          code: `let page = 1; while (true) { api({ page }); if (done) break; page--; }`,
          errors: [{ messageId: "requirePageCounterIncrement", data: { name: "page" } }],
        },
        {
          code: `let page = 1; while (true) { api({ page }); if (done) break; page = page - 1; }`,
          errors: [{ messageId: "requirePageCounterIncrement", data: { name: "page" } }],
        },
      ],
    });
  });
});
