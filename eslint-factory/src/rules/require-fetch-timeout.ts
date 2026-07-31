import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/**
 * Returns true when the fetch call's options object argument (if any) carries
 * a `signal` property, which is how callers wire in `AbortSignal.timeout(...)`
 * or an `AbortController`-backed abort deadline.
 */
function isStaticallyNullish(node: TSESTree.Node): boolean {
  if (node.type === AST_NODE_TYPES.Literal && node.value == null) return true;
  if (node.type === AST_NODE_TYPES.Identifier && node.name === "undefined") return true;
  if (node.type === AST_NODE_TYPES.UnaryExpression && node.operator === "void") return true;
  return false;
}

function hasSignalOption(callExpression: TSESTree.CallExpression): boolean {
  const optionsArg = callExpression.arguments[1];
  if (!optionsArg) return false;
  if (isStaticallyNullish(optionsArg)) return false;

  // Spread arguments (`fetch(url, ...opts)`) can't be statically inspected;
  // assume the caller may have included a signal to avoid false positives.
  if (optionsArg.type === AST_NODE_TYPES.SpreadElement) return true;

  if (optionsArg.type === AST_NODE_TYPES.ObjectExpression) {
    for (const prop of optionsArg.properties) {
      if (prop.type === AST_NODE_TYPES.SpreadElement) return true;
      if (prop.type === AST_NODE_TYPES.Property) {
        const isSignalProp = (!prop.computed && prop.key.type === AST_NODE_TYPES.Identifier && prop.key.name === "signal") || (!prop.computed && prop.key.type === AST_NODE_TYPES.Literal && prop.key.value === "signal");
        if (!isSignalProp) continue;

        if (!isStaticallyNullish(prop.value)) return true;
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
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

    function hasLocalBinding(node: TSESTree.Node, name: string): boolean {
      let scope: SourceCodeScope | null = sourceCode.getScope(node);
      while (scope) {
        const variable = scope.set.get(name);
        if (variable?.defs.length) return true;
        scope = scope.upper;
      }
      return false;
    }

    function isGlobalFetchCall(callee: TSESTree.Expression): callee is TSESTree.MemberExpression {
      if (callee.type !== AST_NODE_TYPES.MemberExpression) return false;

      const propertyName =
        !callee.computed && callee.property.type === AST_NODE_TYPES.Identifier
          ? callee.property.name
          : callee.computed && callee.property.type === AST_NODE_TYPES.Literal && typeof callee.property.value === "string"
            ? callee.property.value
            : null;
      if (propertyName !== "fetch") return false;

      return callee.object.type === AST_NODE_TYPES.Identifier && (callee.object.name === "globalThis" || callee.object.name === "global");
    }

    return {
      CallExpression(node: TSESTree.CallExpression) {
        const callee = node.callee;
        const isBareFetch = callee.type === AST_NODE_TYPES.Identifier && callee.name === "fetch";
        const isMemberFetch = isGlobalFetchCall(callee);
        if (!isBareFetch && !isMemberFetch) return;
        if (isBareFetch && hasLocalBinding(node, "fetch")) return;
        if (isMemberFetch && callee.object.type === AST_NODE_TYPES.Identifier && hasLocalBinding(node, callee.object.name)) return;

        if (hasSignalOption(node)) return;

        context.report({ node, messageId: "requireSignal" });
      },
    };
  },
});
