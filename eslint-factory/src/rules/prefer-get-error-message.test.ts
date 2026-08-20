import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { preferGetErrorMessageRule } from "./prefer-get-error-message";

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

describe("prefer-get-error-message", () => {
  it("valid: already uses getErrorMessage", () => {
    cjsRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [`const msg = getErrorMessage(err);`],
      invalid: [],
    });
  });

  it("valid: .stack variant is intentionally excluded", () => {
    cjsRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [`const stack = err instanceof Error ? err.stack : String(err);`],
      invalid: [],
    });
  });

  it("valid: non-Error instanceof checks are intentionally excluded", () => {
    cjsRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [`const msg = err instanceof MyError ? err.message : String(err);`],
      invalid: [],
    });
  });

  it("valid: computed .message access is intentionally excluded", () => {
    cjsRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [`const msg = err instanceof Error ? err["message"] : String(err);`],
      invalid: [],
    });
  });

  it("valid: mismatched variable names in consequent are intentionally excluded", () => {
    cjsRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [`const msg = err instanceof Error ? other.message : String(err);`],
      invalid: [],
    });
  });

  it("valid: mismatched variable names in alternate are intentionally excluded", () => {
    cjsRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [`const msg = err instanceof Error ? err.message : String(other);`],
      invalid: [],
    });
  });

  it("valid: member-expression LHS in instanceof is intentionally excluded", () => {
    cjsRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [`const msg = this.err instanceof Error ? this.err.message : String(this.err);`],
      invalid: [],
    });
  });

  it("valid: alternate must be String(err)", () => {
    cjsRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [`const msg = err instanceof Error ? err.message : \`\${err}\`;`],
      invalid: [],
    });
  });

  it("valid: ternary is not flagged when getErrorMessage is unavailable", () => {
    cjsRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [`const errorMessage = err instanceof Error ? err.message : String(err);`],
      invalid: [],
    });
  });

  it("invalid: ternary is flagged with suggestion when getErrorMessage is previously imported", () => {
    cjsRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [],
      invalid: [
        {
          code: `const { getErrorMessage } = require("./error_helpers.cjs"); const errorMessage = err instanceof Error ? err.message : String(err);`,
          errors: [
            {
              messageId: "preferGetErrorMessage",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "replaceWithGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `const { getErrorMessage } = require("./error_helpers.cjs"); const errorMessage = getErrorMessage(err);`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: works in template literals", () => {
    cjsRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [],
      invalid: [
        {
          code: 'const { getErrorMessage } = require("./error_helpers.cjs"); core.warning(`failed: ${readErr instanceof Error ? readErr.message : String(readErr)}`);',
          errors: [
            {
              messageId: "preferGetErrorMessage",
              data: { errorVar: "readErr" },
              suggestions: [
                {
                  messageId: "replaceWithGetErrorMessage",
                  data: { errorVar: "readErr" },
                  output: 'const { getErrorMessage } = require("./error_helpers.cjs"); core.warning(`failed: ${getErrorMessage(readErr)}`);',
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: works in ES module files", () => {
    esmRuleTester.run("prefer-get-error-message", preferGetErrorMessageRule, {
      valid: [],
      invalid: [
        {
          code: `import { getErrorMessage } from "./error_helpers.cjs"; console.error(e instanceof Error ? e.message : String(e));`,
          errors: [
            {
              messageId: "preferGetErrorMessage",
              data: { errorVar: "e" },
              suggestions: [
                {
                  messageId: "replaceWithGetErrorMessage",
                  data: { errorVar: "e" },
                  output: `import { getErrorMessage } from "./error_helpers.cjs"; console.error(getErrorMessage(e));`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
