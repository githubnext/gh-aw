import { ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);
const GLOBAL_IS_NAN_OBJECTS = new Set(["globalThis", "window", "global"]);
const NUMERIC_CALL_NAMES = new Set(["parseInt", "parseFloat", "Number"]);
const NUMERIC_METHOD_NAMES = new Set(["getTime", "getTimezoneOffset", "valueOf"]);

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

    /**
     * Returns true when the argument is provably already a number type, making
     * isNaN(x) → Number.isNaN(x) a guaranteed semantics-preserving equivalence.
     *
     * Provably-numeric means: a numeric Literal, or a CallExpression to
     * parseInt / parseFloat / Number / Number.parseInt / Number.parseFloat, or a
     * CallExpression whose callee property is getTime / getTimezoneOffset / valueOf.
     */
    function isProvablyNumeric(arg: TSESTree.Node): boolean {
      if (arg.type === "Literal" && typeof arg.value === "number") {
        return true;
      }
      if (arg.type === "CallExpression") {
        const callee = arg.callee;
        // parseInt(x), parseFloat(x), Number(x)
        if (callee.type === "Identifier" && NUMERIC_CALL_NAMES.has(callee.name)) {
          return true;
        }
        // Number.parseInt(x), Number.parseFloat(x)
        if (
          callee.type === "MemberExpression" &&
          !callee.computed &&
          callee.object.type === "Identifier" &&
          callee.object.name === "Number" &&
          callee.property.type === "Identifier" &&
          (callee.property.name === "parseInt" || callee.property.name === "parseFloat")
        ) {
          return true;
        }
        // x.getTime(), x.getTimezoneOffset(), x.valueOf()
        if (callee.type === "MemberExpression" && !callee.computed && callee.property.type === "Identifier" && NUMERIC_METHOD_NAMES.has(callee.property.name)) {
          return true;
        }
      }
      return false;
    }

    function report(node: TSESTree.CallExpression): void {
      const [arg] = node.arguments;
      const provablyNumeric = arg !== undefined && arg.type !== "SpreadElement" && isProvablyNumeric(arg);

      if (provablyNumeric) {
        context.report({
          node: node.callee,
          messageId: "preferNumberIsNaN",
          fix(fixer: TSESLint.RuleFixer) {
            return fixer.replaceText(node.callee, "Number.isNaN");
          },
        });
      } else {
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
