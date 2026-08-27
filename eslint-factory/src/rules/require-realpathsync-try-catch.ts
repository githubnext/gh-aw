import { ESLintUtils } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, createFsSyncMethodResolver, findEnclosingStatement, isInsideTryBlock } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const FS_SYNC_METHODS = new Set(["realpathSync"]);

export const requireRealpathSyncTryCatchRule = createRule({
  name: "require-realpathsync-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require fs.realpathSync calls in actions/setup/js scripts to be wrapped in try/catch. " +
        "realpathSync is frequently used to canonicalize a path before a path-traversal or symlink-escape check " +
        "(ENOENT when the target does not exist, EACCES on permission errors, ELOOP on symlink cycles). Without a " +
        "call-site try/catch, the security check it feeds is skipped entirely and the failure surfaces as a generic " +
        "engine-level stack instead of a specific message with `{ cause }` that names the path that failed to resolve.",
    },
    schema: [],
    messages: {
      requireTryCatch: "Wrap fs.realpathSync({{arg}}) in try/catch — realpathSync throws on a missing target, permission errors, or symlink cycles, and callers typically use its result for a path-traversal/symlink-escape check that should fail closed.",
      wrapInTryCatch: "Wrap in try { ... } catch { ... } and re-throw with { cause: err } to preserve context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    const resolveFsSyncMethod = createFsSyncMethodResolver(sourceCode, FS_SYNC_METHODS, { allowUnboundFsIdentifier: true });

    return {
      CallExpression(node) {
        const methodName = resolveFsSyncMethod(node);
        if (methodName !== "realpathSync") return;
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
                        todoComment: "TODO: handle filesystem failure for this fs.realpathSync call.",
                        errorPrefix: "fs.realpathSync failed: ",
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
