import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Identifier/member names that indicate a value has already been passed through
// a regex-escaping helper before interpolation, e.g. `escapeRegExp(x)`,
// `escapeRegex(x)`, or a variable named `escapedFoo` / `ESCAPED_FOO`.
const ESCAPE_NAME_PATTERN = /escape/i;

/**
 * Returns true when `node` is a call expression whose callee name looks like
 * a regex-escaping helper (e.g. `escapeRegExp(value)`, `utils.escapeRegex(value)`).
 */
function isEscapeHelperCall(node: TSESTree.Node): boolean {
  if (node.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = node.callee;
  if (callee.type === AST_NODE_TYPES.Identifier) return ESCAPE_NAME_PATTERN.test(callee.name);
  if (callee.type === AST_NODE_TYPES.MemberExpression && !callee.computed && callee.property.type === AST_NODE_TYPES.Identifier) {
    return ESCAPE_NAME_PATTERN.test(callee.property.name);
  }
  return false;
}

/**
 * Returns true when `node` is an identifier or member expression whose name
 * indicates the value has already been escaped, e.g. `ESCAPED_FOO` or
 * `escapedValue`.
 */
function isEscapedNameReference(node: TSESTree.Node): boolean {
  if (node.type === AST_NODE_TYPES.Identifier) return ESCAPE_NAME_PATTERN.test(node.name);
  if (node.type === AST_NODE_TYPES.MemberExpression && !node.computed && node.property.type === AST_NODE_TYPES.Identifier) {
    return ESCAPE_NAME_PATTERN.test(node.property.name);
  }
  return false;
}

/**
 * Returns true when the interpolated expression is recognized as already
 * escaped, either via a call to an escape-like helper function or via a
 * variable/property name that itself signals escaping.
 */
function isRecognizedAsEscaped(node: TSESTree.Node): boolean {
  return isEscapeHelperCall(node) || isEscapedNameReference(node);
}

export const requireEscapedRegexpInterpolationRule = createRule({
  name: "require-escaped-regexp-interpolation",
  meta: {
    type: "problem",
    docs: {
      description:
        "Require values interpolated into a `new RegExp()` template-literal pattern to be passed through a regex-escaping helper first. " +
        "Unescaped interpolation of a value containing regex metacharacters (e.g. `.`, `*`, `+`, `(`, `)`) can produce unintended matches " +
        "or, with attacker-controlled input, a ReDoS-prone pattern.",
    },
    schema: [],
    messages: {
      unescapedInterpolation:
        "Interpolated value `{{expr}}` in `new RegExp()` template literal is not passed through a regex-escaping helper. " +
        "Escape regex metacharacters before interpolating, e.g. `{{expr}}.replace(/[.*+?^${}()|[\\]\\\\]/g, \"\\\\$&\")`.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      NewExpression(node) {
        if (node.callee.type !== AST_NODE_TYPES.Identifier || node.callee.name !== "RegExp") return;

        const patternArg = node.arguments[0];
        if (!patternArg || patternArg.type !== AST_NODE_TYPES.TemplateLiteral) return;
        if (patternArg.expressions.length === 0) return;

        for (const expr of patternArg.expressions) {
          if (isRecognizedAsEscaped(expr)) continue;

          context.report({
            node: expr,
            messageId: "unescapedInterpolation",
            data: { expr: sourceCode.getText(expr) },
          });
        }
      },
    };
  },
});
