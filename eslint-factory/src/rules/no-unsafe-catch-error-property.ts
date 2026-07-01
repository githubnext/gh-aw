import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const UNSAFE_PROPERTIES = new Set(["message", "stack", "code", "status", "cause", "name"]);

interface CatchFrame {
  varName: string;
  hasGuard: boolean;
  hasNonNullGuard: boolean;
  unsafeNodes: Array<{ node: TSESTree.MemberExpression; prop: string }>;
}

function isTypeofObjectCheck(node: TSESTree.Expression, varName: string): boolean {
  if (node.type !== AST_NODE_TYPES.BinaryExpression || node.operator !== "===") return false;
  const { left, right } = node;
  return (
    (left.type === AST_NODE_TYPES.UnaryExpression && left.operator === "typeof" && left.argument.type === AST_NODE_TYPES.Identifier && left.argument.name === varName && right.type === AST_NODE_TYPES.Literal && right.value === "object") ||
    (right.type === AST_NODE_TYPES.UnaryExpression && right.operator === "typeof" && right.argument.type === AST_NODE_TYPES.Identifier && right.argument.name === varName && left.type === AST_NODE_TYPES.Literal && left.value === "object")
  );
}

function isNonNullGuardCheck(node: TSESTree.Expression, varName: string): boolean {
  if (node.type === AST_NODE_TYPES.Identifier) {
    return node.name === varName;
  }

  if (node.type !== AST_NODE_TYPES.BinaryExpression || (node.operator !== "!==" && node.operator !== "!=")) {
    return false;
  }

  const isVarRef = (value: TSESTree.Expression): value is TSESTree.Identifier => value.type === AST_NODE_TYPES.Identifier && value.name === varName;
  const isNullLiteral = (value: TSESTree.Expression): value is TSESTree.Literal => value.type === AST_NODE_TYPES.Literal && value.value === null;

  return (isVarRef(node.left) && isNullLiteral(node.right)) || (isVarRef(node.right) && isNullLiteral(node.left));
}

export const noUnsafeCatchErrorPropertyRule = createRule({
  name: "no-unsafe-catch-error-property",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description: "Disallow direct access to .message, .stack, .code, .status, .cause, or .name on a caught error variable without a getErrorMessage guard",
    },
    schema: [],
    messages: {
      unsafeProperty: "Direct access to .{{prop}} on caught error '{{errorVar}}' is unsafe — the thrown value may not be an Error instance. Use getErrorMessage({{errorVar}}) from error_helpers.cjs instead.",
      useGetErrorMessage: "Replace with getErrorMessage({{errorVar}}) from error_helpers.cjs for safe error message extraction.",
      wrapWithInstanceof: "Wrap with '({{errorVar}} instanceof Error ? {{errorVar}}.{{prop}} : undefined)' to guard against non-Error throws.",
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
          catchStack.push({ varName: "", hasGuard: true, hasNonNullGuard: true, unsafeNodes: [] });
          return;
        }

        catchStack.push({ varName: param.name, hasGuard: false, hasNonNullGuard: false, unsafeNodes: [] });
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
                : [
                    {
                      messageId: "wrapWithInstanceof" as const,
                      data: { errorVar: varName, prop },
                      fix(fixer) {
                        return fixer.replaceText(memberExpr, `(${varName} instanceof Error ? ${varName}.${prop} : undefined)`);
                      },
                    },
                  ],
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
      // Detect typeof catchVar === 'object' with a non-null companion guard
      BinaryExpression(node) {
        if (catchStack.length === 0) return;
        const top = catchStack[catchStack.length - 1];
        if (!top || top.hasGuard || !top.varName) return;

        if (node.operator === "instanceof" && node.left.type === AST_NODE_TYPES.Identifier && node.left.name === top.varName) {
          top.hasGuard = true;
          return;
        }

        if (isTypeofObjectCheck(node, top.varName) && top.hasNonNullGuard) {
          top.hasGuard = true;
        }
      },

      LogicalExpression(node) {
        if (catchStack.length === 0) return;
        const top = catchStack[catchStack.length - 1];
        if (!top || top.hasGuard || !top.varName || node.operator !== "&&") return;

        const conjuncts: TSESTree.Expression[] = [];
        const collectConjuncts = (expr: TSESTree.Expression): void => {
          if (expr.type === AST_NODE_TYPES.LogicalExpression && expr.operator === "&&") {
            collectConjuncts(expr.left);
            collectConjuncts(expr.right);
            return;
          }
          conjuncts.push(expr);
        };
        collectConjuncts(node);

        const hasTypeofObject = conjuncts.some(expr => isTypeofObjectCheck(expr, top.varName));
        const hasNonNullGuard = conjuncts.some(expr => isNonNullGuardCheck(expr, top.varName));
        if (hasTypeofObject && hasNonNullGuard) {
          top.hasGuard = true;
          top.hasNonNullGuard = true;
        }
      },

      IfStatement(node) {
        if (catchStack.length === 0) return;
        const top = catchStack[catchStack.length - 1];
        if (!top || top.hasGuard || !top.varName) return;

        if (
          node.test.type === AST_NODE_TYPES.UnaryExpression &&
          node.test.operator === "!" &&
          node.test.argument.type === AST_NODE_TYPES.Identifier &&
          node.test.argument.name === top.varName &&
          node.consequent.type === AST_NODE_TYPES.ReturnStatement
        ) {
          top.hasNonNullGuard = true;
        }
      },

      // Collect catchVar.message / catchVar.stack / catchVar.code / catchVar.status / catchVar.cause / catchVar.name accesses
      // Also detects computed string-literal access: catchVar["message"], catchVar["status"], etc.
      MemberExpression(node) {
        if (catchStack.length === 0) return;
        const top = catchStack[catchStack.length - 1];
        if (!top || !top.varName) return;

        const obj = node.object;
        const prop = node.property;

        if (obj.type !== AST_NODE_TYPES.Identifier || obj.name !== top.varName) return;

        // Non-computed dot access: err.message / err.stack / err.code / err.status / err.cause / err.name
        if (!node.computed && prop.type === AST_NODE_TYPES.Identifier && UNSAFE_PROPERTIES.has(prop.name)) {
          top.unsafeNodes.push({ node, prop: prop.name });
          return;
        }

        // Computed string-literal access: err["message"] / err["stack"] / err["status"] / etc.
        // Dynamic access (err[prop]) is kept out of scope intentionally.
        if (node.computed && prop.type === AST_NODE_TYPES.Literal && typeof prop.value === "string" && UNSAFE_PROPERTIES.has(prop.value)) {
          top.unsafeNodes.push({ node, prop: prop.value });
        }
      },
    };
  },
});
