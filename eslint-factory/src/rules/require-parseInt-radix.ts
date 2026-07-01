import { ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);
const GLOBAL_PARSE_INT_OBJECTS = new Set(["Number", "globalThis", "window", "global"]);

export const requireParseIntRadixRule = createRule({
  name: "require-parseInt-radix",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description: "Require parseInt() calls in gh-aw JavaScript runtime scripts to include an explicit radix argument to avoid implicit base detection (e.g., 0x prefix silently parsed as hexadecimal)",
    },
    schema: [],
    messages: {
      requireRadix: "parseInt() must be called with an explicit radix (e.g., parseInt(str, 10)) to avoid implicit base detection in gh-aw JavaScript runtime scripts.",
      addRadix10: "Add radix 10 as a safe default, then confirm the intended base for this input (e.g., 16/8 may be correct in some contexts).",
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

        if (variable?.defs.length) {
          return true;
        }

        scope = scope.upper;
      }

      return false;
    }

    /**
     * Checks whether a MemberExpression property is parseInt, either direct or computed.
     * @param node MemberExpression node to inspect.
     * @returns true if the property is parseInt.
     */
    function isParseIntProperty(node: TSESTree.MemberExpression): boolean {
      const property = node.property;
      const isDirectAccess = property.type === "Identifier" && property.name === "parseInt";
      const isComputedAccess = property.type === "Literal" && property.value === "parseInt";

      return isDirectAccess || isComputedAccess;
    }

    /**
     * Returns true when the second argument is a radix value that is safe (not equivalent to "no radix").
     * Literal 0/null and the global identifiers `undefined`/`NaN` all coerce to 0, so they are treated as
     * invalid. Any other value — including non-literal or locally shadowed identifiers — is accepted.
     */
    function isValidSecondArg(arg: TSESTree.CallExpressionArgument): boolean {
      // Literal 0/null are spec-equivalent to no radix
      if (arg.type === "Literal" && (arg.value === 0 || arg.value === null)) {
        return false;
      }
      // The global identifiers `undefined`/`NaN` are equivalent to no radix
      if (arg.type === "Identifier" && (arg.name === "undefined" || arg.name === "NaN") && !hasLocalBinding(arg, arg.name)) {
        return false;
      }
      return true;
    }

    return {
      CallExpression(node) {
        if (node.arguments.length >= 2 && isValidSecondArg(node.arguments[1])) {
          return;
        }

        // Do not offer a fix when the first argument is a spread (e.g. parseInt(...args)):
        // inserting ", 10" after a SpreadElement produces broken output.
        const firstArg = node.arguments[0];
        const secondArg = node.arguments[1];
        const suggest =
          secondArg && !isValidSecondArg(secondArg)
            ? [
                {
                  messageId: "addRadix10" as const,
                  fix(fixer: TSESLint.RuleFixer) {
                    return fixer.replaceText(secondArg, "10");
                  },
                },
              ]
            : node.arguments.length === 1 && firstArg.type !== "SpreadElement"
              ? [
                  {
                    messageId: "addRadix10" as const,
                    fix(fixer: TSESLint.RuleFixer) {
                      return fixer.insertTextAfter(firstArg, ", 10");
                    },
                  },
                ]
              : undefined;

        // Report only the global parseInt binding. Aliased (const p = parseInt; p(x))
        // and destructured (const { parseInt } = Number; parseInt(x)) bindings are
        // intentionally out of scope: tracking them reliably requires deeper
        // scope/alias analysis and is disproportionate to the current risk surface.
        // Global parseInt(x) — missing radix
        if (node.callee.type === "Identifier" && node.callee.name === "parseInt" && !hasLocalBinding(node, "parseInt")) {
          context.report({ node, messageId: "requireRadix", suggest });
          return;
        }

        // Accept both direct property access (Number.parseInt, globalThis.parseInt)
        // and computed string-literal access (Number["parseInt"]).
        if (
          node.callee.type === "MemberExpression" &&
          node.callee.object.type === "Identifier" &&
          GLOBAL_PARSE_INT_OBJECTS.has(node.callee.object.name) &&
          !hasLocalBinding(node, node.callee.object.name) &&
          isParseIntProperty(node.callee)
        ) {
          context.report({ node, messageId: "requireRadix", suggest });
        }
      },
    };
  },
});
