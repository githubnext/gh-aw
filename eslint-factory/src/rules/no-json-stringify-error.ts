import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

interface ErrorScope {
  varName: string;
  isSentinel: boolean;
}

function isCatchCallback(node: TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression): boolean {
  const parent = node.parent;
  if (!parent || parent.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = parent.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return false;
  const prop = callee.property;
  return prop.type === AST_NODE_TYPES.Identifier && prop.name === "catch" && parent.arguments[0] === node;
}

export const noJsonStringifyErrorRule = createRule({
  name: "no-json-stringify-error",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description: "Disallow JSON.stringify() on caught error variables — Error properties (message, stack, etc.) are non-enumerable and produce {} silently",
    },
    schema: [],
    messages: {
      jsonStringifyError:
        "JSON.stringify({{errorVar}}) produces {} for Error objects — Error properties (message, stack, etc.) are non-enumerable. Use getErrorMessage({{errorVar}}) or serialize explicitly: JSON.stringify({ message: {{errorVar}}.message, stack: {{errorVar}}.stack }).",
      useGetErrorMessage: "Replace with getErrorMessage({{errorVar}}) for safe error message extraction.",
    },
  },
  defaultOptions: [],
  create(context) {
    // Stack tracking caught error variable names.
    // Each scope entry holds varName (empty string for sentinel) and isSentinel flag.
    const scopeStack: ErrorScope[] = [];

    function getCaughtVarNames(): Set<string> {
      const names = new Set<string>();
      for (const scope of scopeStack) {
        if (!scope.isSentinel && scope.varName) {
          names.add(scope.varName);
        }
      }
      return names;
    }

    function enterFunction(node: TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression): void {
      if (isCatchCallback(node)) {
        const params = node.params;
        if (params.length === 1 && params[0].type === AST_NODE_TYPES.Identifier) {
          scopeStack.push({ varName: params[0].name, isSentinel: false });
        } else {
          // No-param or destructuring: push sentinel so outer frames are not affected
          scopeStack.push({ varName: "", isSentinel: true });
        }
      } else {
        // Non-.catch() function: sentinel to avoid false positives from shadowed names
        scopeStack.push({ varName: "", isSentinel: true });
      }
    }

    function exitFunction(): void {
      scopeStack.pop();
    }

    return {
      // Track catch clause parameters
      CatchClause(node) {
        const param = node.param;
        if (!param || param.type !== AST_NODE_TYPES.Identifier) {
          scopeStack.push({ varName: "", isSentinel: true });
        } else {
          scopeStack.push({ varName: param.name, isSentinel: false });
        }
      },
      "CatchClause:exit"() {
        scopeStack.pop();
      },

      // Track .catch() callback parameters
      ArrowFunctionExpression: enterFunction,
      "ArrowFunctionExpression:exit": exitFunction,
      FunctionExpression: enterFunction,
      "FunctionExpression:exit": exitFunction,

      // Detect JSON.stringify(caughtErrorVar, ...)
      CallExpression(node) {
        if (scopeStack.length === 0) return;

        const caughtNames = getCaughtVarNames();
        if (caughtNames.size === 0) return;

        const callee = node.callee;
        if (callee.type !== AST_NODE_TYPES.MemberExpression) return;
        if (callee.computed) return;
        const obj = callee.object;
        const prop = callee.property;
        if (obj.type !== AST_NODE_TYPES.Identifier || obj.name !== "JSON") return;
        if (prop.type !== AST_NODE_TYPES.Identifier || prop.name !== "stringify") return;

        const firstArg = node.arguments[0];
        if (!firstArg || firstArg.type !== AST_NODE_TYPES.Identifier) return;
        if (!caughtNames.has(firstArg.name)) return;

        const errorVar = firstArg.name;
        context.report({
          node,
          messageId: "jsonStringifyError",
          data: { errorVar },
          suggest: [
            {
              messageId: "useGetErrorMessage" as const,
              data: { errorVar },
              fix(fixer) {
                return fixer.replaceText(node, `getErrorMessage(${errorVar})`);
              },
            },
          ],
        });
      },
    };
  },
});
