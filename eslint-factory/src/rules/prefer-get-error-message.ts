import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

function isIdentifierNamed(node: TSESTree.Node, name: string): node is TSESTree.Identifier {
  return node.type === AST_NODE_TYPES.Identifier && node.name === name;
}

export const preferGetErrorMessageRule = createRule({
  name: "prefer-get-error-message",
  meta: {
    type: "suggestion",
    hasSuggestions: true,
    docs: {
      description: "Prefer getErrorMessage(err) over err instanceof Error ? err.message : String(err).",
    },
    schema: [],
    messages: {
      preferGetErrorMessage: "Prefer getErrorMessage({{errorVar}}) from error_helpers.cjs. It safely handles non-Error throws and sanitizes HTML error-page responses.",
      replaceWithGetErrorMessage: "Replace with getErrorMessage({{errorVar}}) — ensure getErrorMessage is imported from error_helpers.cjs before applying.",
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      ConditionalExpression(node) {
        const test = node.test;
        if (test.type !== AST_NODE_TYPES.BinaryExpression || test.operator !== "instanceof") return;
        if (test.left.type !== AST_NODE_TYPES.Identifier || !isIdentifierNamed(test.right, "Error")) return;

        const errorVar = test.left.name;
        const consequent = node.consequent;
        if (consequent.type !== AST_NODE_TYPES.MemberExpression || consequent.computed || !isIdentifierNamed(consequent.object, errorVar) || !isIdentifierNamed(consequent.property, "message")) {
          return;
        }

        const alternate = node.alternate;
        if (alternate.type !== AST_NODE_TYPES.CallExpression || !isIdentifierNamed(alternate.callee, "String") || alternate.arguments.length !== 1 || !isIdentifierNamed(alternate.arguments[0], errorVar)) {
          return;
        }

        context.report({
          node,
          messageId: "preferGetErrorMessage",
          data: { errorVar },
          suggest: [
            {
              messageId: "replaceWithGetErrorMessage",
              data: { errorVar },
              fix(fixer) {
                return fixer.replaceText(node, `getErrorMessage(${errorVar})`);
              },
            },
          ],
        });
      },
    };
  },
});
