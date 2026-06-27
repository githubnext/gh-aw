import { ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/actions/setup/js/eslint-factory#${name}`);

export const requireJsonParseTryCatchRule = createRule({
  name: "require-json-parse-try-catch",
  meta: {
    type: "problem",
    docs: {
      description: "Require JSON.parse calls in actions/setup/js scripts to be wrapped in try/catch",
    },
    schema: [],
    messages: {
      requireTryCatch: "Wrap JSON.parse in try/catch to avoid uncaught runtime failures in actions/setup/js.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    function isInsideTryBlock(node: TSESTree.Node): boolean {
      return sourceCode.getAncestors(node).some(ancestor => {
        if (ancestor.type !== "TryStatement") {
          return false;
        }

        const block = ancestor.block;
        return node.range[0] >= block.range[0] && node.range[1] <= block.range[1];
      });
    }

    return {
      CallExpression(node) {
        if (node.callee.type !== "MemberExpression") {
          return;
        }

        if (node.callee.object.type !== "Identifier") {
          return;
        }

        if (node.callee.object.name !== "JSON") {
          return;
        }

        if (node.callee.property.type !== "Identifier") {
          return;
        }

        if (node.callee.property.name !== "parse") {
          return;
        }

        if (!isInsideTryBlock(node)) {
          context.report({ node, messageId: "requireTryCatch" });
        }
      },
    };
  },
});
