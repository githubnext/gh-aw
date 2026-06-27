import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireJsonParseTryCatchRule } from "./require-json-parse-try-catch";

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

describe("require-json-parse-try-catch", () => {
  it("valid: JSON.parse inside try block passes (CommonJS)", () => {
    cjsRuleTester.run("require-json-parse-try-catch", requireJsonParseTryCatchRule, {
      valid: [`try { const x = JSON.parse(str); } catch (e) {}`, `try { return JSON.parse(str); } catch (e) {}`, `function f() { try { JSON.parse(str); } catch (e) {} }`],
      invalid: [],
    });
  });

  it("valid: JSON.parse inside try block passes (ES module)", () => {
    esmRuleTester.run("require-json-parse-try-catch", requireJsonParseTryCatchRule, {
      valid: [`try { const x = JSON.parse(str); } catch (e) {}`],
      invalid: [],
    });
  });

  it("invalid: bare JSON.parse reports requireTryCatch with argument text in message", () => {
    cjsRuleTester.run("require-json-parse-try-catch", requireJsonParseTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const data = JSON.parse(rawInput);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "rawInput" },
              suggestions: [
                {
                  messageId: "useHelper",
                  output: `try {\n  const data = JSON.parse(rawInput);\n} catch (err) {\n  throw err;\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `JSON.parse(response.body);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "response.body" },
              suggestions: [
                {
                  messageId: "useHelper",
                  output: `try {\n  JSON.parse(response.body);\n} catch (err) {\n  throw err;\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("suggestion: wraps enclosing statement in try/catch (ES module)", () => {
    esmRuleTester.run("require-json-parse-try-catch", requireJsonParseTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const data = JSON.parse(rawInput);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "rawInput" },
              suggestions: [
                {
                  messageId: "useHelper",
                  output: `try {\n  const data = JSON.parse(rawInput);\n} catch (err) {\n  throw err;\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("suggestion output is syntactically valid JavaScript", () => {
    cjsRuleTester.run("require-json-parse-try-catch", requireJsonParseTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const result = JSON.parse(text);`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "useHelper",
                  output: `try {\n  const result = JSON.parse(text);\n} catch (err) {\n  throw err;\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
