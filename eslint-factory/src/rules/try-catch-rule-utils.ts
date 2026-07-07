import { AST_NODE_TYPES, TSESTree } from "@typescript-eslint/utils";

const DEFERRED_SINK_NAMES = new Set(["then", "catch", "finally", "on", "once", "addEventListener", "setTimeout", "setInterval", "setImmediate", "queueMicrotask"]);

const MEMBER_ONLY_DEFERRED_SINK_NAMES = new Set(["nextTick"]);

export const SAFE_WRAPPABLE_STATEMENT_TYPES = new Set<AST_NODE_TYPES>([AST_NODE_TYPES.ExpressionStatement, AST_NODE_TYPES.ReturnStatement]);

function escapeRegex(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function isFunctionExpressionLike(node: TSESTree.Node): node is TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression {
  return node.type === AST_NODE_TYPES.ArrowFunctionExpression || node.type === AST_NODE_TYPES.FunctionExpression;
}

/** Returns true when funcNode is passed to a callback sink not protected by the outer try. */
export function isDeferredCallback(funcNode: TSESTree.Node): boolean {
  if (!isFunctionExpressionLike(funcNode)) return false;

  const parent = funcNode.parent;
  if (!parent) return false;

  const isCallLikeParent = parent.type === AST_NODE_TYPES.NewExpression || parent.type === AST_NODE_TYPES.CallExpression;
  const args = isCallLikeParent ? parent.arguments : undefined;
  const isArgument = args?.includes(funcNode) ?? false;
  const isPromiseConstructor = parent.type === AST_NODE_TYPES.NewExpression && parent.callee.type === AST_NODE_TYPES.Identifier && parent.callee.name === "Promise";
  if (isPromiseConstructor && isArgument) {
    return true;
  }

  if (parent.type === AST_NODE_TYPES.CallExpression && isArgument) {
    const callee = parent.callee;
    if (callee.type === AST_NODE_TYPES.Identifier && DEFERRED_SINK_NAMES.has(callee.name)) {
      return true;
    }
    if (callee.type === AST_NODE_TYPES.MemberExpression && !callee.computed && callee.property.type === AST_NODE_TYPES.Identifier) {
      return DEFERRED_SINK_NAMES.has(callee.property.name) || MEMBER_ONLY_DEFERRED_SINK_NAMES.has(callee.property.name);
    }
  }

  return false;
}

type TryCatchSuggestionOptions = {
  indent: string;
  todoComment: string;
  errorPrefix: string;
};

export function buildTryCatchSuggestion(stmtText: string, options: TryCatchSuggestionOptions): string {
  const { indent, todoComment, errorPrefix } = options;
  const normalizedIndent = indent.length > 0 ? new RegExp(`^${escapeRegex(indent)}`) : null;
  const indentedStatement = stmtText
    .split("\n")
    .map((line, index) => {
      const normalizedLine = index === 0 || !normalizedIndent ? line : line.replace(normalizedIndent, "");
      return `${indent}  ${normalizedLine}`;
    })
    .join("\n");

  return [
    "try {",
    indentedStatement,
    `${indent}} catch (err) {`,
    `${indent}  // ${todoComment}`,
    `${indent}  throw new Error(`,
    `${indent}    "${errorPrefix}" + (err instanceof Error ? err.message : String(err)),`,
    `${indent}    { cause: err },`,
    `${indent}  );`,
    `${indent}}`,
  ].join("\n");
}
