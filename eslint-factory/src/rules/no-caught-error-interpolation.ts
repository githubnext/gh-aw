import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

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
 * Returns true when `node` is a bare `Identifier` expression — no member
 * access, no call, no unary/binary operation, no nullish coercion. Used to
 * identify direct `${someVar}` interpolations as opposed to safe forms such
 * as `${someVar.message}` or `${fn(someVar)}`.
 */
function isBareIdentifierExpression(node: TSESTree.Expression): node is TSESTree.Identifier {
  return node.type === AST_NODE_TYPES.Identifier;
}

/**
 * Returns true when the variable definition represents a caught error binding
 * that should not be interpolated directly:
 *   - A try/catch clause with a simple identifier param (not destructured).
 *   - A parameter of an inline promise rejection handler (.catch(fn) /
 *     .then(_, fn)).
 */
function isCaughtErrorVariableDef(def: TSESLint.Scope.Definition): boolean {
  // try/catch binding — only flag simple identifier params, not destructured ones
  // (e.g. `catch ({ message })` introduces string properties, not raw error objects)
  if (def.type === "CatchClause") {
    const catchNode = def.node as TSESTree.CatchClause;
    return catchNode.param?.type === AST_NODE_TYPES.Identifier;
  }

  // Inline rejection handler parameter (.catch(err => ...) / .then(_, err => ...))
  // def.node is the function node for Parameter definitions
  if (def.type === "Parameter") {
    const fn = def.node as TSESTree.Node;
    if (fn.type !== AST_NODE_TYPES.ArrowFunctionExpression && fn.type !== AST_NODE_TYPES.FunctionExpression) {
      return false;
    }
    return isInlineRejectionHandler(fn as TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression);
  }

  return false;
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
    type Scope = ReturnType<typeof sourceCode.getScope>;

    function isDefinitionAvailableAtNode(definition: TSESLint.Scope.Definition, node: TSESTree.Node): boolean {
      if (definition.type === "ImportBinding" || definition.type === "FunctionName") {
        return true;
      }
      const definitionNode = definition.name ?? definition.node;
      if (!definitionNode?.range || !node.range) return false;
      return definitionNode.range[0] < node.range[0];
    }

    function hasResolvableLocalBinding(node: TSESTree.Node, name: string): boolean {
      let scope: Scope | null = sourceCode.getScope(node);
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
      TemplateLiteral(node) {
        // Tagged templates pass values to the tag function as-is; they are not
        // string-coerced by interpolation, so flagging them would be incorrect.
        if (node.parent?.type === AST_NODE_TYPES.TaggedTemplateExpression) return;

        for (const expr of node.expressions) {
          if (!isBareIdentifierExpression(expr)) continue;

          // Resolve the identifier through the full scope chain. This correctly
          // handles closures: an identifier inside a nested function can still
          // resolve to an outer catch binding.
          let scope: Scope | null = sourceCode.getScope(expr);
          let variable: TSESLint.Scope.Variable | null = null;
          while (scope) {
            const v = scope.set.get(expr.name);
            if (v) {
              variable = v;
              break;
            }
            scope = scope.upper;
          }

          if (!variable || !variable.defs.some(isCaughtErrorVariableDef)) continue;

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
