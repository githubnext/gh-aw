import { AST_NODE_TYPES, TSESLint, TSESTree } from "@typescript-eslint/utils";

/**
 * When `identifier` is a write-once local variable binding, returns its
 * initializer expression so the caller can apply further checks. Returns null
 * for parameters, imports, multiply-assigned vars, and vars with no
 * initializer.
 */
function resolveInitializer(identifier: TSESTree.Identifier, sourceCode: TSESLint.SourceCode): TSESTree.Expression | null {
  const startScope = sourceCode.getScope(identifier);
  const functionScope = startScope.variableScope;
  // Only resolve within a concrete function boundary (function declaration,
  // function expression, or arrow function). Module/global scopes are
  // intentionally skipped because those bindings are not a stable proxy for
  // runtime values at call time.
  if (functionScope.type !== "function") return null;

  let scope: TSESLint.Scope.Scope | null = startScope;
  // Stay inside the same function's nested block scopes; do not cross to
  // enclosing function/module scopes.
  while (scope !== null && scope.variableScope === functionScope) {
    const variable = scope.set.get(identifier.name);
    if (variable !== undefined) {
      // Only accept simple, single-definition Variable bindings.
      if (variable.defs.length !== 1) return null;
      const def = variable.defs[0];
      if (def.type !== "Variable") return null;
      // Reject re-assigned bindings (write references that are not the initializer).
      if (variable.references.some(ref => ref.isWrite() && !ref.init)) return null;
      const declarator = def.node as TSESTree.VariableDeclarator;
      return declarator.init ?? null;
    }
    scope = scope.upper;
  }
  return null;
}

export function resolveWriteOnceInitializerChain(expression: TSESTree.Expression, sourceCode: TSESLint.SourceCode): TSESTree.Expression {
  let candidate = expression;
  const seen = new Set<TSESTree.Identifier>();
  while (candidate.type === AST_NODE_TYPES.Identifier && !seen.has(candidate)) {
    seen.add(candidate);
    const resolved = resolveInitializer(candidate, sourceCode);
    if (!resolved) break;
    candidate = resolved;
  }
  return candidate;
}

/**
 * String methods that return a normalized copy of their receiver. Chaining one
 * of these after a command string (for example `` `git checkout ${branch}`.trim() ``)
 * keeps the interpolated value in the resulting command, so the receiver must be
 * inspected instead of the outer call expression.
 */
const STRING_TRANSFORM_METHODS = new Set(["trim", "trimStart", "trimEnd", "toLowerCase", "toUpperCase", "toLocaleLowerCase", "toLocaleUpperCase", "replace", "replaceAll", "normalize"]);

/**
 * When the node is a call to a string-normalizing method (for example
 * `.trim()` or `.toLowerCase()`), returns the receiver expression and the call
 * arguments so callers can inspect the underlying command string. Returns null
 * otherwise.
 */
function getStringTransformCall(node: TSESTree.Expression): { receiver: TSESTree.Expression; args: TSESTree.CallExpressionArgument[] } | null {
  if (node.type !== AST_NODE_TYPES.CallExpression) return null;
  const callee = node.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return null;
  if (callee.property.type !== AST_NODE_TYPES.Identifier || !STRING_TRANSFORM_METHODS.has(callee.property.name)) return null;
  return { receiver: callee.object, args: node.arguments };
}

/**
 * Returns true when the node is a purely static expression (no runtime
 * interpolation): a literal, a no-expression template literal, a binary `+` of
 * two static expressions, or a string-normalizing method call whose receiver
 * and arguments are all static (for example `.replace()` can inject dynamic
 * content through its replacement argument).
 */
export function isStaticExpression(node: TSESTree.Expression): boolean {
  if (node.type === AST_NODE_TYPES.Literal) return true;
  if (node.type === AST_NODE_TYPES.TemplateLiteral) return node.expressions.length === 0;
  if (node.type === AST_NODE_TYPES.BinaryExpression && node.operator === "+") {
    return isStaticExpression(node.left) && isStaticExpression(node.right);
  }
  const transform = getStringTransformCall(node);
  if (transform) {
    if (!isStaticExpression(transform.receiver)) return false;
    return transform.args.every(arg => arg.type !== AST_NODE_TYPES.SpreadElement && isStaticExpression(arg));
  }
  return false;
}

/**
 * Returns true when the node is a dynamic string concatenation (binary `+`
 * that is not entirely static).
 */
export function isDynamicStringConcatenation(node: TSESTree.Expression): boolean {
  return node.type === AST_NODE_TYPES.BinaryExpression && node.operator === "+" && !isStaticExpression(node);
}

/**
 * Returns the display kind string for the problematic command expression, or
 * null when the expression is not one of the flagged shapes.
 *
 * Write-once local bindings and chained string-normalizing calls (for example
 * `` `git checkout ${branch}`.trim() ``) are unwrapped before the check so the
 * underlying command string is inspected. For calls that accept arguments (for
 * example `.replace(pattern, value)`), the arguments are inspected as well.
 */
export function getDynamicCommandKind(expression: TSESTree.Expression, sourceCode: TSESLint.SourceCode, seen: Set<TSESTree.Expression> = new Set()): string | null {
  const candidate = resolveWriteOnceInitializerChain(expression, sourceCode);
  if (seen.has(candidate)) return null;
  seen.add(candidate);

  if (candidate.type === AST_NODE_TYPES.TemplateLiteral && candidate.expressions.length > 0) return "interpolated template literal";
  if (isDynamicStringConcatenation(candidate)) return "dynamic string concatenation";

  const transform = getStringTransformCall(candidate);
  if (!transform) return null;

  const receiverKind = getDynamicCommandKind(transform.receiver, sourceCode, seen);
  if (receiverKind) return receiverKind;
  for (const arg of transform.args) {
    if (arg.type === AST_NODE_TYPES.SpreadElement) continue;
    const argKind = getDynamicCommandKind(arg, sourceCode, seen);
    if (argKind) return argKind;
  }
  return null;
}
