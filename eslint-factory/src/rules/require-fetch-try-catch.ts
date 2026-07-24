import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, findEnclosingStatement } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/** Function node types that form an async boundary. */
const FUNCTION_BOUNDARY_TYPES = new Set<string>([AST_NODE_TYPES.FunctionDeclaration, AST_NODE_TYPES.FunctionExpression, AST_NODE_TYPES.ArrowFunctionExpression]);

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
      requireTryCatch: "Wrap `await fetch({{url}})` in try/catch — fetch throws TypeError on network errors " + "and will crash the action if unhandled.",
      wrapInTryCatch: "Wrap in try { ... } catch { ... } and re-throw with { cause: err } to preserve context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

    /** Returns true when name is bound by a local definition, meaning it shadows the global. */
    function hasLocalBinding(node: TSESTree.Node, name: string): boolean {
      let scope: SourceCodeScope | null = sourceCode.getScope(node);
      while (scope) {
        const variable = scope.set.get(name);
        if (variable?.defs.length) {
          return true;
        }
        scope = scope.upper;
      }
      return false;
    }

    /**
     * Returns true when node is inside a try block within the same function scope.
     * Stops at any function boundary: a try/catch outside the enclosing async function
     * cannot catch a rejected promise from an await inside a nested function.
     */
    function isInsideTryBlock(node: TSESTree.Node): boolean {
      const ancestors = sourceCode.getAncestors(node);

      for (let i = ancestors.length - 1; i >= 0; i--) {
        const ancestor = ancestors[i];

        // Any function boundary (declaration, expression, or arrow) stops the search.
        // A try/catch outside the current async function cannot protect this await.
        if (FUNCTION_BOUNDARY_TYPES.has(ancestor.type)) {
          return false;
        }

        if (ancestor.type === AST_NODE_TYPES.TryStatement && ancestor.handler != null) {
          const block = ancestor.block;
          if (node.range != null && block.range != null && node.range[0] >= block.range[0] && node.range[1] <= block.range[1]) {
            return true;
          }
        }
      }

      return false;
    }

    return {
      AwaitExpression(node) {
        if (!isAwaitFetchCall(node)) return;
        // Skip when fetch is shadowed by a local binding (e.g. a parameter or import named fetch).
        if (hasLocalBinding(node, "fetch")) return;
        if (isInsideTryBlock(node)) return;

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
