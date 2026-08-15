import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { getDynamicCommandKind } from "./command-initializer-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);
type ExecMethodName = "exec" | "getExecOutput";

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

    return {
      CallExpression(node) {
        const method = resolveExecMethod(node);
        if (!method) return;

        const firstArg = node.arguments[0];
        if (!firstArg || firstArg.type === AST_NODE_TYPES.SpreadElement) return;

        const kind = getDynamicCommandKind(firstArg as TSESTree.Expression, sourceCode);
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
