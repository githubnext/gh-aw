import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, isDeferredCallback, SAFE_WRAPPABLE_STATEMENT_TYPES } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Statement types that can be directly wrapped in a try/catch block.
const WRAPPABLE_STATEMENT_TYPES = new Set<AST_NODE_TYPES>([...SAFE_WRAPPABLE_STATEMENT_TYPES, AST_NODE_TYPES.VariableDeclaration]);

export const requireNewUrlTryCatchRule = createRule({
  name: "require-new-url-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require new URL(variable) calls in actions/setup/js scripts to be wrapped in try/catch. " +
        "The URL constructor throws a TypeError when given an invalid or relative URL string, " +
        "which crashes the action with an unhelpful uncaught exception.",
    },
    schema: [],
    messages: {
      requireTryCatch:
        "Wrap new URL({{arg}}) in try/catch — the URL constructor throws TypeError for invalid or relative URLs " +
        "and will crash the action if unhandled.",
      wrapInTryCatch: "Wrap in try { ... } catch { ... } and re-throw with { cause: err } to preserve context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

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
        if (WRAPPABLE_STATEMENT_TYPES.has(ancestor.type)) {
          return ancestor as TSESTree.Statement;
        }
      }
      return null;
    }

    /**
     * Returns true when the first argument to `new URL(...)` is a non-literal expression.
     * String literals are safe because the URL is known at compile time; only variable/dynamic
     * input can be invalid at runtime.
     */
    function isNonLiteralFirstArg(node: TSESTree.NewExpression): boolean {
      const firstArg = node.arguments[0];
      if (!firstArg || firstArg.type === "SpreadElement") return false;
      // Literal strings are compile-time constants — no runtime parse risk.
      if (firstArg.type === AST_NODE_TYPES.Literal && typeof (firstArg as TSESTree.StringLiteral).value === "string") return false;
      // Template literals with no expressions are effectively string constants.
      if (firstArg.type === AST_NODE_TYPES.TemplateLiteral && firstArg.expressions.length === 0) return false;
      return true;
    }

    return {
      NewExpression(node) {
        // Only flag `new URL(...)` — the global URL constructor.
        if (node.callee.type !== AST_NODE_TYPES.Identifier || node.callee.name !== "URL") return;

        // Only flag when the first argument is a non-literal (runtime variable/expression).
        if (!isNonLiteralFirstArg(node)) return;

        if (isInsideTryBlock(node)) return;

        const firstArg = node.arguments[0] as TSESTree.Expression;
        const argText = sourceCode.getText(firstArg);
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
                        todoComment: `TODO: handle invalid URL for this new URL(${argText}) call.`,
                        errorPrefix: `new URL(${argText}) failed: `,
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
