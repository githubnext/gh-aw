import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Statement node types that can be directly wrapped in a try/catch block.
const WRAPPABLE_STATEMENT_TYPES = new Set<AST_NODE_TYPES>([AST_NODE_TYPES.ExpressionStatement, AST_NODE_TYPES.VariableDeclaration, AST_NODE_TYPES.ReturnStatement, AST_NODE_TYPES.ThrowStatement]);

// Method/function names that accept callbacks whose throws are not protected by a try that
// encloses the call site. Most execute later (outside the dynamic extent of the surrounding try);
// Promise executors are synchronous, but Promise captures their throws instead of letting the
// outer try observe them.
const DEFERRED_SINK_NAMES = new Set([
  "then",
  "catch",
  "finally", // Promise methods
  "on",
  "once", // EventEmitter / Node streams
  "addEventListener", // DOM / Node
  "setTimeout",
  "setInterval",
  "setImmediate",
  "queueMicrotask",
  "nextTick", // process.nextTick
]);

function isFunctionExpressionLike(node: TSESTree.Node): node is TSESTree.ArrowFunctionExpression | TSESTree.FunctionExpression {
  return node.type === AST_NODE_TYPES.ArrowFunctionExpression || node.type === AST_NODE_TYPES.FunctionExpression;
}

/** Returns true when funcNode is passed to a callback sink not protected by the outer try. */
function isDeferredCallback(funcNode: TSESTree.Node): boolean {
  if (!isFunctionExpressionLike(funcNode)) return false;

  const parent = funcNode.parent;
  if (!parent) return false;

  const isCallLikeParent = parent.type === "NewExpression" || parent.type === "CallExpression";
  const args = isCallLikeParent ? parent.arguments : undefined;
  const isArgument = args?.includes(funcNode) ?? false;
  // Only direct new Promise(...) is in scope here; aliased constructors are intentionally ignored.
  const isPromiseConstructor = parent.type === "NewExpression" && parent.callee.type === "Identifier" && parent.callee.name === "Promise";
  if (isPromiseConstructor && isArgument) {
    return true;
  }

  // obj.method(cb) or globalFn(cb) where the method/function is a known deferred sink
  if (parent.type === "CallExpression" && isArgument) {
    const callee = parent.callee;
    if (callee.type === "Identifier" && DEFERRED_SINK_NAMES.has(callee.name)) {
      return true;
    }
    if (callee.type === "MemberExpression" && !callee.computed && callee.property.type === "Identifier") {
      return DEFERRED_SINK_NAMES.has(callee.property.name);
    }
  }

  return false;
}

export const requireJsonParseTryCatchRule = createRule({
  name: "require-json-parse-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description: "Require JSON.parse calls in actions/setup/js scripts to be wrapped in try/catch",
    },
    schema: [],
    messages: {
      requireTryCatch: "Wrap JSON.parse({{arg}}) in try/catch to avoid uncaught runtime failures in actions/setup/js.",
      useHelper: "Wrap in try { ... } catch { ... }. For JSONL or possibly-malformed JSON, prefer the established safe-parse helpers: parseJsonWithRepair (collect_ndjson_output.cjs) or parseJsonlContent (jsonl_helpers.cjs).",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    function isInsideTryBlock(node: TSESTree.Node): boolean {
      const ancestors = sourceCode.getAncestors(node);
      // Walk from innermost ancestor outward. If we cross a deferred function boundary
      // (e.g., a .then/.on/setTimeout callback), a try statement further out does NOT
      // protect the node — the callback runs after the try has already returned.
      let crossedDeferredBoundary = false;

      for (let i = ancestors.length - 1; i >= 0; i--) {
        const ancestor = ancestors[i];

        if (isDeferredCallback(ancestor)) {
          crossedDeferredBoundary = true;
        }

        if (ancestor.type === "TryStatement" && !crossedDeferredBoundary) {
          const block = ancestor.block;
          if (node.range[0] >= block.range[0] && node.range[1] <= block.range[1]) {
            return true;
          }
        }
      }

      return false;
    }

    function findEnclosingStatement(node: TSESTree.Node): TSESTree.Statement | null {
      const ancestors = sourceCode.getAncestors(node);
      for (let i = ancestors.length - 1; i >= 0; i--) {
        const ancestor = ancestors[i];
        // Safe cast: WRAPPABLE_STATEMENT_TYPES only contains statement node types.
        if (WRAPPABLE_STATEMENT_TYPES.has(ancestor.type)) {
          return ancestor as TSESTree.Statement;
        }
      }
      return null;
    }

    return {
      CallExpression(node) {
        if (node.callee.type !== "MemberExpression") {
          return;
        }

        if (node.callee.object.type !== "Identifier") {
          return;
        }

        if (node.callee.object.name !== "JSON") {
          return;
        }

        // Accept both direct property access (JSON.parse) and computed string-literal
        // access (JSON["parse"]). Aliased (const p = JSON.parse; p(raw)) and
        // destructured (const { parse } = JSON; parse(raw)) bindings are intentionally
        // out of scope: tracking them reliably requires full scope analysis and is
        // disproportionate to the current risk surface.
        const property = node.callee.property;
        const isParseProperty = (property.type === "Identifier" && property.name === "parse") || (property.type === "Literal" && property.value === "parse");

        if (!isParseProperty) {
          return;
        }

        if (!isInsideTryBlock(node)) {
          const argText = node.arguments.length > 0 ? sourceCode.getText(node.arguments[0]) : "";

          context.report({
            node,
            messageId: "requireTryCatch",
            data: { arg: argText },
            suggest: [
              {
                messageId: "useHelper",
                fix(fixer) {
                  const stmt = findEnclosingStatement(node);
                  if (!stmt) return null;
                  const stmtText = sourceCode.getText(stmt);
                  // ESLint always sets loc on parsed nodes; the optional chain guards
                  // against hypothetical missing loc. loc.start.line is 1-based, so
                  // subtract 1 for the 0-based lines array index.
                  const startLine = stmt.loc?.start.line;
                  const stmtLine = startLine !== undefined ? (sourceCode.lines[startLine - 1] ?? "") : "";
                  const indent = stmtLine.match(/^(\s*)/)?.[1] ?? "";
                  return fixer.replaceText(stmt, `try {\n${indent}  ${stmtText}\n${indent}} catch (err) {\n${indent}  throw err;\n${indent}}`);
                },
              },
            ],
          });
        }
      },
    };
  },
});
