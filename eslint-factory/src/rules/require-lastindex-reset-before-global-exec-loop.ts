import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/**
 * Returns true when `node` is a RegExpLiteral with the global (`g`) or sticky (`y`) flag,
 * the stateful flags that make `.exec()` resume from `.lastIndex` on each call.
 */
function isStatefulRegexLiteral(node: TSESTree.Node): node is TSESTree.RegExpLiteral {
  return node.type === AST_NODE_TYPES.Literal && "regex" in node && typeof node.regex?.flags === "string" && (node.regex.flags.includes("g") || node.regex.flags.includes("y"));
}

/**
 * Matches the common `while ((match = RE.exec(str)) !== null)` idiom and returns the
 * name of the regex identifier being executed, or null if the test doesn't match this shape.
 */
function getExecLoopRegexName(test: TSESTree.Expression): string | null {
  if (test.type !== AST_NODE_TYPES.BinaryExpression || test.operator !== "!==") return null;

  let assignExpr: TSESTree.Expression = test.left;
  // Allow either `(match = RE.exec(x)) !== null` or `null !== (match = RE.exec(x))`
  if (assignExpr.type !== AST_NODE_TYPES.AssignmentExpression) {
    assignExpr = test.right;
  }
  if (assignExpr.type !== AST_NODE_TYPES.AssignmentExpression) return null;

  const rhs = assignExpr.right;
  if (rhs.type !== AST_NODE_TYPES.CallExpression) return null;
  const callee = rhs.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return null;
  if (callee.object.type !== AST_NODE_TYPES.Identifier) return null;
  if (callee.property.type !== AST_NODE_TYPES.Identifier || callee.property.name !== "exec") return null;

  return callee.object.name;
}

export const requireLastIndexResetBeforeGlobalExecLoopRule = createRule({
  name: "require-lastindex-reset-before-global-exec-loop",
  meta: {
    type: "problem",
    docs: {
      description:
        "Require resetting `.lastIndex = 0` on a module-scoped global/sticky regex before a `while ((match = RE.exec(str)))` loop, since the shared stateful regex resumes scanning from wherever the previous call left off across separate invocations.",
    },
    schema: [],
    messages: {
      requireLastIndexReset:
        "Regex '{{name}}' has the 'g' or 'y' flag and is reused across calls, but its 'lastIndex' is never reset before this exec loop. If a prior call ended mid-string (e.g. threw, returned early, or ran out of matches on shorter input), this loop can silently skip matches or miss content entirely. Add '{{name}}.lastIndex = 0;' before the loop.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    // Names of identifiers initialized (at module/outer scope) to a stateful ('g'/'y') regex literal.
    const statefulRegexNames = new Set<string>();

    return {
      VariableDeclarator(node) {
        // Only module/outer-scope declarations are stateful across separate calls into
        // functions that use them; regex literals declared inside a function are freshly
        // created (with lastIndex 0) on every invocation and are not at risk.
        if (node.parent?.parent?.type !== AST_NODE_TYPES.Program) return;
        if (node.id.type === AST_NODE_TYPES.Identifier && node.init && isStatefulRegexLiteral(node.init)) {
          statefulRegexNames.add(node.id.name);
        }
      },

      WhileStatement(node) {
        if (node.test.type === AST_NODE_TYPES.Literal || node.test.type === AST_NODE_TYPES.Identifier) return;
        if (node.test.type !== AST_NODE_TYPES.BinaryExpression) return;

        const regexName = getExecLoopRegexName(node.test);
        if (!regexName || !statefulRegexNames.has(regexName)) return;

        // Search all text preceding this while-loop within the same function/program
        // for an explicit `<regexName>.lastIndex = ...` reset.
        const resetPattern = new RegExp(`\\b${regexName}\\s*\\.\\s*lastIndex\\s*=`);
        const textBefore = sourceCode.getText().slice(0, node.range[0]);
        // Only look within the nearest enclosing function to avoid false negatives from
        // resets that belong to an unrelated, earlier function using the same regex.
        let enclosing: TSESTree.Node | undefined = node.parent;
        while (
          enclosing &&
          enclosing.type !== AST_NODE_TYPES.FunctionDeclaration &&
          enclosing.type !== AST_NODE_TYPES.FunctionExpression &&
          enclosing.type !== AST_NODE_TYPES.ArrowFunctionExpression &&
          enclosing.type !== AST_NODE_TYPES.Program
        ) {
          enclosing = enclosing.parent;
        }
        const scanStart = enclosing ? enclosing.range[0] : 0;
        const relevantTextBefore = textBefore.slice(scanStart);

        if (!resetPattern.test(relevantTextBefore)) {
          context.report({
            node,
            messageId: "requireLastIndexReset",
            data: { name: regexName },
          });
        }
      },
    };
  },
});
