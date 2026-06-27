import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Statement node types that can be directly wrapped in a try/catch block.
const WRAPPABLE_STATEMENT_TYPES = new Set<AST_NODE_TYPES>([AST_NODE_TYPES.ExpressionStatement, AST_NODE_TYPES.VariableDeclaration, AST_NODE_TYPES.ReturnStatement, AST_NODE_TYPES.ThrowStatement]);

export const requireJsonParseTryCatchRule = createRule({
  name: "require-json-parse-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description: "Require JSON.parse calls in actions/setup/js scripts to be wrapped in try/catch",
    },
    schema: [],
    messages: {
      requireTryCatch: "Wrap JSON.parse({{arg}}) in try/catch to avoid uncaught runtime failures in actions/setup/js.",
      useHelper: "Wrap in try { ... } catch { ... }. For JSONL or possibly-malformed JSON, prefer the established safe-parse helpers: parseJsonWithRepair (collect_ndjson_output.cjs) or parseJsonlContent (jsonl_helpers.cjs).",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    function isInsideTryBlock(node: TSESTree.Node): boolean {
      return sourceCode.getAncestors(node).some(ancestor => {
        if (ancestor.type !== "TryStatement") {
          return false;
        }

        const block = ancestor.block;
        return node.range[0] >= block.range[0] && node.range[1] <= block.range[1];
      });
    }

    function findEnclosingStatement(node: TSESTree.Node): TSESTree.Statement | null {
      const ancestors = sourceCode.getAncestors(node);
      for (let i = ancestors.length - 1; i >= 0; i--) {
        const ancestor = ancestors[i];
        // Safe cast: WRAPPABLE_STATEMENT_TYPES only contains statement node types.
        if (WRAPPABLE_STATEMENT_TYPES.has(ancestor.type)) {
          return ancestor as TSESTree.Statement;
        }
      }
      return null;
    }

    return {
      CallExpression(node) {
        if (node.callee.type !== "MemberExpression") {
          return;
        }

        if (node.callee.object.type !== "Identifier") {
          return;
        }

        if (node.callee.object.name !== "JSON") {
          return;
        }

        if (node.callee.property.type !== "Identifier") {
          return;
        }

        if (node.callee.property.name !== "parse") {
          return;
        }

        if (!isInsideTryBlock(node)) {
          const argText = node.arguments.length > 0 ? sourceCode.getText(node.arguments[0]) : "";

          context.report({
            node,
            messageId: "requireTryCatch",
            data: { arg: argText },
            suggest: [
              {
                messageId: "useHelper",
                fix(fixer) {
                  const stmt = findEnclosingStatement(node);
                  if (!stmt) return null;
                  const stmtText = sourceCode.getText(stmt);
                  // ESLint always sets loc on parsed nodes; the optional chain guards
                  // against hypothetical missing loc. loc.start.line is 1-based, so
                  // subtract 1 for the 0-based lines array index.
                  const startLine = stmt.loc?.start.line;
                  const stmtLine = startLine !== undefined ? (sourceCode.lines[startLine - 1] ?? "") : "";
                  const indent = stmtLine.match(/^(\s*)/)?.[1] ?? "";
                  return fixer.replaceText(stmt, `try {\n${indent}  ${stmtText}\n${indent}} catch (err) {\n${indent}  throw err;\n${indent}}`);
                },
              },
            ],
          });
        }
      },
    };
  },
});
