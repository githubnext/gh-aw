import { AST_NODE_TYPES, ESLintUtils } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

export const noThrowPlainObjectRule = createRule({
  name: "no-throw-plain-object",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow throwing plain object literals (`throw { ... }`). Plain objects lack a `.stack` trace and a meaningful `.message` string, making errors hard to debug and incompatible with catch-clause error utilities (getErrorMessage, etc.). Use `new Error(...)` instead, and attach extra context via `Object.assign` or the `cause` option.",
    },
    schema: [],
    messages: {
      noThrowPlainObject:
        "Throwing a plain object literal loses the stack trace. Use `new Error(message)` instead; attach extra fields with `Object.assign(new Error(message), { ... })` if needed.",
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      ThrowStatement(node) {
        const arg = node.argument;
        if (!arg) return;
        if (arg.type === AST_NODE_TYPES.ObjectExpression) {
          context.report({ node: arg, messageId: "noThrowPlainObject" });
        }
      },
    };
  },
});
