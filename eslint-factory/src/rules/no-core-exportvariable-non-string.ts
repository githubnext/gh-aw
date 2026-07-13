import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

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
const NUMERIC_LITERAL_KIND = "numeric literal" as const;
const BOOLEAN_LITERAL_KIND = "boolean literal" as const;
const NULL_KIND = "null" as const;
const UNDEFINED_KIND = "undefined" as const;
const LENGTH_KIND = ".length (number)" as const;

type NonStringKind = typeof NUMERIC_LITERAL_KIND | typeof BOOLEAN_LITERAL_KIND | typeof NULL_KIND | typeof UNDEFINED_KIND | typeof LENGTH_KIND;

function nonStringKind(node: TSESTree.Node): NonStringKind | null {
  if (node.type === AST_NODE_TYPES.Literal) {
    if (typeof node.value === "number") return NUMERIC_LITERAL_KIND;
    if (typeof node.value === "boolean") return BOOLEAN_LITERAL_KIND;
    if (node.value === null) return NULL_KIND;
  }

  if (node.type === AST_NODE_TYPES.Identifier && node.name === "undefined") {
    return UNDEFINED_KIND;
  }

  // expr.length — commonly numeric; computed access (expr["length"]) is intentionally
  // excluded because it is far less common and raises the FP risk slightly.
  if (node.type === AST_NODE_TYPES.MemberExpression && !node.computed && node.property.type === AST_NODE_TYPES.Identifier && node.property.name === "length") {
    return LENGTH_KIND;
  }

  return null;
}

export const noCoreExportVariableNonStringRule = createRule({
  name: "no-core-exportvariable-non-string",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require core.exportVariable value arguments to be explicit strings; passing numbers, booleans, null, undefined, or .length can silently produce unexpected string representations (e.g. 'null', 'true') in downstream GitHub Actions steps that read the exported environment variable. Detects only calls in the form core.exportVariable(name, value).",
    },
    schema: [],
    messages: {
      nonStringValue:
        "The exportVariable value {{valueText}} is a {{kind}}. Implicit coercion may produce unexpected strings such as 'null' or 'true' when the environment variable is read by downstream steps. Use an explicit string conversion and choose the suggestion that matches the intended output semantics.",
      wrapWithString: "Wrap with String({{valueText}}) to make coercion explicit. For null/undefined, use an explicit default (for example '') when empty-string semantics are intended.",
      useEmptyString: "Replace with \"\" (empty string) — use this when the intended output is empty rather than the literal word 'null' or 'undefined'.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node) {
        const callee = node.callee;

        // Must be a member expression: something.exportVariable(...)
        if (callee.type !== AST_NODE_TYPES.MemberExpression) return;

        // Object must be `core` (the @actions/core import convention in actions/setup/js)
        if (callee.object.type !== AST_NODE_TYPES.Identifier || callee.object.name !== "core") return;

        // Property must be `exportVariable` (direct or computed string-literal access)
        const prop = callee.property;
        const isExportVariableProp =
          (!callee.computed && prop.type === AST_NODE_TYPES.Identifier && prop.name === "exportVariable") ||
          (callee.computed && prop.type === AST_NODE_TYPES.Literal && prop.value === "exportVariable");
        if (!isExportVariableProp) return;

        // core.exportVariable expects exactly two arguments: (name, value)
        if (node.arguments.length !== 2) return;

        const valueArg = node.arguments[1];

        const kind = nonStringKind(valueArg);
        if (kind === null) return;

        const valueText = sourceCode.getText(valueArg);

        const isNullOrUndefined = kind === NULL_KIND || kind === UNDEFINED_KIND;

        context.report({
          node,
          messageId: "nonStringValue",
          data: { kind, valueText },
          suggest: [
            ...(isNullOrUndefined
              ? [
                  {
                    messageId: "useEmptyString" as const,
                    fix(fixer: TSESLint.RuleFixer) {
                      return fixer.replaceText(valueArg, `""`);
                    },
                  },
                ]
              : []),
            {
              messageId: "wrapWithString" as const,
              data: { valueText },
              fix(fixer: TSESLint.RuleFixer) {
                return fixer.replaceText(valueArg, `String(${valueText})`);
              },
            },
          ],
        });
      },
    };
  },
});
