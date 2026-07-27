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

        // Only report when the message arguments are provably equivalent (same
        // source text). Calls with different messages are not redundant — the
        // core.error call may log extra context (file names, sizes, stack frames)
        // that setFailed does not repeat.
        const errorArg = getFirstNonSpreadArg(errorCall);
        const setFailedArg = getFirstNonSpreadArg(setFailedCall);
        if (errorArg === null || setFailedArg === null) continue;
        if (sourceCode.getText(errorArg) !== sourceCode.getText(setFailedArg)) continue;

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
