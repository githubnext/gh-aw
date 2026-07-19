import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/**
 * Returns true when the node is a template literal that contains at least one
 * interpolated expression.
 */
function isInterpolatedTemplateLiteral(node: TSESTree.Node): boolean {
  return node.type === "TemplateLiteral" && node.expressions.length > 0;
}

/**
 * Returns true when the node is a purely static expression (no runtime
 * interpolation): a string literal, a no-expression template literal, or a
 * binary `+` of two static expressions.
 */
function isStaticExpression(node: TSESTree.Node): boolean {
  if (node.type === "Literal") return typeof node.value === "string";
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
function isDynamicStringConcatenation(node: TSESTree.Node): boolean {
  return node.type === "BinaryExpression" && node.operator === "+" && !isStaticExpression(node);
}

/**
 * Returns the display kind string for the problematic first argument, or null
 * when the argument is not one of the flagged shapes.
 */
function getDynamicCommandKind(node: TSESTree.Node): string | null {
  if (isInterpolatedTemplateLiteral(node)) return "interpolated template literal";
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
 */
function isExecCall(node: TSESTree.CallExpression): boolean {
  const callee = node.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression) return false;
  if (callee.computed) return false;
  const obj = callee.object;
  const prop = callee.property;
  if (obj.type !== AST_NODE_TYPES.Identifier || obj.name !== "exec") return false;
  if (prop.type !== AST_NODE_TYPES.Identifier) return false;
  return prop.name === "exec" || prop.name === "getExecOutput";
}

export const noExecInterpolatedCommandRule = createRule({
  name: "no-exec-interpolated-command",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow interpolated template literals or dynamic string concatenation as the first (command) argument of exec.exec() or exec.getExecOutput(). " +
        "The @actions/exec runner splits the command string by spaces internally; variables containing spaces silently break argument boundaries. " +
        "Pass a static command string and put all arguments in the second array parameter instead: exec.exec('git', [arg1, arg2]).",
    },
    schema: [],
    messages: {
      interpolatedCommand:
        "Avoid passing a {{kind}} as the exec command — @actions/exec splits the command string by spaces, so values containing spaces silently break argument boundaries. " +
        "Use a static command string and pass all arguments in the args array: exec.exec('git', ['checkout', branchName]).",
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      CallExpression(node) {
        if (!isExecCall(node)) return;

        const firstArg = node.arguments[0];
        if (!firstArg) return;

        const kind = getDynamicCommandKind(firstArg);
        if (!kind) return;

        context.report({
          node: firstArg,
          messageId: "interpolatedCommand",
          data: { kind },
        });
      },
    };
  },
});
