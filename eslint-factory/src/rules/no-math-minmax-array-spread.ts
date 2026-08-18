import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const MATH_MIN_MAX_METHODS = new Set(["min", "max"]);

export const noMathMinMaxArraySpreadRule = createRule({
  name: "no-math-minmax-array-spread",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow spreading a non-literal array into Math.min(...) / Math.max(...). Spreading a large array into call arguments can throw `RangeError: Maximum call stack size exceeded` once the array exceeds the engine argument limit, so arrays whose size depends on runtime data must be reduced instead.",
    },
    schema: [],
    messages: {
      noMathMinMaxArraySpread: "Avoid Math.{{method}}(...{{arg}}) — spreading an array of unknown size can throw `RangeError: Maximum call stack size exceeded`. Use `{{arg}}.reduce((a, b) => Math.{{method}}(a, b))` instead.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

    /**
     * Checks whether a given identifier name is locally bound in the current scope chain,
     * which means the global `Math` object is shadowed at that location.
     */
    function hasLocalBinding(node: TSESTree.Node, name: string): boolean {
      let scope: SourceCodeScope | null = sourceCode.getScope(node);

      while (scope) {
        const variable = scope.set.get(name);
        if (variable && variable.defs.length > 0) return true;
        scope = scope.upper;
      }

      return false;
    }

    /**
     * Returns "min" / "max" when the member expression accesses `Math.min` or `Math.max`
     * (direct or string-literal computed access), and undefined otherwise.
     */
    function getMathMinMaxMethod(node: TSESTree.MemberExpression): string | undefined {
      if (node.object.type !== AST_NODE_TYPES.Identifier || node.object.name !== "Math") return undefined;

      const property = node.property;
      if (!node.computed && property.type === AST_NODE_TYPES.Identifier && MATH_MIN_MAX_METHODS.has(property.name)) return property.name;
      if (node.computed && property.type === AST_NODE_TYPES.Literal && typeof property.value === "string" && MATH_MIN_MAX_METHODS.has(property.value)) return property.value;

      return undefined;
    }

    /**
     * Returns true when the spread argument has a size that is not statically bounded by
     * the source itself. Inline array literals are always bounded, so they are excluded.
     */
    function isUnboundedSpreadArgument(node: TSESTree.Node): boolean {
      return node.type === AST_NODE_TYPES.Identifier || node.type === AST_NODE_TYPES.MemberExpression || node.type === AST_NODE_TYPES.CallExpression;
    }

    return {
      CallExpression(node) {
        if (node.callee.type !== AST_NODE_TYPES.MemberExpression) return;

        const method = getMathMinMaxMethod(node.callee);
        if (!method) return;
        if (hasLocalBinding(node, "Math")) return;

        // Only the single-argument spread form is reported: fixed arguments such as
        // `Math.max(0, ...arr)` suggest an intentional, likely bounded call shape.
        if (node.arguments.length !== 1) return;

        const argument = node.arguments[0];
        if (argument.type !== AST_NODE_TYPES.SpreadElement) return;
        if (!isUnboundedSpreadArgument(argument.argument)) return;

        context.report({
          node,
          messageId: "noMathMinMaxArraySpread",
          data: { method, arg: sourceCode.getText(argument.argument) },
        });
      },
    };
  },
});
