import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireFetchResponseBodyTryCatchRule } from "./require-fetch-response-body-try-catch";

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

describe("require-fetch-response-body-try-catch", () => {
  it("valid: direct chain wrapped in try/catch passes (CommonJS)", () => {
    cjsRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [`async function f() { try { const data = await fetch(url).json(); } catch (e) {} }`, `async function f() { try { const data = await fetch(url).text(); } catch (e) {} }`],
      invalid: [],
    });
  });

  it("valid: variable-resolved response wrapped in try/catch passes (CommonJS)", () => {
    cjsRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [
        `async function f() {
          try {
            const response = await fetch(url);
            const data = await response.json();
          } catch (e) {}
        }`,
      ],
      invalid: [],
    });
  });

  it("valid: response obtained via try-wrapped fetch, .json() call outside is still flagged only when unwrapped (documenting scope)", () => {
    // The rule only tracks whether the *assignment itself* came from a bare await fetch;
    // it does not attempt to prove that the read site is unreachable when the fetch threw.
    // This case documents that a response variable NOT sourced from a direct await fetch call
    // (e.g. a mocked/stubbed response) is left alone.
    cjsRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [`async function f(response) { const data = await response.json(); }`, `async function f() { const response = makeResponse(); const data = await response.json(); }`],
      invalid: [],
    });
  });

  it("valid: variable reassigned to a non-fetch-derived value before the read is not flagged (CommonJS)", () => {
    // The reaching write (the last write before the read) is the safe reassignment,
    // not the earlier bare `await fetch(...)` write — so this path is not risky.
    cjsRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [
        `async function f(needsFallback) {
          let response = await fetch(url1);
          if (needsFallback) {
            response = await getCachedResponse();
          }
          const data = await response.json();
        }`,
      ],
      invalid: [],
    });
  });

  it("invalid: single const-aliased bare-fetch response is flagged (CommonJS)", () => {
    cjsRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() {
            const response = await fetch(url);
            const r = response;
            const data = await r.json();
          }`,
          errors: [
            {
              message:
                "Wrap r.json() in try/catch — even after explicit HTTP-error handling (for example `if (!response.ok) throw ...`), " +
                "reading a fetch() Response body can still reject (malformed JSON, truncated/errored stream). Without this call-site try/catch, " +
                "you lose the original parse error context and get a generic, harder-to-diagnose stack instead of a specific message with `{ cause }`.",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `async function f() {
            const response = await fetch(url);
            const r = response;
            try {
              const data = await r.json();
            } catch (err) {
              // TODO: handle a malformed/errored fetch response body for this call.
              throw new Error(
                "Failed to read fetch response json(): " + (err instanceof Error ? err.message : String(err)),
                { cause: err },
              );
            }
          }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: non-fetch object methods named json/text are ignored (CommonJS)", () => {
    cjsRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [`async function f() { const data = await someOtherThing.json(); }`, `async function f() { const data = await someOtherThing.text(); }`],
      invalid: [],
    });
  });

  it("invalid: direct chain not wrapped in try/catch is flagged (CommonJS)", () => {
    cjsRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() { const data = await fetch(url).json(); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `async function f() { try {\n  const data = await fetch(url).json();\n} catch (err) {\n  // TODO: handle a malformed/errored fetch response body for this call.\n  throw new Error(\n    "Failed to read fetch response json(): " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
        {
          code: `async function f() { const data = await fetch(url).text(); }`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `async function f() { try {\n  const data = await fetch(url).text();\n} catch (err) {\n  // TODO: handle a malformed/errored fetch response body for this call.\n  throw new Error(\n    "Failed to read fetch response text(): " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: variable resolved from bare await fetch, body read outside try is flagged (CommonJS)", () => {
    cjsRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() {
            const response = await fetch(url);
            const data = await response.json();
          }`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `async function f() {
            const response = await fetch(url);
            try {
              const data = await response.json();
            } catch (err) {
              // TODO: handle a malformed/errored fetch response body for this call.
              throw new Error(
                "Failed to read fetch response json(): " + (err instanceof Error ? err.message : String(err)),
                { cause: err },
              );
            }
          }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: variable declaration used later is reported without suggestion (CommonJS)", () => {
    cjsRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() {
            const response = await fetch(url);
            const payload = await response.json();
            const pageArtifacts = Array.isArray(payload?.artifacts) ? payload.artifacts : [];
            return pageArtifacts;
          }`,
          errors: [{ messageId: "requireTryCatch", suggestions: [] }],
        },
      ],
    });
  });

  it("invalid: export declaration is reported without suggestion (ES module)", () => {
    esmRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `export const payload = await fetch(url).json();`,
          errors: [{ messageId: "requireTryCatch", suggestions: [] }],
        },
      ],
    });
  });

  it("invalid: for-loop initializer declaration is reported without suggestion (CommonJS)", () => {
    cjsRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() { for (let payload = await fetch(url).json(); payload; payload = null) {} }`,
          errors: [{ messageId: "requireTryCatch", suggestions: [] }],
        },
      ],
    });
  });

  it("invalid: variable resolved from bare await fetch, body read outside try is flagged (ES module)", () => {
    esmRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() {
            const response = await fetch(url);
            const data = await response.json();
          }`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `async function f() {
            const response = await fetch(url);
            try {
              const data = await response.json();
            } catch (err) {
              // TODO: handle a malformed/errored fetch response body for this call.
              throw new Error(
                "Failed to read fetch response json(): " + (err instanceof Error ? err.message : String(err)),
                { cause: err },
              );
            }
          }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: fetch inside try but .json() read after try block ends is flagged (CommonJS)", () => {
    cjsRuleTester.run("require-fetch-response-body-try-catch", requireFetchResponseBodyTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `async function f() {
            let response;
            try {
              response = await fetch(url);
            } catch (e) {
              throw e;
            }
            const data = await response.json();
          }`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `async function f() {
            let response;
            try {
              response = await fetch(url);
            } catch (e) {
              throw e;
            }
            try {
              const data = await response.json();
            } catch (err) {
              // TODO: handle a malformed/errored fetch response body for this call.
              throw new Error(
                "Failed to read fetch response json(): " + (err instanceof Error ? err.message : String(err)),
                { cause: err },
              );
            }
          }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
