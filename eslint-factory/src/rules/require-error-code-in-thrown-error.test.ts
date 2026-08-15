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
        `const { SAFE_OUTPUT_E099 } = require("./error_codes.cjs"); throw new Error(\`\${SAFE_OUTPUT_E099}: failed to fetch\`);`,
        `const { MY_PREFIX_ERR_CONFIG } = require("./error_codes.cjs"); throw new Error(\`\${MY_PREFIX_ERR_CONFIG}: invalid configuration\`);`,
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
          code: `const { MY_PREFIX_ERROR_CONFIG } = require("./error_codes.cjs"); function f() { throw new Error(\`\${MY_PREFIX_ERROR_CONFIG}: invalid configuration\`); }`,
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

  it("valid: identifiers whose write-once initializer carries an error code are not flagged", () => {
    cjsRuleTester.run("require-error-code-in-thrown-error", requireErrorCodeInThrownErrorRule, {
      valid: [
        `const { ERR_SYSTEM } = require("./error_codes.cjs"); function f(result) { const errorMsg = \`\${ERR_SYSTEM}: \${result.stderr}\`; throw new Error(errorMsg); }`,
        `const { ERR_CONFIG } = require("./error_codes.cjs"); function f(p) { const msg = \`\${ERR_CONFIG}: Source tools file not found at: \${p}\`; throw new Error(msg); }`,
        `const { ERR_API } = require("./error_codes.cjs"); function f() { const msg = ERR_API + ": boom"; throw new Error(msg); }`,
        `const { ERR_CONFIG } = require("./error_codes.cjs"); function f(p) { const inner = \`\${ERR_CONFIG}: bad\`; const outer = inner; throw new Error(outer); }`,
      ],
      invalid: [],
    });
  });

  it("valid: identifiers whose value cannot be statically resolved are not flagged", () => {
    cjsRuleTester.run("require-error-code-in-thrown-error", requireErrorCodeInThrownErrorRule, {
      valid: [
        `const { ERR_CONFIG } = require("./error_codes.cjs"); function f(errorMessage) { throw new Error(errorMessage); }`,
        `const { ERR_CONFIG } = require("./error_codes.cjs"); function f(e) { let msg = "boom"; msg = String(e); throw new Error(msg); }`,
      ],
      invalid: [],
    });
  });

  it("invalid: identifiers whose write-once initializer lacks an error code are still flagged", () => {
    cjsRuleTester.run("require-error-code-in-thrown-error", requireErrorCodeInThrownErrorRule, {
      valid: [],
      invalid: [
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); function f() { const msg = "no code here"; throw new Error(msg); }`,
          errors: [{ messageId: "missingErrorCode" }],
        },
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); function f() { const msg = "no code" + " here"; throw new Error(msg); }`,
          errors: [{ messageId: "missingErrorCode" }],
        },
      ],
    });
  });

  it("valid: identifiers whose write-once initializer is not message-like stay silent", () => {
    cjsRuleTester.run("require-error-code-in-thrown-error", requireErrorCodeInThrownErrorRule, {
      valid: [`const { ERR_API } = require("./error_codes.cjs"); function f(left, right) { const msg = left - right; throw new Error(msg); }`],
      invalid: [],
    });
  });

  it("invalid: non-Error throws and other constructs are ignored", () => {
    cjsRuleTester.run("require-error-code-in-thrown-error", requireErrorCodeInThrownErrorRule, {
      valid: [`const { ERR_API } = require("./error_codes.cjs"); function f() { throw someError; }`, `const { ERR_API } = require("./error_codes.cjs"); function f() { throw new TypeError("bad type"); }`],
      invalid: [],
    });
  });

  it("valid: shadowed constructor (parameter shadows outer class) is not flagged", () => {
    cjsRuleTester.run("require-error-code-in-thrown-error", requireErrorCodeInThrownErrorRule, {
      valid: [
        // 'CustomError' as a parameter shadows the top-level class — cannot determine it extends Error
        `const { ERR_API } = require("./error_codes.cjs"); class CustomError extends Error {} function f(CustomError) { throw new CustomError("not an Error subclass here"); }`,
        // Two separate functions each declaring a local 'CustomError' — they shadow each other, only the outer class would otherwise leak
        `const { ERR_API } = require("./error_codes.cjs"); class CustomError extends Error {} function f(x) { class CustomError { constructor(m) {} } throw new CustomError("plain class, not Error"); }`,
      ],
      invalid: [
        // Outer class is NOT shadowed — scope lookup still finds the Error subclass and flags it
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); class CustomError extends Error {} function f() { throw new CustomError("unshadowed, must be flagged"); }`,
          errors: [{ messageId: "missingErrorCode" }],
        },
      ],
    });
  });

  it("handles helper call message arguments explicitly", () => {
    cjsRuleTester.run("require-error-code-in-thrown-error", requireErrorCodeInThrownErrorRule, {
      valid: [
        `const { ERR_API } = require("./error_codes.cjs"); function helper() { return "ERR_API: failed to fetch"; } function f() { throw new Error(helper()); }`,
        `const { ERR_CONFIG } = require("./error_codes.cjs"); const helper = () => ERR_CONFIG + ": invalid config"; function f() { throw new Error(helper()); }`,
        `const { ERR_API } = require("./error_codes.cjs"); function helper() { const message = ERR_API + ": failed to fetch"; return message; } function f() { throw new Error(helper()); }`,
      ],
      invalid: [
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); function helper() { return "failed to fetch"; } function f() { throw new Error(helper()); }`,
          errors: [{ messageId: "missingErrorCode" }],
        },
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); function helper(reason) { return reason; } function f(reason) { throw new Error(helper(reason)); }`,
          errors: [{ messageId: "callExpressionNeedsReview" }],
        },
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); const { helper } = require("./helper.cjs"); function f() { throw new Error(helper()); }`,
          errors: [{ messageId: "callExpressionNeedsReview" }],
        },
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); function f() { const message = helper(); throw new Error(message); }`,
          errors: [{ messageId: "callExpressionNeedsReview" }],
        },
        {
          code: `const { ERR_API } = require("./error_codes.cjs"); function helper(useFallback) { if (useFallback) return "fallback"; return ERR_API + ": failed"; } function f(useFallback) { throw new Error(helper(useFallback)); }`,
          errors: [{ messageId: "callExpressionNeedsReview" }],
        },
      ],
    });
  });
});
