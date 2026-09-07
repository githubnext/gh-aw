import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Property names that, in actions/setup/js, denote a JSON-Schema-style "default"
// value read from a tool/input schema (e.g. inputSchema.default, schema.default).
// A `||` fallback on these silently discards legitimate falsy defaults such as
// `0`, `false`, and `""`, treating them as if no default were configured.
// This is the exact regression fixed in PR #58291 (numeric `0` input defaults
// were treated as missing because normalization used `||` instead of `??`).
const DEFAULT_PROPERTY_NAME = "default";

function isDefaultMemberAccess(node: TSESTree.Expression): boolean {
  if (node.type !== AST_NODE_TYPES.MemberExpression) return false;
  if (node.computed) return false;
  return node.property.type === AST_NODE_TYPES.Identifier && node.property.name === DEFAULT_PROPERTY_NAME;
}

export const noSchemaDefaultOrFallbackRule = createRule({
  name: "no-schema-default-or-fallback",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow `<schema>.default || fallback` in actions/setup/js scripts. Using `||` to fall back from a schema/input `default` property " +
        "discards legitimate falsy defaults (`0`, `false`, `\"\"`), treating a correctly configured falsy default as if none were set. " +
        "Use `??` (nullish coalescing) instead so only `null`/`undefined` trigger the fallback.",
    },
    schema: [],
    messages: {
      useNullishCoalescing: "Replace `{{objectText}}.default || {{fallbackText}}` with `{{objectText}}.default ?? {{fallbackText}}` — `||` discards falsy-but-valid defaults like 0, false, and \"\".",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      LogicalExpression(node) {
        if (node.operator !== "||") return;
        if (!isDefaultMemberAccess(node.left)) return;

        const objectText = sourceCode.getText((node.left as TSESTree.MemberExpression).object);
        const fallbackText = sourceCode.getText(node.right);

        context.report({
          node,
          messageId: "useNullishCoalescing",
          data: { objectText, fallbackText },
        });
      },
    };
  },
});
