import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noUnsafeCatchErrorPropertyRule } from "./no-unsafe-catch-error-property";

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

describe("no-unsafe-catch-error-property", () => {
  it("valid: bare catch {} without binding is ignored", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [
        `try { f(); } catch { }`,
        // Destructuring binding is also ignored
        `try { f(); } catch ({ message }) { console.log(message); }`,
      ],
      invalid: [],
    });
  });

  it("valid: getErrorMessage guard suppresses all warnings in the catch block", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [`try { f(); } catch (err) { core.setFailed(getErrorMessage(err)); }`, `try { f(); } catch (err) { const msg = getErrorMessage(err); console.log(err.message); }`],
      invalid: [],
    });
  });

  it("valid: instanceof Error guard suppresses all warnings in the catch block", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [`try { f(); } catch (err) { core.setFailed(err instanceof Error ? err.message : String(err)); }`, `try { f(); } catch (err) { if (err instanceof Error) { console.log(err.stack); } }`],
      invalid: [],
    });
  });

  it("valid: property access on a different variable is not flagged", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [`try { f(); } catch (err) { console.log(otherObj.message); }`, `try { f(); } catch (err) { const e = new Error(); console.log(e.message); }`],
      invalid: [],
    });
  });

  it("valid: dynamic computed property access on catch variable is not flagged", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [`try { f(); } catch (err) { const prop = "message"; console.log(err[prop]); }`],
      invalid: [],
    });
  });

  it('invalid: computed string-literal err["message"] is flagged same as err.message', () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { console.log(err["message"]); }`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { console.log(getErrorMessage(err)); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it('invalid: computed string-literal err["stack"] suggests instanceof guard', () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { console.log(err["stack"]); }`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "stack", errorVar: "err" },
              suggestions: [
                {
                  messageId: "wrapWithInstanceof",
                  data: { errorVar: "err", prop: "stack" },
                  output: `try { f(); } catch (err) { console.log((err instanceof Error ? err.stack : undefined)); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it('invalid: computed string-literal err["code"] suggests instanceof guard', () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { if (err["code"] === "ENOENT") { } }`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "code", errorVar: "err" },
              suggestions: [
                {
                  messageId: "wrapWithInstanceof",
                  data: { errorVar: "err", prop: "code" },
                  output: `try { f(); } catch (err) { if ((err instanceof Error ? err.code : undefined) === "ENOENT") { } }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: err.message without guard is flagged", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { core.setFailed(err.message); }`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { core.setFailed(getErrorMessage(err)); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: err.stack without guard is flagged", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { console.log(err.stack); }`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "stack", errorVar: "err" },
              suggestions: [
                {
                  messageId: "wrapWithInstanceof",
                  data: { errorVar: "err", prop: "stack" },
                  output: `try { f(); } catch (err) { console.log((err instanceof Error ? err.stack : undefined)); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: err.code without guard is flagged", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { if (err.code === "ENOENT") { } }`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "code", errorVar: "err" },
              suggestions: [
                {
                  messageId: "wrapWithInstanceof",
                  data: { errorVar: "err", prop: "code" },
                  output: `try { f(); } catch (err) { if ((err instanceof Error ? err.code : undefined) === "ENOENT") { } }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: multiple unsafe property accesses in one catch block are all reported", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { console.log(err.message); console.log(err.stack); }`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { console.log(getErrorMessage(err)); console.log(err.stack); }`,
                },
              ],
            },
            {
              messageId: "unsafeProperty",
              data: { prop: "stack", errorVar: "err" },
              suggestions: [
                {
                  messageId: "wrapWithInstanceof",
                  data: { errorVar: "err", prop: "stack" },
                  output: `try { f(); } catch (err) { console.log(err.message); console.log((err instanceof Error ? err.stack : undefined)); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: .message suggests getErrorMessage replacement", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { core.setFailed(err.message); }`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `try { f(); } catch (err) { core.setFailed(getErrorMessage(err)); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: .stack suggests instanceof Error guard", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `try { f(); } catch (err) { console.log(err.stack); }`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "stack", errorVar: "err" },
              suggestions: [
                {
                  messageId: "wrapWithInstanceof",
                  data: { errorVar: "err", prop: "stack" },
                  output: `try { f(); } catch (err) { console.log((err instanceof Error ? err.stack : undefined)); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: works with ES module syntax", () => {
    esmRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          code: `try { fetch(url); } catch (e) { console.error(e.message); }`,
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "e" },
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

  it("invalid: nested try/catch — each catch block is checked independently", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          // Inner catch has a guard; outer catch does not — outer should still be flagged
          code: `
try {
  f();
} catch (outer) {
  try {
    g();
  } catch (inner) {
    core.setFailed(getErrorMessage(inner));
  }
  core.setFailed(outer.message);
}`.trim(),
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "outer" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "outer" },
                  output: `try {\n  f();\n} catch (outer) {\n  try {\n    g();\n  } catch (inner) {\n    core.setFailed(getErrorMessage(inner));\n  }\n  core.setFailed(getErrorMessage(outer));\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: nested try/catch — inner guard does not affect the outer catch block", () => {
    cjsRuleTester.run("no-unsafe-catch-error-property", noUnsafeCatchErrorPropertyRule, {
      valid: [],
      invalid: [
        {
          // Outer catch has no guard, inner does — outer is still flagged
          code: `
try {
  f();
} catch (err) {
  try {
    g();
  } catch (err2) {
    core.setFailed(getErrorMessage(err2));
  }
  core.setFailed(err.message);
}`.trim(),
          errors: [
            {
              messageId: "unsafeProperty",
              data: { prop: "message", errorVar: "err" },
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `try {\n  f();\n} catch (err) {\n  try {\n    g();\n  } catch (err2) {\n    core.setFailed(getErrorMessage(err2));\n  }\n  core.setFailed(getErrorMessage(err));\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
