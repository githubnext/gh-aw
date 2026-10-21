import { ASTUtils, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

function resolveCallback(callback: TSESTree.Node, sourceCode: TSESLint.SourceCode): TSESTree.Node | null {
  if (callback.type !== "Identifier") return callback;
  const variable = ASTUtils.findVariable(sourceCode.getScope(callback), callback);
  if (!variable || variable.defs.length !== 1 || variable.references.some(ref => ref.isWrite() && !ref.init)) return null;
  const definition = variable.defs[0];
  if (definition.type === "FunctionName" && definition.node.type === "FunctionDeclaration") return definition.node;
  if (definition.type === "Variable" && definition.parent.kind === "const" && definition.node.id.type === "Identifier") return definition.node.init;
  return null;
}

export const noAsyncForEachCallbackRule = createRule({
  name: "no-async-foreach-callback",
  meta: {
    type: "problem",
    docs: {
      description: "Disallow async callbacks passed to Array.prototype.forEach(); the returned promises are discarded, so awaits are not sequenced and rejections go unhandled.",
    },
    schema: [],
    messages: {
      asyncForEach: "forEach() ignores the promise returned by an async callback, so work is not awaited and rejections are unhandled. Use `for (const x of items) { await ... }` or `await Promise.all(items.map(async ...))`.",
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      CallExpression(node: TSESTree.CallExpression) {
        const callee = node.callee;
        if (callee.type !== "MemberExpression") {
          return;
        }
        const isForEach = callee.computed ? callee.property.type === "Literal" && callee.property.value === "forEach" : callee.property.type === "Identifier" && callee.property.name === "forEach";
        if (!isForEach) return;
        const callback = node.arguments[0];
        if (!callback) return;
        const resolved = resolveCallback(callback, context.sourceCode);
        if (!resolved || (resolved.type !== "ArrowFunctionExpression" && resolved.type !== "FunctionExpression" && resolved.type !== "FunctionDeclaration") || !resolved.async) {
          return;
        }
        context.report({ node: callback, messageId: "asyncForEach" });
      },
    };
  },
});
