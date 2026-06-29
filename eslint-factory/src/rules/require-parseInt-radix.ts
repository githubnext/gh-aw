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

    return {
      CallExpression(node) {
        if (node.arguments.length >= 2) {
          return;
        }

        const suggest =
          node.arguments.length === 1
            ? [
                {
                  messageId: "addRadix10" as const,
                  fix(fixer: TSESLint.RuleFixer) {
                    return fixer.insertTextAfter(node.arguments[0], ", 10");
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
