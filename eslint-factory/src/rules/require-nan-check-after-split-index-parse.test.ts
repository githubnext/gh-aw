import { RuleTester } from "eslint";
import { describe, expect, it } from "vitest";
import { requireNanCheckAfterSplitIndexParseRule } from "./require-nan-check-after-split-index-parse";

const cjsRuleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: "commonjs",
  },
});

describe("require-nan-check-after-split-index-parse", () => {
  it("uses the correct docs URL", () => {
    expect(requireNanCheckAfterSplitIndexParseRule.meta.docs.url).toBe("https://github.com/github/gh-aw/tree/main/eslint-factory#require-nan-check-after-split-index-parse");
  });

  it("valid: parseInt from split(...)[index] validated with Number.isNaN", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [`const discussionNumber = parseInt(endpoint.split(":")[1], 10); if (Number.isNaN(discussionNumber)) throw new Error("invalid");`],
      invalid: [],
    });
  });

  it("valid: parseInt from split(...)[index] validated with global isNaN", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [`const discussionNumber = parseInt(endpoint.split(":")[1], 10); if (isNaN(discussionNumber)) throw new Error("invalid");`],
      invalid: [],
    });
  });

  it("valid: Number.parseInt from split(...)[index] validated with Number.isNaN", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [`const discussionNumber = Number.parseInt(endpoint.split(":")[1], 10); if (Number.isNaN(discussionNumber)) throw new Error("invalid");`],
      invalid: [],
    });
  });

  it("valid: truthiness guard on parsed value", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [`const discussionNumber = parseInt(endpoint.split(":")[1], 10); if (!discussionNumber) throw new Error("invalid");`],
      invalid: [],
    });
  });

  it("valid: parseInt not from split(...)[index] is not flagged", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [`const count = parseInt(rawValue, 10); doSomething(count);`],
      invalid: [],
    });
  });

  it("invalid: parseInt from split(...)[index] without NaN check", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [],
      invalid: [
        {
          code: `const discussionNumber = parseInt(endpoint.split(":")[1], 10); getDiscussionNodeId(owner, repo, discussionNumber);`,
          errors: [{ messageId: "requireNaNCheck" }],
        },
      ],
    });
  });

  it("invalid: Number.parseInt from split(...)[index] without NaN check", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [],
      invalid: [
        {
          code: `const discussionNumber = Number.parseInt(endpoint.split(":")[1], 10); getDiscussionNodeId(owner, repo, discussionNumber);`,
          errors: [{ messageId: "requireNaNCheck" }],
        },
      ],
    });
  });

  it("invalid: parseFloat from split(...)[index] without NaN check", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [],
      invalid: [
        {
          code: `const version = parseFloat(tag.split("v")[1]); doSomething(version);`,
          errors: [{ messageId: "requireNaNCheck" }],
        },
      ],
    });
  });

  it("invalid: same-named variable validated in one scope does not suppress an unvalidated occurrence in another scope", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [],
      invalid: [
        {
          code: `
function a() {
  const discussionNumber = parseInt(endpoint.split(":")[1], 10);
  if (!discussionNumber) throw new Error("invalid");
  return discussionNumber;
}
function b() {
  const discussionNumber = parseInt(endpoint.split(":")[1], 10);
  return getDiscussionNodeId(owner, repo, discussionNumber);
}
          `.trim(),
          errors: [{ messageId: "requireNaNCheck" }],
        },
      ],
    });
  });

  it("valid: locally shadowed parseInt is not treated as the global parser", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [`function parseInt(x) { return x; } const discussionNumber = parseInt(endpoint.split(":")[1]); getDiscussionNodeId(owner, repo, discussionNumber);`],
      invalid: [],
    });
  });

  it("valid: locally shadowed Number is not treated as the global Number", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [`function outer(Number) { const discussionNumber = Number.parseInt(endpoint.split(":")[1], 10); getDiscussionNodeId(owner, repo, discussionNumber); }`],
      invalid: [],
    });
  });

  it("invalid: locally shadowed isNaN does not count as validation", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [],
      invalid: [
        {
          code: `function isNaN(x) { return false; } const discussionNumber = parseInt(endpoint.split(":")[1], 10); if (isNaN(discussionNumber)) throw new Error("invalid"); getDiscussionNodeId(owner, repo, discussionNumber);`,
          errors: [{ messageId: "requireNaNCheck" }],
        },
      ],
    });
  });

  it("invalid: locally shadowed Number.isNaN does not count as validation", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [],
      invalid: [
        {
          code: `function outer(Number) { const discussionNumber = parseInt(endpoint.split(":")[1], 10); if (Number.isNaN(discussionNumber)) throw new Error("invalid"); getDiscussionNodeId(owner, repo, discussionNumber); }`,
          errors: [{ messageId: "requireNaNCheck" }],
        },
      ],
    });
  });

  it("invalid: multiple unvalidated split-index parse declarations are each reported", () => {
    cjsRuleTester.run("require-nan-check-after-split-index-parse", requireNanCheckAfterSplitIndexParseRule, {
      valid: [],
      invalid: [
        {
          code: `const a = parseInt(x.split(":")[1], 10); const b = parseInt(y.split(":")[1], 10); use(a, b);`,
          errors: [{ messageId: "requireNaNCheck" }, { messageId: "requireNaNCheck" }],
        },
      ],
    });
  });
});
