import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noErrStackThenStringFallbackRule } from "./no-err-stack-then-string-fallback";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("no-err-stack-then-string-fallback", () => {
  it("valid: already uses getErrorMessage", () => {
    cjsRuleTester.run("no-err-stack-then-string-fallback", noErrStackThenStringFallbackRule, {
      valid: [`const msg = getErrorMessage(err);`],
      invalid: [],
    });
  });

  it("valid: instanceof form is handled by prefer-get-error-message", () => {
    cjsRuleTester.run("no-err-stack-then-string-fallback", noErrStackThenStringFallbackRule, {
      valid: [`const msg = err instanceof Error ? err.message : String(err);`],
      invalid: [],
    });
  });

  it("valid: mismatched variable names are excluded", () => {
    cjsRuleTester.run("no-err-stack-then-string-fallback", noErrStackThenStringFallbackRule, {
      valid: [`const msg = err && err.stack ? err.stack : String(other);`],
      invalid: [],
    });
  });

  it("valid: mismatched object in err.stack check is excluded", () => {
    cjsRuleTester.run("no-err-stack-then-string-fallback", noErrStackThenStringFallbackRule, {
      valid: [`const msg = err && other.stack ? err.stack : String(err);`],
      invalid: [],
    });
  });

  it("valid: mismatched consequent variable is excluded", () => {
    cjsRuleTester.run("no-err-stack-then-string-fallback", noErrStackThenStringFallbackRule, {
      valid: [`const msg = err && err.stack ? other.stack : String(err);`],
      invalid: [],
    });
  });

  it("valid: logical-OR form is intentionally out of scope", () => {
    cjsRuleTester.run("no-err-stack-then-string-fallback", noErrStackThenStringFallbackRule, {
      valid: [`const msg = err.stack || String(err);`],
      invalid: [],
    });
  });

  it("valid: test with different property than stack is excluded", () => {
    cjsRuleTester.run("no-err-stack-then-string-fallback", noErrStackThenStringFallbackRule, {
      valid: [`const msg = err && err.message ? err.message : String(err);`],
      invalid: [],
    });
  });

  it("invalid: core.setFailed(err && err.stack ? err.stack : String(err)) is flagged", () => {
    cjsRuleTester.run("no-err-stack-then-string-fallback", noErrStackThenStringFallbackRule, {
      valid: [],
      invalid: [
        {
          code: `core.setFailed(err && err.stack ? err.stack : String(err));`,
          errors: [
            {
              messageId: "preferGetErrorMessage",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "replaceWithGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `core.setFailed(getErrorMessage(err));`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: standalone assignment is flagged", () => {
    cjsRuleTester.run("no-err-stack-then-string-fallback", noErrStackThenStringFallbackRule, {
      valid: [],
      invalid: [
        {
          code: `const msg = err && err.stack ? err.stack : String(err);`,
          errors: [
            {
              messageId: "preferGetErrorMessage",
              data: { errorVar: "err" },
              suggestions: [
                {
                  messageId: "replaceWithGetErrorMessage",
                  data: { errorVar: "err" },
                  output: `const msg = getErrorMessage(err);`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("invalid: different variable name (error) is also flagged", () => {
    cjsRuleTester.run("no-err-stack-then-string-fallback", noErrStackThenStringFallbackRule, {
      valid: [],
      invalid: [
        {
          code: `console.error(error && error.stack ? error.stack : String(error));`,
          errors: [
            {
              messageId: "preferGetErrorMessage",
              data: { errorVar: "error" },
              suggestions: [
                {
                  messageId: "replaceWithGetErrorMessage",
                  data: { errorVar: "error" },
                  output: `console.error(getErrorMessage(error));`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
