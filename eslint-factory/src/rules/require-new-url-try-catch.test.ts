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
      valid: [
        `const u = new URL("https://github.com");`,
        `const u = new URL("https://github.com/owner/repo");`,
        `const u = new URL(\`https://github.com/static\`);`,
      ],
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

  it("invalid: bare new URL(variable) reports requireTryCatch (CommonJS)", () => {
    cjsRuleTester.run("require-new-url-try-catch", requireNewUrlTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const u = new URL(urlStr);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "urlStr" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  const u = new URL(urlStr);\n} catch (err) {\n  // TODO: handle invalid URL for this new URL(urlStr) call.\n  throw new Error(\n    "new URL(urlStr) failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `new URL(endpoint);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "endpoint" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  new URL(endpoint);\n} catch (err) {\n  // TODO: handle invalid URL for this new URL(endpoint) call.\n  throw new Error(\n    "new URL(endpoint) failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
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
          code: `const u = new URL(process.env.GITHUB_SERVER_URL);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "process.env.GITHUB_SERVER_URL" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  const u = new URL(process.env.GITHUB_SERVER_URL);\n} catch (err) {\n  // TODO: handle invalid URL for this new URL(process.env.GITHUB_SERVER_URL) call.\n  throw new Error(\n    "new URL(process.env.GITHUB_SERVER_URL) failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
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
          code: 'const u = new URL(`https://${host}/path`);',
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: 'try {\n  const u = new URL(`https://${host}/path`);\n} catch (err) {\n  // TODO: handle invalid URL for this new URL(`https://${host}/path`) call.\n  throw new Error(\n    "new URL(`https://${host}/path`) failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}',
                },
              ],
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
          code: `const u = new URL(urlStr);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "urlStr" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  const u = new URL(urlStr);\n} catch (err) {\n  // TODO: handle invalid URL for this new URL(urlStr) call.\n  throw new Error(\n    "new URL(urlStr) failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
