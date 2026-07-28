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
 * Returns true when the variable definition represents a caught error binding:
 *   - A try/catch clause with a simple identifier param (not destructured).
 *   - A parameter of an inline promise rejection handler (.catch(fn) /
 *     .then(_, fn)).
 */
function isCaughtErrorVariableDef(def: TSESLint.Scope.Definition): boolean {
  if (def.type === "CatchClause") {
    const catchNode = def.node as TSESTree.CatchClause;
    return catchNode.param?.type === AST_NODE_TYPES.Identifier;
  }

  if (def.type === "Parameter") {
    const fn = def.node as TSESTree.Node;
    if (fn.type !== AST_NODE_TYPES.ArrowFunctionExpression && fn.type !== AST_NODE_TYPES.FunctionExpression) {
      return false;
    }
    return isInlineRejectionHandler(fn as TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression);
  }

  return false;
}

/**
 * Returns the argument name when `node` is a `String(<identifier>)` call
 * expression, or null otherwise.
 */
function getStringCallArgName(node: TSESTree.CallExpressionArgument): string | null {
  if (node.type !== AST_NODE_TYPES.CallExpression) return null;
  if (node.callee.type !== AST_NODE_TYPES.Identifier || node.callee.name !== "String") return null;
  if (node.arguments.length !== 1) return null;
  const arg = node.arguments[0];
  return arg.type === AST_NODE_TYPES.Identifier ? arg.name : null;
}

export const preferGetErrorMessageOverStringRule = createRule({
  name: "prefer-get-error-message-over-string",
  meta: {
    type: "suggestion",
    hasSuggestions: true,
    docs: {
      description:
        "Prefer getErrorMessage(err) over String(err) when interpolating a caught error into a template literal, when getErrorMessage is already resolvable in scope. " +
        "String(err) on an Error produces the redundant 'Error: message' prefix and does not sanitize GitHub's HTML error-page responses, " +
        "while getErrorMessage(err) handles both correctly. Several actions/setup/js files already import getErrorMessage " +
        "elsewhere yet still call String(err) at other call sites in the same file — this rule catches that inconsistency.",
    },
    schema: [],
    messages: {
      preferGetErrorMessage: "Use getErrorMessage({{errorVar}}) instead of String({{errorVar}}) — getErrorMessage is already available in this scope and produces a cleaner, sanitized message for caught errors.",
      replaceWithGetErrorMessage: "Replace String({{errorVar}}) with getErrorMessage({{errorVar}}).",
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

    function resolvesToCaughtErrorVariable(expr: TSESTree.Identifier): boolean {
      let scope: Scope | null = sourceCode.getScope(expr);
      while (scope) {
        const v = scope.set.get(expr.name);
        if (v) {
          return v.defs.some(isCaughtErrorVariableDef);
        }
        scope = scope.upper;
      }
      return false;
    }

    return {
      TemplateLiteral(node) {
        // Tagged templates pass values to the tag function as-is; not string-coerced.
        if (node.parent?.type === AST_NODE_TYPES.TaggedTemplateExpression) return;

        for (const expr of node.expressions) {
          const errorVar = getStringCallArgName(expr);
          if (!errorVar) continue;
          if (expr.type !== AST_NODE_TYPES.CallExpression) continue;
          const argNode = expr.arguments[0];
          if (argNode.type !== AST_NODE_TYPES.Identifier) continue;
          if (!resolvesToCaughtErrorVariable(argNode)) continue;
          if (!hasResolvableLocalBinding(node, "getErrorMessage")) continue;

          context.report({
            node: expr,
            messageId: "preferGetErrorMessage",
            data: { errorVar },
            suggest: [
              {
                messageId: "replaceWithGetErrorMessage",
                data: { errorVar },
                fix(fixer) {
                  return fixer.replaceText(expr, `getErrorMessage(${errorVar})`);
                },
              },
            ],
          });
        }
      },
    };
  },
});
