import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noThrowPlainObjectRule } from "./no-throw-plain-object";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("no-throw-plain-object", () => {
  it("uses the correct docs URL", () => {
    expect(noThrowPlainObjectRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-throw-plain-object");
  });

  it("hasSuggestions enabled", () => {
    expect(noThrowPlainObjectRule.meta.hasSuggestions).toBe(true);
  });

  it("valid: throwing Error instances is allowed", () => {
    ruleTester.run("no-throw-plain-object", noThrowPlainObjectRule, {
      valid: [
        `throw new Error("something went wrong");`,
        `throw new TypeError("bad type");`,
        `throw new RangeError("out of range");`,
        `throw Object.assign(new Error("msg"), { code: -32602 });`,
        `throw err;`,
        `throw error;`,
        `const e = new Error("x"); throw e;`,
        `throw new Error(JSON.stringify({ code: -32602, message: "bad" }));`,
      ],
      invalid: [],
    });
  });

  it("invalid: throwing a plain object literal is flagged with suggestion", () => {
    ruleTester.run("no-throw-plain-object", noThrowPlainObjectRule, {
      valid: [],
      invalid: [
        {
          code: `throw { code: -32602, message: "Invalid params" };`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [
                {
                  messageId: "useObjectAssign",
                  output: `throw Object.assign(new Error("Invalid params"), { code: -32602 });`,
                },
              ],
            },
          ],
        },
        {
          code: `throw { message: "not found" };`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [
                {
                  messageId: "useObjectAssign",
                  output: `throw new Error("not found");`,
                },
              ],
            },
          ],
        },
        {
          code: `if (bad) { throw { code: 500, message: "internal" }; }`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [
                {
                  messageId: "useObjectAssign",
                  output: `if (bad) { throw Object.assign(new Error("internal"), { code: 500 }); }`,
                },
              ],
            },
          ],
        },
        {
          code: `function f() { throw {}; }`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [
                {
                  messageId: "useObjectAssign",
                  output: `function f() { throw new Error(); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("suggestion: without-message property uses new Error() with full residual", () => {
    ruleTester.run("no-throw-plain-object", noThrowPlainObjectRule, {
      valid: [],
      invalid: [
        {
          code: `throw { code: -32602 };`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [
                {
                  messageId: "useObjectAssign",
                  output: `throw Object.assign(new Error(), { code: -32602 });`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("suggestion: JSON-RPC shape with code, message, data", () => {
    ruleTester.run("no-throw-plain-object", noThrowPlainObjectRule, {
      valid: [],
      invalid: [
        {
          code: `throw { code: -32602, message: "Invalid params", data: { field: "name" } };`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [
                {
                  messageId: "useObjectAssign",
                  output: `throw Object.assign(new Error("Invalid params"), { code: -32602, data: { field: "name" } });`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("skip suggestion: computed key", () => {
    ruleTester.run("no-throw-plain-object", noThrowPlainObjectRule, {
      valid: [],
      invalid: [
        {
          code: `throw { [key]: "value" };`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [],
            },
          ],
        },
      ],
    });
  });

  it("skip suggestion: spread element", () => {
    ruleTester.run("no-throw-plain-object", noThrowPlainObjectRule, {
      valid: [],
      invalid: [
        {
          code: `throw { ...base, code: 1 };`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [],
            },
          ],
        },
      ],
    });
  });
});
