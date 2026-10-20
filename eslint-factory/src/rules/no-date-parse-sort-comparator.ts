import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const SORT_METHODS = new Set(["sort", "toSorted"]);

function isDateParseCall(node: TSESTree.Node): boolean {
  return (
    node.type === AST_NODE_TYPES.CallExpression &&
    node.callee.type === AST_NODE_TYPES.MemberExpression &&
    !node.callee.computed &&
    node.callee.object.type === AST_NODE_TYPES.Identifier &&
    node.callee.object.name === "Date" &&
    node.callee.property.type === AST_NODE_TYPES.Identifier &&
    node.callee.property.name === "parse"
  );
}

/** Collects returned expressions of a comparator (expression body or `return` statements in its own body). */
function collectReturned(fn: TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression): TSESTree.Node[] {
  if (fn.body.type !== AST_NODE_TYPES.BlockStatement) return [fn.body];
  const returned: TSESTree.Node[] = [];
  for (const statement of fn.body.body) {
    if (statement.type === AST_NODE_TYPES.ReturnStatement && statement.argument) returned.push(statement.argument);
  }
  return returned;
}

export const noDateParseSortComparatorRule = createRule({
  name: "no-date-parse-sort-comparator",
  meta: {
    type: "problem",
    docs: {
      description: "Disallow sort comparators that subtract raw Date.parse() results, because NaN from an invalid timestamp makes the comparator inconsistent and the order undefined.",
    },
    schema: [],
    messages: {
      noDateParseSubtraction:
        "Do not return `Date.parse(a) - Date.parse(b)` from a sort comparator: an invalid timestamp yields NaN, which makes the comparator inconsistent and the resulting order implementation-defined. Normalize each value first (e.g. `Number.isFinite(ms) ? ms : 0`, or `|| 0`) or filter invalid entries before sorting.",
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      CallExpression(node) {
        const callee = node.callee;
        if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed || callee.property.type !== AST_NODE_TYPES.Identifier || !SORT_METHODS.has(callee.property.name)) {
          return;
        }
        const comparator = node.arguments[0];
        if (!comparator || (comparator.type !== AST_NODE_TYPES.ArrowFunctionExpression && comparator.type !== AST_NODE_TYPES.FunctionExpression)) {
          return;
        }
        for (const expr of collectReturned(comparator)) {
          if (expr.type === AST_NODE_TYPES.BinaryExpression && expr.operator === "-" && (isDateParseCall(expr.left) || isDateParseCall(expr.right))) {
            context.report({ node: expr, messageId: "noDateParseSubtraction" });
          }
        }
      },
    };
  },
});
