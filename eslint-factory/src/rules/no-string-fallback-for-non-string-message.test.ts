import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { noStringFallbackForNonStringMessageRule } from "./no-string-fallback-for-non-string-message";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("no-string-fallback-for-non-string-message", () => {
  it("uses the correct docs URL", () => {
    expect(noStringFallbackForNonStringMessageRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#no-string-fallback-for-non-string-message");
  });

  it("accepts correct message-coercion fallbacks and unrelated conditionals", () => {
    ruleTester.run("no-string-fallback-for-non-string-message", noStringFallbackForNonStringMessageRule, {
      valid: [
        // Fallback coerces the message itself, not the container.
        `const m = typeof err.message === "string" ? err.message : String(err.message);`,
        // Nested chain, still coerces the same chain.
        `const m = typeof err.response.data.message === "string" ? err.response.data.message : String(err.response.data.message);`,
        // Unrelated typeof check.
        `const m = typeof err.code === "string" ? err.code : String(err);`,
        // Consequent doesn't match the tested chain.
        `const m = typeof err.message === "string" ? err.other : String(err);`,
        // No String() fallback at all.
        `const m = typeof err.message === "string" ? err.message : "Internal error";`,
        // Alternate isn't a String() call.
        `const m = typeof err.message === "string" ? err.message : err.toString();`,
        // A matching container guard narrows the fallback to non-object values.
        `output = typeof data.error === "object" && typeof data.error.message === "string" ? data.error.message : String(data.error);`,
      ],
      invalid: [],
    });
  });

  it("reports fallbacks that stringify a different container instead of the message", () => {
    ruleTester.run("no-string-fallback-for-non-string-message", noStringFallbackForNonStringMessageRule, {
      valid: [],
      invalid: [
        {
          code: `const message = typeof error.message === "string" ? error.message : String(error);`,
          errors: [{ messageId: "stringifiesContainerInsteadOfMessage" }],
        },
        {
          code: `const dispatchErrMessage = typeof err.response.data.message === "string" ? err.response.data.message : String(dispatchError);`,
          errors: [{ messageId: "stringifiesContainerInsteadOfMessage" }],
        },
        {
          code: `const message = typeof foo === "object" && typeof data.error.message === "string" ? data.error.message : String(data.error);`,
          errors: [{ messageId: "stringifiesContainerInsteadOfMessage" }],
        },
      ],
    });
  });
});
