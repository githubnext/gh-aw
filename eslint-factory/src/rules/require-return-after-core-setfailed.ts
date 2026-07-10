import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/**
 * Returns true when the statement is a call to `core.setFailed(...)`.
 */
function isCoreSetFailedStatement(node: TSESTree.Statement): node is TSESTree.ExpressionStatement {
  if (node.type !== AST_NODE_TYPES.ExpressionStatement) return false;
  const expr = node.expression;
  if (expr.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = expr.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return false;
  const obj = callee.object;
  const prop = callee.property;
  return obj.type === AST_NODE_TYPES.Identifier && obj.name === "core" && prop.type === AST_NODE_TYPES.Identifier && prop.name === "setFailed";
}

/**
 * Returns true when the statement unconditionally transfers control out of the
 * current block: `return`, `throw`, `break`, `continue`, or `process.exit(...)`.
 */
function isControlTransfer(node: TSESTree.Statement): boolean {
  if (
    node.type === AST_NODE_TYPES.ReturnStatement ||
    node.type === AST_NODE_TYPES.ThrowStatement ||
    node.type === AST_NODE_TYPES.BreakStatement ||
    node.type === AST_NODE_TYPES.ContinueStatement
  ) {
    return true;
  }
  // process.exit(...)
  if (node.type === AST_NODE_TYPES.ExpressionStatement && node.expression.type === AST_NODE_TYPES.CallExpression) {
    const callee = node.expression.callee;
    if (
      callee.type === AST_NODE_TYPES.MemberExpression &&
      !callee.computed &&
      callee.object.type === AST_NODE_TYPES.Identifier &&
      callee.object.name === "process" &&
      callee.property.type === AST_NODE_TYPES.Identifier &&
      callee.property.name === "exit"
    ) {
      return true;
    }
  }
  return false;
}

/**
 * Checks a list of sequential statements for `core.setFailed(...)` calls that
 * are not immediately followed by a control-transfer statement.
 */
function checkStatementList(stmts: TSESTree.Statement[], report: (node: TSESTree.Node) => void): void {
  for (let i = 0; i < stmts.length; i++) {
    const stmt = stmts[i];
    if (!isCoreSetFailedStatement(stmt)) continue;
    const next = stmts[i + 1];
    if (next && !isControlTransfer(next)) {
      report(stmt);
    }
  }
}

export const requireReturnAfterCoreSetFailedRule = createRule({
  name: "require-return-after-core-setfailed",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require a return, throw, break, continue, or process.exit() statement immediately after core.setFailed() to prevent execution from continuing after a failure is declared. " +
        "core.setFailed() only marks the action as failed at the end; it does not stop execution.",
    },
    schema: [],
    messages: {
      missingReturnAfterSetFailed:
        "core.setFailed() does not stop execution — add a 'return' (or 'throw') immediately after to prevent the action from continuing in a failed state.",
      addReturn: "Add 'return;' after core.setFailed() to stop execution.",
    },
  },
  defaultOptions: [],
  create(context) {
    function report(node: TSESTree.Node): void {
      context.report({
        node,
        messageId: "missingReturnAfterSetFailed",
        suggest: [
          {
            messageId: "addReturn",
            fix(fixer) {
              return fixer.insertTextAfter(node, "\n      return;");
            },
          },
        ],
      });
    }

    return {
      // Check statement blocks: if body, else body, while body, function body, etc.
      BlockStatement(node: TSESTree.BlockStatement) {
        checkStatementList(node.body, report);
      },
      // Handle single-statement arrow functions (no braces) — rare but safe to skip
      // The main case is BlockStatement above.
      SwitchCase(node: TSESTree.SwitchCase) {
        checkStatementList(node.consequent, report);
      },
    };
  },
});
