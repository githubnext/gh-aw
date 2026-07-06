import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

interface ErrorScope {
  varName: string;
  isSentinel: boolean;
}

/**
 * Returns true when the function node is an inline rejection handler passed to
 * a promise method:
 *   - First argument of `.catch(fn)` — the canonical rejection handler.
 *   - Second argument of `.then(onFulfilled, onRejected)` — the rejection-handler
 *     position, analogous to `.catch(fn)`.
 *
 * Named-reference handlers (for example `p.catch(handler)`) are intentionally
 * out of scope because the rule cannot statically follow the reference to
 * inspect its parameter list or body.
 */
function isInlineRejectionHandler(node: TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression): boolean {
  const parent = node.parent;
  if (!parent || parent.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = parent.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return false;
  const prop = callee.property;
  if (prop.type !== AST_NODE_TYPES.Identifier) return false;
  // .catch(fn) — first argument
  if (prop.name === "catch" && parent.arguments[0] === node) return true;
  // .then(onFulfilled, onRejected) — second argument is the rejection handler
  if (prop.name === "then" && parent.arguments[1] === node) return true;
  return false;
}

export const noJsonStringifyErrorRule = createRule({
  name: "no-json-stringify-error",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Disallow JSON.stringify() on caught error variables — Error properties (message, stack, etc.) are non-enumerable and produce {} silently. " +
        "Detected scopes: try/catch bindings, .catch(fn) inline callbacks, and .then(onFulfilled, onRejected) inline rejection handlers. " +
        "Named-reference handlers (e.g. p.catch(handler)) are out of scope because the rule cannot statically follow the reference.",
    },
    schema: [],
    messages: {
      jsonStringifyError:
        "JSON.stringify({{errorVar}}) produces {} for Error objects — Error properties (message, stack, etc.) are non-enumerable. Use getErrorMessage({{errorVar}}) if it is available, or String({{errorVar}}) as a safe import-free alternative.",
      useGetErrorMessage: "Replace with getErrorMessage({{errorVar}}) — ensure getErrorMessage is imported from error_helpers.cjs.",
      useStringFallback: "Replace with String({{errorVar}}) — getErrorMessage is not in scope; String() is a safe import-free alternative.",
      useDetailPreservingForm: 'Replace with (getErrorMessage({{errorVar}}) + "\\n" + ({{errorVar}}?.stack ?? "")) to preserve the stack trace — ensure getErrorMessage is imported from error_helpers.cjs.',
      useDetailPreservingFormFallback: 'Replace with (String({{errorVar}}) + "\\n" + ({{errorVar}}?.stack ?? "")) to preserve the stack trace.',
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

    // Stack tracking caught error variable names.
    // Each scope entry holds varName (empty string for sentinel) and isSentinel flag.
    const scopeStack: ErrorScope[] = [];

    function getCaughtVarNames(): Set<string> {
      const names = new Set<string>();
      // Walk from the innermost active scope outward and stop at the first
      // sentinel so non-.catch() callbacks cannot see shadowed catch vars.
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
          // No-param or destructuring: push sentinel so outer frames are not affected
          scopeStack.push({ varName: "", isSentinel: true });
        }
      } else {
        // Non-rejection-handler function: sentinel to avoid false positives from shadowed names
        scopeStack.push({ varName: "", isSentinel: true });
      }
    }

    function exitFunction(): void {
      scopeStack.pop();
    }

    function isDefinitionAvailableAtNode(definition: { type: string; name?: TSESTree.Node | null; node: TSESTree.Node }, node: TSESTree.Node): boolean {
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
        if (variable && variable.defs.some(definition => isDefinitionAvailableAtNode(definition, node))) {
          return true;
        }

        scope = scope.upper;
      }

      return false;
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

      // Track inline rejection-handler parameters from .catch(fn) and .then(_, fn)
      ArrowFunctionExpression: enterFunction,
      "ArrowFunctionExpression:exit": exitFunction,
      FunctionExpression: enterFunction,
      "FunctionExpression:exit": exitFunction,

      // Detect JSON.stringify(caughtErrorVar, ...)
      CallExpression(node) {
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

        const getErrorMessageInScope = hasResolvableLocalBinding(node, "getErrorMessage");

        context.report({
          node,
          messageId: "jsonStringifyError",
          data: { errorVar },
          suggest: [
            // Suggestion 1: concise message-only replacement
            getErrorMessageInScope
              ? ({
                  messageId: "useGetErrorMessage" as const,
                  data: { errorVar },
                  fix(fixer) {
                    return fixer.replaceText(node, `getErrorMessage(${errorVar})`);
                  },
                } as const)
              : ({
                  messageId: "useStringFallback" as const,
                  data: { errorVar },
                  fix(fixer) {
                    return fixer.replaceText(node, `String(${errorVar})`);
                  },
                } as const),
            // Suggestion 2: detail-preserving replacement (retains stack trace)
            getErrorMessageInScope
              ? ({
                  messageId: "useDetailPreservingForm" as const,
                  data: { errorVar },
                  fix(fixer) {
                    return fixer.replaceText(node, `(getErrorMessage(${errorVar}) + "\\n" + (${errorVar}?.stack ?? ""))`);
                  },
                } as const)
              : ({
                  messageId: "useDetailPreservingFormFallback" as const,
                  data: { errorVar },
                  fix(fixer) {
                    return fixer.replaceText(node, `(String(${errorVar}) + "\\n" + (${errorVar}?.stack ?? ""))`);
                  },
                } as const),
          ],
        });
      },
    };
  },
});
