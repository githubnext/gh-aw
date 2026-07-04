import { ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);
const GLOBAL_IS_NAN_OBJECTS = new Set(["globalThis", "window", "global"]);

export const preferNumberIsNanRule = createRule({
  name: "prefer-number-isnan",
  meta: {
    type: "suggestion",
    hasSuggestions: true,
    docs: {
      description: "Prefer Number.isNaN() over global isNaN() to avoid coercion footguns when validating unknown inputs.",
    },
    schema: [],
    messages: {
      preferNumberIsNaN: "Prefer Number.isNaN(...) over global isNaN(...). Global isNaN() coerces non-number inputs and can hide invalid raw values.",
      replaceWithNumberIsNaN: "Replace callee with Number.isNaN — review whether the argument should be wrapped with Number(...).",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

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

        if (variable?.defs.some(d => d.type !== "ImportBinding")) {
          return true;
        }

        scope = scope.upper;
      }

      return false;
    }

    /**
     * Checks whether a MemberExpression property is isNaN, either direct or computed.
     * @param node MemberExpression node to inspect.
     * @returns true if the property is isNaN.
     */
    function isIsNaNProperty(node: TSESTree.MemberExpression): boolean {
      const property = node.property;
      const isDirectAccess = !node.computed && property.type === "Identifier" && property.name === "isNaN";
      const isComputedAccess = property.type === "Literal" && property.value === "isNaN";

      return isDirectAccess || isComputedAccess;
    }

    function report(node: TSESTree.CallExpression): void {
      context.report({
        node: node.callee,
        messageId: "preferNumberIsNaN",
        suggest: [
          {
            messageId: "replaceWithNumberIsNaN",
            fix(fixer: TSESLint.RuleFixer) {
              return fixer.replaceText(node.callee, "Number.isNaN");
            },
          },
        ],
      });
    }

    return {
      CallExpression(node) {
        if (node.callee.type === "Identifier" && node.callee.name === "isNaN" && !hasLocalBinding(node, "isNaN")) {
          report(node);
          return;
        }

        if (node.callee.type === "MemberExpression" && node.callee.object.type === "Identifier" && GLOBAL_IS_NAN_OBJECTS.has(node.callee.object.name) && !hasLocalBinding(node, node.callee.object.name) && isIsNaNProperty(node.callee)) {
          report(node);
        }
      },
    };
  },
});
