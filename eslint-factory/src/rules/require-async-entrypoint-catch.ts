import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type AsyncFuncNode = TSESTree.FunctionDeclaration | TSESTree.FunctionExpression | TSESTree.ArrowFunctionExpression;

function isAsyncFuncNode(node: TSESTree.Node): node is AsyncFuncNode {
  return node.type === AST_NODE_TYPES.FunctionDeclaration || node.type === AST_NODE_TYPES.FunctionExpression || node.type === AST_NODE_TYPES.ArrowFunctionExpression;
}

export const requireAsyncEntrypointCatchRule = createRule({
  name: "require-async-entrypoint-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description: "Require bare calls to module-scope async functions (e.g. main()) to be chained with .catch() so that unhandled promise rejections are not silently swallowed or reported without context in GitHub Actions scripts.",
    },
    schema: [],
    messages: {
      requireCatch: "Bare call to async function '{{name}}()' outside an async context will produce an unhandled rejection if it rejects. Chain .catch(err => { ... }) to handle errors explicitly.",
      addCatch: "Chain .catch(err => { console.error(err); process.exitCode = 1; }) to handle rejections explicitly. Replace the handler with project-specific failure reporting as appropriate.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    // Names of async functions declared in this module.
    const asyncFunctionNames = new Set<string>();

    /** Returns true if the node is inside an async function body (making `await` available). */
    function isInsideAsyncFunction(node: TSESTree.Node): boolean {
      const ancestors = sourceCode.getAncestors(node);
      for (let i = ancestors.length - 1; i >= 0; i -= 1) {
        const ancestor = ancestors[i];
        if (isAsyncFuncNode(ancestor)) {
          return ancestor.async;
        }
      }
      return false;
    }

    return {
      // Collect module-scope async function declarations.
      FunctionDeclaration(node) {
        if (node.async && node.id?.name && node.parent.type === AST_NODE_TYPES.Program) {
          asyncFunctionNames.add(node.id.name);
        }
      },

      // Flag bare calls: ExpressionStatement whose expression is a direct CallExpression
      // to a tracked async function, and that are not inside an async function body
      // (where `await` would be the right fix instead).
      "ExpressionStatement > CallExpression"(node: TSESTree.CallExpression) {
        const callee = node.callee;

        // Only flag simple identifier calls: main(), run(), etc.
        if (callee.type !== AST_NODE_TYPES.Identifier) return;
        const name = callee.name;
        if (!asyncFunctionNames.has(name)) return;

        // Inside an async context the caller can (and should) use `await fn()` instead.
        if (isInsideAsyncFunction(node)) return;

        context.report({
          node,
          messageId: "requireCatch",
          data: { name },
          suggest: [
            {
              messageId: "addCatch",
              fix(fixer: TSESLint.RuleFixer) {
                return fixer.insertTextAfter(node, ".catch(err => { console.error(err); process.exitCode = 1; })");
              },
            },
          ],
        });
      },
    };
  },
});
