import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

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
    const intentionalIgnoreCommentRe =
      /\bintentional\b|\bbest[- ]effort\b|\bnon[- ]fatal\b|(?<![-\w])(?:safe to )?ignore(?:d|s)?\b|\bsilently swallow(?:ed|s|ing)?\b|(?<!don't )(?<!do not )(?<!not )\bswallow(?:ed|s|ing)?\b(?=[^.!?]*(?:\bbecause\b|\berrors?\b|\bexceptions?\b|\bfailures?\b))|\bfall[- ]through\b|\bno[- ]?op\b/i;

    function commentSignalsIntentionalIgnore(comment: TSESTree.Comment): boolean {
      return intentionalIgnoreCommentRe.test(comment.value);
    }

    function hasAdjacentIntentionalIgnoreComment(node: TSESTree.Node): boolean {
      if (!node.loc) return false;
      return sourceCode.getCommentsBefore(node).some(comment => {
        if (!comment.loc || !commentSignalsIntentionalIgnore(comment)) return false;
        return node.loc.start.line - comment.loc.end.line <= 1;
      });
    }

    function hasIntentionalIgnoreComment(block: TSESTree.BlockStatement, node: TSESTree.Node): boolean {
      if (sourceCode.getCommentsInside(block).some(commentSignalsIntentionalIgnore)) return true;
      if (hasAdjacentIntentionalIgnoreComment(block)) return true;
      if (hasAdjacentIntentionalIgnoreComment(node)) return true;

      const ancestors = sourceCode.getAncestors(node);
      for (let i = ancestors.length - 1; i >= 0; i -= 1) {
        const ancestor = ancestors[i];
        if (ancestor.type === AST_NODE_TYPES.Program) break;
        if (ancestor.type.endsWith("Statement") && hasAdjacentIntentionalIgnoreComment(ancestor)) {
          return true;
        }
      }
      return false;
    }

    return {
      CatchClause(node: TSESTree.CatchClause) {
        const body = node.body.body;
        if (body.length !== 0) return;

        // An explicit intentional-ignore comment inside the otherwise empty
        // braces documents intent, e.g.:
        //   } catch { /* best-effort cleanup */ }
        if (hasIntentionalIgnoreComment(node.body, node)) return;

        context.report({
          node: node.body,
          messageId: "noEmptyCatch",
        });
      },
      "CallExpression[callee.type='MemberExpression'][callee.property.type='Identifier'][callee.property.name='catch']"(node: TSESTree.CallExpression) {
        const [handler] = node.arguments;
        if (!handler) return;
        if (handler.type !== AST_NODE_TYPES.ArrowFunctionExpression && handler.type !== AST_NODE_TYPES.FunctionExpression) return;
        if (handler.body.type !== AST_NODE_TYPES.BlockStatement) return;
        if (handler.body.body.length !== 0) return;
        if (hasIntentionalIgnoreComment(handler.body, handler)) return;

        context.report({
          node: handler.body,
          messageId: "noEmptyCatch",
        });
      },
    };
  },
});
