import { ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

function unwrapChain(node: TSESTree.Node): TSESTree.Node {
  return node.type === "ChainExpression" ? node.expression : node;
}

function isTrimCallOn(node: TSESTree.Node, paramName: string): boolean {
  const expression = unwrapChain(node);
  return (
    expression.type === "CallExpression" &&
    expression.arguments.length === 0 &&
    expression.callee.type === "MemberExpression" &&
    !expression.callee.computed &&
    expression.callee.property.type === "Identifier" &&
    expression.callee.property.name === "trim" &&
    expression.callee.object.type === "Identifier" &&
    expression.callee.object.name === paramName
  );
}

function isTrimLength(node: TSESTree.Node, paramName: string): boolean {
  const expression = unwrapChain(node);
  return expression.type === "MemberExpression" && !expression.computed && expression.property.type === "Identifier" && expression.property.name === "length" && isTrimCallOn(expression.object, paramName);
}

function isEmptyString(node: TSESTree.Node): boolean {
  return node.type === "Literal" && node.value === "";
}

function isNumber(node: TSESTree.Node, value: 0 | 1): boolean {
  return node.type === "Literal" && node.value === value;
}

function isNonEmptyTrimComparison(node: TSESTree.Node, paramName: string): boolean {
  const expression = unwrapChain(node);
  if (expression.type !== "BinaryExpression") return false;
  const { left, operator, right } = expression;

  if ((operator === "!==" || operator === "!=") && ((isTrimCallOn(left, paramName) && isEmptyString(right)) || (isEmptyString(left) && isTrimCallOn(right, paramName)))) return true;
  if ((operator === "!==" || operator === "!=") && ((isTrimLength(left, paramName) && isNumber(right, 0)) || (isNumber(left, 0) && isTrimLength(right, paramName)))) return true;
  if (operator === ">" && isTrimLength(left, paramName) && isNumber(right, 0)) return true;
  if (operator === "<" && isNumber(left, 0) && isTrimLength(right, paramName)) return true;
  if (operator === ">=" && isTrimLength(left, paramName) && isNumber(right, 1)) return true;
  if (operator === "<=" && isNumber(left, 1) && isTrimLength(right, paramName)) return true;
  return false;
}

function isTrimmedNonEmpty(node: TSESTree.Node, paramName: string): boolean {
  let expression = node;
  let negations = 0;
  while (expression.type === "UnaryExpression" && expression.operator === "!") {
    negations++;
    expression = expression.argument;
  }
  if (negations % 2 !== 0) return false;
  return isTrimCallOn(expression, paramName) || isTrimLength(expression, paramName) || isNonEmptyTrimComparison(expression, paramName);
}

type Callback = TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression;

function getReturnExpression(callback: Callback): TSESTree.Node | null {
  if (callback.body.type !== "BlockStatement") return callback.body;
  if (callback.body.body.length !== 1 || callback.body.body[0].type !== "ReturnStatement") return null;
  return callback.body.body[0].argument;
}

function isCallback(node: TSESTree.Node | undefined): node is Callback {
  return !!node && (node.type === "ArrowFunctionExpression" || node.type === "FunctionExpression") && !node.async && node.params.length === 1 && node.params[0].type === "Identifier";
}

function getCallbackParameterName(callback: Callback): string | null {
  const param = callback.params[0];
  return param.type === "Identifier" ? param.name : null;
}

function isTrimMap(callback: TSESTree.Node | undefined): boolean {
  if (!isCallback(callback)) return false;
  const paramName = getCallbackParameterName(callback);
  if (paramName === null) return false;
  const expression = getReturnExpression(callback);
  return expression !== null && isTrimCallOn(expression, paramName);
}

function hasFollowingTrimMap(node: TSESTree.CallExpression): boolean {
  const member = node.parent;
  if (member?.type !== "MemberExpression" || member.computed || member.property.type !== "Identifier" || member.property.name !== "map") return false;
  const mapCall = member.parent;
  return mapCall?.type === "CallExpression" && mapCall.callee === member && isTrimMap(mapCall.arguments[0]);
}

function isSplitCall(node: TSESTree.Node): boolean {
  const expression = unwrapChain(node);
  return expression.type === "CallExpression" && expression.callee.type === "MemberExpression" && !expression.callee.computed && expression.callee.property.type === "Identifier" && expression.callee.property.name === "split";
}

function isSplitSeparator(node: TSESTree.Node | undefined): boolean {
  return !!node && node.type === "Literal" && typeof node.value === "string" && node.value !== "" && !/[\r\n]/.test(node.value);
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
        if (!isCallback(predicate)) return;
        const param = getCallbackParameterName(predicate);
        if (param === null) return;
        if (!isSplitCall(callee.object)) return;
        const source = unwrapChain(callee.object);
        if (source.type !== "CallExpression") return;
        const separator = source.arguments[0];
        // Newline-separated input is line-oriented; the trailing "\r"/indent concern differs, so only flag delimiter lists.
        if (!isSplitSeparator(separator)) return;

        const expression = getReturnExpression(predicate);
        if (!expression || !isTrimmedNonEmpty(expression, param) || hasFollowingTrimMap(node)) return;

        context.report({ node, messageId: "filterTrimKeepsWhitespace", data: { param } });
      },
    };
  },
});
