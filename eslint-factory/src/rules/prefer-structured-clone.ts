import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/**
 * Returns true when the call expression is `JSON.parse(...)` (direct or computed access).
 */
function isJsonParseCall(node: TSESTree.CallExpression): boolean {
  const callee = node.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression) return false;
  if (callee.object.type !== AST_NODE_TYPES.Identifier || callee.object.name !== "JSON") return false;
  const property = callee.property;
  const isDirectAccess = !callee.computed && property.type === AST_NODE_TYPES.Identifier && property.name === "parse";
  const isComputedAccess = callee.computed && property.type === AST_NODE_TYPES.Literal && property.value === "parse";
  return isDirectAccess || isComputedAccess;
}

/**
 * Returns true when the call expression is `JSON.stringify(...)` (direct or computed access)
 * with exactly one argument (a replacer/indent argument changes the round-trip semantics and
 * is intentionally excluded from this check to keep false positives low).
 */
function isPlainJsonStringifyCall(node: TSESTree.CallExpression): boolean {
  const callee = node.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression) return false;
  if (callee.object.type !== AST_NODE_TYPES.Identifier || callee.object.name !== "JSON") return false;
  const property = callee.property;
  const isDirectAccess = !callee.computed && property.type === AST_NODE_TYPES.Identifier && property.name === "stringify";
  const isComputedAccess = callee.computed && property.type === AST_NODE_TYPES.Literal && property.value === "stringify";
  if (!isDirectAccess && !isComputedAccess) return false;
  return node.arguments.length === 1;
}

export const preferStructuredCloneRule = createRule({
  name: "prefer-structured-clone",
  meta: {
    type: "suggestion",
    hasSuggestions: true,
    docs: {
      description:
        "Prefer structuredClone(...) over JSON.parse(JSON.stringify(...)) for deep-cloning plain data in actions/setup/js scripts. The JSON round-trip is slower, silently drops values it cannot represent (undefined, functions, Date becomes a string), and throws on circular references, whereas structuredClone (Node >=17, available globally in the Node 24 runtime this action targets) clones plain objects and JSON-safe data directly.",
    },
    schema: [],
    messages: {
      preferStructuredClone: "Replace JSON.parse(JSON.stringify({{arg}})) with structuredClone({{arg}}) — the JSON round-trip silently drops undefined/function values, converts Dates to strings, and throws on circular references.",
      replaceWithStructuredClone: "Replace with structuredClone(...).",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node) {
        if (!isJsonParseCall(node)) return;
        if (node.arguments.length !== 1) return;

        const innerArg = node.arguments[0];
        if (innerArg.type !== AST_NODE_TYPES.CallExpression) return;
        if (!isPlainJsonStringifyCall(innerArg)) return;

        const clonedExpressionText = sourceCode.getText(innerArg.arguments[0]);

        context.report({
          node,
          messageId: "preferStructuredClone",
          data: { arg: clonedExpressionText },
          suggest: [
            {
              messageId: "replaceWithStructuredClone",
              fix(fixer: TSESLint.RuleFixer) {
                return fixer.replaceText(node, `structuredClone(${clonedExpressionText})`);
              },
            },
          ],
        });
      },
    };
  },
});
