import { ESLintUtils } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, createFsSyncMethodResolver, findEnclosingStatement, isInsideTryBlock } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const FS_SYNC_METHODS = new Set(["mkdirSync"]);

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
    const resolveFsSyncMethod = createFsSyncMethodResolver(sourceCode, FS_SYNC_METHODS);

    return {
      CallExpression(node) {
        const methodName = resolveFsSyncMethod(node);
        if (methodName !== "mkdirSync") return;
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
