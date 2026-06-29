import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const UNSAFE_PROPERTIES = new Set(["message", "stack", "code"]);

interface CatchFrame {
  varName: string;
  hasGuard: boolean;
  unsafeNodes: Array<{ node: TSESTree.MemberExpression; prop: string }>;
}

export const noUnsafeCatchErrorPropertyRule = createRule({
  name: "no-unsafe-catch-error-property",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description: "Disallow direct access to .message, .stack, or .code on a caught error variable without a getErrorMessage guard",
    },
    schema: [],
    messages: {
      unsafeProperty: "Direct access to .{{prop}} on caught error '{{errorVar}}' is unsafe — the thrown value may not be an Error instance. Use getErrorMessage({{errorVar}}) from error_helpers.cjs instead.",
      useGetErrorMessage: "Replace with getErrorMessage({{errorVar}}) from error_helpers.cjs for safe error message extraction.",
    },
  },
  defaultOptions: [],
  create(context) {
    const catchStack: CatchFrame[] = [];

    return {
      CatchClause(node) {
        const param = node.param;

        // Only handle simple identifier bindings; skip bare catch {} and destructuring patterns.
        // Push a sentinel frame so CatchClause:exit always has a matching pop.
        if (!param || param.type !== AST_NODE_TYPES.Identifier) {
          catchStack.push({ varName: "", hasGuard: true, unsafeNodes: [] });
          return;
        }

        catchStack.push({ varName: param.name, hasGuard: false, unsafeNodes: [] });
      },

      "CatchClause:exit"() {
        const frame = catchStack.pop();
        if (!frame || !frame.varName || frame.hasGuard) return;

        for (const { node: memberExpr, prop } of frame.unsafeNodes) {
          const { varName } = frame;
          context.report({
            node: memberExpr,
            messageId: "unsafeProperty",
            data: { prop, errorVar: varName },
            suggest:
              prop === "message"
                ? [
                    {
                      messageId: "useGetErrorMessage" as const,
                      data: { errorVar: varName },
                      fix(fixer) {
                        return fixer.replaceText(memberExpr, `getErrorMessage(${varName})`);
                      },
                    },
                  ]
                : undefined,
          });
        }
      },

      // Detect getErrorMessage(catchVar) call — accepted safe guard
      CallExpression(node) {
        if (catchStack.length === 0) return;
        const top = catchStack[catchStack.length - 1];
        if (!top || top.hasGuard || !top.varName) return;

        const firstArg = node.arguments[0];
        if (node.callee.type === AST_NODE_TYPES.Identifier && node.callee.name === "getErrorMessage" && node.arguments.length >= 1 && firstArg.type === AST_NODE_TYPES.Identifier && firstArg.name === top.varName) {
          top.hasGuard = true;
        }
      },

      // Detect catchVar instanceof Error — also accepted as a safe guard
      BinaryExpression(node) {
        if (catchStack.length === 0) return;
        const top = catchStack[catchStack.length - 1];
        if (!top || top.hasGuard || !top.varName) return;

        if (node.operator === "instanceof" && node.left.type === AST_NODE_TYPES.Identifier && node.left.name === top.varName) {
          top.hasGuard = true;
        }
      },

      // Collect catchVar.message / catchVar.stack / catchVar.code accesses
      MemberExpression(node) {
        if (catchStack.length === 0) return;
        const top = catchStack[catchStack.length - 1];
        if (!top || !top.varName) return;

        const obj = node.object;
        const prop = node.property;
        if (!node.computed && obj.type === AST_NODE_TYPES.Identifier && obj.name === top.varName && prop.type === AST_NODE_TYPES.Identifier && UNSAFE_PROPERTIES.has(prop.name)) {
          top.unsafeNodes.push({ node, prop: prop.name });
        }
      },
    };
  },
});
