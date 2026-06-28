import { ESLintUtils } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/actions/setup/js/eslint-factory#${name}`);

export const requireParseIntRadixRule = createRule({
  name: "require-parseInt-radix",
  meta: {
    type: "problem",
    docs: {
      description: "Require parseInt() calls in actions/setup/js scripts to include an explicit radix argument to avoid implicit base detection (e.g., 0x prefix silently parsed as hexadecimal)",
    },
    schema: [],
    messages: {
      requireRadix: "parseInt() must be called with an explicit radix (e.g., parseInt(str, 10)) to avoid implicit base detection in actions/setup/js.",
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      CallExpression(node) {
        // Global parseInt(x) — missing radix
        if (node.callee.type === "Identifier" && node.callee.name === "parseInt" && node.arguments.length < 2) {
          context.report({ node, messageId: "requireRadix" });
          return;
        }

        // Number.parseInt(x) — missing radix
        if (
          node.callee.type === "MemberExpression" &&
          !node.callee.computed &&
          node.callee.object.type === "Identifier" &&
          node.callee.object.name === "Number" &&
          node.callee.property.type === "Identifier" &&
          node.callee.property.name === "parseInt" &&
          node.arguments.length < 2
        ) {
          context.report({ node, messageId: "requireRadix" });
        }
      },
    };
  },
});
