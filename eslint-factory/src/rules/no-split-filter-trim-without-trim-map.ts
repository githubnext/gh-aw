import { ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

function isTrimCallOn(node: TSESTree.Node, paramName: string): boolean {
  return (
    node.type === "CallExpression" &&
    node.arguments.length === 0 &&
    node.callee.type === "MemberExpression" &&
    !node.callee.computed &&
    node.callee.property.type === "Identifier" &&
    node.callee.property.name === "trim" &&
    node.callee.object.type === "Identifier" &&
    node.callee.object.name === paramName
  );
}

// Matches `x.trim()`, `x.trim().length`, `x.trim().length > 0`, `x.trim() !== ""`, `!!x.trim()`.
function isTrimmedTruthiness(node: TSESTree.Node, paramName: string): boolean {
  if (isTrimCallOn(node, paramName)) return true;
  if (node.type === "UnaryExpression" && node.operator === "!") return isTrimmedTruthiness(node.argument, paramName);
  if (node.type === "MemberExpression" && !node.computed && node.property.type === "Identifier" && node.property.name === "length") return isTrimCallOn(node.object, paramName);
  if (node.type === "BinaryExpression") return isTrimmedTruthiness(node.left, paramName) || isTrimmedTruthiness(node.right, paramName);
  return false;
}

export const noSplitFilterTrimWithoutTrimMapRule = createRule({
  name: "no-split-filter-trim-without-trim-map",
  meta: {
    type: "problem",
    docs: {
      description: 'Disallow `.split(",").filter(x => x.trim())`, which drops blank entries but keeps the surrounding whitespace on the remaining entries.',
    },
    schema: [],
    messages: {
      filterTrimKeepsWhitespace:
        'Filtering split() output with `{{param}}.trim()` only tests for blank entries; the kept values still contain leading/trailing whitespace (e.g. "a, b".split(",") keeps " b"). Trim first: `.split(...).map(v => v.trim()).filter(Boolean)`.',
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      CallExpression(node) {
        const callee = node.callee;
        if (callee.type !== "MemberExpression" || callee.computed || callee.property.type !== "Identifier" || callee.property.name !== "filter") return;
        const predicate = node.arguments[0];
        if (!predicate || predicate.type !== "ArrowFunctionExpression" || predicate.params.length !== 1 || predicate.params[0].type !== "Identifier") return;
        const param = predicate.params[0].name;

        let source: TSESTree.Node = callee.object;
        if (source.type !== "CallExpression" || source.callee.type !== "MemberExpression" || source.callee.computed || source.callee.property.type !== "Identifier" || source.callee.property.name !== "split") return;
        const separator = source.arguments[0];
        // Newline-separated input is line-oriented; the trailing "\r"/indent concern differs, so only flag delimiter lists.
        if (!separator || separator.type !== "Literal" || typeof separator.value !== "string" || separator.value === "" || /[\r\n]/.test(separator.value)) return;

        const body = predicate.body;
        const expression = body.type === "BlockStatement" ? (body.body.length === 1 && body.body[0].type === "ReturnStatement" ? body.body[0].argument : null) : body;
        if (!expression || !isTrimmedTruthiness(expression, param)) return;

        context.report({ node, messageId: "filterTrimKeepsWhitespace", data: { param } });
      },
    };
  },
});
