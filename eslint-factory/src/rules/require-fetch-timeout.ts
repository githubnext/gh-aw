import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/**
 * Returns true when the fetch call's options object argument (if any) carries
 * a `signal` property, which is how callers wire in `AbortSignal.timeout(...)`
 * or an `AbortController`-backed abort deadline.
 */
function hasSignalOption(callExpression: TSESTree.CallExpression): boolean {
  const optionsArg = callExpression.arguments[1];
  if (!optionsArg) return false;

  // Spread arguments (`fetch(url, ...opts)`) can't be statically inspected;
  // assume the caller may have included a signal to avoid false positives.
  if (optionsArg.type === AST_NODE_TYPES.SpreadElement) return true;

  if (optionsArg.type === AST_NODE_TYPES.ObjectExpression) {
    for (const prop of optionsArg.properties) {
      if (prop.type === AST_NODE_TYPES.SpreadElement) return true;
      if (prop.type === AST_NODE_TYPES.Property) {
        if (!prop.computed && prop.key.type === AST_NODE_TYPES.Identifier && prop.key.name === "signal") return true;
        if (!prop.computed && prop.key.type === AST_NODE_TYPES.Literal && prop.key.value === "signal") return true;
      }
    }
    return false;
  }

  // Options passed as an identifier/expression (e.g. a shared config object)
  // can't be statically inspected; assume it may already carry a signal.
  return true;
}

export const requireFetchTimeoutRule = createRule({
  name: "require-fetch-timeout",
  meta: {
    type: "problem",
    docs: {
      description: "Require fetch() calls in actions/setup/js scripts to pass an abort signal so requests cannot hang indefinitely in CI",
    },
    schema: [],
    messages: {
      requireSignal: "fetch() call has no `signal` option. Pass `signal: AbortSignal.timeout(<ms>)` (or an AbortController-backed signal) so a stalled network request cannot hang the job indefinitely.",
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      CallExpression(node: TSESTree.CallExpression) {
        const callee = node.callee;
        if (callee.type !== AST_NODE_TYPES.Identifier || callee.name !== "fetch") return;

        if (hasSignalOption(node)) return;

        context.report({ node, messageId: "requireSignal" });
      },
    };
  },
});
