import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, findEnclosingStatement } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/** Function node types that form an async boundary. */
const FUNCTION_BOUNDARY_TYPES = new Set<string>([AST_NODE_TYPES.FunctionDeclaration, AST_NODE_TYPES.FunctionExpression, AST_NODE_TYPES.ArrowFunctionExpression]);

/**
 * Walks down a call/member chain and returns the root `fetch(...)` CallExpression
 * if `fetch` is the root callee, otherwise returns null.
 *
 * Handles chains like `fetch(url).then(...).catch(...)` where the AST is:
 *   CallExpression(callee=MemberExpression(object=CallExpression(callee=fetch)))
 */
function getRootFetchCall(node: TSESTree.CallExpression): TSESTree.CallExpression | null {
  let current: TSESTree.Node = node;
  while (current.type === AST_NODE_TYPES.CallExpression) {
    const call = current as TSESTree.CallExpression;
    const callee = call.callee as TSESTree.Node;
    if (callee.type === AST_NODE_TYPES.Identifier && (callee as TSESTree.Identifier).name === "fetch") {
      return call;
    }
    if (callee.type === AST_NODE_TYPES.MemberExpression) {
      current = (callee as TSESTree.MemberExpression).object;
    } else {
      return null;
    }
  }
  return null;
}

/**
 * Returns true when the call chain from `outerCall` down to `fetchCall` contains a
 * rejection handler: `.catch(handler)` or a two-argument `.then(onFulfilled, onRejected)`.
 * Such chains already handle network errors and should not be flagged.
 */
function chainHasRejectionHandler(fetchCall: TSESTree.CallExpression, outerCall: TSESTree.CallExpression): boolean {
  let current: TSESTree.Node = outerCall;
  while (current !== fetchCall) {
    if (current.type !== AST_NODE_TYPES.CallExpression) break;
    const call = current as TSESTree.CallExpression;
    const callee = call.callee as TSESTree.Node;
    if (callee.type !== AST_NODE_TYPES.MemberExpression) break;
    const member = callee as TSESTree.MemberExpression;
    const prop = member.property;
    const name = prop.type === AST_NODE_TYPES.Identifier ? (prop as TSESTree.Identifier).name : null;
    if (name === "catch" && call.arguments.length >= 1) return true;
    if (name === "then" && call.arguments.length >= 2) return true;
    current = member.object;
  }
  return false;
}

/**
 * Returns true when the node is an `await` expression whose argument is a call chain
 * rooted in the global `fetch` identifier (e.g. `await fetch(url)` or
 * `await fetch(url).then(...)`).
 */
function isAwaitFetchChain(node: TSESTree.Node): node is TSESTree.AwaitExpression {
  if (node.type !== AST_NODE_TYPES.AwaitExpression) return false;
  const argument = node.argument;
  if (argument.type !== AST_NODE_TYPES.CallExpression) return false;
  return getRootFetchCall(argument) !== null;
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
        if (!isAwaitFetchChain(node)) return;
        const outerCall = node.argument as TSESTree.CallExpression;
        const fetchCall = getRootFetchCall(outerCall)!;
        // Skip when fetch is shadowed by a local binding (e.g. a parameter or import named fetch).
        if (hasLocalBinding(node, "fetch")) return;
        // Chains with a rejection handler (.catch(handler) or .then(ok, err)) are safe.
        if (chainHasRejectionHandler(fetchCall, outerCall)) return;
        if (isInsideTryBlock(node)) return;

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
