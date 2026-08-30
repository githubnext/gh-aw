import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const UNSAFE_PROPERTIES = new Set(["message", "stack", "code", "status", "cause", "name"]);

/**
 * Returns true when a JSDoc-style `/** @type {Error} *\/` (or `{Error|any}`,
 * `{any}` cast used specifically to widen a caught error) comment immediately
 * precedes `node` in the source. This is the pattern used throughout
 * actions/setup/js to silence "possibly not an Error" checks on a caught
 * value without actually narrowing the type at runtime:
 *
 *   } catch (error) {
 *     const err = \/** @type {Error} *\/ error;
 *     log(err.message); // unsafe: `error` may not be an Error instance
 *   }
 */
function hasErrorTypeCastComment(sourceCode: Readonly<import("@typescript-eslint/utils").TSESLint.SourceCode>, node: TSESTree.Node): boolean {
  const commentsBefore = sourceCode.getCommentsBefore(node);
  return commentsBefore.some(comment => /@type\s*\{\s*Error\b/.test(comment.value));
}

/**
 * Returns the identifier being cast when `node` (the init/right-hand side of
 * a declarator or assignment) is a JSDoc `@type {Error}`-annotated caught
 * error identifier, optionally wrapped in parentheses: `error`, `(error)`.
 * Only bare identifier casts of a caught-error-like source are recognized to
 * keep false-positive risk low; casts of arbitrary expressions are ignored.
 */
function getCastSourceIdentifier(sourceCode: Readonly<import("@typescript-eslint/utils").TSESLint.SourceCode>, node: TSESTree.Expression): TSESTree.Identifier | null {
  if (node.type !== AST_NODE_TYPES.Identifier) return null;
  if (!hasErrorTypeCastComment(sourceCode, node)) return null;
  return node;
}

export const noUnsafeJsdocErrorTypeCastRule = createRule({
  name: "no-unsafe-jsdoc-error-type-cast",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow treating a caught error as an Error via a JSDoc `/** @type {Error} */` cast and then accessing .message/.stack/.code/.status/.cause/.name on it without a runtime instanceof check. " +
        "The cast is a compile-time-only annotation for the TypeScript checker (via checkJs) — it has no runtime effect, so a non-Error throw (a string, a plain object, undefined) will still produce `undefined` or a TypeError at the property access. " +
        "Use getErrorMessage(err) from error_helpers.cjs, or guard with `err instanceof Error`, instead of casting.",
    },
    schema: [],
    messages: {
      unsafeCast:
        "'{{varName}}' is assigned from a caught value via a `/** @type {Error} */` JSDoc cast, which only affects type-checking and provides no runtime guarantee. Accessing .{{prop}} on it is unsafe if the caught value is not actually an Error.",
      useGetErrorMessage: "Replace with getErrorMessage(...) from error_helpers.cjs for safe error message extraction instead of casting and accessing .message directly.",
    },
    hasSuggestions: true,
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    // Maps a variable name known to be produced by an unsafe JSDoc Error cast to
    // the identifier node it was assigned from (used only for reporting context).
    const castVariables = new Map<string, TSESTree.Identifier>();

    return {
      VariableDeclarator(node) {
        if (node.id.type !== AST_NODE_TYPES.Identifier || !node.init) return;
        const source = getCastSourceIdentifier(sourceCode, node.init);
        if (!source) return;
        castVariables.set(node.id.name, source);
      },

      AssignmentExpression(node) {
        if (node.operator !== "=" || node.left.type !== AST_NODE_TYPES.Identifier) return;
        const source = getCastSourceIdentifier(sourceCode, node.right);
        if (!source) return;
        castVariables.set(node.left.name, source);
      },

      MemberExpression(node) {
        const obj = node.object;
        if (obj.type !== AST_NODE_TYPES.Identifier) return;
        if (!castVariables.has(obj.name)) return;

        const prop = node.property;
        let propName: string | null = null;
        if (!node.computed && prop.type === AST_NODE_TYPES.Identifier && UNSAFE_PROPERTIES.has(prop.name)) {
          propName = prop.name;
        } else if (node.computed && prop.type === AST_NODE_TYPES.Literal && typeof prop.value === "string" && UNSAFE_PROPERTIES.has(prop.value)) {
          propName = prop.value;
        }
        if (!propName) return;

        context.report({
          node,
          messageId: "unsafeCast",
          data: { varName: obj.name, prop: propName },
          suggest:
            propName === "message"
              ? [
                  {
                    messageId: "useGetErrorMessage",
                    fix(fixer) {
                      return fixer.replaceText(node, `getErrorMessage(${obj.name})`);
                    },
                  },
                ]
              : [],
        });
      },
    };
  },
});
