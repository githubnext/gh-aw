import { ESLintUtils } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, createFsSyncMethodResolver, findEnclosingStatement, isInsideTryBlock } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const FS_SYNC_METHODS = new Set(["unlinkSync", "rmdirSync"]);

export const requireUnlinkSyncRmdirSyncTryCatchRule = createRule({
  name: "require-unlinksync-rmdirsync-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require fs.unlinkSync and fs.rmdirSync calls in actions/setup/js scripts to be wrapped in try/catch. " +
        "Both throw synchronously when the target is missing, is the wrong type (file vs directory), or is denied by " +
        "permissions; without a call-site try/catch, the entrypoint-level catch produces a generic engine-level stack " +
        "instead of a specific message that preserves the error as `{ cause }`.",
    },
    schema: [],
    messages: {
      requireTryCatch:
        "Wrap fs.{{method}}({{arg}}) in try/catch — {{method}} throws when the path does not exist, is the wrong type, or permissions are denied; without a call-site try/catch, you lose the original error context and get a generic engine-level stack instead of a specific message with `{ cause }`.",
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
        if (!methodName || !FS_SYNC_METHODS.has(methodName)) return;
        if (isInsideTryBlock(sourceCode, node)) return;

        const argText = node.arguments.length > 0 ? sourceCode.getText(node.arguments[0]) : "";
        const stmt = findEnclosingStatement(sourceCode, node);

        context.report({
          node,
          messageId: "requireTryCatch",
          data: { method: methodName, arg: argText },
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
                        todoComment: `TODO: handle filesystem failure for this fs.${methodName} call.`,
                        errorPrefix: `fs.${methodName} failed: `,
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
