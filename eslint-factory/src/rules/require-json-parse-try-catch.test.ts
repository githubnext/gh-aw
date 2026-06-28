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
      valid: [`try { const x = JSON.parse(str); } catch (e) {}`, `try { return JSON.parse(str); } catch (e) {}`, `function f() { try { JSON.parse(str); } catch (e) {} }`, `try { const x = JSON["parse"](str); } catch (e) {}`],
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

  it('invalid: computed JSON["parse"] access is flagged when not in try block', () => {
    cjsRuleTester.run("require-json-parse-try-catch", requireJsonParseTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const data = JSON["parse"](rawInput);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "rawInput" },
              suggestions: [
                {
                  messageId: "useHelper",
                  output: `try {\n  const data = JSON["parse"](rawInput);\n} catch (err) {\n  throw err;\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `JSON["parse"](response.body);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "response.body" },
              suggestions: [
                {
                  messageId: "useHelper",
                  output: `try {\n  JSON["parse"](response.body);\n} catch (err) {\n  throw err;\n}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: synchronous callbacks inside try block are protected", () => {
    cjsRuleTester.run("require-json-parse-try-catch", requireJsonParseTryCatchRule, {
      valid: [
        // Array methods are synchronous — the try block is genuinely protective
        `try { const results = lines.map(l => JSON.parse(l)); } catch (e) {}`,
        `try { items.forEach(s => { JSON.parse(s); }); } catch (e) {}`,
        `try { const ok = items.filter(s => JSON.parse(s)); } catch (e) {}`,
        `try { const acc = items.reduce((a, s) => JSON.parse(s), null); } catch (e) {}`,
        // IIFE runs synchronously — still protected
        `try { (function() { JSON.parse(x); })(); } catch (e) {}`,
      ],
      invalid: [],
    });
  });

  it("invalid: JSON.parse in deferred callback is not protected by surrounding try", () => {
    cjsRuleTester.run("require-json-parse-try-catch", requireJsonParseTryCatchRule, {
      valid: [],
      invalid: [
        // EventEmitter .on — callback fires asynchronously
        {
          code: `try { emitter.on("data", chunk => { JSON.parse(chunk); }); } catch (e) {}`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "chunk" },
              suggestions: [
                {
                  messageId: "useHelper",
                  output: `try { emitter.on("data", chunk => { try {\n  JSON.parse(chunk);\n} catch (err) {\n  throw err;\n} }); } catch (e) {}`,
                },
              ],
            },
          ],
        },
        // Promise .then — callback fires asynchronously
        {
          code: `try { promise.then(data => { const x = JSON.parse(data); }); } catch (e) {}`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "data" },
              suggestions: [
                {
                  messageId: "useHelper",
                  output: `try { promise.then(data => { try {\n  const x = JSON.parse(data);\n} catch (err) {\n  throw err;\n} }); } catch (e) {}`,
                },
              ],
            },
          ],
        },
        // setTimeout — callback fires asynchronously
        {
          code: `try { setTimeout(() => { JSON.parse(raw); }, 100); } catch (e) {}`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "raw" },
              suggestions: [
                {
                  messageId: "useHelper",
                  output: `try { setTimeout(() => { try {\n  JSON.parse(raw);\n} catch (err) {\n  throw err;\n} }, 100); } catch (e) {}`,
                },
              ],
            },
          ],
        },
        // new Promise executor — Promise captures thrown errors instead of the outer try
        {
          code: `try { new Promise(resolve => { JSON.parse(data); }); } catch (e) {}`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "data" },
              suggestions: [
                {
                  messageId: "useHelper",
                  output: `try { new Promise(resolve => { try {\n  JSON.parse(data);\n} catch (err) {\n  throw err;\n} }); } catch (e) {}`,
                },
              ],
            },
          ],
        },
        // process.nextTick — deferred
        {
          code: `try { process.nextTick(() => { JSON.parse(payload); }); } catch (e) {}`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { arg: "payload" },
              suggestions: [
                {
                  messageId: "useHelper",
                  output: `try { process.nextTick(() => { try {\n  JSON.parse(payload);\n} catch (err) {\n  throw err;\n} }); } catch (e) {}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
