import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireFetchTryCatchRule } from "./require-fetch-try-catch";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

const esmRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "module",
  },
});

describe("require-fetch-try-catch", () => {
  it("valid: await fetch inside try block (CommonJS)", () => {
    cjsRuleTester.run("require-fetch-try-catch", requireFetchTryCatchRule, {
      valid: [
        `async function f() { try { const res = await fetch(url); } catch (e) {} }`,
        `async function f() { try { return await fetch(url, { method: "POST" }); } catch (e) {} }`,
        `async function f() { try { await fetch(url); } catch (e) { throw e; } }`,
      ],
      invalid: [],
    });
  });

  it("valid: await fetch inside try block (ES module)", () => {
    esmRuleTester.run("require-fetch-try-catch", requireFetchTryCatchRule, {
      valid: [`async function f() { try { const res = await fetch(url); } catch (e) {} }`],
      invalid: [],
    });
  });

  it("valid: non-fetch await calls are not flagged", () => {
    cjsRuleTester.run("require-fetch-try-catch", requireFetchTryCatchRule, {
      valid: [`async function f() { const res = await axios.get(url); }`, `async function f() { const data = await readFile(path); }`, `async function f() { await Promise.resolve(1); }`],
      invalid: [],
    });
  });

  it("invalid: await fetch outside try block (CommonJS)", () => {
    cjsRuleTester.run("require-fetch-try-catch", requireFetchTryCatchRule, {
      valid: [],
      invalid: [
        {
          // VariableDeclaration — error is reported but no suggestion (wrapping would put
          // subsequent uses of `res` outside the try block)
          code: `async function f() { const res = await fetch(url); }`,
          errors: [{ messageId: "requireTryCatch", suggestions: [] }],
        },
        {
          code: `async function f() { const res = await fetch("https://example.com", { method: "GET" }); }`,
          errors: [{ messageId: "requireTryCatch", suggestions: [] }],
        },
        {
          code: `async function f() { await fetch(url); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `async function f() { try {\n  await fetch(url);\n} catch (err) {\n  // TODO: handle fetch network failure (TypeError on DNS/connection errors).\n  throw new Error(\n    "fetch failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: await fetch outside try block (ES module)", () => {
    esmRuleTester.run("require-fetch-try-catch", requireFetchTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() { const res = await fetch(url); }`,
          errors: [{ messageId: "requireTryCatch", suggestions: [] }],
        },
      ],
    });
  });

  it("valid: await fetch inside nested try block", () => {
    cjsRuleTester.run("require-fetch-try-catch", requireFetchTryCatchRule, {
      valid: [
        `async function f() {
          try {
            for (let i = 0; i < 3; i++) {
              const res = await fetch(url);
            }
          } catch (e) {}
        }`,
      ],
      invalid: [],
    });
  });

  it("invalid: await fetch in loop outside try block", () => {
    cjsRuleTester.run("require-fetch-try-catch", requireFetchTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() { for (let i = 0; i < 3; i++) { const res = await fetch(url); } }`,
          errors: [{ messageId: "requireTryCatch" }],
        },
      ],
    });
  });

  it("valid: locally bound fetch is not flagged (shadows the global)", () => {
    cjsRuleTester.run("require-fetch-try-catch", requireFetchTryCatchRule, {
      valid: [
        // fetch as a function parameter
        `async function f(fetch) { await fetch(url); }`,
        // fetch as a local variable
        `async function f() { const fetch = async (u) => null; await fetch(url); }`,
      ],
      invalid: [],
    });
  });

  it("invalid: await fetch inside named function declaration nested in outer try block", () => {
    cjsRuleTester.run("require-fetch-try-catch", requireFetchTryCatchRule, {
      valid: [],
      invalid: [
        {
          // FunctionDeclaration inside outer try — the outer try cannot catch the rejected promise;
          // VariableDeclaration context so no suggestion is emitted (wrapping would move uses out of scope)
          code: `try { async function later() { const r = await fetch(url); } setTimeout(later, 0); } catch (e) {}`,
          errors: [{ messageId: "requireTryCatch", suggestions: [] }],
        },
        {
          // Stored async arrow function inside outer try — same problem
          code: `try { const later = async () => { const r = await fetch(url); }; } catch (e) {}`,
          errors: [{ messageId: "requireTryCatch", suggestions: [] }],
        },
        {
          // Stored async function expression inside outer try
          code: `try { const later = async function() { const r = await fetch(url); }; } catch (e) {}`,
          errors: [{ messageId: "requireTryCatch", suggestions: [] }],
        },
      ],
    });
  });
});
