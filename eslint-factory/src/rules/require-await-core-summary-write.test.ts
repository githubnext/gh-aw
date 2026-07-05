import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireAwaitCoreSummaryWriteRule } from "./require-await-core-summary-write";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-await-core-summary-write", () => {
  it("uses the correct docs URL", () => {
    expect(requireAwaitCoreSummaryWriteRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-await-core-summary-write");
  });

  it("valid: awaited calls are not flagged", () => {
    cjsRuleTester.run("require-await-core-summary-write", requireAwaitCoreSummaryWriteRule, {
      valid: [
        `async function f() { await core.summary.write(); }`,
        `async function f() { await core.summary.addRaw(x).write(); }`,
        `async function f() { await core.summary.addHeading("h").addRaw(x).write(); }`,
        `async function f() { await coreObj.summary.write(); }`,
      ],
      invalid: [],
    });
  });

  it("valid: returned and assigned calls are not flagged", () => {
    cjsRuleTester.run("require-await-core-summary-write", requireAwaitCoreSummaryWriteRule, {
      valid: [
        `async function f() { return core.summary.write(); }`,
        `async function f() { const p = core.summary.write(); }`,
        `async function f() { return core.summary.addRaw(x).write(); }`,
        // assignment expression (without declaration) also propagates the Promise
        `async function f() { let p; p = core.summary.write(); }`,
      ],
      invalid: [],
    });
  });

  it("invalid: bare core.summary.write() is flagged", () => {
    cjsRuleTester.run("require-await-core-summary-write", requireAwaitCoreSummaryWriteRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() { core.summary.write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function f() { await core.summary.write(); }` }] }],
        },
        {
          code: `const f = async () => { core.summary.write(); };`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `const f = async () => { await core.summary.write(); };` }] }],
        },
        {
          code: `const f = async function() { core.summary.write(); };`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `const f = async function() { await core.summary.write(); };` }] }],
        },
      ],
    });
  });

  it("invalid: chained core.summary.addRaw(x).write() is flagged", () => {
    cjsRuleTester.run("require-await-core-summary-write", requireAwaitCoreSummaryWriteRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() { core.summary.addRaw(summary).write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function f() { await core.summary.addRaw(summary).write(); }` }] }],
        },
        {
          code: `async function f() { core.summary.addHeading("Title").addRaw(body).write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function f() { await core.summary.addHeading("Title").addRaw(body).write(); }` }] }],
        },
      ],
    });
  });

  it("invalid: coreObj alias and computed access are flagged", () => {
    cjsRuleTester.run("require-await-core-summary-write", requireAwaitCoreSummaryWriteRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() { coreObj.summary.write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function f() { await coreObj.summary.write(); }` }] }],
        },
        {
          code: `async function f() { core.summary["write"](); }`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function f() { await core.summary["write"](); }` }] }],
        },
      ],
    });
  });

  it("valid: unrelated .write() calls are not flagged", () => {
    cjsRuleTester.run("require-await-core-summary-write", requireAwaitCoreSummaryWriteRule, {
      valid: [`fs.write(fd, buffer);`, `stream.write(data);`, `core.info("hello");`, `foo.bar.write();`, `fs.summary.write();`, `db.summary.write();`, `foo.bar.summary.write();`],
      invalid: [],
    });
  });

  it("valid: identifiers outside CORE_ALIASES are not flagged even with .summary.write() chain", () => {
    // Codifies the tightened heuristic: only exact known aliases ("core", "coreObj") are matched.
    // Objects that merely start with "core" (e.g. coreCache, coreData) must not be flagged.
    cjsRuleTester.run("require-await-core-summary-write", requireAwaitCoreSummaryWriteRule, {
      valid: [`async function f() { coreCache.summary.write(); }`, `async function f() { coreData.summary.write(); }`, `async function f() { coreference.summary.write(); }`],
      invalid: [],
    });
  });

  it("invalid: flagged outside async function — no suggestion offered", () => {
    cjsRuleTester.run("require-await-core-summary-write", requireAwaitCoreSummaryWriteRule, {
      valid: [],
      invalid: [
        {
          // Top-level call: flagged, but no suggestion (await is not valid here)
          code: `core.summary.write();`,
          errors: [{ messageId: "requireAwait", suggestions: [] }],
        },
        {
          // Inside a non-async function: flagged, but no suggestion
          code: `function f() { core.summary.write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [] }],
        },
        {
          code: `const f = () => { core.summary.write(); };`,
          errors: [{ messageId: "requireAwait", suggestions: [] }],
        },
      ],
    });
  });

  it("invalid: local alias `const summary = core.summary` is flagged", () => {
    cjsRuleTester.run("require-await-core-summary-write", requireAwaitCoreSummaryWriteRule, {
      valid: [
        // Awaited alias calls are safe
        `async function f() { const summary = core.summary; await summary.write(); }`,
        // Re-assigned let — alias source is unknown after reassignment; not flagged
        `async function f() { let summary = core.summary; summary = other; summary.write(); }`,
        // Non-core initializer — not a core.summary alias
        `async function f() { const summary = fs.summary; summary.write(); }`,
        // Alias without .write() — addRaw alone must NOT be flagged
        `async function f() { const { summary } = core; summary.addRaw("x"); }`,
        // Chained init: core.summary.addRaw(x) still returns Summary — alias is valid; awaited call is safe
        `async function f() { const s = core.summary.addRaw("x"); await s.write(); }`,
        // Initializer is core.summary.write() — write() returns Promise<Summary>, not Summary; not treated as alias
        `async function f() { const s = core.summary.write(); s.write(); }`,
      ],
      invalid: [
        // Corpus pattern 1: const summary = core.summary; summary.write();
        {
          code: `async function f() { const summary = core.summary; summary.write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function f() { const summary = core.summary; await summary.write(); }` }] }],
        },
        // let variant (e.g. check_workflow_timestamp_api.cjs)
        {
          code: `async function f() { let summary = core.summary; summary.write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function f() { let summary = core.summary; await summary.write(); }` }] }],
        },
        // Corpus pattern 2: const { summary } = core; summary.addRaw(x).write();
        {
          code: `async function f() { const { summary } = core; summary.addRaw(x).write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function f() { const { summary } = core; await summary.addRaw(x).write(); }` }] }],
        },
        // Destructure with rename: const { summary: s } = core; s.write();
        {
          code: `async function f() { const { summary: s } = core; s.write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function f() { const { summary: s } = core; await s.write(); }` }] }],
        },
        // coreObj alias: const summary = coreObj.summary; summary.write();
        {
          code: `async function f() { const summary = coreObj.summary; summary.write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function f() { const summary = coreObj.summary; await summary.write(); }` }] }],
        },
        // coreObj destructuring: const { summary } = coreObj; summary.write();
        {
          code: `async function f() { const { summary } = coreObj; summary.write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function f() { const { summary } = coreObj; await summary.write(); }` }] }],
        },
        // Cross-scope: alias declared in outer scope, write() called inside inner async function
        {
          code: `async function outer() {\n  const summary = core.summary;\n  return (async function inner() { summary.write(); })();\n}`,
          errors: [{ messageId: "requireAwait", suggestions: [{ messageId: "addAwait", output: `async function outer() {\n  const summary = core.summary;\n  return (async function inner() { await summary.write(); })();\n}` }] }],
        },
        // Outside async: flagged, no suggestion
        {
          code: `function f() { const summary = core.summary; summary.write(); }`,
          errors: [{ messageId: "requireAwait", suggestions: [] }],
        },
      ],
    });
  });

  it("suggestion: inserts 'await ' before the expression", () => {
    cjsRuleTester.run("require-await-core-summary-write", requireAwaitCoreSummaryWriteRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() { core.summary.write(); }`,
          errors: [
            {
              messageId: "requireAwait",
              suggestions: [{ messageId: "addAwait", output: `async function f() { await core.summary.write(); }` }],
            },
          ],
        },
        {
          code: `async function f() { core.summary.addRaw(summary).write(); }`,
          errors: [
            {
              messageId: "requireAwait",
              suggestions: [{ messageId: "addAwait", output: `async function f() { await core.summary.addRaw(summary).write(); }` }],
            },
          ],
        },
        {
          code: `async function f() { coreObj.summary.write(); }`,
          errors: [
            {
              messageId: "requireAwait",
              suggestions: [{ messageId: "addAwait", output: `async function f() { await coreObj.summary.write(); }` }],
            },
          ],
        },
        {
          code: `(async () => { core.summary.write(); })()`,
          errors: [
            {
              messageId: "requireAwait",
              suggestions: [{ messageId: "addAwait", output: `(async () => { await core.summary.write(); })()` }],
            },
          ],
        },
      ],
    });
  });
});
