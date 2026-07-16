import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, isDeferredCallback, SAFE_WRAPPABLE_STATEMENT_TYPES } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

export const requireNewUrlTryCatchRule = createRule({
  name: "require-new-url-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require new URL(variable) calls in actions/setup/js scripts to be wrapped in try/catch. " +
        "The URL constructor throws a TypeError when given an invalid or relative URL string, " +
        "which crashes the action with an unhelpful uncaught exception.",
    },
    schema: [],
    messages: {
      requireTryCatch: "Wrap new URL({{arg}}) in try/catch — the URL constructor throws TypeError for invalid or relative URLs " + "and will crash the action if unhandled.",
      wrapInTryCatch: "Wrap in try { ... } catch { ... } and re-throw with { cause: err } to preserve context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

    /** Returns true when name is bound by a local definition, meaning it shadows the global. */
    function hasLocalBinding(node: TSESTree.Node, name: string): boolean {
      let scope: SourceCodeScope | null = sourceCode.getScope(node);
      while (scope) {
        const variable = scope.set.get(name);
        if (variable?.defs.length) {
          return true;
        }
        scope = scope.upper;
      }
      return false;
    }

    function isInsideTryBlock(node: TSESTree.Node): boolean {
      const ancestors = sourceCode.getAncestors(node);
      let crossedDeferredBoundary = false;

      for (let i = ancestors.length - 1; i >= 0; i--) {
        const ancestor = ancestors[i];

        if (isDeferredCallback(ancestor)) {
          crossedDeferredBoundary = true;
        }

        if (ancestor.type === "TryStatement" && !crossedDeferredBoundary && ancestor.handler != null) {
          const block = ancestor.block;
          if (node.range != null && block.range != null && node.range[0] >= block.range[0] && node.range[1] <= block.range[1]) {
            return true;
          }
        }
      }

      return false;
    }

    // Only ExpressionStatement and ReturnStatement are safe to wrap: they are self-contained
    // statements whose removal does not leave other code referencing out-of-scope bindings.
    // VariableDeclaration is intentionally excluded: wrapping `const x = new URL(v)` would
    // place subsequent uses of `x` outside the try block, leaving them unreachable.
    function findEnclosingStatement(node: TSESTree.Node): TSESTree.Statement | null {
      const ancestors = sourceCode.getAncestors(node);
      for (let i = ancestors.length - 1; i >= 0; i--) {
        const ancestor = ancestors[i];
        if (SAFE_WRAPPABLE_STATEMENT_TYPES.has(ancestor.type)) {
          return ancestor as TSESTree.Statement;
        }
      }
      return null;
    }

    /** Returns true when an argument is a runtime-dynamic expression (not a compile-time constant). */
    function isDynamicArg(arg: TSESTree.CallExpressionArgument): boolean {
      if (arg.type === "SpreadElement") return false;
      // Literal strings are compile-time constants — no runtime parse risk.
      if (arg.type === AST_NODE_TYPES.Literal && typeof (arg as TSESTree.StringLiteral).value === "string") return false;
      // Template literals with no expressions are effectively string constants.
      if (arg.type === AST_NODE_TYPES.TemplateLiteral && (arg as TSESTree.TemplateLiteral).expressions.length === 0) return false;
      return true;
    }

    /**
     * Returns true when a base argument is a known-safe compile-time value that never throws.
     * `import.meta.url` is always a valid absolute URL in ES modules.
     */
    function isKnownSafeBase(arg: TSESTree.CallExpressionArgument): boolean {
      // import.meta.url is a MemberExpression: { object: MetaProperty(import.meta), property: Identifier(url) }
      if (arg.type !== AST_NODE_TYPES.MemberExpression) return false;
      const memberExpr = arg as TSESTree.MemberExpression;
      if (memberExpr.computed) return false;
      if (memberExpr.property.type !== AST_NODE_TYPES.Identifier || (memberExpr.property as TSESTree.Identifier).name !== "url") return false;
      if (memberExpr.object.type !== AST_NODE_TYPES.MetaProperty) return false;
      const metaProp = memberExpr.object as TSESTree.MetaProperty;
      return metaProp.meta.name === "import" && metaProp.property.name === "meta";
    }

    return {
      NewExpression(node) {
        // Only flag `new URL(...)` — the global URL constructor.
        if (node.callee.type !== AST_NODE_TYPES.Identifier || node.callee.name !== "URL") return;
        // Skip when URL is shadowed by a local binding (e.g. a parameter or import named URL).
        if (hasLocalBinding(node, "URL")) return;

        const firstArg = node.arguments[0];
        const secondArg = node.arguments[1];

        // `new URL()` with zero arguments always throws TypeError at runtime — always flag it.
        const noArgs = firstArg === undefined;
        // Flag when the first argument is a runtime-dynamic expression.
        const firstArgDynamic = !noArgs && isDynamicArg(firstArg);
        // Flag when the second (base) argument is dynamic and not a known-safe value such as
        // import.meta.url. An invalid base throws the same TypeError as an invalid URL string.
        const secondArgDynamic = secondArg !== undefined && !isKnownSafeBase(secondArg) && isDynamicArg(secondArg);

        if (!noArgs && !firstArgDynamic && !secondArgDynamic) return;

        if (isInsideTryBlock(node)) return;

        // Show the first dynamic argument in the diagnostic message, or empty string for zero-arg calls.
        const reportArgNode = noArgs ? null : firstArgDynamic ? firstArg : (secondArg as TSESTree.CallExpressionArgument);
        const argText = reportArgNode !== null ? sourceCode.getText(reportArgNode as TSESTree.Node) : "";
        const stmt = findEnclosingStatement(node);

        context.report({
          node,
          messageId: "requireTryCatch",
          data: { arg: argText },
          suggest: stmt
            ? [
                {
                  messageId: "wrapInTryCatch",
                  fix(fixer) {
                    const stmtText = sourceCode.getText(stmt);
                    const startLine = stmt.loc?.start.line;
                    const stmtLine = startLine !== undefined ? (sourceCode.lines[startLine - 1] ?? "") : "";
                    const indent = stmtLine.match(/^(\s*)/)?.[1] ?? "";
                    return fixer.replaceText(
                      stmt,
                      buildTryCatchSuggestion(stmtText, {
                        indent,
                        todoComment: "TODO: handle invalid URL for this new URL(...) call.",
                        errorPrefix: "URL constructor call failed: ",
                      })
                    );
                  },
                },
              ]
            : [],
        });
      },
    };
  },
});
