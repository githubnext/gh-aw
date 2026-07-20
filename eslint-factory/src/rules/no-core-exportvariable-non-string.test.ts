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

  it("valid: coreObj alias with string value is accepted", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [`coreObj.exportVariable("GH_AW_AIC", roundedAIC);`, `coreObj.exportVariable("MY_VAR", "hello");`, `coreObj.exportVariable("MY_COUNT", String(items.length));`],
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

  it("invalid: coreObj alias with numeric value is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `coreObj.exportVariable("MY_COUNT", 42);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `coreObj.exportVariable("MY_COUNT", String(42));` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: coreObj alias with boolean value is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `coreObj.exportVariable("MY_FLAG", true);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [{ messageId: "wrapWithString", output: `coreObj.exportVariable("MY_FLAG", String(true));` }],
            },
          ],
        },
      ],
    });
  });

  it("invalid: coreObj alias with null value is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `coreObj.exportVariable("MY_VAR", null);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [
                { messageId: "useEmptyString", output: `coreObj.exportVariable("MY_VAR", "");` },
                { messageId: "wrapWithString", output: `coreObj.exportVariable("MY_VAR", String(null));` },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: single-assignment const alias with string value is accepted", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [`const c = core; c.exportVariable("MY_VAR", "hello");`, `const c = core; c.exportVariable("MY_VAR", someVariable);`, `const c = coreObj; c.exportVariable("MY_VAR", "hello");`],
      invalid: [],
    });
  });

  it("invalid: single-assignment const alias with non-string value is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `const c = core; c.exportVariable("MY_COUNT", items.length);`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `const c = core; c.exportVariable("MY_COUNT", String(items.length));` }] }],
        },
        {
          code: `const c = core; c.exportVariable("READY", true);`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `const c = core; c.exportVariable("READY", String(true));` }] }],
        },
        {
          code: `const c = core; c.exportVariable("MY_VAR", null);`,
          errors: [
            {
              messageId: "nonStringValue",
              suggestions: [
                { messageId: "useEmptyString", output: `const c = core; c.exportVariable("MY_VAR", "");` },
                { messageId: "wrapWithString", output: `const c = core; c.exportVariable("MY_VAR", String(null));` },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: let alias with reassignment is NOT flagged (not a safe const alias)", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [`let c = core; c = other; c.exportVariable("MY_VAR", 1);`],
      invalid: [],
    });
  });

  it("valid: non-core const alias is NOT flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [`const c = other; c.exportVariable("MY_VAR", 1);`],
      invalid: [],
    });
  });

  it("valid: destructured exportVariable from core with string value is accepted", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [`const { exportVariable } = core; exportVariable("MY_VAR", "hello");`, `const { exportVariable } = core; exportVariable("MY_VAR", someVariable);`, `const { exportVariable: ev } = core; ev("MY_VAR", "hello");`],
      invalid: [],
    });
  });

  it("invalid: destructured exportVariable from core with non-string value is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `const { exportVariable } = core; exportVariable("MY_COUNT", items.length);`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `const { exportVariable } = core; exportVariable("MY_COUNT", String(items.length));` }] }],
        },
        {
          code: `const { exportVariable } = core; exportVariable("READY", true);`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `const { exportVariable } = core; exportVariable("READY", String(true));` }] }],
        },
        {
          code: `const { exportVariable: ev } = core; ev("MY_COUNT", items.length);`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `const { exportVariable: ev } = core; ev("MY_COUNT", String(items.length));` }] }],
        },
      ],
    });
  });

  it("valid: standalone exportVariable identifier from non-core source is NOT flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [`function exportVariable(k, v) {} exportVariable("MY_VAR", 1);`, `const { exportVariable } = other; exportVariable("MY_VAR", 1);`],
      invalid: [],
    });
  });

  it("valid: function parameter with core-alias name and string value is accepted", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [`function f(core) { core.exportVariable("MY_VAR", "hello"); }`, `function f(core) { core.exportVariable("MY_VAR", someVariable); }`, `function f(coreObj) { coreObj.exportVariable("MY_VAR", "hello"); }`],
      invalid: [],
    });
  });

  it("invalid: function parameter with core-alias name and non-string value is flagged", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [],
      invalid: [
        {
          code: `function f(core) { core.exportVariable("X", 5); }`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `function f(core) { core.exportVariable("X", String(5)); }` }] }],
        },
        {
          code: `function f(coreObj) { coreObj.exportVariable("MY_FLAG", true); }`,
          errors: [{ messageId: "nonStringValue", suggestions: [{ messageId: "wrapWithString", output: `function f(coreObj) { coreObj.exportVariable("MY_FLAG", String(true)); }` }] }],
        },
      ],
    });
  });

  it("valid: function parameter not in CORE_ALIASES is not treated as core (shadow-exclusion)", () => {
    cjsRuleTester.run("no-core-exportvariable-non-string", noCoreExportVariableNonStringRule, {
      valid: [
        // `coreArg` is not in CORE_ALIASES — must not be treated as a core object
        `function f(coreArg) { coreArg.exportVariable("X", 5); }`,
        `function f(myCore) { myCore.exportVariable("X", 5); }`,
      ],
      invalid: [],
    });
  });
});
