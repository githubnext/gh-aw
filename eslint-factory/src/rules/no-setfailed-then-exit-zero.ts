import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { CORE_ALIASES } from "./core-aliases";
import { isCoreAliasIdentifier, isDestructuredCoreMethodIdentifier } from "./core-method-resolve";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCode = Parameters<typeof isCoreAliasIdentifier>[1];

function isCoreLikeIdentifier(name: string): boolean {
  return CORE_ALIASES.has(name);
}

/**
 * Returns true when `node` is an expression statement containing a call to
 * `core.setFailed(...)` in any recognized form:
 *  - Direct non-computed: core.setFailed(...)
 *  - Computed string literal: core["setFailed"](...)
 *  - Aliased object: const c = core; c.setFailed(...)
 *  - Destructured binding: const { setFailed } = core; setFailed(...)
 */
function isCoreSetFailedStatement(node: TSESTree.Statement, sourceCode: SourceCode): boolean {
  if (node.type !== AST_NODE_TYPES.ExpressionStatement) return false;
  const expr = node.expression;
  if (expr.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = expr.callee;

  if (callee.type === AST_NODE_TYPES.MemberExpression) {
    const obj = callee.object;
    const prop = callee.property;
    const isSetFailedNonComputed = !callee.computed && prop.type === AST_NODE_TYPES.Identifier && prop.name === "setFailed";
    const isSetFailedComputed = callee.computed && prop.type === AST_NODE_TYPES.Literal && prop.value === "setFailed";
    if (!isSetFailedNonComputed && !isSetFailedComputed) return false;
    if (obj.type !== AST_NODE_TYPES.Identifier) return false;
    return isCoreLikeIdentifier(obj.name) || isCoreAliasIdentifier(obj, sourceCode);
  }

  if (callee.type === AST_NODE_TYPES.Identifier) {
    return isDestructuredCoreMethodIdentifier(callee, "setFailed", sourceCode);
  }

  return false;
}

/**
 * Returns true when `node` is `process.exit(0)` or `process.exit()` (no args).
 * Both cause the process to exit with code 0, silently overriding any failure
 * previously declared via `core.setFailed()`.
 */
function isProcessExitZero(node: TSESTree.Statement): node is TSESTree.ExpressionStatement {
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
  // process.exit() with no arguments defaults to exit code 0
  if (expr.arguments.length === 0) return true;
  // process.exit(0) — explicit zero literal
  if (expr.arguments.length === 1) {
    const arg = expr.arguments[0];
    return arg.type === AST_NODE_TYPES.Literal && arg.value === 0;
  }
  return false;
}

/**
 * Returns true when `node` is a control-transfer statement that definitively
 * exits the current block: return, throw, break, continue, or process.exit(...).
 */
function isControlTransferStatement(node: TSESTree.Statement): boolean {
  // prettier-ignore
  if (
    node.type === AST_NODE_TYPES.ReturnStatement ||
    node.type === AST_NODE_TYPES.ThrowStatement ||
    node.type === AST_NODE_TYPES.BreakStatement ||
    node.type === AST_NODE_TYPES.ContinueStatement
  ) {
    return true;
  }
  // process.exit(...) — any call, regardless of exit code
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

export const noSetFailedThenExitZeroRule = createRule({
  name: "no-setfailed-then-exit-zero",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Disallow `process.exit(0)` (or `process.exit()`) after `core.setFailed()` in GitHub Actions scripts. " +
        "`core.setFailed()` marks the step as failed by scheduling a non-zero exit code at process end. " +
        "Calling `process.exit(0)` immediately after overrides that exit code to success, silently hiding the failure " +
        "and causing the GitHub Actions step to appear successful despite the declared error.",
    },
    schema: [],
    messages: {
      noSetFailedThenExitZero: "`process.exit(0)` after `core.setFailed()` silently resets the exit code to success, hiding the failure. " + "Replace `process.exit(0)` with `return;` to preserve the failure signal.",
      replaceWithReturn: "Replace `process.exit(0)` with `return;` to preserve the failure exit code.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    function checkStatements(stmts: readonly TSESTree.Statement[]): void {
      for (let i = 0; i < stmts.length - 1; i++) {
        const current = stmts[i];
        if (!isCoreSetFailedStatement(current, sourceCode)) continue;

        // Scan forward for process.exit(0), stopping at any other control-transfer statement.
        for (let j = i + 1; j < stmts.length; j++) {
          const candidate = stmts[j];

          if (isProcessExitZero(candidate)) {
            const isAdjacent = j === i + 1;
            context.report({
              node: candidate,
              messageId: "noSetFailedThenExitZero",
              suggest: isAdjacent
                ? [
                    {
                      messageId: "replaceWithReturn",
                      fix(fixer: TSESLint.RuleFixer) {
                        return fixer.replaceText(candidate, "return;");
                      },
                    },
                  ]
                : [],
            });
            break;
          }

          // Stop scanning at any control-transfer (return, throw, break, process.exit(nonzero), etc.)
          if (isControlTransferStatement(candidate)) {
            break;
          }
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
