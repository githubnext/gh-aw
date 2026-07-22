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

/**
 * Returns true when `node` is an expression statement containing a call to
 * `core.setFailed(...)` (direct, computed, or aliased).
 * Accepts `sourceCode` for alias resolution via `isCoreAliasIdentifier`; contrast
 * with `isControlTransferStatement` which is a pure syntax check and needs no source-code context.
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
  // process.exit(...) — any call, regardless of exit code; handles both dot and computed access
  if (node.type === AST_NODE_TYPES.ExpressionStatement && node.expression.type === AST_NODE_TYPES.CallExpression) {
    const callee = node.expression.callee;
    if (callee.type === AST_NODE_TYPES.MemberExpression && callee.object.type === AST_NODE_TYPES.Identifier && callee.object.name === "process") {
      const prop = callee.property;
      const isExitDot = !callee.computed && prop.type === AST_NODE_TYPES.Identifier && prop.name === "exit";
      const isExitComputed = callee.computed && prop.type === AST_NODE_TYPES.Literal && prop.value === "exit";
      if (isExitDot || isExitComputed) return true;
    }
  }
  return false;
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
        "Disallow the pattern `core.error(msg); ... ; process.exitCode = nonzero` in GitHub Actions scripts. " +
        "`core.error()` annotates the log but does not mark the action as failed. " +
        "Prefer `core.setFailed(msg)` which correctly marks the action as failed and allows post-action " +
        "cleanup hooks to run. Unlike `process.exit(1)`, `process.exitCode = 1` does not immediately halt " +
        "execution, so subsequent code still runs in the failed state. " +
        "The rule scans forward from `core.error(...)` for a later `process.exitCode = nonzero`, " +
        "stopping at `core.setFailed(...)` or a control-transfer statement.",
    },
    schema: [],
    messages: {
      noCoreErrorThenProcessExitCode:
        "Avoid `core.error()` followed by `process.exitCode = nonzero`. Prefer `core.setFailed(msg)` to signal " +
        "action failure; it marks the action failed and allows post-action cleanup hooks to run. " +
        "Unlike `process.exit(1)`, `process.exitCode = 1` does not halt execution immediately.",
      replaceWithSetFailed: "Replace `core.error(msg); process.exitCode = nonzero` with `core.setFailed(msg)` (at module top level) or `core.setFailed(msg); return;` (inside main()).",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    function checkStatements(stmts: readonly TSESTree.Statement[]): void {
      // Track which process.exitCode nodes have already been reported so that two consecutive
      // core.error() calls before the same exitCode do not each fire their own diagnostic
      // (which could produce conflicting autofixers on the same node).
      const reported = new WeakSet<TSESTree.Statement>();
      for (let i = 0; i < stmts.length - 1; i++) {
        const current = stmts[i];
        if (!isCoreErrorStatement(current, sourceCode)) continue;

        // Scan forward for process.exitCode = nonzero, stopping at setFailed or control-transfer.
        // Adjacent (j === i+1) keeps autofix; non-adjacent reports without suggestion.
        for (let j = i + 1; j < stmts.length; j++) {
          const candidate = stmts[j];

          if (isProcessExitCodeNonZero(candidate)) {
            if (!reported.has(candidate)) {
              reported.add(candidate);
              const isAdjacent = j === i + 1;
              // The autofix suggestion is only safe when the pair is at module top level or directly
              // inside a `main()` entrypoint. Inside helper functions, `return;` only exits the helper
              // and lets the caller continue. For non-adjacent pairs we omit the suggestion to avoid
              // a fixer that leaves intervening statements between a deleted exitCode and the new setFailed.
              const enclosingFn = getImmediateEnclosingFunction(current, sourceCode);
              const errorCall = current.expression as TSESTree.CallExpression;
              const safeToFix = isAdjacent && (enclosingFn === null || isFunctionNamedMain(enclosingFn)) && hasSingleNonSpreadArgument(errorCall);

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

                          // At module top-level (enclosingFn === null) there is nothing to `return` from,
                          // so we just replace with setFailed. Inside main() we append `return;` to exit
                          // the entrypoint cleanly.
                          const replacement = enclosingFn !== null ? `${objectName}.setFailed(${args}); return;` : `${objectName}.setFailed(${args});`;

                          return [fixer.replaceText(current, replacement + "\n"), fixer.remove(candidate)];
                        },
                      },
                    ]
                  : [],
              });
            }
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
        // At module top level, export declarations act as segment boundaries (they separate
        // "regions" of the module). We split the program body at export/import declarations
        // so that an export between core.error and process.exitCode breaks the scan.
        let segment: TSESTree.Statement[] = [];
        for (const stmt of node.body) {
          if (isProgramStatement(stmt)) {
            segment.push(stmt);
          } else {
            checkStatements(segment);
            segment = [];
          }
        }
        checkStatements(segment);
      },
    };
  },
});
