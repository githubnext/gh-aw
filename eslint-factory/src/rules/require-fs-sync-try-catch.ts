import { ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, createFsSyncMethodResolver, findEnclosingStatement, isInsideTryBlock } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// fs module methods that throw on I/O failure and are in scope for this rule.
// readFileSync / writeFileSync / appendFileSync are the highest-frequency, highest-risk callers
// in actions/setup/js. Other sync methods (mkdirSync, unlinkSync, …) are left out of scope for
// now to keep FP risk low on the first iteration.
const FS_SYNC_METHODS = new Set(["readFileSync", "writeFileSync", "appendFileSync"]);

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
    const resolveFsSyncMethod = createFsSyncMethodResolver(sourceCode, FS_SYNC_METHODS, { allowUnboundFsIdentifier: true });

    return {
      CallExpression(node) {
        const methodName = resolveFsSyncMethod(node);

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
