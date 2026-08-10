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
  const isNaNStatic =
    node.callee.type === AST_NODE_TYPES.MemberExpression &&
    node.callee.object.type === AST_NODE_TYPES.Identifier &&
    node.callee.object.name === "Number" &&
    !node.callee.computed &&
    node.callee.property.type === AST_NODE_TYPES.Identifier &&
    node.callee.property.name === "isNaN";

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

/**
 * Returns true when reaching `child` from `parent` requires taking a conditional branch,
 * i.e. `child` is not guaranteed to execute whenever `parent` is reached. Statement positions
 * that always execute (an `if` test, the left operand of `&&`/`||`, a `try` block) are not
 * treated as conditional; branch bodies, loop bodies, catch clauses, and function bodies are.
 */
function isConditionalEdge(parent: TSESTree.Node, child: TSESTree.Node): boolean {
  switch (parent.type) {
    case AST_NODE_TYPES.IfStatement:
      return child === parent.consequent || child === parent.alternate;
    case AST_NODE_TYPES.ConditionalExpression:
      return child === parent.consequent || child === parent.alternate;
    case AST_NODE_TYPES.LogicalExpression:
      return child === parent.right;
    case AST_NODE_TYPES.SwitchCase:
      return parent.consequent.includes(child as TSESTree.Statement);
    case AST_NODE_TYPES.TryStatement:
      return child === parent.handler;
    case AST_NODE_TYPES.ForStatement:
    case AST_NODE_TYPES.ForInStatement:
    case AST_NODE_TYPES.ForOfStatement:
    case AST_NODE_TYPES.WhileStatement:
    case AST_NODE_TYPES.DoWhileStatement:
      return child === parent.body;
    case AST_NODE_TYPES.FunctionDeclaration:
    case AST_NODE_TYPES.FunctionExpression:
    case AST_NODE_TYPES.ArrowFunctionExpression:
      return child === parent.body;
    default:
      return false;
  }
}

/**
 * Returns true when the guard node is guaranteed to have executed by the time the comparison
 * node is evaluated: the guard must finish before the comparison starts in source order, and
 * no conditional branch may be entered on the guard's path below the deepest node the two
 * share. This rejects guards written after the risky comparison as well as guards nested in a
 * sibling branch that only runs on some paths.
 */
function guardDominatesComparison(guardPath: TSESTree.Node[], comparisonPath: TSESTree.Node[]): boolean {
  const guard = guardPath[guardPath.length - 1];
  const comparison = comparisonPath[comparisonPath.length - 1];
  if (guard.range[1] > comparison.range[0]) return false;

  let divergence = 0;
  while (divergence < guardPath.length && divergence < comparisonPath.length && guardPath[divergence] === comparisonPath[divergence]) {
    divergence++;
  }
  // Guards and comparisons always share the Program node, so `divergence` is normally at least 1.
  for (let i = Math.max(divergence, 1); i < guardPath.length; i++) {
    if (isConditionalEdge(guardPath[i - 1], guardPath[i])) return false;
  }
  return true;
}

function endsWithControlFlowExit(statement: TSESTree.Statement): boolean {
  if (statement.type === AST_NODE_TYPES.BlockStatement) {
    const lastStatement = statement.body.at(-1);
    return lastStatement !== undefined && endsWithControlFlowExit(lastStatement);
  }
  return statement.type === AST_NODE_TYPES.ReturnStatement || statement.type === AST_NODE_TYPES.ThrowStatement || statement.type === AST_NODE_TYPES.BreakStatement || statement.type === AST_NODE_TYPES.ContinueStatement;
}

/** Returns true when the invalid-date branch exits before execution can reach a later comparison. */
function isExitingIfGuard(guardPath: TSESTree.Node[]): boolean {
  const guard = guardPath.at(-1);
  const parent = guardPath.at(-2);
  return guard !== undefined && parent?.type === AST_NODE_TYPES.IfStatement && parent.test === guard && endsWithControlFlowExit(parent.consequent);
}

/** Returns true when the check's short-circuit branch directly gates the comparison expression. */
function guardDirectlyGatesComparison(guardPath: TSESTree.Node[], comparisonPath: TSESTree.Node[]): boolean {
  const guard = guardPath.at(-1);
  const comparison = comparisonPath.at(-1);
  if (guard === undefined || comparison === undefined) return false;

  return guardPath.some(node => {
    if (node.type !== AST_NODE_TYPES.LogicalExpression) return false;
    return node.left.range[0] <= guard.range[0] && guard.range[1] <= node.left.range[1] && node.right.range[0] <= comparison.range[0] && comparison.range[1] <= node.right.range[1];
  });
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
      requireInvalidDateCheck:
        "{{subject}} may be an Invalid Date and is compared with a relational operator ({{operator}}) without ever being checked via Number.isNaN({{getTimeTarget}}.getTime()). An unparseable date silently fails every comparison instead of surfacing an error.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    // Variable -> declarator node, for variables assigned `new Date(nonTrivialArg)`.
    const dateVars = new Map<TSESLint.Scope.Variable, TSESTree.Node>();
    // Variables confirmed validated via Number.isNaN(name.getTime()) / isNaN(name.getTime()),
    // each recorded with its ancestor path so ordering and reachability can be checked later.
    const guards = new Map<TSESLint.Scope.Variable, TSESTree.Node[][]>();
    // Relational comparisons to report once all traversal is done, keyed by the involved sides.
    const comparisons: { node: TSESTree.BinaryExpression; operator: string; sides: ComparisonSide[]; path: TSESTree.Node[] }[] = [];

    /** Returns true when at least one recorded guard for the variable is guaranteed to run before the comparison. */
    function isValidatedBefore(variable: TSESLint.Scope.Variable, comparisonPath: TSESTree.Node[]): boolean {
      const paths = guards.get(variable);
      if (paths === undefined) return false;
      return paths.some(guardPath => guardDominatesComparison(guardPath, comparisonPath) && (isExitingIfGuard(guardPath) || guardDirectlyGatesComparison(guardPath, comparisonPath)));
    }

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
            if (variable) {
              const paths = guards.get(variable) ?? [];
              paths.push([...sourceCode.getAncestors(node), node]);
              guards.set(variable, paths);
            }
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
        if (sides.length > 0) comparisons.push({ node, operator: node.operator, sides, path: [...sourceCode.getAncestors(node), node] });
      },

      "Program:exit"() {
        for (const { node, operator, sides, path } of comparisons) {
          const problems = sides.filter(side => (side.kind === "var" ? !isValidatedBefore(side.variable, path) : true));
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
