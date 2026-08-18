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
        // Mirrors extractTemporaryIdReferences: module-scope 'g' regex reused across an
        // outer for-loop's iterations, but the inner while loop's body has no
        // break/return/throw, so it always drains to natural exhaustion and resets
        // lastIndex to 0 on its own before the next field is scanned.
        `const TEMPORARY_ID_PATTERN = /#(aw_[A-Za-z0-9_]{3,12})\\b/gi;
         function extractTemporaryIdReferences(message) {
           const tempIds = new Set();
           const textFields = ["body", "title", "description"];
           for (const field of textFields) {
             if (typeof message[field] === "string") {
               let match;
               while ((match = TEMPORARY_ID_PATTERN.exec(message[field])) !== null) {
                 tempIds.add(match[1]);
               }
             }
           }
           return tempIds;
         }`,
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
        {
          // Reused across an outer for-loop's iterations like the valid case above, but
          // the loop body can return early, so it is not guaranteed to drain the regex
          // to natural exhaustion each time - the reset is still required.
          code: `const TEMPORARY_ID_PATTERN = /#(aw_[A-Za-z0-9_]{3,12})\\b/gi;
                 function extractTemporaryIdReferences(message, stopField) {
                   const tempIds = new Set();
                   const textFields = ["body", "title", "description"];
                   for (const field of textFields) {
                     if (typeof message[field] === "string") {
                       let match;
                       while ((match = TEMPORARY_ID_PATTERN.exec(message[field])) !== null) {
                         if (field === stopField) return tempIds;
                         tempIds.add(match[1]);
                       }
                     }
                   }
                   return tempIds;
                 }`,
          errors: [{ messageId: "requireLastIndexReset" }],
        },
        {
          // A `break` targeting the exec loop's own label still stops the loop before
          // `.exec()` naturally returns null (unlike a `continue` to the same label,
          // which just moves on to the next iteration), so it can leave `lastIndex`
          // dirty and must not be exempted just because it's nested in an outer loop.
          code: `const RE = /foo/g;
                 function scan(items) {
                   for (const item of items) {
                     let m;
                     inner: while ((m = RE.exec(item)) !== null) {
                       if (skip(m)) break inner;
                       use(m);
                     }
                   }
                 }`,
          errors: [{ messageId: "requireLastIndexReset" }],
        },
      ],
    });
  });
});
