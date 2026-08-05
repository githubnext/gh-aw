import { ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

export const requireNanCheckAfterEnvNumericParseRule = createRule({
  name: "require-nan-check-after-env-numeric-parse",
  meta: {
    type: "problem",
    docs: {
      description: "Require NaN validation after parsing numeric values from process.env to prevent silent NaN propagation into comparisons, loop bounds, or API payloads.",
    },
    schema: [],
    messages: {
      requireNaNCheck:
        "Numeric value '{{name}}' parsed from process.env is never validated with Number.isNaN(), isNaN(), Number.isFinite(), isFinite(), or a truthiness check. Parsing functions silently return NaN for malformed environment input.",
    },
  },
  defaultOptions: [],
  create(context) {
    // Map from variable name to the VariableDeclarator node (for reporting)
    const unvalidated = new Map<string, TSESTree.VariableDeclarator>();
    // Set of variable names confirmed to be passed to isNaN / Number.isNaN
    const validated = new Set<string>();

    /**
     * Returns true when the given node contains or is a process.env property access.
     * Handles member expressions, optional chaining, logical fallbacks, and ternaries.
     */
    function containsEnvAccess(node: TSESTree.Node): boolean {
      switch (node.type) {
        case "MemberExpression": {
          const obj = node.object;
          // Direct process.env.FOO or process.env["FOO"]
          if (obj.type === "MemberExpression" && obj.object.type === "Identifier" && obj.object.name === "process" && !obj.computed && obj.property.type === "Identifier" && obj.property.name === "env") {
            return true;
          }
          // Recurse into the object to handle deeper chains
          return containsEnvAccess(obj);
        }
        case "ChainExpression":
          return containsEnvAccess(node.expression);
        case "CallExpression":
          // process.env.FOO?.trim() — method call chained on an env access
          if (node.callee.type === "MemberExpression") {
            return containsEnvAccess(node.callee.object);
          }
          return false;
        case "LogicalExpression":
          // process.env.FOO || "default" or process.env.FOO ?? "default"
          return containsEnvAccess(node.left) || containsEnvAccess(node.right);
        case "ConditionalExpression":
          // ternary: process.env.FOO ? x : y  or  cond ? process.env.FOO : y
          return containsEnvAccess(node.test) || containsEnvAccess(node.consequent) || containsEnvAccess(node.alternate);
        default:
          return false;
      }
    }

    /**
     * Returns true when the call expression is a numeric-parse function whose
     * first argument traces back to a process.env access.
     */
    function isNumericParseCallFromEnv(node: TSESTree.CallExpression): boolean {
      const { callee, arguments: args } = node;

      if (args.length === 0 || args[0].type === "SpreadElement") return false;

      const firstArg = args[0] as TSESTree.Expression;

      // Global parseInt(envExpr, ...) or parseFloat(envExpr)
      if (callee.type === "Identifier" && (callee.name === "parseInt" || callee.name === "parseFloat")) {
        return containsEnvAccess(firstArg);
      }

      // Number.parseInt(envExpr, ...) or Number.parseFloat(envExpr)
      if (
        callee.type === "MemberExpression" &&
        callee.object.type === "Identifier" &&
        callee.object.name === "Number" &&
        !callee.computed &&
        callee.property.type === "Identifier" &&
        (callee.property.name === "parseInt" || callee.property.name === "parseFloat")
      ) {
        return containsEnvAccess(firstArg);
      }

      // Number(envExpr) — Number used as a conversion function
      if (callee.type === "Identifier" && callee.name === "Number") {
        return containsEnvAccess(firstArg);
      }

      return false;
    }

    /**
     * Returns true when the call expression is a NaN-validating global:
     * isNaN(...), Number.isNaN(...), isFinite(...) or Number.isFinite(...).
     */
    function isIsNaNCall(node: TSESTree.CallExpression): boolean {
      const { callee } = node;

      // Global isNaN(x) / isFinite(x)
      if (callee.type === "Identifier" && (callee.name === "isNaN" || callee.name === "isFinite")) {
        return true;
      }

      // Number.isNaN(x) / Number.isFinite(x)
      if (
        callee.type === "MemberExpression" &&
        callee.object.type === "Identifier" &&
        callee.object.name === "Number" &&
        !callee.computed &&
        callee.property.type === "Identifier" &&
        (callee.property.name === "isNaN" || callee.property.name === "isFinite")
      ) {
        return true;
      }

      return false;
    }

    /**
     * Marks a bare identifier (or `!identifier`) used as a condition test as validated,
     * since NaN is falsy and such a truthiness guard rejects it.
     */
    function markTruthinessGuard(test: TSESTree.Node): void {
      let expr = test;
      while (expr.type === "UnaryExpression" && expr.operator === "!") {
        expr = expr.argument;
      }
      if (expr.type === "Identifier") {
        validated.add(expr.name);
      }
    }

    return {
      VariableDeclarator(node) {
        if (node.id.type === "Identifier" && node.init?.type === "CallExpression" && isNumericParseCallFromEnv(node.init as TSESTree.CallExpression)) {
          unvalidated.set(node.id.name, node);
        }
      },

      CallExpression(node) {
        // Track any isNaN(x) / Number.isNaN(x) call where x is an identifier
        if (isIsNaNCall(node) && node.arguments.length === 1 && node.arguments[0].type === "Identifier") {
          validated.add((node.arguments[0] as TSESTree.Identifier).name);
        }
      },

      IfStatement(node) {
        markTruthinessGuard(node.test);
      },

      ConditionalExpression(node) {
        markTruthinessGuard(node.test);
      },

      "Program:exit"() {
        for (const [name, declaratorNode] of unvalidated) {
          if (!validated.has(name)) {
            context.report({
              node: declaratorNode,
              messageId: "requireNaNCheck",
              data: { name },
            });
          }
        }
      },
    };
  },
});
