import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

interface ErrorScope {
  varName: string;
  isSentinel: boolean;
}

/**
 * Returns true when the function node is an inline rejection handler passed to
 * a promise method (.catch(fn) or .then(onFulfilled, onRejected)).
 */
function isInlineRejectionHandler(node: TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression): boolean {
  const parent = node.parent;
  if (!parent || parent.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = parent.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return false;
  const prop = callee.property;
  if (prop.type !== AST_NODE_TYPES.Identifier) return false;
  if (prop.name === "catch" && parent.arguments[0] === node) return true;
  if (prop.name === "then" && parent.arguments[1] === node) return true;
  return false;
}

/**
 * Returns true when `node` is a `TemplateLiteral` expression inside a
 * `TemplateLiteral` expression. Used to identify `${someVar}` directly
 * interpolated (as opposed to `${someVar.message}` or `${fn(someVar)}`).
 *
 * A "bare interpolation" is a `TSESTree.TemplateElement` expression where the
 * expression is exactly an `Identifier` — no member access, no call, no
 * unary/binary operation, no nullish coercion.
 */
function isBareIdentifierExpression(node: TSESTree.Expression): node is TSESTree.Identifier {
  return node.type === AST_NODE_TYPES.Identifier;
}

export const noCaughtErrorInterpolationRule = createRule({
  name: "no-caught-error-interpolation",
  meta: {
    type: "suggestion",
    hasSuggestions: true,
    docs: {
      description:
        "Disallow directly interpolating a caught error variable in a template literal (e.g. `${err}`). " +
        "For Error objects this produces the redundant 'Error: message' prefix; for non-Error throws (plain objects, strings, etc.) " +
        "it silently produces '[object Object]' or another useless string. " +
        "Use getErrorMessage(err) for consistent, safe formatting, or String(err) when getErrorMessage is unavailable. " +
        "Detected scopes: try/catch bindings, .catch(fn) inline callbacks, and .then(onFulfilled, onRejected) inline rejection handlers.",
    },
    schema: [],
    messages: {
      bareErrorInterpolation:
        "Directly interpolating caught error '{{errorVar}}' in a template literal is unsafe — " +
        "for Error objects it produces 'Error: message' (redundant prefix); for non-Error throws it produces '[object Object]'. " +
        "Use ${getErrorMessage({{errorVar}})} if it is available, or ${String({{errorVar}})} as an import-free alternative.",
      useGetErrorMessage: "Replace \\${{{errorVar}}} with \\${getErrorMessage({{errorVar}})} — ensure getErrorMessage is imported from error_helpers.cjs.",
      useStringFallback: "Replace \\${{{errorVar}}} with \\${String({{errorVar}})} — getErrorMessage is not in scope; String() is a safe import-free alternative.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

    const scopeStack: ErrorScope[] = [];

    function getCaughtVarNames(): Set<string> {
      const names = new Set<string>();
      for (let i = scopeStack.length - 1; i >= 0; i--) {
        const scope = scopeStack[i];
        if (scope.isSentinel) break;
        if (scope.varName) names.add(scope.varName);
      }
      return names;
    }

    function enterFunction(node: TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression): void {
      if (isInlineRejectionHandler(node)) {
        const params = node.params;
        if (params.length === 1 && params[0].type === AST_NODE_TYPES.Identifier) {
          scopeStack.push({ varName: params[0].name, isSentinel: false });
        } else {
          scopeStack.push({ varName: "", isSentinel: true });
        }
      } else {
        scopeStack.push({ varName: "", isSentinel: true });
      }
    }

    function exitFunction(): void {
      scopeStack.pop();
    }

    function isDefinitionAvailableAtNode(definition: TSESLint.Scope.Definition, node: TSESTree.Node): boolean {
      if (definition.type === "ImportBinding" || definition.type === "FunctionName") {
        return true;
      }
      const definitionNode = definition.name ?? definition.node;
      return definitionNode.range[0] < node.range[0];
    }

    function hasResolvableLocalBinding(node: TSESTree.Node, name: string): boolean {
      let scope: SourceCodeScope | null = sourceCode.getScope(node);
      while (scope) {
        const variable = scope.set.get(name);
        if (variable && variable.defs.some(def => isDefinitionAvailableAtNode(def, node))) {
          return true;
        }
        scope = scope.upper;
      }
      return false;
    }

    return {
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

      ArrowFunctionExpression: enterFunction,
      "ArrowFunctionExpression:exit": exitFunction,
      FunctionExpression: enterFunction,
      "FunctionExpression:exit": exitFunction,

      TemplateLiteral(node) {
        const caughtNames = getCaughtVarNames();
        if (caughtNames.size === 0) return;

        for (const expr of node.expressions) {
          if (!isBareIdentifierExpression(expr)) continue;
          if (!caughtNames.has(expr.name)) continue;

          const errorVar = expr.name;
          const getErrorMessageAvailable = hasResolvableLocalBinding(node, "getErrorMessage");

          context.report({
            node: expr,
            messageId: "bareErrorInterpolation",
            data: { errorVar },
            suggest: [
              getErrorMessageAvailable
                ? ({
                    messageId: "useGetErrorMessage" as const,
                    data: { errorVar },
                    fix(fixer) {
                      return fixer.replaceText(expr, `getErrorMessage(${errorVar})`);
                    },
                  } as const)
                : ({
                    messageId: "useStringFallback" as const,
                    data: { errorVar },
                    fix(fixer) {
                      return fixer.replaceText(expr, `String(${errorVar})`);
                    },
                  } as const),
            ],
          });
        }
      },
    };
  },
});
