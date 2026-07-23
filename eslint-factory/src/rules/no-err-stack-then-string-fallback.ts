import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

function isIdentifierNamed(node: TSESTree.Node, name: string): node is TSESTree.Identifier {
  return node.type === AST_NODE_TYPES.Identifier && node.name === name;
}

/**
 * Checks whether `node` matches `<errVar> && <errVar>.stack` (LogicalExpression).
 */
function isErrAndErrStack(node: TSESTree.Node, errVar: string): boolean {
  if (node.type !== AST_NODE_TYPES.LogicalExpression || node.operator !== "&&") return false;
  if (!isIdentifierNamed(node.left, errVar)) return false;
  const right = node.right;
  if (right.type !== AST_NODE_TYPES.MemberExpression || right.computed) return false;
  return isIdentifierNamed(right.object, errVar) && isIdentifierNamed(right.property, "stack");
}

/**
 * Checks whether `node` matches `<errVar>.stack` (MemberExpression).
 */
function isErrStack(node: TSESTree.Node, errVar: string): boolean {
  if (node.type !== AST_NODE_TYPES.MemberExpression || node.computed) return false;
  return isIdentifierNamed(node.object, errVar) && isIdentifierNamed(node.property, "stack");
}

/**
 * Checks whether `node` matches `String(<errVar>)`.
 */
function isStringErr(node: TSESTree.Node, errVar: string): boolean {
  if (node.type !== AST_NODE_TYPES.CallExpression) return false;
  if (!isIdentifierNamed(node.callee, "String")) return false;
  if (node.arguments.length !== 1) return false;
  return isIdentifierNamed(node.arguments[0], errVar);
}

export const noErrStackThenStringFallbackRule = createRule({
  name: "no-err-stack-then-string-fallback",
  meta: {
    type: "suggestion",
    hasSuggestions: true,
    docs: {
      description:
        "Prefer getErrorMessage(err) over `err && err.stack ? err.stack : String(err)`. " +
        "The stack-trace form surfaces noisy implementation details; getErrorMessage() returns " +
        "the concise error message and is available in every actions/setup/js script via error_helpers.cjs.",
    },
    schema: [],
    messages: {
      preferGetErrorMessage: "Prefer getErrorMessage({{errorVar}}) from error_helpers.cjs. The `{{errorVar}}.stack` ternary surfaces noisy stack frames; getErrorMessage() returns a clean, consistent message.",
      replaceWithGetErrorMessage: "Replace with getErrorMessage({{errorVar}}) — ensure getErrorMessage is imported from error_helpers.cjs before applying.",
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      ConditionalExpression(node) {
        // Pattern: <errVar> && <errVar>.stack ? <errVar>.stack : String(<errVar>)
        const test = node.test;
        if (test.type !== AST_NODE_TYPES.LogicalExpression) return;

        // Resolve errVar from the test's left-hand side identifier
        if (test.left.type !== AST_NODE_TYPES.Identifier) return;
        const errVar = test.left.name;

        if (!isErrAndErrStack(test, errVar)) return;
        if (!isErrStack(node.consequent, errVar)) return;
        if (!isStringErr(node.alternate, errVar)) return;

        context.report({
          node,
          messageId: "preferGetErrorMessage",
          data: { errorVar: errVar },
          suggest: [
            {
              messageId: "replaceWithGetErrorMessage",
              data: { errorVar: errVar },
              fix(fixer) {
                return fixer.replaceText(node, `getErrorMessage(${errVar})`);
              },
            },
          ],
        });
      },
    };
  },
});
