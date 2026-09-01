import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, findEnclosingStatement, isInsideTryBlock } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const FLAGGED_GLOBALS = new Set(["decodeURIComponent", "decodeURI"]);

export const requireDecodeURIComponentTryCatchRule = createRule({
  name: "require-decodeuricomponent-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require decodeURIComponent(...) and decodeURI(...) calls on dynamic input in actions/setup/js scripts to be wrapped in try/catch. " +
        "Both throw a URIError on malformed percent-encoding (e.g. a lone '%', an incomplete escape sequence, or a surrogate-pair mismatch), " +
        "when the input comes from an external or untrusted source (HTTP headers, URLs, workflow inputs, or other user-controlled text). " +
        "Without a call-site try/catch, the entrypoint-level catch produces a generic engine-level stack instead of a specific message that preserves the error as `{ cause }`.",
    },
    schema: [],
    messages: {
      requireTryCatch:
        "Wrap {{callee}}({{arg}}) in try/catch — malformed percent-encoded input throws URIError; without a call-site try/catch, you lose the original error context and get a generic engine-level stack instead of a specific message with `{ cause }`.",
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

    /** Returns true when an expression coerces to a compile-time constant without percent-encoding (never throws). */
    function isStaticStringExpression(arg: TSESTree.CallExpressionArgument): boolean {
      if (arg.type === AST_NODE_TYPES.Literal && (typeof arg.value === "string" || typeof arg.value === "number" || typeof arg.value === "boolean" || arg.value === null)) return true;
      if (arg.type === AST_NODE_TYPES.TemplateLiteral && (arg as TSESTree.TemplateLiteral).expressions.length === 0) return true;
      if (arg.type === AST_NODE_TYPES.BinaryExpression && arg.operator === "+") {
        return isStaticStringExpression(arg.left) && isStaticStringExpression(arg.right);
      }
      return false;
    }

    function isDynamicArg(arg: TSESTree.CallExpressionArgument): boolean {
      if (arg.type === "SpreadElement") return false;
      return !isStaticStringExpression(arg);
    }

    return {
      CallExpression(node) {
        if (node.callee.type !== AST_NODE_TYPES.Identifier || !FLAGGED_GLOBALS.has(node.callee.name)) return;
        if (hasLocalBinding(node, node.callee.name)) return;

        const firstArg = node.arguments[0];
        if (!firstArg || !isDynamicArg(firstArg)) return;

        if (isInsideTryBlock(sourceCode, node)) return;

        const argText = sourceCode.getText(firstArg as TSESTree.Node);
        const stmt = findEnclosingStatement(sourceCode, node);

        context.report({
          node,
          messageId: "requireTryCatch",
          data: { callee: node.callee.name, arg: argText },
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
                        todoComment: `TODO: handle malformed percent-encoding for this ${node.callee.type === AST_NODE_TYPES.Identifier ? node.callee.name : "decode"}(...) call.`,
                        errorPrefix: "URI decoding failed: ",
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
