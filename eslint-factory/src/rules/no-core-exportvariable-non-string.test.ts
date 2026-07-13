import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noCoreExportVariableNonStringRule } from "./no-core-exportvariable-non-string";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("no-core-exportvariable-non-string", () => {
  it("uses the correct docs URL", () => {
    expect(noCoreExportVariableNonStringRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-core-exportvariable-non-string");
  });

  it("valid: string literal values are accepted", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [
        `core.exportVariable("MY_VAR", "hello");`,
        `core.exportVariable("MY_FLAG", "true");`,
        `core.exportVariable("MY_FLAG", "false");`,
        `core.exportVariable("MY_VAR", someVariable);`,
        `core.exportVariable("MY_COUNT", String(items.length));`,
        `core.exportVariable("MY_COUNT", items.length.toString());`,
        `core.exportVariable("MY_COUNT", \`\${items.length}\`);`,
      ],
      invalid: [],
    });
  });

  it("valid: non-core.exportVariable calls are not flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [`other.exportVariable("MY_VAR", 0);`, `exportVariable("MY_VAR", 0);`, `myCore.exportVariable("MY_VAR", 0);`],
      invalid: [],
    });
  });

  it("valid: computed string-literal exportVariable with string value is accepted", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [`core["exportVariable"]("MY_VAR", "hello");`],
      invalid: [],
    });
  });

  it("invalid: numeric literal value is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core.exportVariable("MY_COUNT", 42);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `core.exportVariable("MY_COUNT", String(42));` }],
            },
          ],
        },
        {
          code: `core.exportVariable("MY_ZERO", 0);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `core.exportVariable("MY_ZERO", String(0));` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: boolean literal value is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core.exportVariable("MY_FLAG", true);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `core.exportVariable("MY_FLAG", String(true));` }],
            },
          ],
        },
        {
          code: `core.exportVariable("MY_FLAG", false);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `core.exportVariable("MY_FLAG", String(false));` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: null value is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core.exportVariable("MY_VAR", null);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [
                { messageId: "useEmptyString", output: `core.exportVariable("MY_VAR", "");` },
                { messageId: "wrapWithString", output: `core.exportVariable("MY_VAR", String(null));` },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: undefined identifier is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core.exportVariable("MY_VAR", undefined);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [
                { messageId: "useEmptyString", output: `core.exportVariable("MY_VAR", "");` },
                { messageId: "wrapWithString", output: `core.exportVariable("MY_VAR", String(undefined));` },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: .length member access is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core.exportVariable("MY_COUNT", items.length);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `core.exportVariable("MY_COUNT", String(items.length));` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: computed exportVariable with numeric value is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `core["exportVariable"]("MY_COUNT", 42);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `core["exportVariable"]("MY_COUNT", String(42));` }],
            },
          ],
        },
      ],
    });
  });

});
