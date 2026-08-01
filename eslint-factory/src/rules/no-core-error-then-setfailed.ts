import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { CORE_ALIASES } from "./core-aliases";
import { isCoreAliasIdentifier } from "./core-method-resolve";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCode = Parameters<typeof isCoreAliasIdentifier>[1];

function isCoreLikeIdentifier(name: string): boolean {
  return CORE_ALIASES.has(name);
}

/**
 * Returns the first non-SpreadElement argument of a call, or null when
 * there are no arguments or the first argument is a spread.
 */
function getFirstNonSpreadArg(call: TSESTree.CallExpression): TSESTree.Expression | null {
  if (call.arguments.length === 0) return null;
  const first = call.arguments[0];
  if (first.type === AST_NODE_TYPES.SpreadElement) return null;
  return first as TSESTree.Expression;
}

/**
 * Returns true when the call has more than one argument (i.e. annotation
 * properties are present, e.g. `core.error(msg, { title: "..." })`).
 * Such calls carry diagnostic context not duplicated in setFailed and must
 * not be flagged as redundant.
 */
function hasAnnotationProperties(call: TSESTree.CallExpression): boolean {
  return call.arguments.length > 1;
}

/**
 * Returns true when `setFailedArg` is a TemplateLiteral whose content ends
 * with the full content of `errorArg` (also a TemplateLiteral), with a
 * non-empty prefix prepended. The prefix may contain template expressions.
 * This captures the common pattern:
 *   core.error(`Failed: ${msg}`)
 *   core.setFailed(`${ERR_CODE}: Failed: ${msg}`)
 * where setFailed emits the same annotation text with only an error-code
 * prefix added in front.
 *
 * A match requires:
 *   1. Both args are TemplateLiterals.
 *   2. setFailed has at least as many expressions as error.
 *   3. The last N expressions of setFailed match error's expressions in
 *      order (same source text), where N = error.expressions.length.
 *   4. The quasis of setFailed at offsets [P+1 .. end] (P = prefix expr
 *      count) equal error's quasis at [1 .. end] (same cooked value).
 *   5. The "junction" quasi in setFailed (at offset P) ends with error's
 *      first quasi (cooked value).
 *   6. The prefix is non-empty (to avoid re-matching exact-equal pairs).
 */
function isSetFailedArgPrefixedVersion(errorArg: TSESTree.Expression, setFailedArg: TSESTree.Expression, sourceCode: SourceCode): boolean {
  if (errorArg.type !== AST_NODE_TYPES.TemplateLiteral) return false;
  if (setFailedArg.type !== AST_NODE_TYPES.TemplateLiteral) return false;

  const errTpl = errorArg as TSESTree.TemplateLiteral;
  const sfTpl = setFailedArg as TSESTree.TemplateLiteral;

  if (sfTpl.expressions.length < errTpl.expressions.length) return false;

  const prefixExprCount = sfTpl.expressions.length - errTpl.expressions.length;

  // Last errTpl.expressions.length expressions of sfTpl must match errTpl's expressions.
  for (let i = 0; i < errTpl.expressions.length; i++) {
    const sfExpr = sfTpl.expressions[prefixExprCount + i] as TSESTree.Expression;
    const errExpr = errTpl.expressions[i] as TSESTree.Expression;
    if (sourceCode.getText(sfExpr) !== sourceCode.getText(errExpr)) return false;
  }

  // Quasis at offsets [prefixExprCount+1 .. end] of sfTpl must equal errTpl quasis [1 .. end].
  // Guard against null cooked values (produced by invalid Unicode escape sequences).
  for (let i = 1; i < errTpl.quasis.length; i++) {
    const sfCooked = sfTpl.quasis[prefixExprCount + i].value.cooked;
    const errCooked = errTpl.quasis[i].value.cooked;
    if (sfCooked === null || errCooked === null) return false;
    if (sfCooked !== errCooked) return false;
  }

  // The junction quasi (sfTpl.quasis[prefixExprCount]) must end with errTpl's first quasi.
  const junctionCooked = sfTpl.quasis[prefixExprCount].value.cooked;
  const errFirstCooked = errTpl.quasis[0].value.cooked;
  if (junctionCooked === null || errFirstCooked === null) return false;
  if (!junctionCooked.endsWith(errFirstCooked)) return false;

  // There must be an actual prefix (not an exact-equal pair).
  return prefixExprCount > 0 || junctionCooked.length > errFirstCooked.length;
}

