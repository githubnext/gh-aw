import { ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

export const requireNanCheckAfterSplitIndexParseRule = createRule({
  name: "require-nan-check-after-split-index-parse",
  meta: {
    type: "problem",
    docs: {
      description: "Require NaN validation after parsing a numeric value out of a string.split(...)[index] expression, since malformed delimited strings silently produce NaN that can propagate into API calls.",
    },
    schema: [],
    messages: {
      requireNaNCheck:
        "Numeric value '{{name}}' parsed from a 'split(...)[index]' expression is never validated with Number.isNaN(), isNaN(), Number.isFinite(), isFinite(), or a truthiness check. A malformed delimited string will silently produce NaN, which can then be passed to downstream API calls.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

    // Set of VariableDeclarator nodes for split(...)[index]-derived parses, keyed by node identity
    // so that same-named variables in different scopes are not conflated.
    const unvalidated = new Set<TSESTree.VariableDeclarator>();
    // Set of VariableDeclarator nodes confirmed to be validated (via isNaN / Number.isNaN or a truthiness guard)
    const validated = new Set<TSESTree.VariableDeclarator>();

    /**
     * Checks whether a given identifier name is locally bound in the current scope chain.
     * @param node AST node to start the scope search from.
     * @param name Identifier name to search for.
     * @returns true if the name has a local binding, false otherwise.
     */
    function hasLocalBinding(node: TSESTree.Node, name: string): boolean {
      let scope: SourceCodeScope | null = sourceCode.getScope(node);

      while (scope) {
        const variable = scope.set.get(name);

        if (variable?.defs.length) {
          return true;
        }

        scope = scope.upper;
      }

      return false;
    }

    /**
     * Resolves an Identifier reference to the VariableDeclarator that declared it,
     * using scope analysis so that same-named variables in different scopes are distinguished.
     */
    function resolveDeclarator(identifier: TSESTree.Identifier): TSESTree.VariableDeclarator | null {
      let scope: SourceCodeScope | null = sourceCode.getScope(identifier);
      while (scope) {
        const variable = scope.variables.find(v => v.name === identifier.name);
        if (variable) {
          const def = variable.defs.find(d => d.node.type === "VariableDeclarator");
          if (def && def.node.type === "VariableDeclarator") {
            return def.node as TSESTree.VariableDeclarator;
          }
          return null;
        }
        scope = scope.upper;
      }
      return null;
    }

    /**
     * Returns true when the given node is a `<expr>.split(...)[<index>]` member access,
     * e.g. `endpoint.split(":")[1]`.
     */
    function isSplitIndexAccess(node: TSESTree.Node): boolean {
      if (node.type !== "MemberExpression" || !node.computed) return false;
      const obj = node.object;
      return obj.type === "CallExpression" && obj.callee.type === "MemberExpression" && !obj.callee.computed && obj.callee.property.type === "Identifier" && obj.callee.property.name === "split";
    }

    /**
     * Returns true when the given node is or contains a split-index access within
     * a logical fallback or conditional branch.
     */
    function isSplitIndexAccessDeep(node: TSESTree.Node): boolean {
      if (node.type === "LogicalExpression") {
        return isSplitIndexAccessDeep(node.left) || isSplitIndexAccessDeep(node.right);
      }
      if (node.type === "ConditionalExpression") {
        return isSplitIndexAccessDeep(node.consequent) || isSplitIndexAccessDeep(node.alternate);
      }
      return isSplitIndexAccess(node);
    }

    /**
     * Returns true when the call expression is a numeric-parse function whose
     * first argument is a `split(...)[index]` access.
     */
    function isNumericParseCallFromSplitIndex(node: TSESTree.CallExpression): boolean {
      const { callee, arguments: args } = node;

      if (args.length === 0 || args[0].type === "SpreadElement") return false;

      const firstArg = args[0] as TSESTree.Expression;

      // Global parseInt(splitExpr, ...) or parseFloat(splitExpr)
      if (callee.type === "Identifier" && (callee.name === "parseInt" || callee.name === "parseFloat") && !hasLocalBinding(callee, callee.name)) {
        return isSplitIndexAccessDeep(firstArg);
      }

      // Number.parseInt(splitExpr, ...) or Number.parseFloat(splitExpr)
      if (
        callee.type === "MemberExpression" &&
        callee.object.type === "Identifier" &&
        callee.object.name === "Number" &&
        !hasLocalBinding(callee.object, "Number") &&
        !callee.computed &&
        callee.property.type === "Identifier" &&
        (callee.property.name === "parseInt" || callee.property.name === "parseFloat")
      ) {
        return isSplitIndexAccessDeep(firstArg);
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
      if (callee.type === "Identifier" && (callee.name === "isNaN" || callee.name === "isFinite") && !hasLocalBinding(callee, callee.name)) {
        return true;
      }

      // Number.isNaN(x) / Number.isFinite(x)
      if (
        callee.type === "MemberExpression" &&
        callee.object.type === "Identifier" &&
        callee.object.name === "Number" &&
        !hasLocalBinding(callee.object, "Number") &&
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
        const declarator = resolveDeclarator(expr);
        if (declarator) validated.add(declarator);
      }
    }

    return {
      VariableDeclarator(node) {
        if (node.id.type === "Identifier" && node.init?.type === "CallExpression" && isNumericParseCallFromSplitIndex(node.init as TSESTree.CallExpression)) {
          unvalidated.add(node);
        }
      },

      CallExpression(node) {
        // Track any isNaN(x) / Number.isNaN(x) call where x is an identifier
        if (isIsNaNCall(node) && node.arguments.length === 1 && node.arguments[0].type === "Identifier") {
          const declarator = resolveDeclarator(node.arguments[0] as TSESTree.Identifier);
          if (declarator) validated.add(declarator);
        }
      },

      IfStatement(node) {
        markTruthinessGuard(node.test);
      },

      ConditionalExpression(node) {
        markTruthinessGuard(node.test);
      },

      "Program:exit"() {
        for (const declaratorNode of unvalidated) {
          if (!validated.has(declaratorNode)) {
            const name = declaratorNode.id.type === "Identifier" ? declaratorNode.id.name : "";
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
