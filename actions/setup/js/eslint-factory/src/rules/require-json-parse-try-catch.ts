import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/actions/setup/js/eslint-factory#${name}`);

// Statement node types that can be directly wrapped in a try/catch block.
const WRAPPABLE_STATEMENT_TYPES = new Set<AST_NODE_TYPES>([AST_NODE_TYPES.ExpressionStatement, AST_NODE_TYPES.VariableDeclaration, AST_NODE_TYPES.ReturnStatement, AST_NODE_TYPES.ThrowStatement]);
const DEFERRED_MEMBER_METHODS = new Set(["then", "catch", "finally", "addEventListener", "on", "once"]);
const DEFERRED_GLOBAL_FUNCTIONS = new Set(["setTimeout", "setInterval", "setImmediate", "queueMicrotask"]);

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

    function isFunctionExpressionNode(node: TSESTree.Node): node is TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression {
      return node.type === "ArrowFunctionExpression" || node.type === "FunctionExpression";
    }

    function isDeferredCallExpression(parent: TSESTree.CallExpression): boolean {
      if (parent.callee.type === "Identifier") {
        return DEFERRED_GLOBAL_FUNCTIONS.has(parent.callee.name);
      }

      if (parent.callee.type !== "MemberExpression") {
        return false;
      }

      if (parent.callee.property.type !== "Identifier") {
        return false;
      }

      const methodName = parent.callee.property.name;
      if (DEFERRED_MEMBER_METHODS.has(methodName)) {
        return true;
      }

      return parent.callee.object.type === "Identifier" && parent.callee.object.name === "process" && methodName === "nextTick";
    }

    function isDeferredCallbackFunction(ancestors: TSESTree.Node[], functionIndex: number): boolean {
      const functionNode = ancestors[functionIndex];
      if (!isFunctionExpressionNode(functionNode) || functionIndex === 0) {
        return false;
      }

      const parent = ancestors[functionIndex - 1];
      if (parent.type === "CallExpression") {
        if (!parent.arguments.includes(functionNode)) {
          return false;
        }
        return isDeferredCallExpression(parent);
      }

      if (parent.type === "NewExpression") {
        return parent.callee.type === "Identifier" && parent.callee.name === "Promise" && parent.arguments[0] === functionNode;
      }

      return false;
    }

    function isProtectedByTry(ancestors: TSESTree.Node[], tryStatement: TSESTree.TryStatement): boolean {
      const tryIndex = ancestors.indexOf(tryStatement);
      if (tryIndex === -1) {
        return false;
      }

      for (let i = tryIndex + 1; i < ancestors.length; i++) {
        if (isDeferredCallbackFunction(ancestors, i)) {
          return false;
        }
      }

      return true;
    }

    function isInsideTryBlock(node: TSESTree.Node): boolean {
      const ancestors = sourceCode.getAncestors(node);
      return ancestors.some(ancestor => {
        if (ancestor.type !== "TryStatement") {
          return false;
        }

        if (node.range[0] < ancestor.block.range[0] || node.range[1] > ancestor.block.range[1]) {
          return false;
        }

        return isProtectedByTry(ancestors, ancestor);
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
