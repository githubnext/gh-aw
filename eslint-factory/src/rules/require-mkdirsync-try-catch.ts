import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, isDeferredCallback, SAFE_WRAPPABLE_STATEMENT_TYPES } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// fs module specifiers recognised as the Node.js built-in file system module.
const FS_MODULE_SPECIFIERS = new Set(["fs", "node:fs"]);

export const requireMkdirSyncTryCatchRule = createRule({
  name: "require-mkdirsync-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require fs.mkdirSync calls in actions/setup/js scripts to be wrapped in try/catch. " +
        "mkdirSync throws synchronously on permission errors, invalid paths, or unexpected filesystem state; " +
        "an unhandled throw crashes the action without surfacing a useful diagnostic.",
    },
    schema: [],
    messages: {
      requireTryCatch: "Wrap fs.mkdirSync({{arg}}) in try/catch — mkdirSync throws on permission denied, invalid path, or filesystem errors and will crash the action if unhandled.",
      wrapInTryCatch: "Wrap in try { ... } catch { ... } and re-throw with { cause: err } to preserve context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

    function isInsideTryBlock(node: TSESTree.Node): boolean {
      const ancestors = sourceCode.getAncestors(node);
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

    function isFsImportBinding(definition: { type: string; node: TSESTree.Node; parent?: TSESTree.Node | null }): boolean {
      if (definition.type !== "ImportBinding") return false;
      if (!definition.parent || definition.parent.type !== AST_NODE_TYPES.ImportDeclaration) return false;
      if (definition.parent.source.type !== AST_NODE_TYPES.Literal) return false;
      return FS_MODULE_SPECIFIERS.has(definition.parent.source.value as string);
    }

    function isMkdirSyncImportBinding(definition: { type: string; node: TSESTree.Node; parent?: TSESTree.Node | null }): boolean {
      return isFsImportBinding(definition) && definition.node.type === AST_NODE_TYPES.ImportSpecifier && definition.node.imported.type === AST_NODE_TYPES.Identifier && definition.node.imported.name === "mkdirSync";
    }

    function isIdentifierBoundToFsModule(identifierName: string, scopeNode: TSESTree.Node): boolean {
      let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
      while (scope) {
        const variable = scope.set.get(identifierName);
        if (variable && variable.defs.length > 0) {
          for (const def of variable.defs) {
            if (isFsImportBinding(def)) {
              return true;
            }
            if (def.type !== "Variable") continue;
            const declarator = def.node as TSESTree.VariableDeclarator;
            if (declarator.id.type === AST_NODE_TYPES.Identifier && isRequireFsCall(declarator.init)) {
              return true;
            }
          }
          return false;
        }
        scope = scope.upper;
      }
      return false;
    }

    /**
     * Resolves the callee identifier to "mkdirSync" if it is bound to fs.mkdirSync
     * via destructuring or aliasing.
     */
    function resolveMkdirSyncFromIdentifier(node: TSESTree.CallExpression): boolean {
      const callee = node.callee;
      if (callee.type !== AST_NODE_TYPES.Identifier) return false;

      let scope: SourceCodeScope | null = sourceCode.getScope(node);
      while (scope) {
        const variable = scope.set.get(callee.name);
        if (variable && variable.defs.length > 0) {
          for (const def of variable.defs) {
            if (isMkdirSyncImportBinding(def)) {
              return true;
            }
            if (isFsImportBinding(def)) {
              continue;
            }
            if (def.type !== "Variable") continue;
            const declarator = def.node as TSESTree.VariableDeclarator;

            // Shape: `const { mkdirSync } = require("fs")`
            if (declarator.id.type === AST_NODE_TYPES.ObjectPattern && isRequireFsCall(declarator.init)) {
              for (const prop of declarator.id.properties) {
                if (prop.type !== AST_NODE_TYPES.Property) continue;
                if (prop.key.type !== AST_NODE_TYPES.Identifier || prop.key.name !== "mkdirSync") continue;
                const boundName = prop.value.type === AST_NODE_TYPES.Identifier ? prop.value.name : null;
                if (boundName === callee.name) return true;
              }
            }

            // Shape: `const mkdirSync = fs.mkdirSync`
            if (declarator.id.type === AST_NODE_TYPES.Identifier && declarator.init?.type === AST_NODE_TYPES.MemberExpression) {
              const init = declarator.init;
              if (init.object.type === AST_NODE_TYPES.Identifier && isIdentifierBoundToFsModule(init.object.name, init.object) && !init.computed && init.property.type === AST_NODE_TYPES.Identifier && init.property.name === "mkdirSync") {
                return true;
              }
            }
          }
          return false;
        }
        scope = scope.upper;
      }
      return false;
    }

    function isMkdirSyncCall(node: TSESTree.CallExpression): boolean {
      const callee = node.callee;

      if (callee.type === AST_NODE_TYPES.MemberExpression) {
        if (callee.object.type !== AST_NODE_TYPES.Identifier) return false;
        if (!isIdentifierBoundToFsModule(callee.object.name, callee.object)) return false;

        if (!callee.computed && callee.property.type === AST_NODE_TYPES.Identifier && callee.property.name === "mkdirSync") return true;
        if (callee.computed && callee.property.type === AST_NODE_TYPES.Literal && callee.property.value === "mkdirSync") return true;

        return false;
      }

      if (callee.type === AST_NODE_TYPES.Identifier) {
        return resolveMkdirSyncFromIdentifier(node);
      }

      return false;
    }

    return {
      CallExpression(node) {
        if (!isMkdirSyncCall(node)) return;
        if (isInsideTryBlock(node)) return;

        const argText = node.arguments.length > 0 ? sourceCode.getText(node.arguments[0]) : "";
        const stmt = findEnclosingStatement(node);

        context.report({
          node,
          messageId: "requireTryCatch",
          data: { arg: argText },
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
                        todoComment: "TODO: handle filesystem failure for this fs.mkdirSync call.",
                        errorPrefix: "fs.mkdirSync failed: ",
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
