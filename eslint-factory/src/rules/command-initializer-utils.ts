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
