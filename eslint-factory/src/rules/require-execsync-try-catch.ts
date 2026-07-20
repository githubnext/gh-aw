import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, findEnclosingStatement, isDeferredCallback, isInsideTryBlock } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const CHILD_PROCESS_SPECIFIERS = new Set(["child_process", "node:child_process"]);

type SourceCodeScope = ReturnType<TSESLint.SourceCode["getScope"]>;

function isRequireChildProcess(node: TSESTree.Node | null | undefined): boolean {
  if (!node) return false;
  return (
    node.type === AST_NODE_TYPES.CallExpression &&
    node.callee.type === AST_NODE_TYPES.Identifier &&
    node.callee.name === "require" &&
    node.arguments.length >= 1 &&
    node.arguments[0].type === AST_NODE_TYPES.Literal &&
    typeof (node.arguments[0] as TSESTree.Literal).value === "string" &&
    CHILD_PROCESS_SPECIFIERS.has((node.arguments[0] as TSESTree.Literal).value as string)
  );
}

function isChildProcessImportBinding(def: { type: string; node: TSESTree.Node; parent?: TSESTree.Node | null }): boolean {
  if (def.type !== "ImportBinding") return false;
  if (!def.parent || def.parent.type !== AST_NODE_TYPES.ImportDeclaration) return false;
  if (def.parent.source.type !== AST_NODE_TYPES.Literal) return false;
  return typeof def.parent.source.value === "string" && CHILD_PROCESS_SPECIFIERS.has(def.parent.source.value);
}

/**
 * Walks the scope chain to decide whether `identifierName` resolves to
 * `execSync` from `child_process`.
 */
function isExecSyncBinding(identifierName: string, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode): boolean {
  let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
  while (scope) {
    const variable = scope.set.get(identifierName);
    if (variable && variable.defs.length > 0) {
      for (const def of variable.defs) {
        // ESM: import { execSync } from "child_process"
        if (isChildProcessImportBinding(def) && def.node.type === AST_NODE_TYPES.ImportSpecifier) {
          const specifier = def.node as TSESTree.ImportSpecifier;
          const importedName = specifier.imported.type === AST_NODE_TYPES.Identifier ? specifier.imported.name : null;
          if (importedName === "execSync") return true;
        }
        // CJS: const { execSync } = require("child_process")
        if (def.type === "Variable") {
          const declarator = def.node as TSESTree.VariableDeclarator;
          if (declarator.id.type === AST_NODE_TYPES.ObjectPattern && isRequireChildProcess(declarator.init)) {
            for (const prop of declarator.id.properties) {
              if (prop.type !== AST_NODE_TYPES.Property) continue;
              if (prop.key.type !== AST_NODE_TYPES.Identifier || prop.key.name !== "execSync") continue;
              const boundName = prop.value.type === AST_NODE_TYPES.Identifier ? prop.value.name : null;
              if (boundName === identifierName) return true;
            }
          }
          // const execSync = childProcess.execSync
          if (declarator.id.type === AST_NODE_TYPES.Identifier && declarator.init?.type === AST_NODE_TYPES.MemberExpression) {
            const init = declarator.init;
            if (
              !init.computed &&
              init.object.type === AST_NODE_TYPES.Identifier &&
              isChildProcessObjectBinding(init.object.name, init.object, sourceCode) &&
              init.property.type === AST_NODE_TYPES.Identifier &&
              init.property.name === "execSync"
            ) {
              return true;
            }
          }
        }
      }
      return false;
    }
    scope = scope.upper;
  }
  return false;
}

function isChildProcessObjectBinding(name: string, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode): boolean {
  let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
  while (scope) {
    const variable = scope.set.get(name);
    if (variable && variable.defs.length > 0) {
      for (const def of variable.defs) {
        if (def.type === "Variable") {
          const declarator = def.node as TSESTree.VariableDeclarator;
          if (declarator.id.type === AST_NODE_TYPES.Identifier && isRequireChildProcess(declarator.init)) {
            return true;
          }
        }
        if (isChildProcessImportBinding(def) && def.node.type === AST_NODE_TYPES.ImportNamespaceSpecifier) {
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
 * Returns true if the CallExpression is an `execSync(...)` call sourced from
 * the `child_process` module.
 */
function isExecSyncCall(node: TSESTree.CallExpression, sourceCode: TSESLint.SourceCode): boolean {
  const callee = node.callee;

  // execSync(...) — destructured or aliased
  if (callee.type === AST_NODE_TYPES.Identifier) {
    return isExecSyncBinding(callee.name, callee, sourceCode);
  }

  // childProcess.execSync(...) or cp.execSync(...)
  if (callee.type === AST_NODE_TYPES.MemberExpression && !callee.computed && callee.object.type === AST_NODE_TYPES.Identifier && callee.property.type === AST_NODE_TYPES.Identifier && callee.property.name === "execSync") {
    return isChildProcessObjectBinding(callee.object.name, callee.object, sourceCode);
  }

  return false;
}

export const requireExecSyncTryCatchRule = createRule({
  name: "require-execsync-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require execSync calls in actions/setup/js scripts to be wrapped in try/catch. " +
        "execSync throws a ChildProcessError when the child process exits with a non-zero status code or is killed by a signal; " +
        "an unhandled throw crashes the action without surfacing a useful diagnostic.",
    },
    schema: [],
    messages: {
      requireTryCatch: "Wrap execSync({{arg}}) in try/catch — execSync throws when the process exits non-zero or is killed by a signal, " + "and will crash the action if the error is unhandled.",
      wrapInTryCatch: "Wrap in try { ... } catch { ... } and re-throw with { cause: err } to preserve context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node) {
        if (!isExecSyncCall(node, sourceCode)) return;
        if (isInsideTryBlock(sourceCode, node)) return;

        // Ignore execSync inside deferred callbacks — the parent try block does not protect them.
        // isInsideTryBlock already handles this, but we skip reporting when the node itself is
        // inside a deferred callback that has no enclosing try block (same FP-avoidance as other rules).
        const ancestors = sourceCode.getAncestors(node);
        let withinDeferredBoundary = false;
        for (let i = ancestors.length - 1; i >= 0; i--) {
          if (isDeferredCallback(ancestors[i])) {
            withinDeferredBoundary = true;
            break;
          }
        }
        // Still flag it even in deferred callbacks — execSync in async callbacks is still risky.
        void withinDeferredBoundary;

        const argText = node.arguments.length > 0 ? sourceCode.getText(node.arguments[0]) : "";
        const stmt = findEnclosingStatement(sourceCode, node);

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
                        todoComment: "TODO: handle execSync failure (non-zero exit / signal termination).",
                        errorPrefix: "execSync failed: ",
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
