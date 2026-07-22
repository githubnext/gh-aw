import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, findEnclosingStatement, isChildProcessImportBinding, isChildProcessObjectBinding, isInsideTryBlock, isRequireChildProcess } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCodeScope = ReturnType<TSESLint.SourceCode["getScope"]>;

/**
 * Walks the scope chain to decide whether `identifierName` resolves to
 * `execFileSync` from `child_process`.
 */
function isExecFileSyncBinding(identifierName: string, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode): boolean {
  let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
  while (scope) {
    const variable = scope.set.get(identifierName);
    if (variable && variable.defs.length > 0) {
      for (const def of variable.defs) {
        // ESM: import { execFileSync } from "child_process"
        if (isChildProcessImportBinding(def) && def.node.type === AST_NODE_TYPES.ImportSpecifier) {
          const specifier = def.node as TSESTree.ImportSpecifier;
          const importedName = specifier.imported.type === AST_NODE_TYPES.Identifier ? specifier.imported.name : null;
          if (importedName === "execFileSync") return true;
        }
        // CJS: const { execFileSync } = require("child_process")
        if (def.type === "Variable") {
          const declarator = def.node as TSESTree.VariableDeclarator;
          if (declarator.id.type === AST_NODE_TYPES.ObjectPattern && isRequireChildProcess(declarator.init)) {
            for (const prop of declarator.id.properties) {
              if (prop.type !== AST_NODE_TYPES.Property) continue;
              if (prop.key.type !== AST_NODE_TYPES.Identifier || prop.key.name !== "execFileSync") continue;
              const boundName = prop.value.type === AST_NODE_TYPES.Identifier ? prop.value.name : null;
              if (boundName === identifierName) return true;
            }
          }
          // const execFileSync = childProcess.execFileSync
          if (declarator.id.type === AST_NODE_TYPES.Identifier && declarator.init?.type === AST_NODE_TYPES.MemberExpression) {
            const init = declarator.init;
            if (
              !init.computed &&
              init.object.type === AST_NODE_TYPES.Identifier &&
              isChildProcessObjectBinding(init.object.name, init.object, sourceCode) &&
              init.property.type === AST_NODE_TYPES.Identifier &&
              init.property.name === "execFileSync"
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

/**
 * Returns true if the CallExpression is an `execFileSync(...)` call sourced from
 * the `child_process` module.
 */
function isExecFileSyncCall(node: TSESTree.CallExpression, sourceCode: TSESLint.SourceCode): boolean {
  const callee = node.callee;

  // execFileSync(...) — destructured or aliased
  if (callee.type === AST_NODE_TYPES.Identifier) {
    return isExecFileSyncBinding(callee.name, callee, sourceCode);
  }

  // childProcess.execFileSync(...) or cp.execFileSync(...)
  if (callee.type === AST_NODE_TYPES.MemberExpression && !callee.computed && callee.object.type === AST_NODE_TYPES.Identifier && callee.property.type === AST_NODE_TYPES.Identifier && callee.property.name === "execFileSync") {
    return isChildProcessObjectBinding(callee.object.name, callee.object, sourceCode);
  }

  return false;
}

export const requireExecFileSyncTryCatchRule = createRule({
  name: "require-execfilesync-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require execFileSync calls in actions/setup/js scripts to be wrapped in try/catch. " +
        "execFileSync throws an Error containing child-process result fields when the child process exits with a non-zero status code or is killed by a signal; " +
        "an unhandled throw crashes the action without surfacing a useful diagnostic.",
    },
    schema: [],
    messages: {
      requireTryCatch: "Wrap execFileSync({{arg}}) in try/catch — execFileSync throws when the process exits non-zero or is killed by a signal, " + "and will crash the action if the error is unhandled.",
      wrapInTryCatch: "Wrap in try { ... } catch { ... } and re-throw with { cause: err } to preserve context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node) {
        if (!isExecFileSyncCall(node, sourceCode)) return;
        if (isInsideTryBlock(sourceCode, node)) return;

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
                        todoComment: "TODO: handle execFileSync failure (non-zero exit / signal termination).",
                        errorPrefix: "execFileSync failed: ",
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
