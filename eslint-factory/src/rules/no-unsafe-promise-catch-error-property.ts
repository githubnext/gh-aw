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

function isCatchCallback(node: TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression): boolean {
  const parent = node.parent;
  if (!parent || parent.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = parent.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return false;
  const prop = callee.property;
  return prop.type === AST_NODE_TYPES.Identifier && prop.name === "catch" && parent.arguments[0] === node;
}

export const noUnsafePromiseCatchErrorPropertyRule = createRule({
  name: "no-unsafe-promise-catch-error-property",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description: "Disallow direct access to .message, .stack, .code, .status, .cause, or .name on a promise .catch() callback parameter without a getErrorMessage guard",
    },
    schema: [],
    messages: {
      unsafeProperty: "Direct access to .{{prop}} on promise rejection '{{errorVar}}' is unsafe — the rejection value may not be an Error instance. Use getErrorMessage({{errorVar}}) from error_helpers.cjs instead.",
      useGetErrorMessage: "Replace with getErrorMessage({{errorVar}}) from error_helpers.cjs for safe error message extraction.",
    },
  },
  defaultOptions: [],
  create(context) {
    // Stack of frames — one per ArrowFunctionExpression/FunctionExpression.
    // Non-.catch() frames are sentinels (hasGuard: true) that shield outer frames
    // from false positives when a nested callback shadows the same parameter name.
    const stack: CatchFrame[] = [];

    function enterFunction(node: TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression): void {
      if (isCatchCallback(node)) {
        const params = node.params;
        // Only handle simple identifier bindings; skip no-param and destructuring callbacks.
        if (params.length === 1 && params[0].type === AST_NODE_TYPES.Identifier) {
          stack.push({ varName: params[0].name, hasGuard: false, hasNonNullGuard: false, unsafeNodes: [] });
        } else {
          stack.push({ varName: "", hasGuard: true, hasNonNullGuard: true, unsafeNodes: [] });
        }
      } else {
        // Sentinel: prevents the outer .catch() frame from collecting accesses
        // to a shadowed parameter name inside this nested function.
        stack.push({ varName: "", hasGuard: true, hasNonNullGuard: true, unsafeNodes: [] });
      }
    }

    function exitFunction(): void {
      const frame = stack.pop();
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
    }

    return {
      ArrowFunctionExpression: enterFunction,
      "ArrowFunctionExpression:exit": exitFunction,
      FunctionExpression: enterFunction,
      "FunctionExpression:exit": exitFunction,

      // Detect getErrorMessage(catchVar) call — accepted safe guard
      CallExpression(node) {
        if (stack.length === 0) return;
        const top = stack[stack.length - 1];
        if (!top || top.hasGuard || !top.varName) return;

        const firstArg = node.arguments[0];
        if (node.callee.type === AST_NODE_TYPES.Identifier && node.callee.name === "getErrorMessage" && node.arguments.length >= 1 && firstArg.type === AST_NODE_TYPES.Identifier && firstArg.name === top.varName) {
          top.hasGuard = true;
        }
      },

      // Detect catchVar instanceof Error — also accepted as a safe guard
      // Detect typeof catchVar === 'object' with a non-null companion guard
      BinaryExpression(node) {
        if (stack.length === 0) return;
        const top = stack[stack.length - 1];
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
        if (stack.length === 0) return;
        const top = stack[stack.length - 1];
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
        if (stack.length === 0) return;
        const top = stack[stack.length - 1];
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
      MemberExpression(node) {
        if (stack.length === 0) return;
        const top = stack[stack.length - 1];
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
