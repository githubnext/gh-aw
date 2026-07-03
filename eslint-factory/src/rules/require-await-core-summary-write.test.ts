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
