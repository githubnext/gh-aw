import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { CORE_ALIASES } from "./core-aliases";
import { isCoreAliasIdentifier } from "./core-method-resolve";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCode = Parameters<typeof isCoreAliasIdentifier>[1];

function isCoreLikeIdentifier(name: string): boolean {
  return CORE_ALIASES.has(name);
}

type FunctionNode = TSESTree.FunctionDeclaration | TSESTree.FunctionExpression | TSESTree.ArrowFunctionExpression;

function getImmediateEnclosingFunction(node: TSESTree.Node, sourceCode: SourceCode): FunctionNode | null {
  const ancestors = sourceCode.getAncestors(node);
  for (let i = ancestors.length - 1; i >= 0; i--) {
    const ancestor = ancestors[i];
    if (ancestor.type === AST_NODE_TYPES.FunctionDeclaration || ancestor.type === AST_NODE_TYPES.FunctionExpression || ancestor.type === AST_NODE_TYPES.ArrowFunctionExpression) {
      return ancestor as FunctionNode;
    }
  }
  return null;
}

function isFunctionNamedMain(fn: FunctionNode): boolean {
  if (fn.type === AST_NODE_TYPES.FunctionDeclaration) {
    if (fn.id?.name !== "main") return false;
    if (fn.parent?.type === AST_NODE_TYPES.Program) return true;
    return fn.parent?.type === AST_NODE_TYPES.ExportNamedDeclaration && fn.parent.parent?.type === AST_NODE_TYPES.Program;
  }
  const declarator = fn.parent;
  if (declarator == null || declarator.type !== AST_NODE_TYPES.VariableDeclarator || declarator.id.type !== AST_NODE_TYPES.Identifier || declarator.id.name !== "main") {
    return false;
  }
  const varDecl = declarator.parent;
  return varDecl?.type === AST_NODE_TYPES.VariableDeclaration && varDecl.parent?.type === AST_NODE_TYPES.Program;
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
 * Returns true when `node` is `process.exitCode = expr` where `expr` is a
 * non-zero integer literal (e.g. process.exitCode = 1).
 * Returns false for process.exitCode = 0 which is a deliberate clean exit.
 */
function isProcessExitCodeNonZero(node: TSESTree.Statement): node is TSESTree.ExpressionStatement {
  if (node.type !== AST_NODE_TYPES.ExpressionStatement) return false;
  const expr = node.expression;
  if (expr.type !== AST_NODE_TYPES.AssignmentExpression || expr.operator !== "=") return false;
  const left = expr.left;
  if (
    left.type !== AST_NODE_TYPES.MemberExpression ||
    left.computed ||
    left.object.type !== AST_NODE_TYPES.Identifier ||
    left.object.name !== "process" ||
    left.property.type !== AST_NODE_TYPES.Identifier ||
    left.property.name !== "exitCode"
  ) {
    return false;
  }
  const right = expr.right;
  if (right.type !== AST_NODE_TYPES.Literal || typeof right.value !== "number" || !Number.isInteger(right.value) || right.value === 0) return false;
  return true;
}

function hasSingleNonSpreadArgument(call: TSESTree.CallExpression): boolean {
  return call.arguments.length === 1 && call.arguments[0].type !== AST_NODE_TYPES.SpreadElement;
}

function isProgramStatement(node: TSESTree.ProgramStatement): node is TSESTree.Statement {
  return node.type !== AST_NODE_TYPES.ImportDeclaration && node.type !== AST_NODE_TYPES.ExportAllDeclaration && node.type !== AST_NODE_TYPES.ExportDefaultDeclaration && node.type !== AST_NODE_TYPES.ExportNamedDeclaration;
}

export const noCoreErrorThenProcessExitCodeRule = createRule({
  name: "no-core-error-then-process-exitcode",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Disallow the pattern `core.error(msg); process.exitCode = nonzero` in GitHub Actions scripts. " +
        "`core.error()` annotates the log but does not mark the action as failed. " +
        "Prefer `core.setFailed(msg)` which correctly marks the action as failed and allows post-action " +
        "cleanup hooks to run. Unlike `process.exit(1)`, `process.exitCode = 1` does not immediately halt " +
        "execution, so subsequent code still runs in the failed state.",
    },
    schema: [],
    messages: {
      noCoreErrorThenProcessExitCode:
        "Avoid `core.error()` followed by `process.exitCode = nonzero`. Prefer `core.setFailed(msg)` to signal " +
        "action failure; it marks the action failed and allows post-action cleanup hooks to run. " +
        "Unlike `process.exit(1)`, `process.exitCode = 1` does not halt execution immediately.",
      replaceWithSetFailed: "Replace `core.error(msg); process.exitCode = nonzero` with `core.setFailed(msg); return;`.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    function checkStatements(stmts: readonly TSESTree.Statement[]): void {
      for (let i = 0; i < stmts.length - 1; i++) {
        const current = stmts[i];
        const next = stmts[i + 1];
        if (isCoreErrorStatement(current, sourceCode) && isProcessExitCodeNonZero(next)) {
          const enclosingFn = getImmediateEnclosingFunction(current, sourceCode);
          const errorCall = current.expression as TSESTree.CallExpression;
          const safeToFix = enclosingFn !== null && isFunctionNamedMain(enclosingFn) && hasSingleNonSpreadArgument(errorCall);

          context.report({
            node: current,
            messageId: "noCoreErrorThenProcessExitCode",
            suggest: safeToFix
              ? [
                  {
                    messageId: "replaceWithSetFailed",
                    fix(fixer: TSESLint.RuleFixer) {
                      const args = errorCall.arguments.map(a => sourceCode.getText(a)).join(", ");
                      const callee = errorCall.callee as TSESTree.MemberExpression;
                      const objectName = sourceCode.getText(callee.object);
                      return [fixer.replaceText(current, `${objectName}.setFailed(${args}); return;\n`), fixer.remove(next)];
                    },
                  },
                ]
              : [],
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
        for (let i = 0; i < node.body.length - 1; i++) {
          const current = node.body[i];
          const next = node.body[i + 1];
          if (isProgramStatement(current) && isProgramStatement(next)) {
            checkStatements([current, next]);
          }
        }
      },
    };
  },
});
