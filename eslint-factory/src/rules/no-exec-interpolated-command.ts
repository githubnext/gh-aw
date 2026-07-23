import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);
type ExecMethodName = "exec" | "getExecOutput";

/**
 * Returns true when the node is a purely static expression (no runtime
 * interpolation): a literal, a no-expression template literal, or a binary
 * `+` of two static expressions.
 */
function isStaticExpression(node: TSESTree.Expression): boolean {
  if (node.type === "Literal") return true;
  if (node.type === "TemplateLiteral") return node.expressions.length === 0;
  if (node.type === "BinaryExpression" && node.operator === "+") {
    return isStaticExpression(node.left) && isStaticExpression(node.right);
  }
  return false;
}

/**
 * Returns true when the node is a dynamic string concatenation (binary `+`
 * that is not entirely static).
 */
function isDynamicStringConcatenation(node: TSESTree.Expression): boolean {
  return node.type === "BinaryExpression" && node.operator === "+" && !isStaticExpression(node);
}

/**
 * Returns the display kind string for the problematic first argument, or null
 * when the argument is not one of the flagged shapes.
 */
function getDynamicCommandKind(node: TSESTree.Expression): string | null {
  if (node.type === "TemplateLiteral" && node.expressions.length > 0) return "interpolated template literal";
  if (isDynamicStringConcatenation(node)) return "dynamic string concatenation";
  return null;
}

/**
 * Returns true when the call expression looks like `exec.exec(...)` or
 * `exec.getExecOutput(...)` — the `exec` global injected by github-script.
 *
 * Recognized shapes:
 *   exec.exec(cmd, args?, opts?)
 *   exec.getExecOutput(cmd, args?, opts?)
 *
 * This rule intentionally matches only the `exec` global injected by
 * github-script in CommonJS action scripts.
 */
function resolveExecMethod(node: TSESTree.CallExpression): ExecMethodName | null {
  const callee = node.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return null;
  const obj = callee.object;
  const prop = callee.property;
  if (obj.type !== AST_NODE_TYPES.Identifier || obj.name !== "exec") return null;
  if (prop.type !== AST_NODE_TYPES.Identifier) return null;
  return prop.name === "exec" || prop.name === "getExecOutput" ? prop.name : null;
}

export const noExecInterpolatedCommandRule = createRule({
  name: "no-exec-interpolated-command",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow interpolated template literals or dynamic string concatenation as the first (command) argument of github-script's injected exec.exec() or exec.getExecOutput() calls in CommonJS action scripts. " +
        "The @actions/exec runner splits the command string by spaces internally; variables containing spaces silently break argument boundaries. " +
        "Pass a static command string and put all arguments in the second array parameter instead: exec.exec('git', [arg1, arg2]).",
    },
    schema: [],
    messages: {
      interpolatedCommand:
        "Avoid passing a {{kind}} as the exec command — @actions/exec splits the command string by spaces, so values containing spaces silently break argument boundaries. " +
        "Use a static command string and pass all arguments in the args array, preserving the current method: exec.{{method}}('git', ['checkout', branchName]).",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    /**
     * When `identifier` is a write-once local variable binding, returns its
     * initializer expression so the caller can apply further checks.  Returns
     * null for parameters, imports, multiply-assigned vars, and vars with no
     * initializer.
     */
    function resolveInitializer(identifier: TSESTree.Identifier): TSESTree.Expression | null {
      const startScope = sourceCode.getScope(identifier);
      const functionScope = startScope.variableScope;
      // Only resolve within a concrete function boundary (function declaration,
      // function expression, or arrow function). Module/global scopes are
      // intentionally skipped because those bindings are not a stable proxy for
      // runtime values at call time.
      if (functionScope.type !== "function") return null;

      let scope: TSESLint.Scope.Scope | null = startScope;
      // Stay inside the same function's nested block scopes; do not cross to
      // enclosing function/module scopes.
      while (scope !== null && scope.variableScope === functionScope) {
        const variable = scope.set.get(identifier.name);
        if (variable !== undefined) {
          // Only accept simple, single-definition Variable bindings.
          if (variable.defs.length !== 1) return null;
          const def = variable.defs[0];
          if (def.type !== "Variable") return null;
          // Reject re-assigned bindings (write references that are not the initializer).
          if (variable.references.some(ref => ref.isWrite() && !ref.init)) return null;
          const declarator = def.node as TSESTree.VariableDeclarator;
          return declarator.init ?? null;
        }
        scope = scope.upper;
      }
      return null;
    }

    return {
      CallExpression(node) {
        const method = resolveExecMethod(node);
        if (!method) return;

        const firstArg = node.arguments[0];
        if (!firstArg || firstArg.type === AST_NODE_TYPES.SpreadElement) return;

        let candidate: TSESTree.Expression = firstArg as TSESTree.Expression;
        const seen = new Set<TSESTree.Identifier>();
        while (candidate.type === AST_NODE_TYPES.Identifier && !seen.has(candidate)) {
          seen.add(candidate);
          const resolved = resolveInitializer(candidate);
          if (!resolved) break;
          candidate = resolved;
        }

        const kind = getDynamicCommandKind(candidate);
        if (!kind) return;

        context.report({
          node: firstArg,
          messageId: "interpolatedCommand",
          data: { kind, method },
        });
      },
    };
  },
});
