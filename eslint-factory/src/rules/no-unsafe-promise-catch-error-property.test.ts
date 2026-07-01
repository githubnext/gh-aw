import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noUnsafePromiseCatchErrorPropertyRule } from "./no-unsafe-promise-catch-error-property";

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

describe("no-unsafe-promise-catch-error-property", () => {
  it("valid: instanceof Error guard suppresses warnings in .catch() callback", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [`promise.catch(err => { if (err instanceof Error) { console.log(err.message); } });`, `fetch(url).catch(err => err instanceof Error ? err.message : String(err));`],
      invalid: [],
    });
  });

  it("valid: getErrorMessage() guard suppresses warnings in .catch() callback", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [`promise.catch(err => { core.setFailed(getErrorMessage(err)); });`, `promise.catch(err => { const msg = getErrorMessage(err); console.log(err.message); });`],
      invalid: [],
    });
  });

  it("valid: .catch() with no params or destructuring params is ignored", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [`promise.catch(() => { console.log("error"); });`, `promise.catch(({ message }) => { console.log(message); });`],
      invalid: [],
    });
  });

  it("valid: non-.catch() methods (.then, .finally) are not tracked", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [`promise.then(err => { console.log(err.message); });`, `promise.finally(err => { console.log(err.message); });`],
      invalid: [],
    });
  });

  it("valid: named function reference passed to .catch() is not tracked", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [`promise.catch(handleError);`, `promise.catch(console.error);`],
      invalid: [],
    });
  });

  it("valid: computed .catch access is not tracked", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [`const method = "catch"; promise[method](err => { console.log(err.message); });`],
      invalid: [],
    });
  });

  it("valid: property access on a different variable is not flagged", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [`promise.catch(err => { console.log(otherObj.message); });`],
      invalid: [],
    });
  });

  it("valid: computed property access on .catch() callback param is not flagged", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [`promise.catch(err => { console.log(err["message"]); });`, `promise.catch(err => { const prop = "message"; console.log(err[prop]); });`],
      invalid: [],
    });
  });

  it("valid: nested callback with same param name does not cause false positive on outer catch param", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [`promise.catch(err => { [1, 2].map(err => err.message); });`],
      invalid: [],
    });
  });

  it("invalid: err.message without guard is flagged in arrow function callback", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `promise.catch(err => { core.setFailed(err.message); });`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `promise.catch(err => { core.setFailed(getErrorMessage(err)); });`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: err.stack without guard is flagged", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `promise.catch(err => { console.log(err.stack); });`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "stack", errorVar: "err" },
              suggestions: [],
            },
          ],
        },
      ],
    });
  });

  it("invalid: err.code without guard is flagged", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `promise.catch(err => { if (err.code === "ENOENT") { } });`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "code", errorVar: "err" },
            },
          ],
        },
      ],
    });
  });

  it("invalid: FunctionExpression callback is also tracked", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `promise.catch(function(err) { core.setFailed(err.message); });`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `promise.catch(function(err) { core.setFailed(getErrorMessage(err)); });`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: multiple unsafe accesses in one .catch() callback are all reported", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `promise.catch(err => { console.log(err.message); console.log(err.stack); });`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `promise.catch(err => { console.log(getErrorMessage(err)); console.log(err.stack); });`,
                },
              ],
            },
            {
              messageId: "unsafeProperty",
              data: { prop: "stack", errorVar: "err" },
              suggestions: [],
            },
          ],
        },
      ],
    });
  });

  it("invalid: .message suggests getErrorMessage replacement", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `fetch(url).catch(err => { throw new Error(err.message); });`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `fetch(url).catch(err => { throw new Error(getErrorMessage(err)); });`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: works with ES module syntax", () => {
    esmRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `fetch(url).catch(e => { console.error(e.message); });`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "e" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "e" },
                  output: `fetch(url).catch(e => { console.error(getErrorMessage(e)); });`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: typeof err === 'object' with non-null guard suppresses warnings in .catch() callback", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [
        `promise.catch(err => { if (typeof err === 'object' && err !== null) { console.log(err.status); } });`,
        `promise.catch(err => { if ('object' === typeof err && null !== err) { console.log(err.status); } });`,
        `promise.catch(err => { if (typeof err === 'object' && err != null) { console.log(err.status); } });`,
        `promise.catch(err => { if (err && typeof err === 'object') { console.log(err.status); } });`,
        `promise.catch(err => { if (!err) return; if (typeof err === 'object') { console.log(err.status); } });`,
      ],
      invalid: [],
    });
  });

  it("invalid: bare typeof err === 'object' guard is insufficient in .catch() callback", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `promise.catch(err => { if (typeof err === 'object') { console.log(err.status); } });`,
          errors: [{ messageId: "unsafeProperty", data: { prop: "status", errorVar: "err" } }],
        },
        {
          code: `promise.catch(err => { if ('object' === typeof err) { console.log(err.status); } });`,
          errors: [{ messageId: "unsafeProperty", data: { prop: "status", errorVar: "err" } }],
        },
        {
          // Standalone err !== null in a separate if (without return) does not count as companion guard
          code: `promise.catch(err => { if (err !== null) { } if (typeof err === 'object') { console.log(err.status); } });`,
          errors: [{ messageId: "unsafeProperty", data: { prop: "status", errorVar: "err" } }],
        },
      ],
    });
  });

  it("invalid: err.status without guard is flagged in .catch() callback", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `promise.catch(err => { if (err.status === 404) { } });`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "status", errorVar: "err" },
            },
          ],
        },
      ],
    });
  });

  it("invalid: err.cause without guard is flagged in .catch() callback", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `promise.catch(err => { console.log(err.cause); });`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "cause", errorVar: "err" },
            },
          ],
        },
      ],
    });
  });

  it("invalid: err.name without guard is flagged in .catch() callback", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `promise.catch(err => { console.log(err.name); });`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "name", errorVar: "err" },
            },
          ],
        },
      ],
    });
  });

  it("invalid: nested .catch() callbacks are checked independently", () => {
    cjsRuleTester.run("no-unsafe-promise-catch-error-property", noUnsafePromiseCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          // Inner .catch has a guard; outer .catch does not — outer should still be flagged
          code: `outer.catch(err => { inner.catch(err2 => { core.setFailed(getErrorMessage(err2)); }); core.setFailed(err.message); });`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `outer.catch(err => { inner.catch(err2 => { core.setFailed(getErrorMessage(err2)); }); core.setFailed(getErrorMessage(err)); });`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
