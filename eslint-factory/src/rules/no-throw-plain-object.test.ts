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

  it("valid: JSON-RPC error shape is exempt (negative code + message + optional data)", () => {
    ruleTester.run("no-throw-plain-object", noThrowPlainObjectRule, {
      valid: [
        // Acceptance-criteria examples — must NOT be flagged
        `throw { code: -32602, message: "Invalid params" };`,
        `throw { code: -32603, message: msg };`,
        `throw { code: -32603, message: msg, data: {} };`,
        // Template-literal message (common in mcp_server_core.cjs)
        "throw { code: -32601, message: `Method not found: ${method}` };",
        // Function-call message (common in safe_outputs_handlers.cjs)
        `throw { code: -32602, message: getErrorMessage(error) };`,
      ],
      invalid: [
        // Positive code → not a valid JSON-RPC error code, still flagged
        {
          code: `throw { code: 500, message: "internal" };`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [
                {
                  messageId: "useObjectAssign",
                  output: `throw Object.assign(new Error("internal"), { code: 500 });`,
                },
              ],
            },
          ],
        },
        // Fractional negative code → not a valid JSON-RPC integer code, still flagged
        {
          code: `throw { code: -1.5, message: "fractional" };`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [
                {
                  messageId: "useObjectAssign",
                  output: `throw Object.assign(new Error("fractional"), { code: -1.5 });`,
                },
              ],
            },
          ],
        },
        // Negative zero is not strictly negative, still flagged
        {
          code: `throw { code: -0, message: "zero" };`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [
                {
                  messageId: "useObjectAssign",
                  output: `throw Object.assign(new Error("zero"), { code: -0 });`,
                },
              ],
            },
          ],
        },
        // No message → still flagged
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
        // Extra key beyond code/message/data → still flagged
        {
          code: `throw { code: -32602, message: "oops", extra: "field" };`,
          errors: [
            {
              messageId: "noThrowPlainObject",
              suggestions: [
                {
                  messageId: "useObjectAssign",
                  output: `throw Object.assign(new Error("oops"), { code: -32602, extra: "field" });`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: throwing a plain object literal is flagged with suggestion", () => {
    ruleTester.run("no-throw-plain-object", noThrowPlainObjectRule, {
      valid: [],
      invalid: [
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

  it("valid: JSON-RPC shape with code, message, data is exempt", () => {
    ruleTester.run("no-throw-plain-object", noThrowPlainObjectRule, {
      valid: [`throw { code: -32602, message: "Invalid params", data: { field: "name" } };`],
      invalid: [],
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
