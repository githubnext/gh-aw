import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireNewUrlTryCatchRule } from "./require-new-url-try-catch";

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

describe("require-new-url-try-catch", () => {
  it("valid: new URL with string literal is always safe (CommonJS)", () => {
    cjsRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [`const u = new URL("https://github.com");`, `const u = new URL("https://github.com/owner/repo");`, `const u = new URL(\`https://github.com/static\`);`],
      invalid: [],
    });
  });

  it("valid: new URL inside try block passes (CommonJS)", () => {
    cjsRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [
        `try { const u = new URL(urlStr); } catch (e) {}`,
        `try { return new URL(urlStr); } catch (e) {}`,
        `function f() { try { new URL(urlStr); } catch (e) {} }`,
        `try { const u = new URL(process.env.GITHUB_SERVER_URL); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("valid: new URL inside try block passes (ES module)", () => {
    esmRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [`try { const u = new URL(urlStr); } catch (e) {}`],
      invalid: [],
    });
  });

  it("valid: URL shadowed by a local binding is not the global constructor (CommonJS)", () => {
    cjsRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [
        // URL is a parameter — not the global; should not flag
        `function parse(URL, value) { return new URL(value); }`,
        // URL is locally imported/assigned
        `const URL = require("./my-url"); const u = new URL(variable);`,
      ],
      invalid: [],
    });
  });

  it("valid: new URL with import.meta.url as base is safe (ES module)", () => {
    esmRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [
        // First arg is static, base is import.meta.url — never throws
        `new URL("./relative/path", import.meta.url);`,
        // First arg is dynamic but we're inside try
        `try { new URL(path, import.meta.url); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("invalid: bare new URL(variable) reports requireTryCatch (CommonJS)", () => {
    cjsRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [],
      invalid: [
        {
          // ExpressionStatement — suggestion is safe (no bindings go out of scope)
          code: `new URL(endpoint);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "endpoint" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  new URL(endpoint);\n} catch (err) {\n  // TODO: handle invalid URL for this new URL(...) call.\n  throw new Error(\n    "URL constructor call failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          // VariableDeclaration — error is reported but no suggestion (wrapping would put
          // subsequent uses of `u` out of scope).
          code: `const u = new URL(urlStr);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "urlStr" },
            },
          ],
        },
      ],
    });
  });

  it("invalid: new URL(process.env.VAR) without fallback (CommonJS)", () => {
    cjsRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [],
      invalid: [
        {
          // VariableDeclaration — no suggestion (wrapping would put `u` out of scope)
          code: `const u = new URL(process.env.GITHUB_SERVER_URL);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "process.env.GITHUB_SERVER_URL" },
            },
          ],
        },
      ],
    });
  });

  it("invalid: new URL with template literal containing expressions (CommonJS)", () => {
    cjsRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [],
      invalid: [
        {
          // VariableDeclaration — no suggestion
          code: "const u = new URL(`https://${host}/path`);",
          errors: [
            {
              messageId: "requireTryCatch",
            },
          ],
        },
      ],
    });
  });

  it("invalid: new URL reports in ES module", () => {
    esmRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [],
      invalid: [
        {
          // VariableDeclaration — no suggestion
          code: `const u = new URL(urlStr);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "urlStr" },
            },
          ],
        },
      ],
    });
  });

  it("invalid: dynamic second argument (base) without try/catch is flagged (ES module)", () => {
    esmRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [],
      invalid: [
        {
          // ExpressionStatement with dynamic base — suggestion is safe
          code: `new URL("/path", base);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "base" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  new URL("/path", base);\n} catch (err) {\n  // TODO: handle invalid URL for this new URL(...) call.\n  throw new Error(\n    "URL constructor call failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          // Both args are dynamic — flag on first arg; ExpressionStatement so suggestion is provided
          code: `new URL(urlStr, base);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "urlStr" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  new URL(urlStr, base);\n} catch (err) {\n  // TODO: handle invalid URL for this new URL(...) call.\n  throw new Error(\n    "URL constructor call failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: new URL() with no arguments always throws (CommonJS)", () => {
    cjsRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [],
      invalid: [
        {
          // ExpressionStatement — suggestion wraps the whole call
          code: `new URL();`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  new URL();\n} catch (err) {\n  // TODO: handle invalid URL for this new URL(...) call.\n  throw new Error(\n    "URL constructor call failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: new URL in arrow-expression body has no wrappable ancestor — no suggestion emitted (CommonJS)", () => {
    cjsRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [],
      invalid: [
        {
          // Arrow expression body is not a statement — findEnclosingStatement returns null, so suggestions is []
          code: `const f = () => new URL(urlStr);`,
          errors: [{ messageId: "requireTryCatch", suggestions: [] }],
        },
      ],
    });
  });

  it("invalid: new URL inside setTimeout callback is not protected by outer try (CommonJS)", () => {
    cjsRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [],
      invalid: [
        {
          // The outer try does NOT protect the URL call: isDeferredCallback detects the
          // setTimeout boundary and crossedDeferredBoundary = true.
          code: `try { setTimeout(() => { new URL(urlStr); }, 0); } catch(e) {}`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try { setTimeout(() => { try {\n  new URL(urlStr);\n} catch (err) {\n  // TODO: handle invalid URL for this new URL(...) call.\n  throw new Error(\n    "URL constructor call failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }, 0); } catch(e) {}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
