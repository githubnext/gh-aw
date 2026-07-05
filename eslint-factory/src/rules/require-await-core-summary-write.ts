import { ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const ASYNC_FUNCTION_TYPES = new Set(["FunctionDeclaration", "FunctionExpression", "ArrowFunctionExpression"]);

/**
 * Checks whether a MemberExpression property is "write" (direct or computed string-literal access).
 */
function isWriteProperty(node: TSESTree.MemberExpression): boolean {
  const property = node.property;
  const isDirectAccess = !node.computed && property.type === "Identifier" && property.name === "write";
  const isComputedAccess = node.computed && property.type === "Literal" && property.value === "write";
  return isDirectAccess || isComputedAccess;
}

function getRootIdentifier(node: TSESTree.Node): string | null {
  if (node.type === "Identifier") return node.name;
  if (node.type === "MemberExpression") return getRootIdentifier(node.object);
  if (node.type === "CallExpression") return getRootIdentifier(node.callee);
  return null;
}

// Tightened from the prior `/^core/i` heuristic to an exact allow-list.
// Only identifiers that are verified @actions/core bindings in the corpus are
// matched; broad prefix matching would silently flag unrelated objects such as
// `coreCache`, `coreData`, or `coreference` that happen to start with "core".
//
// Known aliases (extend here when a new verified binding is introduced):
//   "core"    — conventional require("@actions/core") name in github-script steps
//   "coreObj" — alias used in parse_mcp_gateway_log.cjs
const CORE_ALIASES = new Set(["core", "coreObj"]);

function isCoreLikeIdentifier(name: string): boolean {
  return CORE_ALIASES.has(name);
}

/**
 * Checks whether a node is rooted in a `.summary` member access, possibly through
 * a chain of method calls (e.g., `core.summary.addRaw(x).write()`).
 *
 * Accepted patterns (non-exhaustive):
 *   - `core.summary`
 *   - `core.summary.addRaw(x)`
 *   - `core.summary.addHeading(...).addRaw(x)`
 *   - `coreObj.summary` (known @actions/core alias — see CORE_ALIASES)
 */
function rootsSummary(node: TSESTree.Node): boolean {
  const rootIdentifier = getRootIdentifier(node);
  if (!rootIdentifier || !isCoreLikeIdentifier(rootIdentifier)) return false;
  if (node.type === "MemberExpression") {
    const property = node.property;
    const isSummaryProp = (!node.computed && property.type === "Identifier" && property.name === "summary") || (node.computed && property.type === "Literal" && property.value === "summary");
    if (isSummaryProp) return true;
  }
  if (node.type === "CallExpression" && node.callee.type === "MemberExpression") {
    return rootsSummary(node.callee.object);
  }
  return false;
}

/**
 * Returns true when the statement is directly inside an async function body.
 * Walking up the ancestors, the first function boundary found determines
 * whether `await` is currently valid — the suggestion is only safe to apply
 * in an async context.
 */
function isInsideAsyncFunction(ancestors: TSESTree.Node[]): boolean {
  for (let i = ancestors.length - 1; i >= 0; i--) {
    const ancestor = ancestors[i];
    if (ASYNC_FUNCTION_TYPES.has(ancestor.type)) {
      return (ancestor as TSESTree.FunctionDeclaration | TSESTree.FunctionExpression | TSESTree.ArrowFunctionExpression).async;
    }
  }
  return false;
}

export const requireAwaitCoreSummaryWriteRule = createRule({
  name: "require-await-core-summary-write",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require core.summary.write() calls to be awaited; the returned Promise<Summary> is silently discarded when called without await, which can truncate or drop the step summary if the process exits before the microtask queue drains.",
    },
    schema: [],
    messages: {
      requireAwait: "core.summary.write() returns a Promise<Summary> that must be awaited; omitting await silently discards the promise and can cause the step summary to be truncated or missing.",
      addAwait: "Insert 'await' before the expression.",
    },
  },
  defaultOptions: [],
  create(context) {
    /**
     * Checks whether an Identifier in the current scope is a single-assignment
     * local alias for `core.summary` (or any CORE_ALIASES member's `.summary`).
     *
     * Accepted initializer patterns:
     *   - `const/let summaryVar = core.summary;`
     *   - `const { summary } = core;` or `const { summary: alias } = core;`
     *
     * Re-assigned `let` bindings are rejected (conservative: source unknown after
     * reassignment).
     */
    function isCoreSummaryAlias(identifier: TSESTree.Identifier): boolean {
      let currentScope: TSESLint.Scope.Scope | null = context.sourceCode.getScope(identifier);
      while (currentScope !== null) {
        const variable = currentScope.set.get(identifier.name);
        if (variable !== undefined) {
          // Only handle single-declaration bindings (no overloads / duplicate lets).
          if (variable.defs.length !== 1) return false;
          const def = variable.defs[0];

          // Must be a VariableDeclarator (not a function parameter, import, etc.).
          if (def.type !== "Variable") return false;

          // Reject let bindings that are re-assigned after initialisation.
          // ref.init === true marks the write that comes from the initialiser itself.
          if (variable.references.some(ref => ref.isWrite() && !ref.init)) return false;

          const declarator = def.node as TSESTree.VariableDeclarator;
          if (!declarator.init) return false;

          // Pattern 1: const/let summaryVar = core.summary;
          // rootsSummary covers `core.summary` and chained inits like
          // `core.summary.addRaw(x)` — addRaw returns Summary, so the alias
          // still wraps a Summary object.
          // .write() is explicitly rejected: write() returns Promise<Summary>,
          // not Summary, so `const s = core.summary.write()` must not be
          // treated as a Summary alias.
          if (declarator.id.type === "Identifier") {
            if (declarator.init.type === "CallExpression" && declarator.init.callee.type === "MemberExpression" && isWriteProperty(declarator.init.callee)) return false;
            return rootsSummary(declarator.init);
          }

          // Pattern 2: const { summary } = core; or const { summary: alias } = core;
          if (declarator.id.type === "ObjectPattern" && declarator.init.type === "Identifier" && isCoreLikeIdentifier(declarator.init.name)) {
            return declarator.id.properties.some(prop => {
              if (prop.type !== "Property" || prop.computed) return false;
              const keyIsSummary = prop.key.type === "Identifier" && prop.key.name === "summary";
              const valueIsAlias = prop.value.type === "Identifier" && prop.value.name === identifier.name;
              return keyIsSummary && valueIsAlias;
            });
          }

          return false;
        }
        currentScope = currentScope.upper;
      }
      return false;
    }

    /**
     * Extended version of rootsSummary that also handles local aliases of
     * `core.summary` (e.g. `const summary = core.summary; summary.write()`).
     * The depth guard prevents unbounded recursion on pathologically deep
     * call chains (> 32 levels); real source code is well below this limit.
     */
    function rootsSummaryOrAlias(node: TSESTree.Node, depth = 0): boolean {
      if (depth > 32) return false;
      // Fast path: already handles core.summary and chained core.summary.*() calls.
      if (rootsSummary(node)) return true;

      // Direct alias: `summary.write()` — callee.object is an Identifier.
      if (node.type === "Identifier") return isCoreSummaryAlias(node);

      // Chained alias: `summary.addRaw(x).write()` — callee.object is a CallExpression
      // rooted on the alias identifier.
      if (node.type === "CallExpression" && node.callee.type === "MemberExpression") {
        return rootsSummaryOrAlias(node.callee.object, depth + 1);
      }

      return false;
    }

    return {
      ExpressionStatement(node) {
        const expr = node.expression;

        // Only flag bare expression statements — AwaitExpression, ReturnStatement,
        // VariableDeclaration, and AssignmentExpression propagate the Promise to the
        // caller and are not flagged (zero false positives on existing correct uses).
        if (expr.type !== "CallExpression") return;

        const callee = expr.callee;
        if (callee.type !== "MemberExpression") return;

        // Property must be `write` (direct or computed string-literal access)
        if (!isWriteProperty(callee)) return;

        // Object must trace back through a `.summary` member access (or an alias)
        if (!rootsSummaryOrAlias(callee.object)) return;

        // Only offer the `await` suggestion when already inside an async function —
        // applying `await` outside an async context would produce a syntax error.
        const ancestors = context.sourceCode.getAncestors(node);
        const suggest = isInsideAsyncFunction(ancestors)
          ? [
              {
                messageId: "addAwait" as const,
                fix(fixer: TSESLint.RuleFixer) {
                  return fixer.insertTextBefore(expr, "await ");
                },
              },
            ]
          : [];

        context.report({
          node: expr,
          messageId: "requireAwait",
          suggest,
        });
      },
    };
  },
});
