import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type AsyncFuncNode = TSESTree.FunctionDeclaration | TSESTree.FunctionExpression | TSESTree.ArrowFunctionExpression;
type SourceCodeScope = TSESLint.Scope.Scope;
type FunctionDeclarationDefinition = TSESLint.Scope.Definitions.FunctionNameDefinition & { node: TSESTree.FunctionDeclaration };
type VariableDefinition = TSESLint.Scope.Definitions.VariableDefinition;

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

/** Walks a chained call expression to find the root identifier node. */
function getRootCallIdentifier(node: TSESTree.CallExpression): TSESTree.Identifier | null {
  const callee = node.callee;
  if (callee.type === AST_NODE_TYPES.Identifier) {
    return callee;
  }
  if (callee.type === AST_NODE_TYPES.MemberExpression) {
    const obj = callee.object;
    if (obj.type === AST_NODE_TYPES.CallExpression) {
      return getRootCallIdentifier(obj);
    }
  }
  return null;
}

function isFunctionDeclarationDefinition(definition: TSESLint.Scope.Definition): definition is FunctionDeclarationDefinition {
  return definition.type === TSESLint.Scope.DefinitionType.FunctionName && definition.node.type === AST_NODE_TYPES.FunctionDeclaration;
}

function isVariableDefinition(definition: TSESLint.Scope.Definition): definition is VariableDefinition {
  return definition.type === TSESLint.Scope.DefinitionType.Variable && definition.node.type === AST_NODE_TYPES.VariableDeclarator;
}

function isModuleScopeVariableDeclaration(node: TSESTree.VariableDeclaration): boolean {
  return node.parent.type === AST_NODE_TYPES.Program || (node.parent.type === AST_NODE_TYPES.ExportNamedDeclaration && node.parent.parent.type === AST_NODE_TYPES.Program);
}

function isAsyncVariableEntrypoint(definition: VariableDefinition): boolean {
  const declaration = definition.parent;
  if (!declaration || !isModuleScopeVariableDeclaration(declaration)) return false;
  const init = definition.node.init;
  return (init?.type === AST_NODE_TYPES.FunctionExpression || init?.type === AST_NODE_TYPES.ArrowFunctionExpression) && init.async;
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

    /** Returns true when the identifier resolves to a module-scope async entrypoint. */
    function isAsyncModuleScopeEntrypoint(identifier: TSESTree.Identifier): boolean {
      let scope: SourceCodeScope | null = sourceCode.getScope(identifier);

      while (scope) {
        const variable = scope.set.get(identifier.name);
        const functionDeclarationDefinition = variable?.defs.find(isFunctionDeclarationDefinition);
        if (functionDeclarationDefinition) {
          return functionDeclarationDefinition.node.async && functionDeclarationDefinition.node.parent.type === AST_NODE_TYPES.Program;
        }

        const variableDefinition = variable?.defs.find(isVariableDefinition);
        if (variableDefinition) {
          return isAsyncVariableEntrypoint(variableDefinition);
        }
        if (variable && variable.defs.length > 0) {
          return false;
        }

        scope = scope.upper;
      }

      return false;
    }

    return {
      // Flag bare calls: ExpressionStatement whose expression is a direct CallExpression
      // to a tracked module-scope async function/variable entrypoint, and that are not inside an async function body
      // (where `await` would be the right fix instead).
      "ExpressionStatement > CallExpression"(node: TSESTree.CallExpression) {
        const callee = node.callee;
        let rootIdentifier: TSESTree.Identifier | null = null;

        if (callee.type === AST_NODE_TYPES.Identifier) {
          rootIdentifier = callee;
        } else if (callee.type === AST_NODE_TYPES.MemberExpression) {
          // Chained call: main().then(...) etc.
          // If the chain contains .catch(...), it's handled — skip.
          if (chainHasCatch(node)) return;
          rootIdentifier = getRootCallIdentifier(node);
        }

        if (!rootIdentifier) return;
        const name = rootIdentifier.name;
        if (!isAsyncModuleScopeEntrypoint(rootIdentifier)) return;

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
