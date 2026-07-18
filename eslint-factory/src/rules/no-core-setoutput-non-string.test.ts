import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noCoreSetOutputNonStringRule } from "./no-core-setoutput-non-string";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("no-core-setoutput-non-string", () => {
  it("uses the correct docs URL", () => {
    expect(noCoreSetOutputNonStringRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-core-setoutput-non-string");
  });

  it("valid: string literal values are accepted", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [
        `core.setOutput("count", "42");`,
        `core.setOutput("flag", "true");`,
        `core.setOutput("flag", "false");`,
        `core.setOutput("url", html_url);`,
        `core.setOutput("result", someVariable);`,
        `core.setOutput("count", String(items.length));`,
        `core.setOutput("count", items.length.toString());`,
        `core.setOutput("count", \`\${items.length}\`);`,
        `core.setOutput("count", -1);`,
      ],
      invalid: [],
    });
  });

  it("valid: non-core.setOutput calls are not flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [`other.setOutput("count", 0);`, `setOutput("count", 0);`, `myCore.setOutput("count", 0);`],
      invalid: [],
    });
  });

  it("valid: coreObj alias with string value is accepted", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [`coreObj.setOutput("aic", roundedAIC);`, `coreObj.setOutput("result", "hello");`, `coreObj.setOutput("count", String(items.length));`],
      invalid: [],
    });
  });

  it("valid: computed string-literal setOutput with string value is accepted", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [`core["setOutput"]("count", "42");`],
      invalid: [],
    });
  });

  it("invalid: numeric literal value is flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core.setOutput("processed_count", 0);`,
          errors: [
            {
              messageId: "nonStringValue",
              data: { kind: "numeric literal", valueText: "0" },
              suggestions: [{ messageId: "wrapWithString", data: { valueText: "0" }, output: `core.setOutput("processed_count", String(0));` }],
            },
          ],
        },
        {
          code: `core.setOutput("findings_count", 42);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `core.setOutput("findings_count", String(42));` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: boolean literal value is flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core.setOutput("success", true);`,
          errors: [
            {
              messageId: "nonStringValue",
              data: { kind: "boolean literal", valueText: "true" },
              suggestions: [{ messageId: "wrapWithString", data: { valueText: "true" }, output: `core.setOutput("success", String(true));` }],
            },
          ],
        },
        {
          code: `core.setOutput("ok", false);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `core.setOutput("ok", String(false));` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: undefined identifier value is flagged with empty-string suggestion first", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core.setOutput("result", undefined);`,
          errors: [
            {
              messageId: "nonStringValue",
              data: { kind: "undefined", valueText: "undefined" },
              suggestions: [
                { messageId: "useEmptyString", output: `core.setOutput("result", "");` },
                { messageId: "wrapWithString", data: { valueText: "undefined" }, output: `core.setOutput("result", String(undefined));` },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: null literal value is flagged with empty-string suggestion first", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core.setOutput("result", null);`,
          errors: [
            {
              messageId: "nonStringValue",
              data: { kind: "null", valueText: "null" },
              suggestions: [
                { messageId: "useEmptyString", output: `core.setOutput("result", "");` },
                { messageId: "wrapWithString", data: { valueText: "null" }, output: `core.setOutput("result", String(null));` },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: .length member access is flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core.setOutput("findings_count", validFindings.length);`,
          errors: [
            {
              messageId: "nonStringValue",
              data: { kind: ".length (number)", valueText: "validFindings.length" },
              suggestions: [
                {
                  messageId: "wrapWithString",
                  data: { valueText: "validFindings.length" },
                  output: `core.setOutput("findings_count", String(validFindings.length));`,
                },
              ],
            },
          ],
        },
        {
          code: `core.setOutput("item_count", items.length);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `core.setOutput("item_count", String(items.length));` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: computed string-literal setOutput with non-string value is also flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core["setOutput"]("count", 0);`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `core["setOutput"]("count", String(0));` }] }],
        },
      ],
    });
  });

  it("invalid: coreObj alias with numeric value is flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `coreObj.setOutput("aic", 0);`,
          errors: [
            {
              messageId: "nonStringValue",
              data: { kind: "numeric literal", valueText: "0" },
              suggestions: [{ messageId: "wrapWithString", data: { valueText: "0" }, output: `coreObj.setOutput("aic", String(0));` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: coreObj alias with boolean value is flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `coreObj.setOutput("success", true);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `coreObj.setOutput("success", String(true));` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: coreObj alias with null value is flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `coreObj.setOutput("result", null);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [
                { messageId: "useEmptyString", output: `coreObj.setOutput("result", "");` },
                { messageId: "wrapWithString", output: `coreObj.setOutput("result", String(null));` },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: single-assignment const alias with string value is accepted", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [`const c = core; c.setOutput("n", "str");`, `const c = core; c.setOutput("n", someVariable);`, `const c = coreObj; c.setOutput("n", "str");`],
      invalid: [],
    });
  });

  it("invalid: single-assignment const alias with non-string value is flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `const c = core; c.setOutput("count", items.length);`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `const c = core; c.setOutput("count", String(items.length));` }] }],
        },
        {
          code: `const c = core; c.setOutput("flag", true);`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `const c = core; c.setOutput("flag", String(true));` }] }],
        },
        {
          code: `const c = core; c.setOutput("result", null);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [
                { messageId: "useEmptyString", output: `const c = core; c.setOutput("result", "");` },
                { messageId: "wrapWithString", output: `const c = core; c.setOutput("result", String(null));` },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: let alias with reassignment is NOT flagged (not a safe const alias)", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [`let c = core; c = other; c.setOutput("n", 1);`],
      invalid: [],
    });
  });

  it("valid: non-core const alias is NOT flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [`const c = other; c.setOutput("n", 1);`],
      invalid: [],
    });
  });

  it("valid: destructured setOutput from core with string value is accepted", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [`const { setOutput } = core; setOutput("n", "str");`, `const { setOutput } = core; setOutput("n", someVariable);`, `const { setOutput: so } = core; so("n", "str");`],
      invalid: [],
    });
  });

  it("invalid: destructured setOutput from core with non-string value is flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `const { setOutput } = core; setOutput("count", items.length);`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `const { setOutput } = core; setOutput("count", String(items.length));` }] }],
        },
        {
          code: `const { setOutput } = core; setOutput("flag", true);`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `const { setOutput } = core; setOutput("flag", String(true));` }] }],
        },
        {
          code: `const { setOutput: so } = core; so("n", items.length);`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `const { setOutput: so } = core; so("n", String(items.length));` }] }],
        },
      ],
    });
  });

  it("valid: standalone setOutput identifier from non-core source is NOT flagged", () => {
    cjsRuleTester.run("no-core-setoutput-non-string", noCoreSetOutputNonStringRule, {
      valid: [`function setOutput(k, v) {} setOutput("n", 1);`, `const { setOutput } = other; setOutput("n", 1);`],
      invalid: [],
    });
  });
});
