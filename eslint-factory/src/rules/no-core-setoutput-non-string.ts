import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/**
 * Returns a description of the non-string value kind if the node is one of the
 * low-false-positive forms targeted by this rule, or null if the value may already
 * be a string or cannot be determined without type information.
 *
 * Targeted forms (low false-positive risk):
 *   - Numeric literal: 0, 42, 3.14
 *   - Boolean literal: true, false
 *   - Null literal
 *   - Identifier `undefined`
 *   - .length member access: commonly numeric in practice
 */
function nonStringKind(node: TSESTree.Node): string | null {
  if (node.type === AST_NODE_TYPES.Literal) {
    if (typeof node.value === "number") return "numeric literal";
    if (typeof node.value === "boolean") return "boolean literal";
    if (node.value === null) return "null";
  }

  if (node.type === AST_NODE_TYPES.Identifier && node.name === "undefined") {
    return "undefined";
  }

  // expr.length — commonly numeric; computed access (expr["length"]) is intentionally
  // excluded because it is far less common and raises the FP risk slightly.
  if (node.type === AST_NODE_TYPES.MemberExpression && !node.computed && node.property.type === AST_NODE_TYPES.Identifier && node.property.name === "length") {
    return ".length (number)";
  }

  return null;
}

export const noCoreSetOutputNonStringRule = createRule({
  name: "no-core-setoutput-non-string",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require core.setOutput value arguments to be explicit strings; passing numbers, booleans, null, undefined, or .length can silently produce unexpected string representations (e.g. 'null', 'true') in downstream GitHub Actions workflow expressions. Detects only calls in the form core.setOutput(name, value).",
    },
    schema: [],
    messages: {
      nonStringValue:
        "The setOutput value {{valueText}} is a {{kind}}. Implicit coercion may produce unexpected strings such as 'null' or 'true' in downstream workflow expressions. Wrap with String({{valueText}}) or use an explicit string literal.",
      wrapWithString: "Wrap with String({{valueText}}) to make coercion explicit. For null/undefined, use an explicit default (for example '') when empty-string semantics are intended.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node) {
        const callee = node.callee;

        // Must be a member expression: something.setOutput(...)
        if (callee.type !== AST_NODE_TYPES.MemberExpression) return;

        // Object must be `core` (the @actions/core import convention in actions/setup/js)
        if (callee.object.type !== AST_NODE_TYPES.Identifier || callee.object.name !== "core") return;

        // Property must be `setOutput` (direct or computed string-literal access)
        const prop = callee.property;
        const isSetOutputProp = (!callee.computed && prop.type === AST_NODE_TYPES.Identifier && prop.name === "setOutput") || (callee.computed && prop.type === AST_NODE_TYPES.Literal && prop.value === "setOutput");
        if (!isSetOutputProp) return;

        // core.setOutput expects exactly two arguments: (name, value)
        if (node.arguments.length !== 2) return;

        const valueArg = node.arguments[1];

        const kind = nonStringKind(valueArg);
        if (kind === null) return;

        const valueText = sourceCode.getText(valueArg);

        context.report({
          node,
          messageId: "nonStringValue",
          data: { kind, valueText },
          suggest: [
            {
              messageId: "wrapWithString",
              data: { valueText },
              fix(fixer) {
                return fixer.replaceText(valueArg, `String(${valueText})`);
              },
            },
          ],
        });
      },
    };
  },
});
