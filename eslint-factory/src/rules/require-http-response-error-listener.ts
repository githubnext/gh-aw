import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const HTTP_MODULE_SPECIFIERS = new Set(["http", "https", "node:http", "node:https"]);
const REQUEST_METHODS = new Set(["request", "get"]);

/** Returns true when the node is `require("http")` / `require("https")` / `require("node:http")` / `require("node:https")`. */
function isRequireHttpModule(node: TSESTree.Node | null | undefined): boolean {
  if (!node) return false;
  if (node.type !== AST_NODE_TYPES.CallExpression) return false;
  if (node.callee.type !== AST_NODE_TYPES.Identifier || node.callee.name !== "require") return false;
  const firstArg = node.arguments[0];
  if (!firstArg || firstArg.type !== AST_NODE_TYPES.Literal) return false;
  return typeof firstArg.value === "string" && HTTP_MODULE_SPECIFIERS.has(firstArg.value);
}

/** Returns true when `name` resolves to a variable bound to Node's `http`/`https` module via `require(...)`. */
function isHttpModuleBinding(name: string, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode): boolean {
  let scope: ReturnType<typeof sourceCode.getScope> | null = sourceCode.getScope(scopeNode);
  while (scope) {
    const variable = scope.set.get(name);
    if (variable && variable.defs.length > 0) {
      for (const def of variable.defs) {
        if (def.type !== "Variable") continue;
        const declarator = def.node as TSESTree.VariableDeclarator;
        if (declarator.id.type === AST_NODE_TYPES.Identifier && isRequireHttpModule(declarator.init)) {
          return true;
        }
      }
      return false;
    }
    scope = scope.upper;
  }
  return false;
}

/** Returns true when `call` is `<http>.request(...)` / `<http>.get(...)` on a resolved http/https module binding. */
function isHttpRequestCall(call: TSESTree.CallExpression, sourceCode: TSESLint.SourceCode): boolean {
  const callee = call.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return false;
  if (callee.object.type !== AST_NODE_TYPES.Identifier) return false;
  if (callee.property.type !== AST_NODE_TYPES.Identifier || !REQUEST_METHODS.has(callee.property.name)) return false;
  return isHttpModuleBinding(callee.object.name, callee.object, sourceCode);
}

type ResponseCallback = TSESTree.FunctionExpression | TSESTree.ArrowFunctionExpression;

/** Returns the response callback argument of an http request call, when it has a single named response parameter. */
function getResponseCallback(call: TSESTree.CallExpression): ResponseCallback | null {
  for (const arg of call.arguments) {
    if (arg.type !== AST_NODE_TYPES.FunctionExpression && arg.type !== AST_NODE_TYPES.ArrowFunctionExpression) continue;
    const firstParam = arg.params[0];
    if (!firstParam || firstParam.type !== AST_NODE_TYPES.Identifier) return null;
    return arg;
  }
  return null;
}

/** Returns true when `call` is `<name>.on("error", ...)` / `<name>.once("error", ...)`. */
function isErrorListenerCall(call: TSESTree.CallExpression, name: string): boolean {
  const callee = call.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return false;
  if (callee.object.type !== AST_NODE_TYPES.Identifier || callee.object.name !== name) return false;
  if (callee.property.type !== AST_NODE_TYPES.Identifier || (callee.property.name !== "on" && callee.property.name !== "once")) return false;
  const firstArg = call.arguments[0];
  return firstArg !== undefined && firstArg.type === AST_NODE_TYPES.Literal && firstArg.value === "error";
}

export const requireHttpResponseErrorListenerRule = createRule({
  name: "require-http-response-error-listener",
  meta: {
    type: "problem",
    docs: {
      description:
        "Require an 'error' event listener on the response object passed to http.request()/http.get()/https.request()/https.get() callbacks. " +
        "Node emits 'error' on the IncomingMessage itself for socket-level failures that occur while the body is streamed " +
        "(reset connections, decompression failures, aborted sockets); a listener on the request does not catch these, " +
        "so an unhandled response 'error' event crashes the action. " +
        'Scope: only fires when the http/https module identifier is statically resolved through a `require("http")`-style binding.',
    },
    schema: [],
    messages: {
      missingResponseErrorListener:
        "The response passed to this http request callback must have an 'error' event listener attached (e.g. `res.on(\"error\", reject)`). " +
        "Node emits 'error' on the response for socket-level failures while the body streams; `req.on(\"error\", ...)` does not catch those and the unhandled event crashes the action.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node: TSESTree.CallExpression) {
        if (!isHttpRequestCall(node, sourceCode)) return;

        const callback = getResponseCallback(node);
        if (!callback) return;

        const param = callback.params[0];
        if (!param || param.type !== AST_NODE_TYPES.Identifier) return;
        const responseName = param.name;

        const variable = sourceCode.getDeclaredVariables(callback).find(candidate => candidate.name === responseName);
        if (!variable) return;

        const hasErrorListener = variable.references.some(ref => {
          const id = ref.identifier;
          const parent = id.parent;
          if (!parent || parent.type !== AST_NODE_TYPES.MemberExpression || parent.object !== id) return false;
          const grandparent = parent.parent;
          return grandparent !== undefined && grandparent.type === AST_NODE_TYPES.CallExpression && grandparent.callee === parent && isErrorListenerCall(grandparent, responseName);
        });

        if (!hasErrorListener) {
          context.report({ node: param, messageId: "missingResponseErrorListener" });
        }
      },
    };
  },
});
