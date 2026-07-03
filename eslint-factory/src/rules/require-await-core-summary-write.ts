import { ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const ASYNC_FUNCTION_TYPES = new Set(["FunctionDeclaration", "FunctionExpression", "ArrowFunctionExpression"]);

/**
 * Checks whether a MemberExpression property is "write" (direct or computed string-literal access).
 */
function isWriteProperty(node: TSESTree.MemberExpression): boolean {
  const property = node.property;
  const isDirectAccess = !node.computed && property.type === "Identifier" && property.name === "write";
  const isComputedAccess = node.computed && property.type === "Literal" && property.value === "write";
  return isDirectAccess || isComputedAccess;
}

function getRootIdentifier(node: TSESTree.Node): string | null {
  if (node.type === "Identifier") return node.name;
  if (node.type === "MemberExpression") return getRootIdentifier(node.object);
  if (node.type === "CallExpression") return getRootIdentifier(node.callee);
  return null;
}

function isCoreLikeIdentifier(name: string): boolean {
  return /^core/i.test(name);
}

/**
 * Checks whether a node is rooted in a `.summary` member access, possibly through
 * a chain of method calls (e.g., `core.summary.addRaw(x).write()`).
 *
 * Accepted patterns (non-exhaustive):
 *   - `core.summary`
 *   - `core.summary.addRaw(x)`
 *   - `core.summary.addHeading(...).addRaw(x)`
 *   - `coreObj.summary` (any identifier alias)
 */
function rootsSummary(node: TSESTree.Node): boolean {
  const rootIdentifier = getRootIdentifier(node);
  if (!rootIdentifier || !isCoreLikeIdentifier(rootIdentifier)) return false;
  if (node.type === "MemberExpression") {
    const property = node.property;
    const isSummaryProp = (!node.computed && property.type === "Identifier" && property.name === "summary") || (node.computed && property.type === "Literal" && property.value === "summary");
    if (isSummaryProp) return true;
  }
  if (node.type === "CallExpression" && node.callee.type === "MemberExpression") {
    return rootsSummary(node.callee.object);
  }
  return false;
}

/**
 * Returns true when the statement is directly inside an async function body.
 * Walking up the ancestors, the first function boundary found determines
 * whether `await` is currently valid — the suggestion is only safe to apply
 * in an async context.
 */
function isInsideAsyncFunction(ancestors: TSESTree.Node[]): boolean {
  for (let i = ancestors.length - 1; i >= 0; i--) {
    const ancestor = ancestors[i];
    if (ASYNC_FUNCTION_TYPES.has(ancestor.type)) {
      return (ancestor as TSESTree.FunctionDeclaration | TSESTree.FunctionExpression | TSESTree.ArrowFunctionExpression).async;
    }
  }
  return false;
}

export const requireAwaitCoreSummaryWriteRule = createRule({
  name: "require-await-core-summary-write",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require core.summary.write() calls to be awaited; the returned Promise<Summary> is silently discarded when called without await, which can truncate or drop the step summary if the process exits before the microtask queue drains.",
    },
    schema: [],
    messages: {
      requireAwait: "core.summary.write() returns a Promise<Summary> that must be awaited; omitting await silently discards the promise and can cause the step summary to be truncated or missing.",
      addAwait: "Insert 'await' before the expression.",
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      ExpressionStatement(node) {
        const expr = node.expression;

        // Only flag bare expression statements — AwaitExpression, ReturnStatement,
        // VariableDeclaration, and AssignmentExpression propagate the Promise to the
        // caller and are not flagged (zero false positives on existing correct uses).
        if (expr.type !== "CallExpression") return;

        const callee = expr.callee;
        if (callee.type !== "MemberExpression") return;

        // Property must be `write` (direct or computed string-literal access)
        if (!isWriteProperty(callee)) return;

        // Object must trace back through a `.summary` member access
        if (!rootsSummary(callee.object)) return;

        // Only offer the `await` suggestion when already inside an async function —
        // applying `await` outside an async context would produce a syntax error.
        const ancestors = context.sourceCode.getAncestors(node);
        const suggest = isInsideAsyncFunction(ancestors)
          ? [
              {
                messageId: "addAwait" as const,
                fix(fixer: TSESLint.RuleFixer) {
                  return fixer.insertTextBefore(expr, "await ");
                },
              },
            ]
          : [];

        context.report({
          node: expr,
          messageId: "requireAwait",
          suggest,
        });
      },
    };
  },
});
