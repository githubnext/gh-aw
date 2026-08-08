import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const RELATIONAL_OPERATORS = new Set(["<", ">", "<=", ">="]);

/**
 * Returns true when the call expression is `new Date(arg)` with a non-trivial argument
 * (i.e., not a bare `new Date()` or `new Date(Date.now())`, both of which are always valid).
 */
function isPotentiallyInvalidDateConstruction(node: TSESTree.NewExpression): boolean {
  if (node.callee.type !== AST_NODE_TYPES.Identifier || node.callee.name !== "Date") return false;
  if (node.arguments.length === 0) return false;

  const arg = node.arguments[0];
  // `new Date(Date.now())` and `new Date(Date.now() + n)` are always valid — Date.now() cannot
  // produce NaN, and arithmetic on a finite number stays finite.
  if (isDateNowDerived(arg)) return false;

  return true;
}

function isDateNowDerived(node: TSESTree.Node): boolean {
  if (node.type === AST_NODE_TYPES.CallExpression && node.callee.type === AST_NODE_TYPES.MemberExpression) {
    const { object, property } = node.callee;
    if (object.type === AST_NODE_TYPES.Identifier && object.name === "Date" && property.type === AST_NODE_TYPES.Identifier && property.name === "now") {
      return true;
    }
  }
  if (node.type === AST_NODE_TYPES.BinaryExpression) {
    return isDateNowDerived(node.left) || isDateNowDerived(node.right);
  }
  return false;
}

/** Returns true when the call expression is `Number.isNaN(x.getTime())` or `isNaN(x.getTime())`. */
function isGetTimeNaNCheck(node: TSESTree.CallExpression): boolean {
  const isNaNGlobal = node.callee.type === AST_NODE_TYPES.Identifier && node.callee.name === "isNaN";
  const isNaNStatic = node.callee.type === AST_NODE_TYPES.MemberExpression && node.callee.object.type === AST_NODE_TYPES.Identifier && node.callee.object.name === "Number" && !node.callee.computed && node.callee.property.type === AST_NODE_TYPES.Identifier && node.callee.property.name === "isNaN";

  if (!isNaNGlobal && !isNaNStatic) return false;
  if (node.arguments.length !== 1) return false;

  const arg = node.arguments[0];
  return arg.type === AST_NODE_TYPES.CallExpression && arg.callee.type === AST_NODE_TYPES.MemberExpression && arg.callee.property.type === AST_NODE_TYPES.Identifier && arg.callee.property.name === "getTime";
}

export const requireInvalidDateCheckBeforeCompareRule = createRule({
  name: "require-invalid-date-check-before-compare",
  meta: {
    type: "problem",
    docs: {
      description:
        "Require validating `new Date(x)` with Number.isNaN(d.getTime()) (or isNaN(d.getTime())) before using it in a relational comparison (<, >, <=, >=). " +
        "An Invalid Date compares as neither greater than nor less than any other date (all relational comparisons involving NaN return false), " +
        "which silently defeats time-window/threshold checks such as rate-limit windows or freshness cutoffs instead of raising a visible parse error.",
    },
    schema: [],
    messages: {
      requireInvalidDateCheck: "{{subject}} is constructed with new Date(...) from a non-literal value and compared with a relational operator ({{operator}}) without ever being checked via Number.isNaN({{getTimeTarget}}.getTime()). An unparseable date silently fails every comparison instead of surfacing an error.",
    },
  },
  defaultOptions: [],
  create(context) {
    // Variable name -> declarator node, for names assigned `new Date(nonTrivialArg)`.
    const dateVars = new Map<string, TSESTree.Node>();
    // Variable names confirmed validated via Number.isNaN(name.getTime()) / isNaN(name.getTime()).
    const validated = new Set<string>();
    // Relational comparisons referencing an identifier, to report once all traversal is done.
    const comparisons: { name: string; operator: string; node: TSESTree.Node }[] = [];

    return {
      VariableDeclarator(node) {
        if (node.id.type === AST_NODE_TYPES.Identifier && node.init?.type === AST_NODE_TYPES.NewExpression && isPotentiallyInvalidDateConstruction(node.init)) {
          dateVars.set(node.id.name, node);
        }
      },

      CallExpression(node) {
        if (isGetTimeNaNCheck(node)) {
          const arg = node.arguments[0] as TSESTree.CallExpression;
          const obj = (arg.callee as TSESTree.MemberExpression).object;
          if (obj.type === AST_NODE_TYPES.Identifier) {
            validated.add(obj.name);
          }
        }
      },

      BinaryExpression(node) {
        if (!RELATIONAL_OPERATORS.has(node.operator)) return;

        for (const side of [node.left, node.right]) {
          // Direct relational use of an inline `new Date(...)` expression.
          if (side.type === AST_NODE_TYPES.NewExpression && isPotentiallyInvalidDateConstruction(side)) {
            comparisons.push({ name: "<inline>", operator: node.operator, node });
            continue;
          }
          if (side.type === AST_NODE_TYPES.Identifier && dateVars.has(side.name)) {
            comparisons.push({ name: side.name, operator: node.operator, node });
          }
        }
      },

      "Program:exit"() {
        for (const { name, operator, node } of comparisons) {
          if (name === "<inline>") {
            context.report({ node, messageId: "requireInvalidDateCheck", data: { subject: "An inline `new Date(...)` expression", operator, getTimeTarget: "it" } });
            continue;
          }
          if (!validated.has(name)) {
            context.report({ node, messageId: "requireInvalidDateCheck", data: { subject: `'${name}'`, operator, getTimeTarget: name } });
          }
        }
      },
    };
  },
});
