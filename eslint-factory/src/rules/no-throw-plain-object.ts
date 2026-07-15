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

/** Returns true when the node is a negative numeric literal (e.g. -32602 in source code). */
function isNegativeNumericLiteral(node: TSESTree.Node): boolean {
  // In JS source, -32602 is parsed as UnaryExpression(-) + Literal(32602).
  if (node.type === AST_NODE_TYPES.UnaryExpression && node.operator === "-" && node.argument.type === AST_NODE_TYPES.Literal) {
    const value = (node.argument as TSESTree.Literal).value;
    return typeof value === "number" && Number.isInteger(value) && value > 0;
  }
  // A raw negative numeric literal is also accepted for completeness.
  return node.type === AST_NODE_TYPES.Literal && typeof node.value === "number" && Number.isInteger(node.value) && node.value < 0;
}

/**
 * Returns true when the thrown ObjectExpression matches the intentional JSON-RPC error idiom.
 * All three conditions must hold:
 *   1. Keys come only from { code, message, data } (no extra properties allowed).
 *   2. `code` is present and its value is a negative numeric literal (e.g. -32602).
 *   3. `message` is present (the value may be a literal, template, variable, or call expression).
 * The protocol boundary reads these fields directly instead of using a stack trace, so the
 * rule's regression-guard value does not apply to them.
 */
function isJsonRpcErrorShape(props: TSESTree.ObjectLiteralElement[]): boolean {
  if (props.length === 0) return false;
  if (hasUnsafeProperties(props)) return false;

  const ALLOWED_KEYS = new Set(["code", "message", "data"]);
  let hasNegativeCode = false;
  let hasMessage = false;

  for (const prop of props) {
    if (prop.type !== AST_NODE_TYPES.Property) return false;
    if (prop.computed) return false;

    const { key } = prop;
    let keyName: string;
    if (key.type === AST_NODE_TYPES.Identifier) {
      keyName = key.name;
    } else if (key.type === AST_NODE_TYPES.Literal && typeof key.value === "string") {
      keyName = key.value;
    } else {
      return false;
    }

    if (!ALLOWED_KEYS.has(keyName)) return false;

    if (keyName === "code") {
      if (!isNegativeNumericLiteral(prop.value)) return false;
      hasNegativeCode = true;
    }

    if (keyName === "message") {
      hasMessage = true;
    }
  }

  return hasNegativeCode && hasMessage;
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

        // Exempt intentional JSON-RPC error throws: { code: <negative int>, message: <any>, data?: <any> }
        if (isJsonRpcErrorShape(props)) return;

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
