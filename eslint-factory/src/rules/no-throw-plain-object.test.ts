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

  it("invalid: throwing a plain object literal is flagged", () => {
    ruleTester.run("no-throw-plain-object", noThrowPlainObjectRule, {
      valid: [],
      invalid: [
        {
          code: `throw { code: -32602, message: "Invalid params" };`,
          errors: [{ messageId: "noThrowPlainObject" }],
        },
        {
          code: `throw { message: "not found" };`,
          errors: [{ messageId: "noThrowPlainObject" }],
        },
        {
          code: `if (bad) { throw { code: 500, message: "internal" }; }`,
          errors: [{ messageId: "noThrowPlainObject" }],
        },
        {
          code: `function f() { throw {}; }`,
          errors: [{ messageId: "noThrowPlainObject" }],
        },
      ],
    });
  });
});
