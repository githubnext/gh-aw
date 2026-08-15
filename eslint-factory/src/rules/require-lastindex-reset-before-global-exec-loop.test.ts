import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireLastIndexResetBeforeGlobalExecLoopRule } from "./require-lastindex-reset-before-global-exec-loop";

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-lastindex-reset-before-global-exec-loop", () => {
  it("uses the correct docs URL", () => {
    expect(requireLastIndexResetBeforeGlobalExecLoopRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-lastindex-reset-before-global-exec-loop");
  });

  it("accepts loops that reset lastIndex, non-global regexes, and local regex literals", () => {
    ruleTester.run("require-lastindex-reset-before-global-exec-loop", requireLastIndexResetBeforeGlobalExecLoopRule, {
      valid: [
        // Explicit reset right before the loop.
        `const RE = /foo/g;
         function scan(text) {
           RE.lastIndex = 0;
           let match;
           while ((match = RE.exec(text)) !== null) {
             use(match);
           }
         }`,
        // Reset anywhere earlier in the same function.
        `const RE = /foo/g;
         function scan(text) {
           RE.lastIndex = 0;
           doOtherWork();
           let match;
           while ((match = RE.exec(text)) !== null) {
             use(match);
           }
         }`,
        // Non-global, non-sticky regex is not stateful across calls in the same way.
        `const RE = /foo/;
         function scan(text) {
           let match;
           while ((match = RE.exec(text)) !== null) {
             use(match);
           }
         }`,
        // Regex declared locally inside the function is freshly created each call.
        `function scan(text) {
           const RE = /foo/g;
           let match;
           while ((match = RE.exec(text)) !== null) {
             use(match);
           }
         }`,
        // Unrelated while loop.
        `while (x < 10) { x++; }`,
      ],
      invalid: [
        {
          code: `const RE = /foo/g;
                 function scan(text) {
                   let match;
                   while ((match = RE.exec(text)) !== null) {
                     use(match);
                   }
                 }`,
          errors: [{ messageId: "requireLastIndexReset" }],
        },
        {
          code: `const TEMPORARY_ID_PATTERN = /#(aw_[A-Za-z0-9_]{3,12})\\b/gi;
                 function extract(message) {
                   let match;
                   while ((match = TEMPORARY_ID_PATTERN.exec(message.body)) !== null) {
                     use(match);
                   }
                 }`,
          errors: [{ messageId: "requireLastIndexReset" }],
        },
        {
          code: `const RE = /foo/y;
                 function scan(text) {
                   let match;
                   while ((match = RE.exec(text)) !== null) {
                     use(match);
                   }
                 }`,
          errors: [{ messageId: "requireLastIndexReset" }],
        },
      ],
    });
  });
});
