import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const UNSAFE_PROPERTIES = new Set(["message", "stack", "code"]);

interface Guard {
  /** Source start offset of the guard expression. */
  start: number;
  /** Snapshot of the block-nesting stack at the point the guard was seen. */
  blockStack: readonly number[];
}

interface UnsafeNode {
  node: TSESTree.MemberExpression;
  prop: string;
  /** Snapshot of the block-nesting stack at the point the access was seen. */
  blockStack: readonly number[];
}

interface CatchFrame {
  varName: string;
  guards: Guard[];
  unsafeNodes: UnsafeNode[];
  /** Mutable block-nesting stack — updated by BlockStatement enter/exit. */
  blockStack: number[];
}

/**
 * Returns true when guardStack is a prefix of (or equal to) accessStack.
 * This means the guard is at the same or an outer block level as the access,
 * so every execution path that reaches the access also passed through the guard's scope.
 */
function dominates(guardStack: readonly number[], accessStack: readonly number[]): boolean {
  if (guardStack.length > accessStack.length) return false;
  for (let i = 0; i < guardStack.length; i++) {
    if (guardStack[i] !== accessStack[i]) return false;
  }
  return true;
}

/**
 * Returns true when a guard lexically dominates an unsafe access:
 * the guard appears before the access in source order AND its block stack
 * is a prefix of the access's block stack (same or outer scope level).
 */
function guardProtects(guard: Guard, accessStart: number, accessStack: readonly number[]): boolean {
  return guard.start < accessStart && dominates(guard.blockStack, accessStack);
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
          catchStack.push({ varName: "", guards: [], unsafeNodes: [], blockStack: [] });
          return;
        }

        catchStack.push({ varName: param.name, guards: [], unsafeNodes: [], blockStack: [] });
      },

      "CatchClause:exit"() {
        const frame = catchStack.pop();
        if (!frame || !frame.varName) return;

        for (const { node: memberExpr, prop, blockStack: accessStack } of frame.unsafeNodes) {
          // An access is safe only when a guard dominates it: the guard's block stack is a prefix
          // of (or equal to) the access's block stack, and the guard appears first in source order.
          const accessStart = memberExpr.range[0];
          const isSafe = frame.guards.some(guard => guardProtects(guard, accessStart, accessStack));
          if (isSafe) continue;

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

      // Track block nesting within the active catch frame so guards and accesses
      // can be compared by their scope depth (used in dominance check above).
      BlockStatement(node) {
        if (catchStack.length === 0) return;
        catchStack[catchStack.length - 1].blockStack.push(node.range[0]);
      },

      "BlockStatement:exit"() {
        if (catchStack.length === 0) return;
        catchStack[catchStack.length - 1].blockStack.pop();
      },

      // Detect getErrorMessage(catchVar) call — accepted safe guard
      CallExpression(node) {
        if (catchStack.length === 0) return;
        const top = catchStack[catchStack.length - 1];
        if (!top || !top.varName) return;

        const firstArg = node.arguments[0];
        if (node.callee.type === AST_NODE_TYPES.Identifier && node.callee.name === "getErrorMessage" && node.arguments.length >= 1 && firstArg.type === AST_NODE_TYPES.Identifier && firstArg.name === top.varName) {
          top.guards.push({ start: node.range[0], blockStack: [...top.blockStack] });
        }
      },

      // Detect catchVar instanceof Error — also accepted as a safe guard
      BinaryExpression(node) {
        if (catchStack.length === 0) return;
        const top = catchStack[catchStack.length - 1];
        if (!top || !top.varName) return;

        if (node.operator === "instanceof" && node.left.type === AST_NODE_TYPES.Identifier && node.left.name === top.varName) {
          top.guards.push({ start: node.range[0], blockStack: [...top.blockStack] });
        }
      },

      // Collect catchVar.message / catchVar.stack / catchVar.code accesses
      // Also detects computed string-literal access: catchVar["message"], catchVar["stack"], catchVar["code"]
      MemberExpression(node) {
        if (catchStack.length === 0) return;
        const top = catchStack[catchStack.length - 1];
        if (!top || !top.varName) return;

        const obj = node.object;
        const prop = node.property;

        if (obj.type !== AST_NODE_TYPES.Identifier || obj.name !== top.varName) return;

        // Non-computed dot access: err.message / err.stack / err.code
        if (!node.computed && prop.type === AST_NODE_TYPES.Identifier && UNSAFE_PROPERTIES.has(prop.name)) {
          top.unsafeNodes.push({ node, prop: prop.name, blockStack: [...top.blockStack] });
          return;
        }

        // Computed string-literal access: err["message"] / err["stack"] / err["code"]
        // Dynamic access (err[prop]) is kept out of scope intentionally.
        if (node.computed && prop.type === AST_NODE_TYPES.Literal && typeof prop.value === "string" && UNSAFE_PROPERTIES.has(prop.value)) {
          top.unsafeNodes.push({ node, prop: prop.value, blockStack: [...top.blockStack] });
        }
      },
    };
  },
});
