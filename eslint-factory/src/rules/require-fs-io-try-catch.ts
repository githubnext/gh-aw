import { ESLintUtils } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, createFsSyncMethodResolver, findEnclosingStatement, isInsideTryBlock } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// fs methods beyond readFileSync/writeFileSync/appendFileSync that throw on I/O failure
// and appear frequently unguarded in actions/setup/js.
const FS_IO_METHODS = new Set(["statSync", "readdirSync", "copyFileSync", "unlinkSync", "renameSync"]);

export const requireFsIoTryCatchRule = createRule({
  name: "require-fs-io-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require fs.statSync, fs.readdirSync, fs.copyFileSync, fs.unlinkSync, and fs.renameSync calls in actions/setup/js scripts to be wrapped in try/catch. " +
        "These methods throw synchronously on missing files, permission errors, and other I/O failures; " +
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
    const resolveFsIoMethod = createFsSyncMethodResolver(sourceCode, FS_IO_METHODS, { allowUnboundFsIdentifier: true });

    return {
      CallExpression(node) {
        const methodName = resolveFsIoMethod(node);

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
