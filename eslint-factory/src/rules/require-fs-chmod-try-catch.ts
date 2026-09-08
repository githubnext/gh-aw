import { ESLintUtils } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, createFsSyncMethodResolver, findEnclosingStatement, isInsideTryBlock } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// fs.chmodSync and fs.fchmodSync throw synchronously on missing files/descriptors, permission
// errors (EPERM), or unsupported filesystems (EINVAL on some mounts). actions/setup/js relies on
// these calls to lock down permissions on files that may contain secrets (config files, staged
// artifacts, temp directories); an unguarded throw here crashes the action with a generic stack
// trace instead of a specific message that preserves the original error as `{ cause }`.
const FS_CHMOD_METHODS = new Set(["chmodSync", "fchmodSync"]);

export const requireFsChmodTryCatchRule = createRule({
  name: "require-fs-chmod-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require fs.chmodSync and fs.fchmodSync calls in actions/setup/js scripts to be wrapped in try/catch. " +
        "These methods throw synchronously on missing files, invalid descriptors, or permission errors; " +
        "without a call-site try/catch, the entrypoint-level catch produces a generic engine-level stack instead of a specific message that preserves the error as `{ cause }`.",
    },
    schema: [],
    messages: {
      requireTryCatch:
        "Wrap fs.{{method}}({{arg}}) in try/catch — synchronous chmod calls throw on missing files, invalid descriptors, " +
        "or permission errors; without a call-site try/catch, you lose the original error context and get a generic engine-level stack instead of a specific message with `{ cause }`.",
      wrapInTryCatch: "Wrap in try { ... } catch { ... } and re-throw with { cause: err } to preserve context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    const resolveFsChmodMethod = createFsSyncMethodResolver(sourceCode, FS_CHMOD_METHODS, { allowUnboundFsIdentifier: true });

    return {
      CallExpression(node) {
        const methodName = resolveFsChmodMethod(node);

        if (!methodName) return;

        if (isInsideTryBlock(sourceCode, node)) return;

        const argText = node.arguments.length > 0 ? sourceCode.getText(node.arguments[0]) : "";
        const method = methodName;
        const stmt = findEnclosingStatement(sourceCode, node);

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
                        todoComment: `TODO: handle permission-change failure for this fs.${method} call.`,
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
