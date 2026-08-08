import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { requireErrorCodeInThrownErrorRule } from "./require-error-code-in-thrown-error";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-error-code-in-thrown-error", () => {
  it("valid: files that do not import error_codes.cjs are not flagged", () => {
    cjsRuleTester.run("require-error-code-in-thrown-error", requireErrorCodeInThrownErrorRule, {
      valid: [`throw new Error("Something went wrong");`, `throw new Error(\`Missing field: \${name}\`);`],
      invalid: [],
    });
  });

  it("valid: thrown errors that reference an ERR_ code are not flagged", () => {
    cjsRuleTester.run("require-error-code-in-thrown-error", requireErrorCodeInThrownErrorRule, {
      valid: [
        `const { ERR_NOT_FOUND } = require("./error_codes.cjs"); throw new Error(\`\${ERR_NOT_FOUND}: Issue #1 not found\`);`,
        `const { ERR_API } = require("./error_codes.cjs"); throw new Error(ERR_API + ": failed to fetch");`,
        `const { ERR_VALIDATION } = require("./error_codes.cjs"); throw new Error("ERR_VALIDATION: missing field");`,
        `const { ERR_API } = require("./error_codes.cjs"); class CustomError extends Error {} throw new CustomError(\`\${ERR_API}: failed to fetch\`);`,
        `const { ERR_API } = require("./error_codes.cjs"); class A extends Error {} class B extends A {} throw new B(ERR_API + ": failed to fetch");`,
      ],
      invalid: [],
    });
  });

  it("invalid: thrown Error in a file importing error_codes.cjs without a code is flagged", () => {
    cjsRuleTester.run("require-error-code-in-thrown-error", requireErrorCodeInThrownErrorRule, {
      valid: [],
      invalid: [
        {
          code: `const { ERR_NOT_FOUND } = require("./error_codes.cjs"); function f() { throw new Error(\`node_id missing for issue\`); }`,
          errors: [{ messageId: "missingErrorCode" }],
        },
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); function f(id) { throw new Error("Cannot mark issue as duplicate of " + id); }`,
          errors: [{ messageId: "missingErrorCode" }],
        },
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); class CustomError extends Error {} function f() { throw new CustomError("failed to fetch"); }`,
          errors: [{ messageId: "missingErrorCode" }],
        },
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); class A extends Error {} class B extends A {} function f() { throw new B("failed to fetch"); }`,
          errors: [{ messageId: "missingErrorCode" }],
        },
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); function f() { throw new CustomError("failed to fetch"); } class CustomError extends Error {}`,
          errors: [{ messageId: "missingErrorCode" }],
        },
      ],
    });
  });

  it("invalid: non-Error throws and other constructs are ignored", () => {
    cjsRuleTester.run("require-error-code-in-thrown-error", requireErrorCodeInThrownErrorRule, {
      valid: [`const { ERR_API } = require("./error_codes.cjs"); function f() { throw someError; }`, `const { ERR_API } = require("./error_codes.cjs"); function f() { throw new TypeError("bad type"); }`],
      invalid: [],
    });
  });
});
