import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, isDeferredCallback, SAFE_WRAPPABLE_STATEMENT_TYPES } from "./try-catch-rule-utils";

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

        if (ancestor.type === "TryStatement" && !crossedDeferredBoundary) {
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

    return {
      CallExpression(node) {
        const callee = node.callee;
        if (callee.type !== AST_NODE_TYPES.MemberExpression) return;

        // Object must be the `fs` identifier (the standard import alias in actions/setup/js).
        // Aliased references (const r = fs.readFileSync; r(path)) are intentionally out of scope.
        if (callee.object.type !== AST_NODE_TYPES.Identifier || callee.object.name !== "fs") return;

        // Accept both direct property access (fs.readFileSync) and computed string-literal access
        // (fs["readFileSync"]). Dynamic computed access (fs[varName]) is excluded.
        const property = callee.property;
        let methodName: string | null = null;
        if (!callee.computed && property.type === AST_NODE_TYPES.Identifier && FS_SYNC_METHODS.has(property.name)) {
          methodName = property.name;
        } else if (callee.computed && property.type === AST_NODE_TYPES.Literal && typeof property.value === "string" && FS_SYNC_METHODS.has(property.value)) {
          methodName = property.value;
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
