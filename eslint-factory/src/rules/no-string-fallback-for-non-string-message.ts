import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/**
 * Renders a MemberExpression chain of identifiers/optional-member-accesses
 * (e.g. `err.response.data.message`, `err?.message`) to a normalized string
 * key for structural comparison, or null if any segment isn't a simple
 * (optionally-chained) `.property` access rooted at an Identifier.
 */
function memberChainKey(node: TSESTree.Node): string | null {
  if (node.type === AST_NODE_TYPES.Identifier) {
    return node.name;
  }
  if (node.type === AST_NODE_TYPES.ChainExpression) {
    return memberChainKey(node.expression);
  }
  if (node.type === AST_NODE_TYPES.MemberExpression && !node.computed) {
    if (node.property.type !== AST_NODE_TYPES.Identifier) return null;
    const objectKey = memberChainKey(node.object);
    if (objectKey === null) return null;
    return `${objectKey}.${node.property.name}`;
  }
  return null;
}

/**
 * Returns the `.message`-chain key when `node` is `typeof <chain>.message === "string"`,
 * or null otherwise.
 */
function getTypeofMessageCheckChainKey(node: TSESTree.Node): string | null {
  // Support `<guard> && typeof <chain>.message === "string"` by checking the
  // rightmost operand of a right-associated `&&` chain.
  if (node.type === AST_NODE_TYPES.LogicalExpression && node.operator === "&&") {
    return getTypeofMessageCheckChainKey(node.right);
  }
  if (node.type !== AST_NODE_TYPES.BinaryExpression || node.operator !== "===") return null;
  const { left, right } = node;
  const literalSide = right.type === AST_NODE_TYPES.Literal && right.value === "string" ? right : left.type === AST_NODE_TYPES.Literal && left.value === "string" ? left : null;
  if (!literalSide) return null;
  const typeofSide = literalSide === right ? left : right;
  if (typeofSide.type !== AST_NODE_TYPES.UnaryExpression || typeofSide.operator !== "typeof") return null;
  const chainKey = memberChainKey(typeofSide.argument);
  if (!chainKey || !chainKey.endsWith(".message")) return null;
  return chainKey;
}

/** Returns whether an `&&` chain contains `typeof <chain> === "object"`. */
function hasTypeofObjectCheck(node: TSESTree.Node, chainKey: string): boolean {
  if (node.type === AST_NODE_TYPES.LogicalExpression && node.operator === "&&") {
    return hasTypeofObjectCheck(node.left, chainKey) || hasTypeofObjectCheck(node.right, chainKey);
  }
  if (node.type !== AST_NODE_TYPES.BinaryExpression || node.operator !== "===") return false;
  const { left, right } = node;
  const literalSide = right.type === AST_NODE_TYPES.Literal && right.value === "object" ? right : left.type === AST_NODE_TYPES.Literal && left.value === "object" ? left : null;
  if (!literalSide) return false;
  const typeofSide = literalSide === right ? left : right;
  return typeofSide.type === AST_NODE_TYPES.UnaryExpression && typeofSide.operator === "typeof" && memberChainKey(typeofSide.argument) === chainKey;
}

/** Returns the argument's member-chain key when `node` is `String(<chain>)`, or null. */
function getStringCallArgChainKey(node: TSESTree.Node): string | null {
  if (node.type !== AST_NODE_TYPES.CallExpression) return null;
  if (node.callee.type !== AST_NODE_TYPES.Identifier || node.callee.name !== "String") return null;
  if (node.arguments.length !== 1) return null;
  const arg = node.arguments[0];
  if (arg.type === AST_NODE_TYPES.SpreadElement) return null;
  return memberChainKey(arg);
}

export const noStringFallbackForNonStringMessageRule = createRule({
  name: "no-string-fallback-for-non-string-message",
  meta: {
    type: "problem",
    docs: {
      description:
        'Disallow `typeof <x>.message === "string" ? <x>.message : String(<container>)` where the String() fallback stringifies a different (container) ' +
        "expression than the `.message` chain being tested. When `.message` exists but isn't a string, this pattern silently produces `[object Object]` " +
        "instead of coercing the message value itself (e.g. `String(<x>.message)`). Mirrors the real bug found in error_helpers.cjs's getErrorMessage().",
    },
    schema: [],
    messages: {
      stringifiesContainerInsteadOfMessage:
        'This falls back to String({{containerExpr}}) instead of coercing {{messageChain}} itself when it exists but isn\'t a string — risks producing "[object Object]". Use String({{messageChain}}) instead.',
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      ConditionalExpression(node: TSESTree.ConditionalExpression) {
        const messageChainKey = getTypeofMessageCheckChainKey(node.test);
        if (!messageChainKey) return;

        const consequentKey = memberChainKey(node.consequent);
        if (consequentKey !== messageChainKey) return;

        const alternateChainKey = getStringCallArgChainKey(node.alternate);
        if (!alternateChainKey) return;
        // Only report when String() is applied to a *different* expression
        // than the `.message` chain (i.e. stringifying the container object).
        if (alternateChainKey === messageChainKey) return;

        const messageContainerKey = messageChainKey.slice(0, -".message".length);
        if (alternateChainKey === messageContainerKey && hasTypeofObjectCheck(node.test, messageContainerKey)) return;

        const sourceCode = context.sourceCode;
        context.report({
          node: node.alternate,
          messageId: "stringifiesContainerInsteadOfMessage",
          data: {
            containerExpr: sourceCode.getText(node.alternate.type === AST_NODE_TYPES.CallExpression ? node.alternate.arguments[0] : node.alternate),
            messageChain: messageChainKey,
          },
        });
      },
    };
  },
});