/**
 * Returns true when `setFailedArg` is a BinaryExpression of the form
 * `<string> + <expr>` where the string literal is non-empty (the prefix)
 * and the right operand is identical in source text to `errorArg`.
 * This captures:
 *   core.error(`Failed: ${msg}`)
 *   core.setFailed("ERR: " + `Failed: ${msg}`)
 */
function isSetFailedArgStringConcatPrefixedVersion(errorArg: TSESTree.Expression, setFailedArg: TSESTree.Expression, sourceCode: SourceCode): boolean {
  if (setFailedArg.type !== AST_NODE_TYPES.BinaryExpression) return false;
  const be = setFailedArg as TSESTree.BinaryExpression;
  if (be.operator !== "+") return false;

  // Left side must be a non-empty string literal (the prefix).
  const left = be.left as TSESTree.Expression;
  if (left.type !== AST_NODE_TYPES.Literal) return false;
  if (typeof (left as TSESTree.Literal).value !== "string") return false;
  if ((left as TSESTree.Literal).value === "") return false;

  // Right side must be identical in source to errorArg.
  const right = be.right as TSESTree.Expression;
  return sourceCode.getText(right) === sourceCode.getText(errorArg);
}

/**
 * Returns true when the expression is provably side-effect-free: no call,
 * new, or assignment expression at any nesting level. Conservatively returns
 * false for any node type not listed here.
 */
function isSideEffectFree(node: TSESTree.Expression): boolean {
  switch (node.type) {
    case AST_NODE_TYPES.Literal:
    case AST_NODE_TYPES.Identifier:
      return true;
    case AST_NODE_TYPES.TemplateLiteral:
      return (node as TSESTree.TemplateLiteral).expressions.every(e => isSideEffectFree(e as TSESTree.Expression));
    case AST_NODE_TYPES.MemberExpression: {
      const me = node as TSESTree.MemberExpression;
      return isSideEffectFree(me.object as TSESTree.Expression) && (!me.computed || isSideEffectFree(me.property as TSESTree.Expression));
    }
    case AST_NODE_TYPES.BinaryExpression: {
      const be = node as TSESTree.BinaryExpression;
      return isSideEffectFree(be.left as TSESTree.Expression) && isSideEffectFree(be.right as TSESTree.Expression);
    }
    case AST_NODE_TYPES.UnaryExpression:
      return isSideEffectFree((node as TSESTree.UnaryExpression).argument as TSESTree.Expression);
    default:
      return false;
  }
}

/**
 * Returns true when `node` is an expression statement containing a call to
 * `<coreObj>.<methodName>(...)` where the receiver is a known core alias
 * (direct or assigned alias). Also returns the receiver identifier name via
 * the `objectName` out-param so the caller can enforce same-object pairing.
 */
function isCoreMethodCallStatement(node: TSESTree.Statement, sourceCode: SourceCode, methodName: string): node is TSESTree.ExpressionStatement {
  if (node.type !== AST_NODE_TYPES.ExpressionStatement) return false;
  const expr = node.expression;
  if (expr.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = expr.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression) return false;

  const obj = callee.object;
  const prop = callee.property;
  const isNonComputed = !callee.computed && prop.type === AST_NODE_TYPES.Identifier && (prop as TSESTree.Identifier).name === methodName;
  const isComputed = callee.computed && prop.type === AST_NODE_TYPES.Literal && (prop as TSESTree.Literal).value === methodName;
  if (!isNonComputed && !isComputed) return false;
  if (obj.type !== AST_NODE_TYPES.Identifier) return false;

  return isCoreLikeIdentifier((obj as TSESTree.Identifier).name) || isCoreAliasIdentifier(obj as TSESTree.Identifier, sourceCode);
}

function isCoreErrorStatement(node: TSESTree.Statement, sourceCode: SourceCode): node is TSESTree.ExpressionStatement {
  return isCoreMethodCallStatement(node, sourceCode, "error");
}

