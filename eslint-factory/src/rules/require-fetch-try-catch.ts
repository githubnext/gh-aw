import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, findEnclosingStatement } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/** Function node types that form an async boundary. */
const FUNCTION_BOUNDARY_TYPES = new Set<string>([AST_NODE_TYPES.FunctionDeclaration, AST_NODE_TYPES.FunctionExpression, AST_NODE_TYPES.ArrowFunctionExpression]);

function getMemberPropertyName(node: TSESTree.MemberExpression): string | null {
  const property = node.property;
  if (!node.computed && property.type === AST_NODE_TYPES.Identifier) return property.name;
  if (node.computed && property.type === AST_NODE_TYPES.Literal && typeof property.value === "string") return property.value;
  return null;
}

/**
 * Returns true when a call argument is statically non-callable.
 * Promise rejection callbacks of `null`, `undefined`, any literal value, or spread elements
 * are replaced by the default thrower and do NOT suppress rejection.
 */
function isStaticallyNonCallable(node: TSESTree.Expression | TSESTree.SpreadElement): boolean {
  if (node.type === AST_NODE_TYPES.SpreadElement) return true;
  if (node.type === AST_NODE_TYPES.Literal) return true;
  if (node.type === AST_NODE_TYPES.Identifier && node.name === "undefined") return true;
  return false;
}

interface AwaitedFetchInfo {
  fetchCall: TSESTree.CallExpression;
  hasRejectionHandler: boolean;
}

/**
 * Returns info when the node is an awaited fetch call, including member-chained forms like
 * `await fetch(url).then(...)` and whether the chain already carries a rejection handler.
 */
function getAwaitedFetchInfo(node: TSESTree.Node): AwaitedFetchInfo | null {
  if (node.type !== AST_NODE_TYPES.AwaitExpression) return null;

  let current: TSESTree.Expression | TSESTree.Super = node.argument;
  let hasRejectionHandler = false;

  while (true) {
    if (current.type === AST_NODE_TYPES.Super) return null;

    // Unwrap optional chains: `fetch(url)?.then(ok)` is wrapped in a ChainExpression by Espree.
    if (current.type === AST_NODE_TYPES.ChainExpression) {
      current = current.expression;
      continue;
    }

    if (current.type === AST_NODE_TYPES.CallExpression) {
      const callee = current.callee;

      if (callee.type === AST_NODE_TYPES.Identifier && callee.name === "fetch") {
        return { fetchCall: current, hasRejectionHandler };
      }

      if (callee.type !== AST_NODE_TYPES.MemberExpression) return null;

      const methodName = getMemberPropertyName(callee);
      if (methodName === "catch" && current.arguments.length >= 1 && !isStaticallyNonCallable(current.arguments[0])) hasRejectionHandler = true;
      if (methodName === "then" && current.arguments.length >= 2 && !isStaticallyNonCallable(current.arguments[1])) hasRejectionHandler = true;

      current = callee.object;
      continue;
    }

    if (current.type === AST_NODE_TYPES.MemberExpression) {
      current = current.object;
      continue;
    }

    return null;
  }
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
     * Stops at any function boundary: a try/catch outside a non-awaited (fire-and-forget)
     * callback cannot catch a rejected promise from an await inside that callback.
     *
     * Exception: if the enclosing function is an inline callback passed to a call expression
     * that is itself awaited inside a try block, the rejected promise propagates through the
     * awaited chain and IS caught by the outer catch.
     */
    function isInsideTryBlock(node: TSESTree.Node): boolean {
      const ancestors = sourceCode.getAncestors(node);

      for (let i = ancestors.length - 1; i >= 0; i--) {
        const ancestor = ancestors[i];

        // Any function boundary stops the search for non-awaited (fire-and-forget) callbacks.
        // Exception: inline FunctionExpression/ArrowFunctionExpression whose parent call is
        // itself immediately awaited inside a try block — the rejection propagates up.
        if (FUNCTION_BOUNDARY_TYPES.has(ancestor.type)) {
          // FunctionDeclarations are never inline callback arguments.
          if (ancestor.type === AST_NODE_TYPES.FunctionDeclaration) {
            return false;
          }
          // Check for the directly-awaited inline callback pattern:
          //   await someWrapper(async () => { await fetch(...) })
          // ancestors[i-1] must be the CallExpression, ancestors[i-2] the AwaitExpression.
          if (
            i >= 2 &&
            ancestors[i - 1].type === AST_NODE_TYPES.CallExpression &&
            ancestors[i - 2].type === AST_NODE_TYPES.AwaitExpression
          ) {
            const outerAwait = ancestors[i - 2];
            // Now search ancestors outward from i-3 to see if the outer AwaitExpression
            // is inside a try block (stopping at the next function boundary).
            for (let j = i - 3; j >= 0; j--) {
              const outer = ancestors[j];
              if (FUNCTION_BOUNDARY_TYPES.has(outer.type)) {
                break;
              }
              if (outer.type === AST_NODE_TYPES.TryStatement && outer.handler != null) {
                const block = outer.block;
                if (
                  outerAwait.range != null &&
                  block.range != null &&
                  outerAwait.range[0] >= block.range[0] &&
                  outerAwait.range[1] <= block.range[1]
                ) {
                  return true;
                }
              }
            }
          }
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
        const fetchInfo = getAwaitedFetchInfo(node);
        if (!fetchInfo) return;
        // Skip when fetch is shadowed by a local binding (e.g. a parameter or import named fetch).
        if (hasLocalBinding(node, "fetch")) return;
        if (fetchInfo.hasRejectionHandler) return;
        if (isInsideTryBlock(node)) return;

        const { fetchCall } = fetchInfo;
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
