import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { CORE_ALIASES } from "./core-aliases";
import { isCoreAliasIdentifier } from "./core-method-resolve";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCode = Parameters<typeof isCoreAliasIdentifier>[1];

function isCoreLikeIdentifier(name: string): boolean {
  return CORE_ALIASES.has(name);
}

type FunctionNode = TSESTree.FunctionDeclaration | TSESTree.FunctionExpression | TSESTree.ArrowFunctionExpression;

/**
 * Returns the innermost enclosing function node for `node`, or null when
 * `node` is at module top level (not inside any function).
 */
function getImmediateEnclosingFunction(node: TSESTree.Node, sourceCode: SourceCode): FunctionNode | null {
  const ancestors = sourceCode.getAncestors(node);
  for (let i = ancestors.length - 1; i >= 0; i--) {
    const ancestor = ancestors[i];
    // prettier-ignore
    if (
      ancestor.type === AST_NODE_TYPES.FunctionDeclaration ||
      ancestor.type === AST_NODE_TYPES.FunctionExpression ||
      ancestor.type === AST_NODE_TYPES.ArrowFunctionExpression
    ) {
      return ancestor as FunctionNode;
    }
  }
  return null;
}

/**
 * Returns true when `fn` is a conventional module entrypoint named `main`
 * declared at module top level (not nested inside another function):
 *   - `function main() {}` / `async function main() {}`  at Program scope
 *   - `const main = function() {}` / `const main = async () => {}`  at Program scope
 */
function isFunctionNamedMain(fn: FunctionNode): boolean {
  if (fn.type === AST_NODE_TYPES.FunctionDeclaration) {
    // Must be named `main` and declared directly inside the Program (module top level)
    return fn.id?.name === "main" && fn.parent?.type === AST_NODE_TYPES.Program;
  }
  // FunctionExpression or ArrowFunctionExpression assigned to a variable named `main`
  const declarator = fn.parent;
  // prettier-ignore
  if (
    declarator == null ||
    declarator.type !== AST_NODE_TYPES.VariableDeclarator ||
    declarator.id.type !== AST_NODE_TYPES.Identifier ||
    declarator.id.name !== "main"
  ) {
    return false;
  }
  // The VariableDeclaration containing this declarator must be at module top level
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
 * Returns true when `node` is an expression statement containing a call to
 * `core.setFailed(...)` (direct, computed, or aliased).
 */
function isCoreSetFailedStatement(node: TSESTree.Statement, sourceCode: SourceCode): boolean {
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

/**
 * Returns true when `node` is a control-transfer statement that definitively
 * exits the current block: return, throw, break, continue, or process.exit(...).
 *
 * Any `process.exit(...)` call is treated as a barrier regardless of the exit
 * code value — process.exit(0) and process.exit(variable) both terminate the
 * process, so subsequent statements are unreachable and should not be flagged.
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
        "Prefer `core.setFailed(msg)` which correctly marks the action as failed and allows post-action " +
        "cleanup hooks to run. Note: in a standalone `node` script, `process.exit(nonzero)` does fail the " +
        "step, but `core.setFailed` is more portable and is still recommended.",
    },
    schema: [],
    messages: {
      noCoreErrorThenProcessExit:
        "Avoid `core.error()` followed by `process.exit(nonzero)`. Prefer `core.setFailed(msg)` to signal " +
        "action failure; it marks the action failed and allows post-action cleanup hooks to run. " +
        "In standalone `node` scripts, `process.exit(nonzero)` does fail the step, but `core.setFailed` is more portable.",
      replaceWithSetFailed: "Replace `core.error(msg); process.exit(...)` with `core.setFailed(msg); return;`.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    function checkStatements(stmts: readonly TSESTree.Statement[]): void {
      for (let i = 0; i < stmts.length - 1; i++) {
        const current = stmts[i];
        if (!isCoreErrorStatement(current, sourceCode)) continue;

        // Scan forward for process.exit(nonzero), stopping at setFailed or control-transfer.
        // Adjacent (j === i+1) keeps autofix; non-adjacent reports without suggestion.
        for (let j = i + 1; j < stmts.length; j++) {
          const candidate = stmts[j];

          if (isProcessExitNonZero(candidate)) {
            const isAdjacent = j === i + 1;
            // The autofix suggestion is only safe when the pair is at module top level or directly
            // inside a `main()` entrypoint. Inside helper functions, `return;` only exits the helper
            // and lets the caller continue — it does NOT abort the process like `process.exit` does.
            // For non-adjacent pairs we omit the suggestion to avoid a fixer that leaves intervening
            // statements between a deleted process.exit and the new setFailed.
            const enclosingFn = getImmediateEnclosingFunction(current, sourceCode);
            const safeToFix = isAdjacent && (enclosingFn === null || isFunctionNamedMain(enclosingFn));

            context.report({
              node: current,
              messageId: "noCoreErrorThenProcessExit",
              suggest: safeToFix
                ? [
                    {
                      messageId: "replaceWithSetFailed",
                      fix(fixer: TSESLint.RuleFixer) {
                        const errorCall = (current as TSESTree.ExpressionStatement).expression as TSESTree.CallExpression;
                        const args = errorCall.arguments.map(a => sourceCode.getText(a)).join(", ");

                        // Detect the core object name (e.g. "core")
                        const callee = errorCall.callee as TSESTree.MemberExpression;
                        const objectName = sourceCode.getText(callee.object);

                        // At module top-level (enclosingFn === null) there is nothing to `return` from,
                        // so we just replace with setFailed. Inside main() we append `return;` to exit
                        // the entrypoint in the same way process.exit would.
                        const replacement = enclosingFn !== null ? `${objectName}.setFailed(${args}); return;` : `${objectName}.setFailed(${args});`;

                        return [fixer.replaceText(current, replacement + "\n"), fixer.remove(candidate)];
                      },
                    },
                  ]
                : [],
            });
            break;
          }

          // Stop scanning if setFailed already handles the failure or a control-transfer exits the block.
          if (isCoreSetFailedStatement(candidate, sourceCode) || isControlTransferStatement(candidate)) {
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
