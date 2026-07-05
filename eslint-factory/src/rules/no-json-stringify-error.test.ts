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
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { core.error(getErrorMessage(err)); }`,
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
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "error" },
                  output: `try { f(); } catch (error) { core.error(\`details: \${getErrorMessage(error)}\`); }`,
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
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `p.catch(err => core.error(getErrorMessage(err)));`,
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
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `p.catch(function(err) { core.error(getErrorMessage(err)); });`,
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
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "outer" },
                  output: `try { f(); } catch (outer) { try { g(); } catch (inner) { } core.error(getErrorMessage(outer)); }`,
                },
              ],
            },
          ],
        },
      ],
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
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "e" },
                  output: `try { fetch(url); } catch (e) { console.error(getErrorMessage(e)); }`,
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
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "inner" },
                  output: `try { f(); } catch (outer) { try { g(); } catch (inner) { core.error(getErrorMessage(inner)); } }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
