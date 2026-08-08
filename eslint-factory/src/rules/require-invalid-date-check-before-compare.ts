import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const RELATIONAL_OPERATORS = new Set(["<", ">", "<=", ">="]);

/** Returns true when the node is exactly the call expression `Date.now()`. */
function isExactDateNowCall(node: TSESTree.Node): boolean {
  if (node.type !== AST_NODE_TYPES.CallExpression || node.arguments.length !== 0) return false;
  if (node.callee.type !== AST_NODE_TYPES.MemberExpression) return false;
  const { object, property, computed } = node.callee;
  return !computed && object.type === AST_NODE_TYPES.Identifier && object.name === "Date" && property.type === AST_NODE_TYPES.Identifier && property.name === "now";
}

/**
 * Returns true when the call expression is `new Date(arg)` with a non-trivial argument
 * (i.e., not a bare `new Date()` or `new Date(Date.now())`, both of which are always valid).
 * Any arithmetic on `Date.now()` (e.g. `Date.now() - windowMs`) is treated as potentially
 * invalid since the other operand is not guaranteed to be a finite number.
 */
function isPotentiallyInvalidDateConstruction(node: TSESTree.NewExpression): boolean {
  if (node.callee.type !== AST_NODE_TYPES.Identifier || node.callee.name !== "Date") return false;
  if (node.arguments.length === 0) return false;

  const arg = node.arguments[0];
  if (isExactDateNowCall(arg)) return false;

  return true;
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

/**
 * Resolves an identifier to its scope-bound `Variable`, walking up the scope chain.
 * Using the resolved `Variable` (rather than the bare name string) as a map key ensures
 * same-named locals in different functions are never conflated.
 */
function resolveVariable(sourceCode: TSESLint.SourceCode, identifier: TSESTree.Identifier): TSESLint.Scope.Variable | null {
  let scope: TSESLint.Scope.Scope | null = sourceCode.getScope(identifier);
  while (scope !== null) {
    const variable = scope.set.get(identifier.name);
    if (variable !== undefined) return variable;
    scope = scope.upper;
  }
  return null;
}

type ComparisonSide = { kind: "inline" } | { kind: "var"; variable: TSESLint.Scope.Variable };

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
      requireInvalidDateCheck: "{{subject}} may be an Invalid Date and is compared with a relational operator ({{operator}}) without ever being checked via Number.isNaN({{getTimeTarget}}.getTime()). An unparseable date silently fails every comparison instead of surfacing an error.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    // Variable -> declarator node, for variables assigned `new Date(nonTrivialArg)`.
    const dateVars = new Map<TSESLint.Scope.Variable, TSESTree.Node>();
    // Variables confirmed validated via Number.isNaN(name.getTime()) / isNaN(name.getTime()).
    const validated = new Set<TSESLint.Scope.Variable>();
    // Relational comparisons to report once all traversal is done, keyed by the involved sides.
    const comparisons: { node: TSESTree.BinaryExpression; operator: string; sides: ComparisonSide[] }[] = [];

    return {
      VariableDeclarator(node) {
        if (node.id.type === AST_NODE_TYPES.Identifier && node.init?.type === AST_NODE_TYPES.NewExpression && isPotentiallyInvalidDateConstruction(node.init)) {
          const variable = resolveVariable(sourceCode, node.id);
          if (variable) dateVars.set(variable, node);
        }
      },

      CallExpression(node) {
        if (isGetTimeNaNCheck(node)) {
          const arg = node.arguments[0] as TSESTree.CallExpression;
          const obj = (arg.callee as TSESTree.MemberExpression).object;
          if (obj.type === AST_NODE_TYPES.Identifier) {
            const variable = resolveVariable(sourceCode, obj);
            if (variable) validated.add(variable);
          }
        }
      },

      BinaryExpression(node) {
        if (!RELATIONAL_OPERATORS.has(node.operator)) return;

        const sides: ComparisonSide[] = [];
        for (const side of [node.left, node.right]) {
          // Direct relational use of an inline `new Date(...)` expression.
          if (side.type === AST_NODE_TYPES.NewExpression && isPotentiallyInvalidDateConstruction(side)) {
            sides.push({ kind: "inline" });
            continue;
          }
          if (side.type === AST_NODE_TYPES.Identifier) {
            const variable = resolveVariable(sourceCode, side);
            if (variable && dateVars.has(variable)) {
              sides.push({ kind: "var", variable });
            }
          }
        }
        if (sides.length > 0) comparisons.push({ node, operator: node.operator, sides });
      },

      "Program:exit"() {
        for (const { node, operator, sides } of comparisons) {
          const problems = sides.filter(side => (side.kind === "var" ? !validated.has(side.variable) : true));
          if (problems.length === 0) continue;

          if (problems.length === 1) {
            const problem = problems[0];
            if (problem.kind === "inline") {
              context.report({ node, messageId: "requireInvalidDateCheck", data: { subject: "An inline `new Date(...)` expression", operator, getTimeTarget: "it" } });
            } else {
              const name = problem.variable.name;
              context.report({ node, messageId: "requireInvalidDateCheck", data: { subject: `'${name}'`, operator, getTimeTarget: name } });
            }
            continue;
          }

          // Both sides of the comparison are unvalidated: report a single combined diagnostic
          // instead of one per side, to avoid two identical errors on the same node.
          context.report({ node, messageId: "requireInvalidDateCheck", data: { subject: "Both operands of this comparison", operator, getTimeTarget: "each value" } });
        }
      },
    };
  },
});
