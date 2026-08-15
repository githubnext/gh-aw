import { ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

export const noEmptyCatchBlockRule = createRule({
  name: "no-empty-catch-block",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow empty catch blocks in actions/setup/js scripts. Swallowing an error with no logging, fallback assignment, or explicit intentional-ignore comment hides real failures (corrupted state files, cleanup errors) that are hard to diagnose from CI logs.",
    },
    schema: [],
    messages: {
      noEmptyCatch: "Empty catch block silently swallows the error. Log it (e.g. core.debug/core.warning), assign a fallback value, or add an explicit comment explaining why the error is intentionally ignored.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CatchClause(node: TSESTree.CatchClause) {
        const body = node.body.body;
        if (body.length !== 0) return;

        // An explicit intentional-ignore comment inside the otherwise empty
        // braces documents intent, e.g.:
        //   } catch { /* best-effort cleanup */ }
        const commentsInside = sourceCode.getCommentsInside(node.body);
        if (commentsInside.some(comment => /\bintentional\b|\bbest[- ]effort\b/i.test(comment.value))) return;

        context.report({
          node: node.body,
          messageId: "noEmptyCatch",
        });
      },
    };
  },
});
