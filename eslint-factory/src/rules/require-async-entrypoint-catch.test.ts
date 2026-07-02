import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireAsyncEntrypointCatchRule } from "./require-async-entrypoint-catch";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-async-entrypoint-catch", () => {
  it("uses the correct docs URL", () => {
    expect(requireAsyncEntrypointCatchRule.meta.docs.url).toBe(
      "https://github.com/github/gh-aw/tree/main/eslint-factory#require-async-entrypoint-catch",
    );
  });

  it("valid: non-async function call is not flagged", () => {
    cjsRuleTester.run("require-async-entrypoint-catch", requireAsyncEntrypointCatchRule, {
      valid: [
        // Synchronous main — no problem
        `function main() { return 42; }
if (require.main === module) { main(); }`,
      ],
      invalid: [],
    });
  });

  it("valid: async main chained with .catch() is not flagged", () => {
    cjsRuleTester.run("require-async-entrypoint-catch", requireAsyncEntrypointCatchRule, {
      valid: [
        `async function main() { return 42; }
if (require.main === module) { main().catch(err => { throw err; }); }`,

        `async function main() { return 42; }
main().catch(err => { process.exit(1); });`,
      ],
      invalid: [],
    });
  });

  it("valid: async main called with await inside an async context is not flagged", () => {
    cjsRuleTester.run("require-async-entrypoint-catch", requireAsyncEntrypointCatchRule, {
      valid: [
        // Inside an async IIFE
        `async function main() { return 42; }
(async () => { await main(); })();`,

        // Inside an async arrow function
        `async function main() { return 42; }
const run = async () => { main(); };`,
      ],
      invalid: [],
    });
  });

  it("valid: async function called as part of an expression (not a bare statement) is not flagged", () => {
    cjsRuleTester.run("require-async-entrypoint-catch", requireAsyncEntrypointCatchRule, {
      valid: [
        // Assigned to a variable — caller handles the Promise
        `async function main() { return 42; }
const p = main();`,

        // Passed as argument
        `async function main() { return 42; }
Promise.resolve().then(() => main());`,

        // main() is the object of a .then() chain
        `async function main() { return 42; }
main().then(() => {}).catch(err => { throw err; });`,
      ],
      invalid: [],
    });
  });

  it("invalid: bare async main() call outside async context is flagged", () => {
    cjsRuleTester.run("require-async-entrypoint-catch", requireAsyncEntrypointCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function main() { return 42; }
if (require.main === module) { main(); }`,
          errors: [
            {
              messageId: "requireCatch",
              data: { name: "main" },
              suggestions: [
                {
                  messageId: "addCatch",
                  output: `async function main() { return 42; }
if (require.main === module) { main().catch(err => { throw err; }); }`,
                },
              ],
            },
          ],
        },
        {
          // Top-level bare call
          code: `async function main() { return 42; }
main();`,
          errors: [
            {
              messageId: "requireCatch",
              data: { name: "main" },
              suggestions: [
                {
                  messageId: "addCatch",
                  output: `async function main() { return 42; }
main().catch(err => { throw err; });`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: bare call to other async entrypoints (run, start) is flagged", () => {
    cjsRuleTester.run("require-async-entrypoint-catch", requireAsyncEntrypointCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function run() { }
if (require.main === module) { run(); }`,
          errors: [
            {
              messageId: "requireCatch",
              data: { name: "run" },
              suggestions: [
                {
                  messageId: "addCatch",
                  output: `async function run() { }
if (require.main === module) { run().catch(err => { throw err; }); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("suggests chaining .catch(err => { throw err; })", () => {
    cjsRuleTester.run("require-async-entrypoint-catch", requireAsyncEntrypointCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function main() {}
main();`,
          errors: [
            {
              messageId: "requireCatch",
              suggestions: [
                {
                  messageId: "addCatch",
                  output: `async function main() {}
main().catch(err => { throw err; });`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
