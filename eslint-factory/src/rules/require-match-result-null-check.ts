import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/**
 * Returns true when the call expression is `<expr>.match(...)`, i.e. a call to
 * `String.prototype.match`, which returns `null` when the pattern does not match.
 * `matchAll` is excluded because it returns an iterator, not a nullable result.
 */
function isStringMatchCall(node: TSESTree.CallExpression): boolean {
  const { callee } = node;
  if (callee.type !== AST_NODE_TYPES.MemberExpression) return false;
  if (callee.computed) return false;
  return callee.property.type === AST_NODE_TYPES.Identifier && callee.property.name === "match";
}

export const requireMatchResultNullCheckRule = createRule({
  name: "require-match-result-null-check",
  meta: {
    type: "problem",
    docs: {
      description: "Require the result of String.prototype.match() to be null-checked (via optional chaining, a guard, or a logical fallback) before its properties are accessed, since match() returns null when the pattern does not match.",
    },
    schema: [],
    messages: {
      requireNullCheck: "The result of '{{name}}.match(...)' is accessed at '{{name}}{{access}}' without a null check. match() returns null when the pattern does not match, so this throws a TypeError on unmatched input. Use optional chaining ({{name}}?.{{accessNoDot}}), an if-guard, or a logical fallback before accessing the result.",
    },
  },
  defaultOptions: [],
  create(context) {
    // Map from variable name to its VariableDeclarator, for identifiers assigned directly from a .match(...) call.
    const matchResults = new Map<string, TSESTree.VariableDeclarator>();
    // Names confirmed to be guarded (via if-check, logical, ternary test, or optional chaining) before unguarded use.
    const guarded = new Set<string>();
    // Names that have already been reported, to avoid duplicate reports per variable.
    const reported = new Set<string>();

    function markGuard(node: TSESTree.Node): void {
      let expr = node;
      while (expr.type === AST_NODE_TYPES.UnaryExpression && expr.operator === "!") {
        expr = expr.argument;
      }
      if (expr.type === AST_NODE_TYPES.Identifier) {
        guarded.add(expr.name);
        return;
      }
      // `m?.[1]` or `m?.groups` used as a test (e.g. `if (m?.[1]) { use(m[1]); }`) — the optional
      // chain itself proves the author is aware the value may be null, so treat `m` as guarded.
      if (expr.type === AST_NODE_TYPES.ChainExpression) {
        expr = expr.expression;
      }
      if (expr.type === AST_NODE_TYPES.MemberExpression && expr.optional && expr.object.type === AST_NODE_TYPES.Identifier) {
        guarded.add(expr.object.name);
        return;
      }
      // `m === null`, `m !== null`, `m == null`, `m != null` (either operand order).
      if (expr.type === AST_NODE_TYPES.BinaryExpression && ["===", "!==", "==", "!="].includes(expr.operator)) {
        const { left, right } = expr;
        if (left.type === AST_NODE_TYPES.Identifier && right.type === AST_NODE_TYPES.Literal && right.value === null) {
          guarded.add(left.name);
        } else if (right.type === AST_NODE_TYPES.Identifier && left.type === AST_NODE_TYPES.Literal && left.value === null) {
          guarded.add(right.name);
        }
      }
    }

    return {
      VariableDeclarator(node) {
        if (node.id.type === AST_NODE_TYPES.Identifier && node.init?.type === AST_NODE_TYPES.CallExpression && isStringMatchCall(node.init)) {
          matchResults.set(node.id.name, node);
        }
      },

      IfStatement(node) {
        markGuard(node.test);
      },

      ConditionalExpression(node) {
        markGuard(node.test);
      },

      LogicalExpression(node) {
        // `result && result[1]` or `result || fallback` treat `result` as guarded on the right-hand side.
        // Also handles `result !== null && result[1]` (a BinaryExpression left operand).
        markGuard(node.left);
      },

      // Member access like `result[1]` or `result.groups` — flag only when NOT part of an
      // optional-chained ChainExpression (those are already safe).
      MemberExpression(node) {
        if (node.object.type !== AST_NODE_TYPES.Identifier) return;
        const name = node.object.name;
        const declarator = matchResults.get(name);
        if (!declarator) return;
        if (node.optional) return; // `result?.[1]` is safe.
        if (guarded.has(name)) return;
        if (reported.has(name)) return;

        // Skip if this MemberExpression is nested inside a ChainExpression with optional links
        // (e.g. `result?.match ... `) — handled above via node.optional already covering direct case.

        const access = node.computed ? `[${context.sourceCode.getText(node.property)}]` : `.${(node.property as TSESTree.Identifier).name}`;
        reported.add(name);
        context.report({
          node,
          messageId: "requireNullCheck",
          data: { name, access, accessNoDot: access.startsWith(".") ? access.slice(1) : access },
        });
      },
    };
  },
});
