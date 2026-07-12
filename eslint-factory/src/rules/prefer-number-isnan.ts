import { ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);
const GLOBAL_IS_NAN_OBJECTS = new Set(["globalThis", "window", "global"]);

export const preferNumberIsNanRule = createRule({
  name: "prefer-number-isnan",
  meta: {
    type: "suggestion",
    fixable: "code",
    hasSuggestions: true,
    docs: {
      description: "Prefer Number.isNaN() over global isNaN() to avoid coercion footguns when validating unknown inputs.",
    },
    schema: [],
    messages: {
      preferNumberIsNaN: "Prefer Number.isNaN(...) over global isNaN(...).",
      preferNumberIsNaNWithCoercionCaveat: "Prefer Number.isNaN(...) over global isNaN(...). Global isNaN() coerces non-number inputs and can hide invalid raw values.",
      replaceWithNumberIsNaN: "Replace callee with Number.isNaN.",
      replaceWithNumberIsNaNWithNumberWrapReview: "Replace callee with Number.isNaN — review whether the argument should be wrapped with Number(...).",
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

        // Any local definition shadows the global (including ESM ImportBinding).
        if (variable && variable.defs.length > 0) {
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

    function isNumberParseMethod(node: TSESTree.MemberExpression): boolean {
      if (node.object.type !== "Identifier" || node.object.name !== "Number") {
        return false;
      }

      const property = node.property;
      const isDirectAccess = !node.computed && property.type === "Identifier" && (property.name === "parseInt" || property.name === "parseFloat");
      const isComputedAccess = node.computed && property.type === "Literal" && (property.value === "parseInt" || property.value === "parseFloat");

      return isDirectAccess || isComputedAccess;
    }

    function isProvablyNumericArgument(node: TSESTree.CallExpressionArgument): boolean {
      if (node.type === "Literal") {
        return typeof node.value === "number";
      }

      if (node.type !== "CallExpression") {
        return false;
      }

      // Unqualified global functions are only provably numeric when unshadowed.
      if (node.callee.type === "Identifier") {
        const name = node.callee.name;

        return (name === "parseInt" || name === "parseFloat" || name === "Number") && !hasLocalBinding(node, name);
      }

      // Number.parseInt / Number.parseFloat are only provably numeric when Number is unshadowed.
      if (node.callee.type === "MemberExpression" && isNumberParseMethod(node.callee) && !hasLocalBinding(node, "Number")) {
        return true;
      }

      // Arbitrary method calls (getTime, valueOf, etc.) are not provably numeric —
      // the receiver type is unknown and the method can be overridden to return anything.
      return false;
    }

    function report(node: TSESTree.CallExpression): void {
      const numberUnshadowed = !hasLocalBinding(node, "Number");
      const hasSingleProvablyNumericArgument = node.arguments.length === 1 && isProvablyNumericArgument(node.arguments[0]);

      if (hasSingleProvablyNumericArgument && numberUnshadowed) {
        context.report({
          node: node.callee,
          messageId: "preferNumberIsNaN",
          fix(fixer: TSESLint.RuleFixer) {
            return fixer.replaceText(node.callee, "Number.isNaN");
          },
        });
        return;
      }

      context.report({
        node: node.callee,
        messageId: "preferNumberIsNaNWithCoercionCaveat",
        suggest: [
          {
            messageId: "replaceWithNumberIsNaNWithNumberWrapReview",
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
