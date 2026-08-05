import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { preferStructuredCloneRule } from "./prefer-structured-clone";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("prefer-structured-clone", () => {
  it("uses the correct docs URL", () => {
    expect(preferStructuredCloneRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#prefer-structured-clone");
  });

  it("valid: unrelated JSON.parse / JSON.stringify usages are accepted", () => {
    cjsRuleTester.run("prefer-structured-clone", preferStructuredCloneRule, {
      valid: [
        `const data = JSON.parse(rawText);`,
        `const text = JSON.stringify(obj);`,
        `structuredClone(obj);`,
        // Replacer/indent argument changes stringify semantics; excluded to avoid false positives.
        `const clone = JSON.parse(JSON.stringify(obj, null, 2));`,
        // Custom reviver on parse changes semantics too; still matched today since only the
        // parse call itself is checked for a single stringify argument — but the inner
        // stringify call must have exactly one argument, so this is out of scope here.
        `const clone = JSON.parse(JSON.stringify(obj), reviver);`,
      ],
      invalid: [],
    });
  });

  it("invalid: JSON.parse(JSON.stringify(x)) is flagged and suggests structuredClone(x)", () => {
    cjsRuleTester.run("prefer-structured-clone", preferStructuredCloneRule, {
      valid: [],
      invalid: [
        {
          code: `const clone = JSON.parse(JSON.stringify(tool));`,
          errors: [
            {
              messageId: "preferStructuredClone",
              suggestions: [
                {
                  messageId: "replaceWithStructuredClone",
                  output: `const clone = structuredClone(tool);`,
                },
              ],
            },
          ],
        },
        {
          code: `const runs = state.runs.map(run => JSON.parse(JSON.stringify(run)));`,
          errors: [
            {
              messageId: "preferStructuredClone",
              suggestions: [
                {
                  messageId: "replaceWithStructuredClone",
                  output: `const runs = state.runs.map(run => structuredClone(run));`,
                },
              ],
            },
          ],
        },
        {
          code: `const clone = JSON["parse"](JSON["stringify"](tool));`,
          errors: [
            {
              messageId: "preferStructuredClone",
              suggestions: [
                {
                  messageId: "replaceWithStructuredClone",
                  output: `const clone = structuredClone(tool);`,
                },
              ],
            },
          ],
        },
      ],
    });
  });
});
