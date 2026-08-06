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

/**
 * Returns the property name of a non-computed member expression, or a string literal
 * computed access, and undefined otherwise.
 */
function getStaticPropertyName(node: TSESTree.MemberExpression): string | undefined {
  if (!node.computed && node.property.type === AST_NODE_TYPES.Identifier) return node.property.name;
  if (node.computed && node.property.type === AST_NODE_TYPES.Literal && typeof node.property.value === "string") return node.property.value;
  return undefined;
}

/**
 * Returns true when the expression is a function literal (`function () {}` or `() => {}`).
 */
function isFunctionLiteral(node: TSESTree.Node): boolean {
  return node.type === AST_NODE_TYPES.FunctionExpression || node.type === AST_NODE_TYPES.ArrowFunctionExpression;
}

export const preferStructuredCloneRule = createRule({
  name: "prefer-structured-clone",
  meta: {
    type: "suggestion",
    hasSuggestions: true,
    docs: {
      description:
        'Prefer structuredClone(...) over JSON.parse(JSON.stringify(...)) for deep-cloning plain data in actions/setup/js scripts. The JSON round-trip is slower, silently drops values it cannot represent (undefined, functions, Date becomes a string), and throws on circular references, whereas structuredClone (Node >=17, available globally in the Node 24 runtime this action targets) clones plain objects and JSON-safe data directly. The autofix suggestion assumes the cloned value never carries function-valued properties: JSON.stringify silently drops functions while structuredClone throws DataCloneError, so code that intentionally relies on the drop-and-reattach idiom must not be rewritten. The suggestion is therefore withheld when the cloned expression is an identifier whose properties are assigned function literals (or checked with `typeof x.prop === "function"`) anywhere in the same file; the diagnostic is still reported so the round trip can be reviewed by hand.',
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

    // Identifiers observed carrying function-valued properties somewhere in the file.
    // Cloning those with structuredClone would throw DataCloneError, so the suggestion
    // is withheld for them (the diagnostic is still reported).
    const identifiersWithFunctionProperties = new Set<string>();
    const candidates: { node: TSESTree.CallExpression; clonedExpression: TSESTree.Node }[] = [];

    return {
      AssignmentExpression(node) {
        if (node.left.type !== AST_NODE_TYPES.MemberExpression) return;
        if (node.left.object.type !== AST_NODE_TYPES.Identifier) return;
        if (!isFunctionLiteral(node.right)) return;
        identifiersWithFunctionProperties.add(node.left.object.name);
      },

      VariableDeclarator(node) {
        // `const x = { handler: () => {} }` also carries a function-valued property.
        if (node.id.type !== AST_NODE_TYPES.Identifier) return;
        if (!node.init || node.init.type !== AST_NODE_TYPES.ObjectExpression) return;
        const hasFunctionProperty = node.init.properties.some(property => property.type === AST_NODE_TYPES.Property && (isFunctionLiteral(property.value) || property.method));
        if (hasFunctionProperty) identifiersWithFunctionProperties.add(node.id.name);
      },

      BinaryExpression(node) {
        // `typeof x.prop === "function"` is strong evidence that `x` carries a function property.
        if (node.operator !== "===" && node.operator !== "==" && node.operator !== "!==" && node.operator !== "!=") return;
        const [typeofSide, literalSide] = node.left.type === AST_NODE_TYPES.UnaryExpression ? [node.left, node.right] : [node.right, node.left];
        if (typeofSide.type !== AST_NODE_TYPES.UnaryExpression || typeofSide.operator !== "typeof") return;
        if (literalSide.type !== AST_NODE_TYPES.Literal || literalSide.value !== "function") return;
        const argument = typeofSide.argument;
        if (argument.type !== AST_NODE_TYPES.MemberExpression) return;
        if (argument.object.type !== AST_NODE_TYPES.Identifier) return;
        if (getStaticPropertyName(argument) === undefined) return;
        identifiersWithFunctionProperties.add(argument.object.name);
      },

      CallExpression(node) {
        if (!isJsonParseCall(node)) return;
        if (node.arguments.length !== 1) return;

        const innerArg = node.arguments[0];
        if (innerArg.type !== AST_NODE_TYPES.CallExpression) return;
        if (!isPlainJsonStringifyCall(innerArg)) return;

        candidates.push({ node, clonedExpression: innerArg.arguments[0] });
      },

      "Program:exit"() {
        for (const { node, clonedExpression } of candidates) {
          const clonedExpressionText = sourceCode.getText(clonedExpression);
          const carriesFunctionProperties = clonedExpression.type === AST_NODE_TYPES.Identifier && identifiersWithFunctionProperties.has(clonedExpression.name);

          context.report({
            node,
            messageId: "preferStructuredClone",
            data: { arg: clonedExpressionText },
            suggest: carriesFunctionProperties
              ? []
              : [
                  {
                    messageId: "replaceWithStructuredClone",
                    fix(fixer: TSESLint.RuleFixer) {
                      return fixer.replaceText(node, `structuredClone(${clonedExpressionText})`);
                    },
                  },
                ],
          });
        }
      },
    };
  },
});
