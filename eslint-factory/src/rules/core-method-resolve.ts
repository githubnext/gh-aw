import { AST_NODE_TYPES, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { CORE_ALIASES } from "./core-aliases";

/**
 * Matches `@param {typeof import('@actions/core')} <name>` (single or double quotes)
 * in a JSDoc block-comment value. Used as an unambiguous signal that a parameter
 * is a dependency-injected `@actions/core`-like object.
 */
const JSDOC_CORE_PARAM_RE = /@param\s*\{typeof\s+import\(['"]@actions\/core['"]\)\}\s+([$\w]+)/g;

/**
 * Returns true when the enclosing function carries a JSDoc block comment with
 * `@param {typeof import('@actions/core')} <paramName>`.
 * For function expressions / arrow functions assigned to a variable the JSDoc is
 * typically before the VariableDeclaration, not the function node itself, so both
 * positions are checked.
 */
function hasJSDocCoreParamAnnotation(functionNode: TSESTree.Node, paramName: string, sourceCode: TSESLint.SourceCode): boolean {
  const nodesToCheck: TSESTree.Node[] = [functionNode];
  const parent = functionNode.parent;
  if (parent?.type === AST_NODE_TYPES.VariableDeclarator && parent.parent) {
    nodesToCheck.push(parent.parent);
  }
  // JSDoc before `export function` / `export async function` / `export default function`
  // is attached to the ExportNamedDeclaration/ExportDefaultDeclaration, not the inner
  // FunctionDeclaration/FunctionExpression, so check the export wrapper as well.
  if (parent?.type === AST_NODE_TYPES.ExportNamedDeclaration || parent?.type === AST_NODE_TYPES.ExportDefaultDeclaration) {
    nodesToCheck.push(parent);
  }
  for (const node of nodesToCheck) {
    for (const comment of sourceCode.getCommentsBefore(node)) {
      if (comment.type !== "Block" || !comment.value.startsWith("*")) continue;
      JSDOC_CORE_PARAM_RE.lastIndex = 0;
      let m: RegExpExecArray | null;
      while ((m = JSDOC_CORE_PARAM_RE.exec(comment.value)) !== null) {
        if (m[1] === paramName) return true;
      }
    }
  }
  return false;
}

/**
 * Returns true when `identifier` resolves (via scope chain) to a function parameter
 * that carries a JSDoc `@param {typeof import('@actions/core')}` annotation.
 * Used as the slow-path for DI parameters not covered by CORE_ALIASES.
 */
function isJSDocCoreParamInScope(identifier: TSESTree.Identifier, sourceCode: TSESLint.SourceCode): boolean {
  let currentScope: TSESLint.Scope.Scope | null = sourceCode.getScope(identifier);
  while (currentScope !== null) {
    const variable = currentScope.set.get(identifier.name);
    if (variable !== undefined) {
      if (variable.defs.length !== 1) return false;
      const def = variable.defs[0];
      if (def.type !== "Parameter" || def.name.type !== AST_NODE_TYPES.Identifier) return false;
      return hasJSDocCoreParamAnnotation(def.node, identifier.name, sourceCode);
    }
    currentScope = currentScope.upper;
  }
  return false;
}

/**
 * Checks whether an Identifier is a single-assignment alias for a core-like
 * object (e.g., `const c = core`). Re-assigned let bindings are rejected.
 * Local shadows (e.g., a parameter also named `c`) are excluded because they
 * are found first in the scope chain and their definition type will not match.
 *
 * Additionally, a plain function parameter is accepted to support the
 * dependency-injection pattern:
 *   `async function f(core) { core.setFailed(msg); }`
 * Two gating strategies prevent arbitrary parameters from being treated as core:
 *   1. Fast path — parameter name is in CORE_ALIASES (exact known aliases).
 *   2. Slow path — enclosing function has a JSDoc `@param {typeof import('@actions/core')}`
 *      annotation for this parameter (strong, unambiguous signal).
 * Destructured parameters (e.g. `{ core }`) are excluded in both paths.
 */
export function isCoreAliasIdentifier(identifier: TSESTree.Identifier, sourceCode: TSESLint.SourceCode): boolean {
  let currentScope: TSESLint.Scope.Scope | null = sourceCode.getScope(identifier);
  while (currentScope !== null) {
    const variable = currentScope.set.get(identifier.name);
    if (variable !== undefined) {
      if (variable.defs.length !== 1) return false;
      const def = variable.defs[0];
      if (def.type === "Parameter") {
        // Only plain (non-destructured) parameters are accepted.
        if (def.name.type !== AST_NODE_TYPES.Identifier) return false;
        // Fast path: known exact alias names (e.g. `core`, `coreObj`).
        if (CORE_ALIASES.has(identifier.name)) return true;
        // Slow path: JSDoc @param {typeof import('@actions/core')} annotation on
        // the enclosing function is an unambiguous signal for DI-style parameters
        // with non-canonical names (e.g. `coreArg`, `coreLib`).
        return hasJSDocCoreParamAnnotation(def.node, identifier.name, sourceCode);
      }
      if (def.type !== "Variable") return false;
      if (variable.references.some(ref => ref.isWrite() && !ref.init)) return false;
      const declarator = def.node as TSESTree.VariableDeclarator;
      if (!declarator.init) return false;
      return declarator.id.type === AST_NODE_TYPES.Identifier && declarator.init.type === AST_NODE_TYPES.Identifier && (CORE_ALIASES.has(declarator.init.name) || isJSDocCoreParamInScope(declarator.init as TSESTree.Identifier, sourceCode));
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
      if (declarator.id.type === AST_NODE_TYPES.ObjectPattern && declarator.init.type === AST_NODE_TYPES.Identifier && (CORE_ALIASES.has(declarator.init.name) || isJSDocCoreParamInScope(declarator.init, sourceCode))) {
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
