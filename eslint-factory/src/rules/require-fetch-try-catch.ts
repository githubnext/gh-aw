import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, findEnclosingStatement, isInsideTryBlock } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/**
 * Returns true when the node is an `await fetch(...)` expression (AwaitExpression wrapping
 * a CallExpression whose callee is the global `fetch` identifier).
 */
function isAwaitFetchCall(node: TSESTree.Node): node is TSESTree.AwaitExpression {
  if (node.type !== AST_NODE_TYPES.AwaitExpression) return false;
  const argument = node.argument;
  if (argument.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = argument.callee;
  return callee.type === AST_NODE_TYPES.Identifier && callee.name === "fetch";
}

export const requireFetchTryCatchRule = createRule({
  name: "require-fetch-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require `await fetch(...)` calls in actions/setup/js scripts to be wrapped in try/catch. " +
        "The fetch API throws a TypeError on network failures (DNS errors, connection refused, etc.) " +
        "which will crash the action with an unhelpful uncaught exception if unhandled.",
    },
    schema: [],
    messages: {
      requireTryCatch:
        "Wrap `await fetch({{url}})` in try/catch — fetch throws TypeError on network errors " +
        "and will crash the action if unhandled.",
      wrapInTryCatch: "Wrap in try { ... } catch { ... } and re-throw with { cause: err } to preserve context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      AwaitExpression(node) {
        if (!isAwaitFetchCall(node)) return;
        if (isInsideTryBlock(sourceCode, node)) return;

        const fetchCall = node.argument as TSESTree.CallExpression;
        const firstArg = fetchCall.arguments[0];
        const urlText = firstArg !== undefined ? sourceCode.getText(firstArg as TSESTree.Node) : "";
        const stmt = findEnclosingStatement(sourceCode, node);

        context.report({
          node,
          messageId: "requireTryCatch",
          data: { url: urlText },
          suggest: stmt
            ? [
                {
                  messageId: "wrapInTryCatch",
                  fix(fixer) {
                    const stmtText = sourceCode.getText(stmt);
                    const startLine = stmt.loc?.start.line;
                    const stmtLine = startLine !== undefined ? (sourceCode.lines[startLine - 1] ?? "") : "";
                    const indent = stmtLine.match(/^(\s*)/)?.[1] ?? "";
                    return fixer.replaceText(
                      stmt,
                      buildTryCatchSuggestion(stmtText, {
                        indent,
                        todoComment: "TODO: handle fetch network failure (TypeError on DNS/connection errors).",
                        errorPrefix: "fetch failed: ",
                      })
                    );
                  },
                },
              ]
            : [],
        });
      },
    };
  },
});
