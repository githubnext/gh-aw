import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/** Returns true when the object has properties that cannot be safely rewritten (spreads, computed keys, methods, getters/setters). */
function hasUnsafeProperties(props: TSESTree.ObjectLiteralElement[]): boolean {
  return props.some(prop => {
    if (prop.type === AST_NODE_TYPES.SpreadElement) return true;
    if (prop.type !== AST_NODE_TYPES.Property) return true;
    if (prop.computed) return true;
    if (prop.kind === "get" || prop.kind === "set") return true;
    if (prop.method) return true;
    return false;
  });
}

/** Returns the first Property whose key is the identifier or string literal "message". */
function findMessageProp(props: TSESTree.ObjectLiteralElement[]): TSESTree.Property | null {
  for (const prop of props) {
    if (prop.type !== AST_NODE_TYPES.Property) continue;
    if (prop.computed) continue;
    const { key } = prop;
    if (key.type === AST_NODE_TYPES.Identifier && key.name === "message") return prop;
    if (key.type === AST_NODE_TYPES.Literal && key.value === "message") return prop;
  }
  return null;
}

export const noThrowPlainObjectRule = createRule({
  name: "no-throw-plain-object",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Disallow throwing plain object literals (`throw { ... }`). Plain objects lack a `.stack` trace and a meaningful `.message` string, making errors hard to debug and incompatible with catch-clause error utilities (getErrorMessage, etc.). Use `new Error(...)` instead, and attach extra context via `Object.assign` or the `cause` option.",
    },
    schema: [],
    messages: {
      noThrowPlainObject: "Throwing a plain object literal loses the stack trace. Use `new Error(message)` instead; attach extra fields with `Object.assign(new Error(message), { ... })` if needed.",
      useObjectAssign: "Rewrite as `Object.assign(new Error(...), { ... })`.",
    },
  },
  defaultOptions: [],
  create(context) {
    return {
      ThrowStatement(node) {
        const arg = node.argument;
        if (!arg) return;
        if (arg.type !== AST_NODE_TYPES.ObjectExpression) return;

        const { properties: props } = arg;

        if (hasUnsafeProperties(props)) {
          context.report({ node: arg, messageId: "noThrowPlainObject", suggest: [] });
          return;
        }

        context.report({
          node: arg,
          messageId: "noThrowPlainObject",
          suggest: [
            {
              messageId: "useObjectAssign",
              fix(fixer: TSESLint.RuleFixer) {
                const src = context.sourceCode;

                // throw {} → throw new Error()
                if (props.length === 0) {
                  return fixer.replaceText(arg, "new Error()");
                }

                const msgProp = findMessageProp(props);
                const errorArg = msgProp ? src.getText(msgProp.value) : "";
                const residual = props.filter(p => p !== msgProp);
                const newErr = errorArg ? `new Error(${errorArg})` : "new Error()";

                // throw { message: x } → throw new Error(x)
                if (residual.length === 0) {
                  return fixer.replaceText(arg, newErr);
                }

                // throw { code, message, data } → throw Object.assign(new Error(message), { code, data })
                const residualText = residual.map(p => src.getText(p)).join(", ");
                return fixer.replaceText(arg, `Object.assign(${newErr}, { ${residualText} })`);
              },
            },
          ],
        });
      },
    };
  },
});
