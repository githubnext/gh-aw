import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

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
    return {
      CallExpression(node) {
        const method = resolveExecMethod(node);
        if (!method) return;

        const firstArg = node.arguments[0];
        if (!firstArg || firstArg.type === AST_NODE_TYPES.SpreadElement) return;

        const kind = getDynamicCommandKind(firstArg);
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
