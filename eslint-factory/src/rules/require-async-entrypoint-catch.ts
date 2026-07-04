import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type AsyncFuncNode = TSESTree.FunctionDeclaration | TSESTree.FunctionExpression | TSESTree.ArrowFunctionExpression;

function isAsyncFuncNode(node: TSESTree.Node): node is AsyncFuncNode {
  return node.type === AST_NODE_TYPES.FunctionDeclaration || node.type === AST_NODE_TYPES.FunctionExpression || node.type === AST_NODE_TYPES.ArrowFunctionExpression;
}

/** Returns true if any call in the chain is `.catch(...)`. */
function chainHasCatch(node: TSESTree.CallExpression): boolean {
  const callee = node.callee;
  if (callee.type === AST_NODE_TYPES.MemberExpression) {
    const prop = callee.property;
    if (prop.type === AST_NODE_TYPES.Identifier && prop.name === "catch") {
      return true;
    }
    const obj = callee.object;
    if (obj.type === AST_NODE_TYPES.CallExpression) {
      return chainHasCatch(obj);
    }
  }
  return false;
}

/**
 * Walks a chained call expression to find the root identifier name.
 * e.g. for `main().then(cb)`, returns "main".
 * Returns null if the root call is not a simple Identifier call.
 */
function getRootCallName(node: TSESTree.CallExpression): string | null {
  const callee = node.callee;
  if (callee.type === AST_NODE_TYPES.Identifier) {
    return callee.name;
  }
  if (callee.type === AST_NODE_TYPES.MemberExpression) {
    const obj = callee.object;
    if (obj.type === AST_NODE_TYPES.CallExpression) {
      return getRootCallName(obj);
    }
  }
  return null;
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

      // Collect module-scope async function expressions and arrow functions:
      // const/let/var X = async function() {} or X = async () => {}
      VariableDeclaration(node) {
        const isModuleScope = node.parent.type === AST_NODE_TYPES.Program || (node.parent.type === AST_NODE_TYPES.ExportNamedDeclaration && node.parent.parent.type === AST_NODE_TYPES.Program);
        if (!isModuleScope) return;
        for (const declarator of node.declarations) {
          if (
            declarator.id.type === AST_NODE_TYPES.Identifier &&
            declarator.init !== null &&
            declarator.init !== undefined &&
            (declarator.init.type === AST_NODE_TYPES.FunctionExpression || declarator.init.type === AST_NODE_TYPES.ArrowFunctionExpression) &&
            declarator.init.async
          ) {
            asyncFunctionNames.add(declarator.id.name);
          }
        }
      },

      // Flag bare calls: ExpressionStatement whose expression is a direct CallExpression
      // to a tracked async function, and that are not inside an async function body
      // (where `await` would be the right fix instead).
      "ExpressionStatement > CallExpression"(node: TSESTree.CallExpression) {
        const callee = node.callee;

        let name: string | null = null;

        if (callee.type === AST_NODE_TYPES.Identifier) {
          // Bare call: main()
          name = callee.name;
        } else if (callee.type === AST_NODE_TYPES.MemberExpression) {
          // Chained call: main().then(...) etc.
          // If the chain contains .catch(...), it's handled — skip.
          if (chainHasCatch(node)) return;
          // Otherwise find the root call name.
          name = getRootCallName(node);
        }

        if (!name || !asyncFunctionNames.has(name)) return;

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
