import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const EQUALITY_OPERATORS = new Set(["===", "!==", "==", "!="]);

/**
 * True when `node` is a `JSON.stringify(...)` call expression.
 */
function isJsonStringifyCall(node: TSESTree.Expression | TSESTree.PrivateIdentifier): node is TSESTree.CallExpression {
  if (node.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = node.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression) return false;
  if (callee.computed) return false;
  const obj = callee.object;
  const prop = callee.property;
  if (obj.type !== AST_NODE_TYPES.Identifier || obj.name !== "JSON") return false;
  if (prop.type !== AST_NODE_TYPES.Identifier || prop.name !== "stringify") return false;
  return true;
}

export const noJsonStringifyEqualityRule = createRule({
  name: "no-json-stringify-equality",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow comparing two JSON.stringify() calls for equality — key order is not guaranteed to be stable across differently-constructed objects, so JSON.stringify(a) === JSON.stringify(b) can report unequal for objects that are deeply equal.",
    },
    schema: [],
    messages: {
      jsonStringifyEquality:
        "Comparing JSON.stringify(...) results with '{{operator}}' is unreliable: two deeply-equal objects with different key insertion order produce different strings, causing false negatives. Use a structural deep-equality check (e.g. a recursive deepEqual helper) instead.",
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      BinaryExpression(node) {
        if (!EQUALITY_OPERATORS.has(node.operator)) return;
        if (!isJsonStringifyCall(node.left) || !isJsonStringifyCall(node.right)) return;

        context.report({
          node,
          messageId: "jsonStringifyEquality",
          data: { operator: node.operator },
        });
      },
    };
  },
});
