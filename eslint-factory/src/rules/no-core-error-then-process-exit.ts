import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { CORE_ALIASES } from "./core-aliases";
import { isCoreAliasIdentifier } from "./core-method-resolve";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCode = Parameters<typeof isCoreAliasIdentifier>[1];

function isCoreLikeIdentifier(name: string): boolean {
  return CORE_ALIASES.has(name);
}

/**
 * Returns true when `node` is an expression statement containing a call to
 * `core.error(...)` (direct, computed, or aliased).
 */
function isCoreErrorStatement(node: TSESTree.Statement, sourceCode: SourceCode): node is TSESTree.ExpressionStatement {
  if (node.type !== AST_NODE_TYPES.ExpressionStatement) return false;
  const expr = node.expression;
  if (expr.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = expr.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression) return false;

  const obj = callee.object;
  const prop = callee.property;
  const isErrorNonComputed = !callee.computed && prop.type === AST_NODE_TYPES.Identifier && prop.name === "error";
  const isErrorComputed = callee.computed && prop.type === AST_NODE_TYPES.Literal && prop.value === "error";
  if (!isErrorNonComputed && !isErrorComputed) return false;
  if (obj.type !== AST_NODE_TYPES.Identifier) return false;

  return isCoreLikeIdentifier(obj.name) || isCoreAliasIdentifier(obj, sourceCode);
}

/**
 * Returns true when `node` is `process.exit(expr)` where `expr` evaluates to
 * a non-zero literal integer (e.g. process.exit(1), process.exit(2)).
 * Returns false for process.exit(0) which is a deliberate clean exit.
 */
function isProcessExitNonZero(node: TSESTree.Statement): node is TSESTree.ExpressionStatement {
  if (node.type !== AST_NODE_TYPES.ExpressionStatement) return false;
  const expr = node.expression;
  if (expr.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = expr.callee;
  if (
    callee.type !== AST_NODE_TYPES.MemberExpression ||
    callee.computed ||
    callee.object.type !== AST_NODE_TYPES.Identifier ||
    callee.object.name !== "process" ||
    callee.property.type !== AST_NODE_TYPES.Identifier ||
    callee.property.name !== "exit"
  ) {
    return false;
  }
  // Must have exactly one argument
  if (expr.arguments.length !== 1) return false;
  const arg = expr.arguments[0];
  // Only match explicit non-zero integer numeric literals (e.g. process.exit(1), process.exit(2)).
  // Variables, function calls, and non-numeric literals are not flagged — their runtime value
  // cannot be proven non-zero at static analysis time.
  if (arg.type !== AST_NODE_TYPES.Literal || typeof arg.value !== "number" || !Number.isInteger(arg.value) || arg.value === 0) return false;
  return true;
}

export const noCoreErrorThenProcessExitRule = createRule({
  name: "no-core-error-then-process-exit",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Disallow the pattern `core.error(msg); process.exit(nonzero)` in GitHub Actions scripts. " +
        "`core.error()` annotates the log but does not mark the action as failed. " +
        "Using `process.exit(nonzero)` after it bypasses the proper GitHub Actions failure lifecycle. " +
        "Use `core.setFailed(msg); return;` instead so the action is correctly marked as failed and all cleanup hooks run.",
    },
    schema: [],
    messages: {
      noCoreErrorThenProcessExit: "Avoid `core.error()` followed by `process.exit(nonzero)`. Use `core.setFailed(msg); return;` instead to correctly signal action failure without bypassing cleanup hooks.",
      replaceWithSetFailed: "Replace `core.error(msg); process.exit(...)` with `core.setFailed(msg); return;`.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    function checkStatements(stmts: readonly TSESTree.Statement[]): void {
      for (let i = 0; i < stmts.length - 1; i++) {
        const current = stmts[i];
        const next = stmts[i + 1];
        if (isCoreErrorStatement(current, sourceCode) && isProcessExitNonZero(next)) {
          // Report on the pair (the core.error call)
          context.report({
            node: current,
            messageId: "noCoreErrorThenProcessExit",
            suggest: [
              {
                messageId: "replaceWithSetFailed",
                fix(fixer: TSESLint.RuleFixer) {
                  const errorCall = (current as TSESTree.ExpressionStatement).expression as TSESTree.CallExpression;
                  const args = errorCall.arguments.map(a => sourceCode.getText(a)).join(", ");

                  // Detect the core object name (e.g. "core")
                  const callee = errorCall.callee as TSESTree.MemberExpression;
                  const objectName = sourceCode.getText(callee.object);

                  const isInsideFunction = (() => {
                    const ancestors = sourceCode.getAncestors(current);
                    for (let j = ancestors.length - 1; j >= 0; j--) {
                      const a = ancestors[j];
                      if (a.type === AST_NODE_TYPES.FunctionDeclaration || a.type === AST_NODE_TYPES.FunctionExpression || a.type === AST_NODE_TYPES.ArrowFunctionExpression) {
                        return true;
                      }
                    }
                    return false;
                  })();

                  const replacement = isInsideFunction ? `${objectName}.setFailed(${args}); return;` : `${objectName}.setFailed(${args});`;

                  return [fixer.replaceText(current, replacement + "\n"), fixer.remove(next)];
                },
              },
            ],
          });
        }
      }
    }

    return {
      BlockStatement(node: TSESTree.BlockStatement) {
        checkStatements(node.body);
      },
      SwitchCase(node: TSESTree.SwitchCase) {
        checkStatements(node.consequent);
      },
      Program(node: TSESTree.Program) {
        checkStatements(node.body.filter((s): s is TSESTree.Statement => s.type !== AST_NODE_TYPES.ImportDeclaration && s.type !== AST_NODE_TYPES.ExportAllDeclaration));
      },
    };
  },
});
