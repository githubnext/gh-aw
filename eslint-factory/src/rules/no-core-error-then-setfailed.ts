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
 * Returns true when `node` is an expression statement containing a call to
 * `core.setFailed(...)` (direct, computed, or aliased).
 */
function isCoreSetFailedStatement(node: TSESTree.Statement, sourceCode: SourceCode): node is TSESTree.ExpressionStatement {
  if (node.type !== AST_NODE_TYPES.ExpressionStatement) return false;
  const expr = node.expression;
  if (expr.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = expr.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression) return false;

  const obj = callee.object;
  const prop = callee.property;
  const isSetFailedNonComputed = !callee.computed && prop.type === AST_NODE_TYPES.Identifier && prop.name === "setFailed";
  const isSetFailedComputed = callee.computed && prop.type === AST_NODE_TYPES.Literal && prop.value === "setFailed";
  if (!isSetFailedNonComputed && !isSetFailedComputed) return false;
  if (obj.type !== AST_NODE_TYPES.Identifier) return false;

  return isCoreLikeIdentifier(obj.name) || isCoreAliasIdentifier(obj, sourceCode);
}

function hasSingleNonSpreadArgument(call: TSESTree.CallExpression): boolean {
  return call.arguments.length === 1 && call.arguments[0].type !== AST_NODE_TYPES.SpreadElement;
}

export const noCoreErrorThenSetFailedRule = createRule({
  name: "no-core-error-then-setfailed",
  meta: {
    type: "suggestion",
    hasSuggestions: true,
    docs: {
      description:
        "Disallow the redundant pattern `core.error(msg); core.setFailed(msg)` in GitHub Actions scripts. " +
        "`core.setFailed()` already logs the message as an error annotation and marks the action as failed. " +
        "Preceding it with `core.error()` using the same or similar message creates a duplicate error annotation " +
        "in the GitHub Actions log, adding noise without benefit. Use `core.setFailed(msg)` alone.",
    },
    schema: [],
    messages: {
      noCoreErrorThenSetFailed:
        "`core.error()` immediately before `core.setFailed()` is redundant: `core.setFailed()` already logs " +
        "an error annotation and marks the action failed. Remove the `core.error()` call.",
      removeErrorCall: "Remove the redundant `core.error()` call — `core.setFailed()` already logs an error annotation.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    function checkStatements(stmts: readonly TSESTree.Statement[]): void {
      for (let i = 0; i < stmts.length - 1; i++) {
        const current = stmts[i];
        if (!isCoreErrorStatement(current, sourceCode)) continue;

        const next = stmts[i + 1];
        if (!isCoreSetFailedStatement(next, sourceCode)) continue;

        const errorCall = current.expression as TSESTree.CallExpression;
        const safeToFix = hasSingleNonSpreadArgument(errorCall);

        context.report({
          node: current,
          messageId: "noCoreErrorThenSetFailed",
          suggest: safeToFix
            ? [
                {
                  messageId: "removeErrorCall",
                  fix(fixer: TSESLint.RuleFixer) {
                    return fixer.remove(current);
                  },
                },
              ]
            : [],
        });
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
        const stmts = node.body.filter((s): s is TSESTree.Statement => s.type !== AST_NODE_TYPES.ImportDeclaration && s.type !== AST_NODE_TYPES.ExportAllDeclaration);
        checkStatements(stmts);
      },
    };
  },
});
