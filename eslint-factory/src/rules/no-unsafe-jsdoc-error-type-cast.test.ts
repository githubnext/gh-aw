import { RuleTester } from "eslint";
import { describe, it } from "vitest";
import { noUnsafeJsdocErrorTypeCastRule } from "./no-unsafe-jsdoc-error-type-cast";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("no-unsafe-jsdoc-error-type-cast", () => {
  it("invalid: declarator cast then unsafe .message/.name access", () => {
    cjsRuleTester.run("no-unsafe-jsdoc-error-type-cast", noUnsafeJsdocErrorTypeCastRule, {
      valid: [],
      invalid: [
        {
          code: `try {} catch (error) { const err = /** @type {Error} */ error; log(err.message); }`,
          errors: [
            {
              messageId: "unsafeCast",
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  output: `try {} catch (error) { const err = /** @type {Error} */ error; log(getErrorMessage(err)); }`,
                },
              ],
            },
          ],
        },
        {
          code: `try {} catch (err) { const e = /** @type {Error} */ err; if (e.name === "AbortError") {} }`,
          errors: [{ messageId: "unsafeCast" }],
        },
      ],
    });
  });

  it("invalid: assignment-form cast then unsafe .stack/.code access", () => {
    cjsRuleTester.run("no-unsafe-jsdoc-error-type-cast", noUnsafeJsdocErrorTypeCastRule, {
      valid: [],
      invalid: [
        {
          code: `let e; try {} catch (err) { e = /** @type {Error} */ err; log(e.stack); }`,
          errors: [{ messageId: "unsafeCast" }],
        },
        {
          code: `try {} catch (err) { const e = /** @type {Error} */ err; log(e["code"]); }`,
          errors: [{ messageId: "unsafeCast" }],
        },
      ],
    });
  });

  it("invalid: suggests getErrorMessage for .message access", () => {
    cjsRuleTester.run("no-unsafe-jsdoc-error-type-cast", noUnsafeJsdocErrorTypeCastRule, {
      valid: [],
      invalid: [
        {
          code: `try {} catch (error) { const err = /** @type {Error} */ error; log(err.message); }`,
          errors: [
            {
              messageId: "unsafeCast",
              suggestions: [
                {
                  messageId: "useGetErrorMessage",
                  output: `try {} catch (error) { const err = /** @type {Error} */ error; log(getErrorMessage(err)); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });

  it("valid: no JSDoc cast, or safe access patterns", () => {
    cjsRuleTester.run("no-unsafe-jsdoc-error-type-cast", noUnsafeJsdocErrorTypeCastRule, {
      valid: [
        // No cast at all — plain reassignment.
        `try {} catch (error) { const err = error; log(err.message); }`,
        // Cast variable never accesses an unsafe property.
        `try {} catch (error) { const err = /** @type {Error} */ error; log(String(err)); }`,
        // Cast to a different type is unrelated to this rule.
        `try {} catch (error) { const err = /** @type {string} */ error; log(err.length); }`,
      ],
      invalid: [],
    });
  });
});
