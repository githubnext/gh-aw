import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireFetchTimeoutRule } from "./require-fetch-timeout";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-fetch-timeout", () => {
  it("valid: fetch with AbortSignal.timeout", () => {
    cjsRuleTester.run("require-fetch-timeout", requireFetchTimeoutRule, {
      valid: [
        `async function f() { const res = await fetch(url, { signal: AbortSignal.timeout(10000) }); }`,
        `async function f() { const res = await fetch(url, { method: "POST", signal: ac.signal }); }`,
        `async function f() { const res = await globalThis.fetch(url, { signal: AbortSignal.timeout(10000) }); }`,
      ],
      invalid: [],
    });
  });

  it("valid: fetch with spread options or identifier options object (unresolvable statically)", () => {
    cjsRuleTester.run("require-fetch-timeout", requireFetchTimeoutRule, {
      valid: [`async function f() { const res = await fetch(url, ...opts); }`, `async function f() { const res = await fetch(url, options); }`, `async function f() { const res = await fetch(url, { ...baseOptions }); }`],
      invalid: [],
    });
  });

  it("valid: non-fetch calls are not flagged", () => {
    cjsRuleTester.run("require-fetch-timeout", requireFetchTimeoutRule, {
      valid: [
        `async function f() { const res = await axios.get(url); }`,
        `function fetch2() { return 1; }`,
        `obj.fetch(url);`,
        `async function f(fetch) { return fetch(url); }`,
        `async function f() { const fetch = (u) => Promise.resolve(u); return fetch(url); }`,
      ],
      invalid: [],
    });
  });

  it("invalid: fetch with no options argument", () => {
    cjsRuleTester.run("require-fetch-timeout", requireFetchTimeoutRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() { const res = await fetch(url); }`,
          errors: [{ messageId: "requireSignal" }],
        },
      ],
    });
  });

  it("invalid: fetch with options object missing signal", () => {
    cjsRuleTester.run("require-fetch-timeout", requireFetchTimeoutRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() { const res = await fetch(url, { method: "POST", headers: { "Content-Type": "application/json" } }); }`,
          errors: [{ messageId: "requireSignal" }],
        },
        {
          code: `async function f() { const res = await fetch(url, { method: "GET" }); }`,
          errors: [{ messageId: "requireSignal" }],
        },
        {
          code: `async function f() { const res = await globalThis.fetch(url, { method: "GET" }); }`,
          errors: [{ messageId: "requireSignal" }],
        },
        {
          code: `async function f() { const res = await global["fetch"](url, { method: "GET" }); }`,
          errors: [{ messageId: "requireSignal" }],
        },
        {
          code: `async function f() { const res = await fetch(url, { signal: null }); }`,
          errors: [{ messageId: "requireSignal" }],
        },
        {
          code: `async function f() { const res = await fetch(url, { signal: undefined }); }`,
          errors: [{ messageId: "requireSignal" }],
        },
        {
          code: `async function f() { const res = await fetch(url, { signal: void 0 }); }`,
          errors: [{ messageId: "requireSignal" }],
        },
        {
          code: `async function f() { const res = await fetch(url, undefined); }`,
          errors: [{ messageId: "requireSignal" }],
        },
        {
          code: `async function f() { const res = await fetch(url, null); }`,
          errors: [{ messageId: "requireSignal" }],
        },
      ],
    });
  });
});
