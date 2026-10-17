import { ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

export const noMapParseIntCallbackRule = createRule({
  name: "no-map-parseint-callback",
  meta: {
    type: "problem",
    docs: {
      description: "Disallow passing parseInt directly as an array callback, because the element index is used as the radix.",
    },
    schema: [],
    messages: {
      noMapParseInt: 'Do not pass parseInt directly to .map(); the index becomes the radix (["10","10","10"].map(parseInt) yields [10, NaN, 2]). Use .map(value => parseInt(value, 10)) or .map(Number).',
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    function isShadowed(node: TSESTree.Node, name: string): boolean {
      let scope: ReturnType<typeof sourceCode.getScope> | null = sourceCode.getScope(node);
      while (scope) {
        const variable = scope.set.get(name);
        if (variable && variable.defs.length > 0) {
          return true;
        }
        scope = scope.upper;
      }
      return false;
    }

    function isParseInt(node: TSESTree.Node): boolean {
      if (node.type === "Identifier") {
        return node.name === "parseInt" && !isShadowed(node, "parseInt");
      }
      return (
        node.type === "MemberExpression" && !node.computed && node.object.type === "Identifier" && node.object.name === "Number" && !isShadowed(node, "Number") && node.property.type === "Identifier" && node.property.name === "parseInt"
      );
    }

    return {
      CallExpression(node) {
        const callee = node.callee;
        if (callee.type !== "MemberExpression" || callee.computed || callee.property.type !== "Identifier" || callee.property.name !== "map") {
          return;
        }
        const arg = node.arguments[0];
        if (arg && isParseInt(arg)) {
          context.report({ node: arg, messageId: "noMapParseInt" });
        }
      },
    };
  },
});
