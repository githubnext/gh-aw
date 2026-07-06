import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireErrorCauseInRethrowRule } from "./require-error-cause-in-rethrow";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-error-cause-in-rethrow", () => {
  it("uses the correct docs URL", () => {
    expect(requireErrorCauseInRethrowRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-error-cause-in-rethrow");
  });

  it("valid: rethrow with { cause: err } passes", () => {
    cjsRuleTester.run("require-error-cause-in-rethrow", requireErrorCauseInRethrowRule, {
      valid: [
        // Single message arg + cause
        `try { doSomething(); } catch (err) { throw new Error("Failed: " + getErrorMessage(err), { cause: err }); }`,
        // Template literal with cause
        `try { doSomething(); } catch (err) { throw new Error(\`Failed: \${getErrorMessage(err)}\`, { cause: err }); }`,
        // Cause with extra properties
        `try { doSomething(); } catch (err) { throw new Error("msg: " + getErrorMessage(err), { cause: err, extra: 1 }); }`,
        // Error that does NOT reference the catch var — no violation expected
        `try { doSomething(); } catch (err) { throw new Error("Something went wrong"); }`,
        // Bare catch with no binding — should not flag
        `try { doSomething(); } catch { throw new Error("Something went wrong"); }`,
        // Error subclass with cause (not flagged — rule only checks `new Error`)
        `try { doSomething(); } catch (err) { throw new TypeError(\`Failed: \${getErrorMessage(err)}\`, { cause: err }); }`,
        // Error subclass without cause (not flagged — rule only checks `new Error`)
        `try { doSomething(); } catch (err) { throw new TypeError(\`Failed: \${getErrorMessage(err)}\`); }`,
        // Direct catch var reference with cause
        `try { doSomething(); } catch (err) { throw new Error(\`Outer: \${err.message}\`, { cause: err }); }`,
        // Nested function inside catch — should NOT flag (deferred execution boundary)
        `try { doSomething(); } catch (err) { const fn = function() { throw new Error(\`msg: \${getErrorMessage(err)}\`); }; }`,
        // Nested arrow function inside catch — should NOT flag (deferred execution boundary)
        `try { doSomething(); } catch (err) { const fn = () => { throw new Error(\`msg: \${getErrorMessage(err)}\`); }; }`,
        // Constructing Error without throw is not a rethrow
        `try { doSomething(); } catch (err) { const e = new Error(\`msg: \${getErrorMessage(err)}\`); log(e); }`,
        // Existing cause property with wrapped expression should not be flagged.
        `try { doSomething(); } catch (err) { throw new Error("Failed: " + getErrorMessage(err), { cause: new Error(err.message), code: 500 }); }`,
      ],
      invalid: [],
    });
  });

  it("invalid: rethrow references catch var but omits { cause }", () => {
    cjsRuleTester.run("require-error-cause-in-rethrow", requireErrorCauseInRethrowRule, {
      valid: [],
      invalid: [
        {
          code: `try { doSomething(); } catch (err) { throw new Error("Failed: " + getErrorMessage(err)); }`,
          errors: [
            {
              messageId: "missingCause",
              data: { catchVar: "err" },
              suggestions: [
                {
                  messageId: "addCause",
                  output: `try { doSomething(); } catch (err) { throw new Error("Failed: " + getErrorMessage(err), { cause: err }); }`,
                },
              ],
            },
          ],
        },
        {
          code: `try { doSomething(); } catch (err) { throw new Error(\`Failed: \${getErrorMessage(err)}\`); }`,
          errors: [
            {
              messageId: "missingCause",
              data: { catchVar: "err" },
              suggestions: [
                {
                  messageId: "addCause",
                  output: `try { doSomething(); } catch (err) { throw new Error(\`Failed: \${getErrorMessage(err)}\`, { cause: err }); }`,
                },
              ],
            },
          ],
        },
        {
          code: `try { doSomething(); } catch (error) { throw new Error(\`\${ERR_PARSE}: \${getErrorMessage(error)}\`); }`,
          errors: [
            {
              messageId: "missingCause",
              data: { catchVar: "error" },
              suggestions: [
                {
                  messageId: "addCause",
                  output: `try { doSomething(); } catch (error) { throw new Error(\`\${ERR_PARSE}: \${getErrorMessage(error)}\`, { cause: error }); }`,
                },
              ],
            },
          ],
        },
        {
          // Second arg exists but no cause property
          code: `try { doSomething(); } catch (err) { throw new Error("Failed: " + getErrorMessage(err), { code: 500 }); }`,
          errors: [
            {
              messageId: "missingCause",
              data: { catchVar: "err" },
              suggestions: [
                {
                  messageId: "addCause",
                  output: `try { doSomething(); } catch (err) { throw new Error("Failed: " + getErrorMessage(err), { cause: err, code: 500 }); }`,
                },
              ],
            },
          ],
        },
        {
          // Direct reference to catch var in message (not via getErrorMessage)
          code: `try { doSomething(); } catch (err) { throw new Error(\`Failed: \${err}\`); }`,
          errors: [
            {
              messageId: "missingCause",
              data: { catchVar: "err" },
              suggestions: [
                {
                  messageId: "addCause",
                  output: `try { doSomething(); } catch (err) { throw new Error(\`Failed: \${err}\`, { cause: err }); }`,
                },
              ],
            },
          ],
        },
        {
          // Catch var reference via member access in message
          code: `try { doSomething(); } catch (err) { throw new Error(\`Failed: \${err.message}\`); }`,
          errors: [
            {
              messageId: "missingCause",
              data: { catchVar: "err" },
              suggestions: [
                {
                  messageId: "addCause",
                  output: `try { doSomething(); } catch (err) { throw new Error(\`Failed: \${err.message}\`, { cause: err }); }`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
