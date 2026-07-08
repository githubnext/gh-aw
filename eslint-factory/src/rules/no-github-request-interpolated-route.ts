import { ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const OCTOKIT_CLIENT_NAMES = new Set(["github", "octokit", "githubClient", "octokitClient"]);

/**
 * Returns true when the node is a template literal that contains at least one
 * interpolated expression (i.e. a tagged or untagged TemplateLiteral with
 * one or more `expressions` entries).
 */
function isInterpolatedTemplateLiteral(node: TSESTree.Node): boolean {
  return node.type === "TemplateLiteral" && node.expressions.length > 0;
}

/**
 * Returns true when the node is a binary `+` expression, which indicates
 * string concatenation. Nested concatenations such as
 * `"GET /repos/" + owner + "/" + repo` parse as left-associative
 * BinaryExpressions, so the outermost node is still a `+` BinaryExpression
 * and is caught by this check.
 */
function isStringConcatenation(node: TSESTree.Node): boolean {
  return node.type === "BinaryExpression" && node.operator === "+";
}

export const noGithubRequestInterpolatedRouteRule = createRule({
  name: "no-github-request-interpolated-route",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow template literals with interpolations or string concatenation as the route argument of Octokit " +
        "github/octokit/githubClient/octokitClient .request() calls. " +
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
    return {
      CallExpression(node) {
        const callee = node.callee;

        // Only match <client>.request(...)
        if (callee.type !== "MemberExpression") return;
        if (callee.computed) return;
        if (callee.object.type !== "Identifier") return;
        if (!OCTOKIT_CLIENT_NAMES.has(callee.object.name)) return;
        if (callee.property.type !== "Identifier") return;
        if (callee.property.name !== "request") return;

        const firstArg = node.arguments[0];
        if (!firstArg) return;

        const clientName = callee.object.name;

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
