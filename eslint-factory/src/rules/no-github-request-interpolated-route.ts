import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const OCTOKIT_CLIENT_NAMES = new Set(["github", "octokit", "githubClient", "octokitClient"]);
const GET_OCTOKIT_MEMBER_OBJECT_NAMES = new Set(["github", "actions"]);

/**
 * Returns true when the node is a template literal that contains at least one
 * interpolated expression (i.e. a tagged or untagged TemplateLiteral with
 * one or more `expressions` entries).
 */
function isInterpolatedTemplateLiteral(node: TSESTree.Node): boolean {
  return node.type === "TemplateLiteral" && node.expressions.length > 0;
}

/**
 * Returns true when the node is a compile-time constant route expression built
 * from literals only.
 */
function isStaticRouteExpression(node: TSESTree.Node): boolean {
  if (node.type === "Literal") return true;
  if (node.type === "TemplateLiteral") return node.expressions.length === 0;
  if (node.type === "BinaryExpression" && node.operator === "+") {
    return isStaticRouteExpression(node.left) && isStaticRouteExpression(node.right);
  }
  return false;
}

/**
 * Returns true when the node is a binary `+` expression, which indicates
 * string concatenation. Nested concatenations such as
 * `"GET /repos/" + owner + "/" + repo` parse as left-associative
 * BinaryExpressions, so the outermost node is still a `+` BinaryExpression
 * and is caught by this check. Compile-time constant concatenations built
 * from literals only are intentionally excluded.
 */
function isStringConcatenation(node: TSESTree.Node): boolean {
  return node.type === "BinaryExpression" && node.operator === "+" && !isStaticRouteExpression(node);
}

/**
 * Returns true when `node` is the `context.github` member expression.
 */
function isContextGithubExpression(node: TSESTree.Node): boolean {
  return (
    node.type === AST_NODE_TYPES.MemberExpression && !node.computed && node.object.type === AST_NODE_TYPES.Identifier && node.object.name === "context" && node.property.type === AST_NODE_TYPES.Identifier && node.property.name === "github"
  );
}

/**
 * Returns true when a syntactic expression node directly represents a known
 * Octokit client source without scope resolution. Recognizes:
 * - Direct known names: github, octokit, githubClient, octokitClient
 * - `getOctokit(...)` call results (bare or via known module objects, e.g.
 *   `github.getOctokit(...)` or `actions.getOctokit(...)`)
 * - `context.github` member expression
 */
function isOctokitSourceExpression(node: TSESTree.Node): boolean {
  if (node.type === AST_NODE_TYPES.Identifier && OCTOKIT_CLIENT_NAMES.has(node.name)) return true;

  if (node.type === AST_NODE_TYPES.CallExpression) {
    const callee = node.callee;
    if (callee.type === AST_NODE_TYPES.Identifier && callee.name === "getOctokit") return true;
    if (
      callee.type === AST_NODE_TYPES.MemberExpression &&
      !callee.computed &&
      callee.object.type === AST_NODE_TYPES.Identifier &&
      GET_OCTOKIT_MEMBER_OBJECT_NAMES.has(callee.object.name) &&
      callee.property.type === AST_NODE_TYPES.Identifier &&
      callee.property.name === "getOctokit"
    ) {
      return true;
    }
  }

  if (isContextGithubExpression(node)) return true;

  return false;
}

export const noGithubRequestInterpolatedRouteRule = createRule({
  name: "no-github-request-interpolated-route",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow template literals with interpolations or string concatenation as the route argument of Octokit.request() calls. " +
        "Octokit clients are detected by well-known names (github, octokit, githubClient, octokitClient), " +
        "identifiers initialized from getOctokit(...) call results, context.github, and simple const aliases of any of these. " +
        'Use the typed placeholder form instead: "GET /repos/{owner}/{repo}" with a separate params object.',
    },
    schema: [],
    messages: {
      interpolatedRoute:
        "Avoid using a {{kind}} as the route argument of {{client}}.request(). " +
        'Use the typed placeholder form instead — e.g. github.request("GET /repos/{owner}/{repo}", { owner, repo }) — ' +
        "to preserve typed dispatch and prevent malformed paths.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

    /**
     * Resolves an identifier name to check if it is bound to a known Octokit
     * client source in the visible scope chain. Handles simple single-level
     * assignments:
     *   const x = github
     *   const x = getOctokit(token)
     *   const x = context.github
     */
    function isIdentifierBoundToOctokitClient(name: string, scopeNode: TSESTree.Node): boolean {
      if (OCTOKIT_CLIENT_NAMES.has(name)) return true;

      let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
      while (scope) {
        const variable = scope.set.get(name);
        if (variable && variable.defs.length > 0) {
          for (const def of variable.defs) {
            if (def.type !== "Variable") continue;
            const declarator = def.node as TSESTree.VariableDeclarator;
            const declaration = declarator.parent;
            if (!declaration || declaration.type !== AST_NODE_TYPES.VariableDeclaration || declaration.kind !== "const") continue;
            if (declarator.init && isOctokitSourceExpression(declarator.init)) return true;
          }
          return false;
        }
        scope = scope.upper;
      }
      return false;
    }

    /**
     * Returns the display name (for error messages) if the callee object
     * resolves to a known Octokit client, or null otherwise.
     *
     * Recognized shapes:
     * - `<name>.request(...)` where name is in OCTOKIT_CLIENT_NAMES or is a
     *   simple alias of an Octokit source (scope-resolved)
     * - `context.github.request(...)`
     */
    function resolveOctokitClientName(calleeObject: TSESTree.Expression | TSESTree.Super, callNode: TSESTree.Node): string | null {
      if (calleeObject.type === AST_NODE_TYPES.Identifier) {
        return isIdentifierBoundToOctokitClient(calleeObject.name, callNode) ? calleeObject.name : null;
      }

      if (isContextGithubExpression(calleeObject)) {
        return "context.github";
      }

      return null;
    }

    return {
      CallExpression(node) {
        const callee = node.callee;

        // Only match <client>.request(...)
        if (callee.type !== AST_NODE_TYPES.MemberExpression) return;
        if (callee.computed) return;
        if (callee.property.type !== AST_NODE_TYPES.Identifier) return;
        if (callee.property.name !== "request") return;

        const clientName = resolveOctokitClientName(callee.object, node);
        if (!clientName) return;

        const firstArg = node.arguments[0];
        if (!firstArg) return;

        if (isInterpolatedTemplateLiteral(firstArg)) {
          context.report({
            node: firstArg,
            messageId: "interpolatedRoute",
            data: { kind: "template literal with interpolations", client: clientName },
          });
          return;
        }

        if (isStringConcatenation(firstArg)) {
          context.report({
            node: firstArg,
            messageId: "interpolatedRoute",
            data: { kind: "string concatenation expression", client: clientName },
          });
        }
      },
    };
  },
});
