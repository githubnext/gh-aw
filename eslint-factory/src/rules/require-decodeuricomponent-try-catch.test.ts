import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireDecodeURIComponentTryCatchRule } from "./require-decodeuricomponent-try-catch";

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

describe("require-decodeuricomponent-try-catch", () => {
  it("valid: decodeURIComponent/decodeURI with string literal is always safe (CommonJS)", () => {
    cjsRuleTester.run("require-decodeuricomponent-try-catch", requireDecodeURIComponentTryCatchRule, {
      valid: [`const v = decodeURIComponent("abc");`, `const v = decodeURI(\`static\`);`, `const v = decodeURIComponent("a" + "b");`],
      invalid: [],
    });
  });

  it("valid: decodeURIComponent with primitive literals is always safe (CommonJS)", () => {
    cjsRuleTester.run("require-decodeuricomponent-try-catch", requireDecodeURIComponentTryCatchRule, {
      valid: [`decodeURIComponent(42);`, `decodeURIComponent(true);`, `decodeURIComponent(null);`],
      invalid: [],
    });
  });

  it("valid: calls inside try block pass (CommonJS)", () => {
    cjsRuleTester.run("require-decodeuricomponent-try-catch", requireDecodeURIComponentTryCatchRule, {
      valid: [`try { const v = decodeURIComponent(raw); } catch (e) {}`, `try { return decodeURI(raw); } catch (e) {}`, `function f() { try { decodeURIComponent(raw); } catch (e) {} }`],
      invalid: [],
    });
  });

  it("valid: shadowed by a local binding is not the global function (CommonJS)", () => {
    cjsRuleTester.run("require-decodeuricomponent-try-catch", requireDecodeURIComponentTryCatchRule, {
      valid: [`function decodeURIComponent(v) { return v; } const x = decodeURIComponent(raw);`, `const decodeURI = v => v; const x = decodeURI(raw);`],
      invalid: [],
    });
  });

  it("valid: no arguments is not flagged (returns 'undefined' string, does not throw)", () => {
    cjsRuleTester.run("require-decodeuricomponent-try-catch", requireDecodeURIComponentTryCatchRule, {
      valid: [`decodeURIComponent();`],
      invalid: [],
    });
  });

  it("invalid: bare decodeURIComponent(variable) reports requireTryCatch (CommonJS)", () => {
    cjsRuleTester.run("require-decodeuricomponent-try-catch", requireDecodeURIComponentTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `decodeURIComponent(raw);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { callee: "decodeURIComponent", arg: "raw" },
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try {\n  decodeURIComponent(raw);\n} catch (err) {\n  // TODO: handle malformed percent-encoding for this decodeURIComponent(...) call.\n  throw new Error(\n    "URI decoding failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n}`,
                },
              ],
            },
          ],
        },
        {
          code: `const v = decodeURI(raw);`,
          errors: [
            {
              messageId: "requireTryCatch",
              data: { callee: "decodeURI", arg: "raw" },
            },
          ],
        },
      ],
    });
  });

  it("invalid: template literal containing expressions (CommonJS)", () => {
    cjsRuleTester.run("require-decodeuricomponent-try-catch", requireDecodeURIComponentTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: "const v = decodeURIComponent(`${prefix}${raw}`);",
          errors: [{ messageId: "requireTryCatch" }],
        },
      ],
    });
  });

  it("invalid: reports in ES module", () => {
    esmRuleTester.run("require-decodeuricomponent-try-catch", requireDecodeURIComponentTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `const v = decodeURIComponent(raw);`,
          errors: [{ messageId: "requireTryCatch", data: { callee: "decodeURIComponent", arg: "raw" } }],
        },
      ],
    });
  });

  it("invalid: decodeURIComponent inside setTimeout callback is not protected by outer try (CommonJS)", () => {
    cjsRuleTester.run("require-decodeuricomponent-try-catch", requireDecodeURIComponentTryCatchRule, {
      valid: [],
      invalid: [
        {
          code: `try { setTimeout(() => { decodeURIComponent(raw); }, 0); } catch(e) {}`,
          errors: [
            {
              messageId: "requireTryCatch",
              suggestions: [
                {
                  messageId: "wrapInTryCatch",
                  output: `try { setTimeout(() => { try {\n  decodeURIComponent(raw);\n} catch (err) {\n  // TODO: handle malformed percent-encoding for this decodeURIComponent(...) call.\n  throw new Error(\n    "URI decoding failed: " + (err instanceof Error ? err.message : String(err)),\n    { cause: err },\n  );\n} }, 0); } catch(e) {}`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
