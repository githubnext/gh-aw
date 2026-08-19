import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { getDynamicCommandKind } from "./command-initializer-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);
type ExecMethodName = "exec" | "getExecOutput";

/**
 * Returns true when `identifier` is bound as a function parameter (as opposed
 * to a variable, import, or global). Used to recognize the common
 * `execApi` / `execAlias` parameter-alias convention for the injected
 * `exec.exec` / `exec.getExecOutput` API without colliding with unrelated
 * receivers such as `RegExp.prototype.exec`.
 */
function isFunctionParameter(identifier: TSESTree.Identifier, sourceCode: TSESLint.SourceCode): boolean {
  const scope = sourceCode.getScope(identifier);
  let current: TSESLint.Scope.Scope | null = scope;
  while (current !== null) {
    const variable = current.set.get(identifier.name);
    if (variable !== undefined) {
      return variable.defs.length === 1 && variable.defs[0].type === "Parameter";
    }
    current = current.upper;
  }
  return false;
}

/**
 * Returns true when the call's second argument is shaped like the exec API's
 * `args` array parameter: an array literal, or an identifier (assumed to hold
 * an array). `RegExp.prototype.exec(str)` takes exactly one string argument
 * and never a second array-shaped argument, so this shape is a safe
 * disambiguator for the parameter-alias case.
 */
function hasArrayShapedSecondArgument(node: TSESTree.CallExpression): boolean {
  const secondArg = node.arguments[1];
  if (!secondArg) return false;
  return secondArg.type === AST_NODE_TYPES.ArrayExpression || secondArg.type === AST_NODE_TYPES.Identifier;
}

/**
 * Returns true when the call expression looks like `exec.exec(...)` or
 * `exec.getExecOutput(...)` — the `exec` global injected by github-script —
 * or the common `execApi.exec(...)` / `execApi.getExecOutput(...)`
 * parameter-alias convention used to thread the exec API through helper
 * functions.
 *
 * Recognized shapes:
 *   exec.exec(cmd, args?, opts?)
 *   exec.getExecOutput(cmd, args?, opts?)
 *   <param>.exec(cmd, args, opts?)            (param is a function parameter, args is array-shaped)
 *   <param>.getExecOutput(cmd, args, opts?)   (param is a function parameter, args is array-shaped)
 *
 * The parameter-alias shape additionally requires a second, array-shaped
 * argument so it cannot collide with `RegExp.prototype.exec(str)`, which
 * takes exactly one string argument.
 */
function resolveExecMethod(node: TSESTree.CallExpression, sourceCode: TSESLint.SourceCode): ExecMethodName | null {
  const callee = node.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return null;
  const obj = callee.object;
  const prop = callee.property;
  if (obj.type !== AST_NODE_TYPES.Identifier) return null;
  if (prop.type !== AST_NODE_TYPES.Identifier) return null;
  const methodName = prop.name === "exec" || prop.name === "getExecOutput" ? prop.name : null;
  if (!methodName) return null;

  if (obj.name === "exec") return methodName;

  // Parameter-alias convention (e.g. execApi.exec(...)): only recognized
  // when the receiver is a function parameter and the call has an
  // array-shaped second argument, ruling out RegExp.prototype.exec(str).
  if (isFunctionParameter(obj, sourceCode) && hasArrayShapedSecondArgument(node)) return methodName;

  return null;
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
        const method = resolveExecMethod(node, sourceCode);
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
