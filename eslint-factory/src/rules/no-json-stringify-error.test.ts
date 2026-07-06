import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noJsonStringifyErrorRule } from "./no-json-stringify-error";

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

describe("no-json-stringify-error", () => {
  it("valid: JSON.stringify on a non-caught variable is not flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [`const obj = { a: 1 }; JSON.stringify(obj);`, `JSON.stringify({ message: "hello" });`, `JSON.stringify("a string");`, `const data = fetchData(); JSON.stringify(data);`],
      invalid: [],
    });
  });

  it("valid: JSON.stringify on a non-error variable inside a catch block is not flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [`try { f(); } catch (err) { const data = { a: 1 }; JSON.stringify(data); }`, `try { f(); } catch (err) { JSON.stringify(someOtherVar); }`, `try { f(); } catch (err) { JSON.stringify({ message: err.message }); }`],
      invalid: [],
    });
  });

  it("valid: JSON.stringify with explicit error properties is not flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [`try { f(); } catch (err) { JSON.stringify({ message: err.message, stack: err.stack }); }`, `try { f(); } catch (err) { JSON.stringify({ error: String(err) }); }`],
      invalid: [],
    });
  });

  it("valid: JSON.stringify on catch param that shadows outer scope is not flagged outside the catch", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [
        `const err = { a: 1 }; try { f(); } catch (err) { } JSON.stringify(err);`,
        `try { f(); } catch (err) { [1].forEach(function(err) { JSON.stringify(err); }); }`,
        `try { f(); } catch (err) { items.map(err => JSON.stringify(err)); }`,
      ],
      invalid: [],
    });
  });

  it("valid: JSON.stringify on promise .catch() non-error param is not flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [`p.catch(function(err) { JSON.stringify(otherVar); })`, `p.catch(err => JSON.stringify(notErr))`],
      invalid: [],
    });
  });

  it("valid: bare catch {} without binding is not flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [`try { f(); } catch { JSON.stringify(someObj); }`],
      invalid: [],
    });
  });

  it("invalid: JSON.stringify(err) in catch block is flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { core.error(JSON.stringify(err)); }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { core.error(String(err)); }`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "err" },
                  output: 'try { f(); } catch (err) { core.error((String(err) + "\\n" + (err?.stack ?? ""))); }',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: JSON.stringify(error, null, 2) in catch block is flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (error) { core.error(\`details: \${JSON.stringify(error, null, 2)}\`); }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "error" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "error" },
                  output: `try { f(); } catch (error) { core.error(\`details: \${String(error)}\`); }`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "error" },
                  output: 'try { f(); } catch (error) { core.error(`details: ${(String(error) + "\\n" + (error?.stack ?? ""))}`); }',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: JSON.stringify(err) in promise .catch() arrow callback is flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `p.catch(err => core.error(JSON.stringify(err)));`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `p.catch(err => core.error(String(err)));`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "err" },
                  output: 'p.catch(err => core.error((String(err) + "\\n" + (err?.stack ?? ""))));',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: JSON.stringify(err) in promise .catch() function callback is flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `p.catch(function(err) { core.error(JSON.stringify(err)); });`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `p.catch(function(err) { core.error(String(err)); });`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "err" },
                  output: 'p.catch(function(err) { core.error((String(err) + "\\n" + (err?.stack ?? ""))); });',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: nested catch — each catch variable is tracked independently", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (outer) { try { g(); } catch (inner) { } core.error(JSON.stringify(outer)); }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "outer" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "outer" },
                  output: `try { f(); } catch (outer) { try { g(); } catch (inner) { } core.error(String(outer)); }`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "outer" },
                  output: 'try { f(); } catch (outer) { try { g(); } catch (inner) { } core.error((String(outer) + "\\n" + (outer?.stack ?? ""))); }',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: p.catch(handler) named-reference is not flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [`const handler = err => JSON.stringify(err); p.catch(handler);`],
      invalid: [],
    });
  });

  it("valid: p.then(ok, handler) named-reference rejection handler is not flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [`function onRejected(err) { JSON.stringify(err); } p.then(ok, onRejected);`],
      invalid: [],
    });
  });

  it("invalid: JSON.stringify(err) in .then() second-argument rejection handler (arrow) is flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `p.then(result => result, err => core.error(JSON.stringify(err)));`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `p.then(result => result, err => core.error(String(err)));`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "err" },
                  output: 'p.then(result => result, err => core.error((String(err) + "\\n" + (err?.stack ?? ""))));',
                },
              ],
            },
          ],
        },
        {
          code: `p.then(null, err => core.error(JSON.stringify(err, null, 2)));`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `p.then(null, err => core.error(String(err)));`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "err" },
                  output: 'p.then(null, err => core.error((String(err) + "\\n" + (err?.stack ?? ""))));',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: JSON.stringify(err) in .then() second-argument rejection handler (function) is flagged", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `p.then(null, function(err) { core.error(JSON.stringify(err)); });`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `p.then(null, function(err) { core.error(String(err)); });`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "err" },
                  output: 'p.then(null, function(err) { core.error((String(err) + "\\n" + (err?.stack ?? ""))); });',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: first argument of .then() (onFulfilled) is not treated as a rejection handler", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [`p.then(result => JSON.stringify(result));`, `p.then(function(result) { JSON.stringify(result); });`, `p.then(err => core.info(JSON.stringify(err)));`],
      invalid: [],
    });
  });

  it("invalid: works with ES module syntax", () => {
    esmRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `try { fetch(url); } catch (e) { console.error(JSON.stringify(e)); }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "e" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "e" },
                  output: `try { fetch(url); } catch (e) { console.error(String(e)); }`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "e" },
                  output: 'try { fetch(url); } catch (e) { console.error((String(e) + "\\n" + (e?.stack ?? ""))); }',
                },
              ],
            },
          ],
        },
        {
          code: `p.then(ok, err => console.error(JSON.stringify(err)));`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `p.then(ok, err => console.error(String(err)));`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "err" },
                  output: 'p.then(ok, err => console.error((String(err) + "\\n" + (err?.stack ?? ""))));',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: inner catch variable also flagged when outer has JSON.stringify", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (outer) { try { g(); } catch (inner) { core.error(JSON.stringify(inner)); } }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "inner" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "inner" },
                  output: `try { f(); } catch (outer) { try { g(); } catch (inner) { core.error(String(inner)); } }`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "inner" },
                  output: 'try { f(); } catch (outer) { try { g(); } catch (inner) { core.error((String(inner) + "\\n" + (inner?.stack ?? ""))); } }',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: when getErrorMessage is in scope, suggests getErrorMessage and detail-preserving form", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `const getErrorMessage = require('./error_helpers').getErrorMessage; try { f(); } catch (err) { core.error(JSON.stringify(err)); }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `const getErrorMessage = require('./error_helpers').getErrorMessage; try { f(); } catch (err) { core.error(getErrorMessage(err)); }`,
                },
                {
                  messageId: "useDetailPreservingForm",
                  data: { errorVar: "err" },
                  output: 'const getErrorMessage = require(\'./error_helpers\').getErrorMessage; try { f(); } catch (err) { core.error((getErrorMessage(err) + "\\n" + (err?.stack ?? ""))); }',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: when getErrorMessage is in scope via destructured require, detail-preserving form uses getErrorMessage", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `const { getErrorMessage } = require('./error_helpers'); try { f(); } catch (error) { core.error(\`details: \${JSON.stringify(error, null, 2)}\`); }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "error" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "error" },
                  output: `const { getErrorMessage } = require('./error_helpers'); try { f(); } catch (error) { core.error(\`details: \${getErrorMessage(error)}\`); }`,
                },
                {
                  messageId: "useDetailPreservingForm",
                  data: { errorVar: "error" },
                  output: 'const { getErrorMessage } = require(\'./error_helpers\'); try { f(); } catch (error) { core.error(`details: ${(getErrorMessage(error) + "\\n" + (error?.stack ?? ""))}`); }',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: when getErrorMessage is imported via ESM, suggestions use getErrorMessage", () => {
    esmRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `import { getErrorMessage } from "./error_helpers"; try { f(); } catch (err) { console.error(JSON.stringify(err)); }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `import { getErrorMessage } from "./error_helpers"; try { f(); } catch (err) { console.error(getErrorMessage(err)); }`,
                },
                {
                  messageId: "useDetailPreservingForm",
                  data: { errorVar: "err" },
                  output: 'import { getErrorMessage } from "./error_helpers"; try { f(); } catch (err) { console.error((getErrorMessage(err) + "\\n" + (err?.stack ?? ""))); }',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: when getErrorMessage is declared earlier in the catch block, suggestions use getErrorMessage", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { const getErrorMessage = customHelper; console.error(JSON.stringify(err)); }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { const getErrorMessage = customHelper; console.error(getErrorMessage(err)); }`,
                },
                {
                  messageId: "useDetailPreservingForm",
                  data: { errorVar: "err" },
                  output: 'try { f(); } catch (err) { const getErrorMessage = customHelper; console.error((getErrorMessage(err) + "\\n" + (err?.stack ?? ""))); }',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: when getErrorMessage is declared after the call site, suggestions fall back to String()", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { console.error(JSON.stringify(err)); const getErrorMessage = customHelper; }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { console.error(String(err)); const getErrorMessage = customHelper; }`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "err" },
                  output: 'try { f(); } catch (err) { console.error((String(err) + "\\n" + (err?.stack ?? ""))); const getErrorMessage = customHelper; }',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: detail-preserving suggestion is parenthesized when the call is chained", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { console.error(JSON.stringify(err).slice(0, 20)); }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { console.error(String(err).slice(0, 20)); }`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "err" },
                  output: 'try { f(); } catch (err) { console.error((String(err) + "\\n" + (err?.stack ?? "")).slice(0, 20)); }',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: when getErrorMessage is in scope inside a promise rejection handler, suggestions use getErrorMessage", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `const { getErrorMessage } = require('./error_helpers'); p.catch(err => console.error(JSON.stringify(err)));`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `const { getErrorMessage } = require('./error_helpers'); p.catch(err => console.error(getErrorMessage(err)));`,
                },
                {
                  messageId: "useDetailPreservingForm",
                  data: { errorVar: "err" },
                  output: 'const { getErrorMessage } = require(\'./error_helpers\'); p.catch(err => console.error((getErrorMessage(err) + "\\n" + (err?.stack ?? ""))));',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: when getErrorMessage is not in scope, first suggestion uses String() fallback", () => {
    cjsRuleTester.run("no-json-stringify-error", noJsonStringifyErrorRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { console.error(JSON.stringify(err)); }`,
          errors: [
            {
              messageId: "jsonStringifyError",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "useStringFallback",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { console.error(String(err)); }`,
                },
                {
                  messageId: "useDetailPreservingFormFallback",
                  data: { errorVar: "err" },
                  output: 'try { f(); } catch (err) { console.error((String(err) + "\\n" + (err?.stack ?? ""))); }',
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