function isCoreSetFailedStatement(node: TSESTree.Statement, sourceCode: SourceCode): node is TSESTree.ExpressionStatement {
  return isCoreMethodCallStatement(node, sourceCode, "setFailed");
}

/**
 * Returns the receiver identifier name from a matched core-method call statement.
 * Precondition: `isCoreErrorStatement` or `isCoreSetFailedStatement` returned true.
 */
function getCoreObjectName(stmt: TSESTree.ExpressionStatement): string {
  const call = stmt.expression as TSESTree.CallExpression;
  const callee = call.callee as TSESTree.MemberExpression;
  return (callee.object as TSESTree.Identifier).name;
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
        "Preceding it with `core.error()` using the same message creates a duplicate error annotation " +
        "in the GitHub Actions log, adding noise without benefit. Use `core.setFailed(msg)` alone.",
    },
    schema: [],
    messages: {
      noCoreErrorThenSetFailed: "`core.error()` immediately before `core.setFailed()` with the same message is redundant: `core.setFailed()` already logs an error annotation and marks the action failed. Remove the `core.error()` call.",
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

        // Both calls must reference the same receiver identifier to avoid
        // flagging `c1.error("x"); c2.setFailed("x")` where c1 and c2 are
        // different objects that happen to both be in CORE_ALIASES.
        if (getCoreObjectName(current) !== getCoreObjectName(next)) continue;

        const errorCall = (current as TSESTree.ExpressionStatement).expression as TSESTree.CallExpression;
        const setFailedCall = (next as TSESTree.ExpressionStatement).expression as TSESTree.CallExpression;

        // Do not flag core.error calls that carry annotation properties (e.g.
        // core.error(msg, { title: "..." })). The second argument provides
        // diagnostic context that is not duplicated by setFailed.
        if (hasAnnotationProperties(errorCall)) continue;

        // Only report when the message arguments are provably equivalent.
        // Two shapes are recognised:
        //   1. Exact source-text match — the classic identical-message case.
        //   2. Prefixed match — setFailedArg is a TemplateLiteral whose
        //      content ends with the full content of errorArg (also a
        //      TemplateLiteral), with only a non-empty prefix added in front
        //      (e.g. `${ERR_CODE}: ${msg}` vs `${msg}`). This is the dominant
        //      anti-pattern in the codebase: core.error duplicates the same
        //      annotation that core.setFailed already emits, just without the
        //      leading error-code prefix.
        // Calls where setFailed adds suffix or interleaved context are NOT
        // flagged, since they may carry information not duplicated by setFailed.
        const errorArg = getFirstNonSpreadArg(errorCall);
        const setFailedArg = getFirstNonSpreadArg(setFailedCall);
        if (errorArg === null || setFailedArg === null) continue;
        const exactMatch = sourceCode.getText(errorArg) === sourceCode.getText(setFailedArg);
        if (!exactMatch && !isSetFailedArgPrefixedVersion(errorArg, setFailedArg, sourceCode) && !isSetFailedArgStringConcatPrefixedVersion(errorArg, setFailedArg, sourceCode)) continue;

        // The auto-remove suggestion is semantics-preserving only when the shared
        // argument is provably side-effect-free. For example,
        // `core.error(nextMessage()); core.setFailed(nextMessage())` must not have
        // the first call silently removed because that would drop a side-effectful
        // function invocation.
        const safeToFix = isSideEffectFree(errorArg);

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
        // Filter out all module declarations (ImportDeclaration, ExportAllDeclaration,
        // ExportNamedDeclaration, ExportDefaultDeclaration) — they are not Statements
        // and their type assertion would be incorrect. BlockStatement visitor handles
        // the bodies of any exported function/class declarations separately.
        const stmts = node.body.filter(
          (s): s is TSESTree.Statement =>
            s.type !== AST_NODE_TYPES.ImportDeclaration && s.type !== AST_NODE_TYPES.ExportAllDeclaration && s.type !== AST_NODE_TYPES.ExportNamedDeclaration && s.type !== AST_NODE_TYPES.ExportDefaultDeclaration
        );
        checkStatements(stmts);
      },
    };
  },
});
