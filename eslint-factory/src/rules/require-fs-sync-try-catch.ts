import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, isDeferredCallback, SAFE_WRAPPABLE_STATEMENT_TYPES } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// fs module methods that throw on I/O failure and are in scope for this rule.
// readFileSync / writeFileSync / appendFileSync are the highest-frequency, highest-risk callers
// in actions/setup/js. Other sync methods (mkdirSync, unlinkSync, …) are left out of scope for
// now to keep FP risk low on the first iteration.
const FS_SYNC_METHODS = new Set(["readFileSync", "writeFileSync", "appendFileSync"]);

// fs module specifiers recognised as the Node.js built-in file system module.
const FS_MODULE_SPECIFIERS = new Set(["fs", "node:fs"]);

export const requireFsSyncTryCatchRule = createRule({
  name: "require-fs-sync-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require fs.readFileSync, fs.writeFileSync, and fs.appendFileSync calls in actions/setup/js scripts to be wrapped in try/catch. " +
        "These methods throw synchronously on missing files, permission errors, and disk failures; " +
        "an unhandled throw crashes the action without surfacing a useful error message.",
    },
    schema: [],
    messages: {
      requireTryCatch: "Wrap fs.{{method}}({{arg}}) in try/catch — synchronous fs methods throw on I/O errors " + "(missing file, permission denied, disk full) and will crash the action if unhandled.",
      wrapInTryCatch: "Wrap in try { ... } catch { ... } and re-throw with { cause: err } to preserve context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

    function isInsideTryBlock(node: TSESTree.Node): boolean {
      const ancestors = sourceCode.getAncestors(node);
      // Walk from innermost ancestor outward. Stop marking a try as protective once a deferred
      // callback boundary has been crossed (the callback fires after the try has returned).
      let crossedDeferredBoundary = false;

      for (let i = ancestors.length - 1; i >= 0; i--) {
        const ancestor = ancestors[i];

        if (isDeferredCallback(ancestor)) {
          crossedDeferredBoundary = true;
        }

        if (ancestor.type === "TryStatement" && !crossedDeferredBoundary && ancestor.handler != null) {
          const block = ancestor.block;
          if (node.range != null && block.range != null && node.range[0] >= block.range[0] && node.range[1] <= block.range[1]) {
            return true;
          }
        }
      }

      return false;
    }

    function findEnclosingStatement(node: TSESTree.Node): TSESTree.Statement | null {
      const ancestors = sourceCode.getAncestors(node);
      for (let i = ancestors.length - 1; i >= 0; i--) {
        const ancestor = ancestors[i];
        if (SAFE_WRAPPABLE_STATEMENT_TYPES.has(ancestor.type)) {
          return ancestor as TSESTree.Statement;
        }
      }
      return null;
    }

    /**
     * Returns true if `node` is a `require("fs")` or `require("node:fs")` call.
     */
    function isRequireFsCall(node: TSESTree.Node | null | undefined): boolean {
      if (!node) return false;
      return (
        node.type === AST_NODE_TYPES.CallExpression &&
        node.callee.type === AST_NODE_TYPES.Identifier &&
        node.callee.name === "require" &&
        node.arguments.length >= 1 &&
        node.arguments[0].type === AST_NODE_TYPES.Literal &&
        FS_MODULE_SPECIFIERS.has(node.arguments[0].value as string)
      );
    }

    /**
     * Resolves the original fs sync method name for an Identifier callee via scope analysis.
     * Handles two binding shapes:
     *   1. Destructured: `const { appendFileSync } = require("fs")`
     *              or   `const { appendFileSync: alias } = require("fs")`
     *   2. Member-expression alias: `const alias = fs.appendFileSync`
     *
     * Returns the canonical method name (e.g. "appendFileSync") or null if the identifier
     * does not trace back to an in-scope fs sync method.
     */
    function resolveFsSyncMethodFromIdentifier(node: TSESTree.CallExpression): string | null {
      const callee = node.callee;
      if (callee.type !== AST_NODE_TYPES.Identifier) return null;

      let scope: SourceCodeScope | null = sourceCode.getScope(node);
      while (scope) {
        const variable = scope.set.get(callee.name);
        if (variable && variable.defs.length > 0) {
          for (const def of variable.defs) {
            if (def.type !== "Variable") continue;
            const declarator = def.node as TSESTree.VariableDeclarator;

            // Shape 1: `const { appendFileSync } = require("fs")`
            //       or `const { appendFileSync: alias } = require("fs")`
            if (declarator.id.type === AST_NODE_TYPES.ObjectPattern && isRequireFsCall(declarator.init)) {
              for (const prop of declarator.id.properties) {
                if (prop.type !== AST_NODE_TYPES.Property) continue;
                if (prop.key.type !== AST_NODE_TYPES.Identifier) continue;
                if (!FS_SYNC_METHODS.has(prop.key.name)) continue;
                // prop.value is the bound identifier (same as key for shorthand)
                const boundName = prop.value.type === AST_NODE_TYPES.Identifier ? prop.value.name : null;
                if (boundName === callee.name) {
                  return prop.key.name;
                }
              }
            }

            // Shape 2: `const alias = fs.appendFileSync`
            // Only matches when the `fs` identifier is itself bound to require("fs") / require("node:fs").
            if (declarator.id.type === AST_NODE_TYPES.Identifier && declarator.init?.type === AST_NODE_TYPES.MemberExpression) {
              const init = declarator.init;
              if (init.object.type === AST_NODE_TYPES.Identifier && isIdentifierBoundToFsModule(init.object.name, init.object)) {
                const methodName = getFsSyncMethodFromProperty(init);
                if (methodName !== null) return methodName;
              }
            }
          }
          // Variable is locally defined but not as an fs sync method binding — stop searching.
          return null;
        }
        scope = scope.upper;
      }
      return null;
    }

    /**
     * Returns the fs sync method name from a MemberExpression property, or null if the property
     * is not one of the in-scope fs sync methods. Handles both direct and computed string-literal access.
     */
    function getFsSyncMethodFromProperty(memberExpr: TSESTree.MemberExpression): string | null {
      const property = memberExpr.property;
      if (!memberExpr.computed && property.type === AST_NODE_TYPES.Identifier && FS_SYNC_METHODS.has(property.name)) {
        return property.name;
      }
      if (memberExpr.computed && property.type === AST_NODE_TYPES.Literal && typeof property.value === "string" && FS_SYNC_METHODS.has(property.value)) {
        return property.value;
      }
      return null;
    }

    /**
     * Returns true if the named identifier is bound to `require("fs")` or `require("node:fs")`
     * anywhere in the scope chain visible from `scopeNode`.
     */
    function isIdentifierBoundToFsModule(identifierName: string, scopeNode: TSESTree.Node): boolean {
      let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
      while (scope) {
        const variable = scope.set.get(identifierName);
        if (variable && variable.defs.length > 0) {
          for (const def of variable.defs) {
            if (def.type !== "Variable") continue;
            const declarator = def.node as TSESTree.VariableDeclarator;
            if (declarator.id.type === AST_NODE_TYPES.Identifier && isRequireFsCall(declarator.init)) {
              return true;
            }
          }
          return false; // Identifier found but not bound to require("fs")
        }
        scope = scope.upper;
      }
      return false;
    }

    return {
      CallExpression(node) {
        const callee = node.callee;
        let methodName: string | null = null;

        if (callee.type === AST_NODE_TYPES.MemberExpression) {
          // Object may be `fs` or any identifier bound to require("fs") / require("node:fs").
          if (callee.object.type !== AST_NODE_TYPES.Identifier) return;
          if (callee.object.name !== "fs" && !isIdentifierBoundToFsModule(callee.object.name, callee.object)) return;

          // Accept both direct property access (fs.readFileSync) and computed string-literal access
          // (fs["readFileSync"]). Dynamic computed access (fs[varName]) is excluded.
          methodName = getFsSyncMethodFromProperty(callee);
        } else if (callee.type === AST_NODE_TYPES.Identifier) {
          // Destructured or aliased fs binding — resolve via scope analysis.
          methodName = resolveFsSyncMethodFromIdentifier(node);
        }

        if (!methodName) return;

        if (isInsideTryBlock(node)) return;

        const argText = node.arguments.length > 0 ? sourceCode.getText(node.arguments[0]) : "";
        const method = methodName;
        const stmt = findEnclosingStatement(node);

        context.report({
          node,
          messageId: "requireTryCatch",
          data: { method, arg: argText },
          suggest: stmt
            ? [
                {
                  messageId: "wrapInTryCatch",
                  fix(fixer) {
                    const stmtText = sourceCode.getText(stmt);
                    const startLine = stmt.loc?.start.line;
                    const stmtLine = startLine !== undefined ? (sourceCode.lines[startLine - 1] ?? "") : "";
                    const indent = stmtLine.match(/^(\s*)/)?.[1] ?? "";
                    return fixer.replaceText(
                      stmt,
                      buildTryCatchSuggestion(stmtText, {
                        indent,
                        todoComment: `TODO: handle I/O failure for this fs.${method} call.`,
                        errorPrefix: `fs.${method} failed: `,
                      })
                    );
                  },
                },
              ]
            : [],
        });
      },
    };
  },
});
