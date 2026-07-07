import { AST_NODE_TYPES, TSESTree } from "@typescript-eslint/utils";

/** Callback sinks that are safe to recognize as either bare function calls or member calls. */
const DEFERRED_SINK_NAMES = new Set(["then", "catch", "finally", "on", "once", "addEventListener", "setTimeout", "setInterval", "setImmediate", "queueMicrotask"]);

/**
 * Callback sinks that must only be recognized as member calls.
 * This avoids false negatives for user-defined synchronous helpers like `nextTick(fn)`.
 */
const MEMBER_ONLY_DEFERRED_SINK_NAMES = new Set(["nextTick"]);

export const SAFE_WRAPPABLE_STATEMENT_TYPES = new Set<AST_NODE_TYPES>([AST_NODE_TYPES.ExpressionStatement, AST_NODE_TYPES.ReturnStatement]);

function escapeRegex(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

function getCommonContinuationIndent(lines: string[]): string {
  const indents = lines.filter(line => line.trim().length > 0).map(line => line.match(/^(\s*)/)?.[1] ?? "");

  if (indents.length === 0) return "";

  let commonIndent = indents[0];
  for (const indent of indents.slice(1)) {
    let shared = 0;
    while (shared < commonIndent.length && shared < indent.length && commonIndent[shared] === indent[shared]) {
      shared++;
    }
    commonIndent = commonIndent.slice(0, shared);
  }

  return commonIndent;
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
  const lines = stmtText.split("\n");
  const continuationIndent = getCommonContinuationIndent(lines.slice(1));
  const continuationIndentPattern = continuationIndent.length > 0 ? new RegExp(`^${escapeRegex(continuationIndent)}`) : null;
  const indentedStatement = lines
    .map((line, index) => {
      const normalizedLine = index === 0 ? line.trimStart() : continuationIndentPattern ? line.replace(continuationIndentPattern, "") : line;
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
