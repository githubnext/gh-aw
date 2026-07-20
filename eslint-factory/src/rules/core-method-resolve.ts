import { AST_NODE_TYPES, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { CORE_ALIASES } from "./core-aliases";

/**
 * Checks whether an Identifier is a single-assignment alias for a core-like
 * object (e.g., `const c = core`). Re-assigned let bindings are rejected.
 * Local shadows (e.g., a parameter also named `c`) are excluded because they
 * are found first in the scope chain and their definition type will not match.
 *
 * Additionally, a plain function parameter whose name is in CORE_ALIASES is
 * accepted to support the dependency-injection pattern:
 *   `async function f(core) { core.setFailed(msg); }`
 * Gating: only parameters whose name exactly matches a known alias (i.e. is in
 * CORE_ALIASES) are accepted, preventing arbitrary parameters from being treated
 * as core. Destructured parameters (e.g. `{ core }`) are excluded.
 */
export function isCoreAliasIdentifier(identifier: TSESTree.Identifier, sourceCode: TSESLint.SourceCode): boolean {
  let currentScope: TSESLint.Scope.Scope | null = sourceCode.getScope(identifier);
  while (currentScope !== null) {
    const variable = currentScope.set.get(identifier.name);
    if (variable !== undefined) {
      if (variable.defs.length !== 1) return false;
      const def = variable.defs[0];
      if (def.type === "Parameter") {
        // Accept a plain function-parameter whose name is a known core alias.
        // This covers the DI pattern: `function f(core) { core.setFailed(...) }`.
        // For Parameter defs, `def.node` is the enclosing function node; `def.name`
        // is the binding pattern (Identifier for simple params, ObjectPattern for
        // destructured params).  We gate on `def.name.type === Identifier` to exclude
        // destructured parameters, and on `CORE_ALIASES` to avoid false positives.
        // `identifier.name` equals the parameter name because the scope look-up
        // above resolved the variable by that name.
        return def.name.type === AST_NODE_TYPES.Identifier && CORE_ALIASES.has(identifier.name);
      }
      if (def.type !== "Variable") return false;
      if (variable.references.some(ref => ref.isWrite() && !ref.init)) return false;
      const declarator = def.node as TSESTree.VariableDeclarator;
      if (!declarator.init) return false;
      return declarator.id.type === AST_NODE_TYPES.Identifier && declarator.init.type === AST_NODE_TYPES.Identifier && CORE_ALIASES.has(declarator.init.name);
    }
    currentScope = currentScope.upper;
  }
  return false;
}

/**
 * Checks whether an Identifier is a destructured binding for a specific
 * @actions/core method from a core-like object (e.g., `const { setOutput } = core`
 * or `const { setOutput: alias } = core` where `alias` is the identifier).
 * Re-assigned let bindings are rejected. Local `function setOutput()` or
 * parameter shadows are excluded via the `def.type !== "Variable"` guard.
 */
export function isDestructuredCoreMethodIdentifier(identifier: TSESTree.Identifier, methodName: string, sourceCode: TSESLint.SourceCode): boolean {
  let currentScope: TSESLint.Scope.Scope | null = sourceCode.getScope(identifier);
  while (currentScope !== null) {
    const variable = currentScope.set.get(identifier.name);
    if (variable !== undefined) {
      if (variable.defs.length !== 1) return false;
      const def = variable.defs[0];
      if (def.type !== "Variable") return false;
      if (variable.references.some(ref => ref.isWrite() && !ref.init)) return false;
      const declarator = def.node as TSESTree.VariableDeclarator;
      if (!declarator.init) return false;
      if (declarator.id.type === AST_NODE_TYPES.ObjectPattern && declarator.init.type === AST_NODE_TYPES.Identifier && CORE_ALIASES.has(declarator.init.name)) {
        return declarator.id.properties.some(prop => {
          if (prop.type !== AST_NODE_TYPES.Property || prop.computed) return false;
          const keyIsMethod = prop.key.type === AST_NODE_TYPES.Identifier && prop.key.name === methodName;
          const valueIsAlias = prop.value.type === AST_NODE_TYPES.Identifier && prop.value.name === identifier.name;
          return keyIsMethod && valueIsAlias;
        });
      }
      return false;
    }
    currentScope = currentScope.upper;
  }
  return false;
}
