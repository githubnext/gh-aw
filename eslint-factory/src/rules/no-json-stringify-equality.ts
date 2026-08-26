import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCodeScope = ReturnType<TSESLint.SourceCode["getScope"]>;

const EQUALITY_OPERATORS = new Set(["===", "!==", "==", "!="]);

/**
 * Returns true when `name` at `node` resolves to the global binding, i.e. no
 * enclosing scope declares a local variable, import, class or function with
 * that name shadowing the built-in.
 */
function isGlobalReference(sourceCode: TSESLint.SourceCode, node: TSESTree.Node, name: string): boolean {
  let scope: SourceCodeScope | null = sourceCode.getScope(node);

  while (scope) {
    const variable = scope.set.get(name);
    if (variable && variable.defs.length > 0) return false;
    scope = scope.upper;
  }

  return true;
}

/** Returns true when `expr` is a call to the global `JSON.stringify(...)`. */
function isJsonStringifyCall(sourceCode: TSESLint.SourceCode, expr: TSESTree.Node): boolean {
  if (expr.type !== AST_NODE_TYPES.CallExpression) return false;

  const callee = expr.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return false;

  const obj = callee.object;
  const prop = callee.property;
  if (obj.type !== AST_NODE_TYPES.Identifier || obj.name !== "JSON") return false;
  if (!isGlobalReference(sourceCode, obj, "JSON")) return false;
  if (prop.type !== AST_NODE_TYPES.Identifier || prop.name !== "stringify") return false;

  return true;
}

export const noJsonStringifyEqualityRule = createRule({
  name: "no-json-stringify-equality",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow comparing two JSON.stringify() results for equality — JSON.stringify() output depends on object key insertion order, so deeply-equal values built with a different key order serialize to different strings and compare as unequal. " +
        "Use a structural deep-equality check instead.",
    },
    schema: [],
    messages: {
      jsonStringifyEquality:
        "Comparing JSON.stringify(...) results with '{{operator}}' is unreliable: two deeply-equal objects with different key insertion order produce different strings, causing false negatives. Use a structural deep-equality check (e.g. a recursive deepEqual helper) instead.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      BinaryExpression(node: TSESTree.BinaryExpression) {
        if (!EQUALITY_OPERATORS.has(node.operator)) return;
        if (!isJsonStringifyCall(sourceCode, node.left)) return;
        if (!isJsonStringifyCall(sourceCode, node.right)) return;

        context.report({
          node,
          messageId: "jsonStringifyEquality",
          data: { operator: node.operator },
        });
      },
    };
  },
});
