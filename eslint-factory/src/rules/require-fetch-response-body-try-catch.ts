import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";
import { buildTryCatchSuggestion, isDeferredCallback, SAFE_WRAPPABLE_STATEMENT_TYPES } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Statement node types that can be directly wrapped in a try/catch block.
const WRAPPABLE_STATEMENT_TYPES = new Set<AST_NODE_TYPES>([...SAFE_WRAPPABLE_STATEMENT_TYPES, AST_NODE_TYPES.VariableDeclaration]);

const BODY_METHOD_NAMES = new Set(["json", "text"]);

function unwrapChain(node: TSESTree.Expression | TSESTree.Super): TSESTree.Expression | TSESTree.Super {
  return node.type === AST_NODE_TYPES.ChainExpression ? node.expression : node;
}

function getMemberPropertyName(node: TSESTree.MemberExpression): string | null {
  const property = node.property;
  if (!node.computed && property.type === AST_NODE_TYPES.Identifier) return property.name;
  if (node.computed && property.type === AST_NODE_TYPES.Literal && typeof property.value === "string") return property.value;
  return null;
}

/** Returns true when expr is `await fetch(...)` (direct global fetch call, no member chain). */
function isDirectAwaitFetchCall(expr: TSESTree.Expression): boolean {
  if (expr.type !== AST_NODE_TYPES.AwaitExpression) return false;
  const argument = unwrapChain(expr.argument);
  if (argument.type !== AST_NODE_TYPES.CallExpression) return false;
  return argument.callee.type === AST_NODE_TYPES.Identifier && argument.callee.name === "fetch";
}

export const requireFetchResponseBodyTryCatchRule = createRule({
  name: "require-fetch-response-body-try-catch",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require .json()/.text() calls on a fetch() Response in actions/setup/js scripts to be wrapped in try/catch. " +
        "Both methods reject when the body stream errors mid-read or (for .json()) when the payload is not valid JSON, " +
        "so a call-site try/catch preserves the original error as `{ cause }` and produces a specific message instead of a generic engine-level stack.",
    },
    schema: [],
    messages: {
      requireTryCatch:
        "Wrap {{call}} in try/catch — even after explicit HTTP-error handling (for example `if (!response.ok) throw ...`), " +
        "reading a fetch() Response body can still reject (malformed JSON, truncated/errored stream). Without this call-site try/catch, " +
        "you lose the original parse error context and get a generic, harder-to-diagnose stack instead of a specific message with `{ cause }`.",
      wrapInTryCatch: "Wrap in try { ... } catch { ... } and re-throw with { cause: err } to preserve context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;
    type SourceCodeScope = ReturnType<typeof sourceCode.getScope>;

    function isInsideTryBlock(node: TSESTree.Node): boolean {
      const ancestors = sourceCode.getAncestors(node);
      let crossedDeferredBoundary = false;

      for (let i = ancestors.length - 1; i >= 0; i--) {
        const ancestor = ancestors[i];

        if (isDeferredCallback(ancestor)) {
          crossedDeferredBoundary = true;
        }

        if (ancestor.type === AST_NODE_TYPES.TryStatement && !crossedDeferredBoundary && ancestor.handler != null) {
          const block = ancestor.block;
          if (node.range != null && block.range != null && node.range[0] >= block.range[0] && node.range[1] <= block.range[1]) {
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
        if (WRAPPABLE_STATEMENT_TYPES.has(ancestor.type)) {
          return ancestor as TSESTree.Statement;
        }
      }
      return null;
    }

    function canSuggestWrapStatement(stmt: TSESTree.Statement): boolean {
      if (stmt.type !== AST_NODE_TYPES.VariableDeclaration) {
        return true;
      }

      const parent = stmt.parent;
      const isStandaloneVariableDeclaration =
        parent != null &&
        ((parent.type === AST_NODE_TYPES.Program && parent.body.includes(stmt)) ||
          (parent.type === AST_NODE_TYPES.BlockStatement && parent.body.includes(stmt)) ||
          (parent.type === AST_NODE_TYPES.SwitchCase && parent.consequent.includes(stmt)));
      if (!isStandaloneVariableDeclaration) {
        return false;
      }

      if (stmt.kind === "var") {
        return true;
      }

      const statementRange = stmt.range;
      if (!statementRange) return false;
      const [statementStart, statementEnd] = statementRange;

      const hasReferenceOutsideStatement = sourceCode.getDeclaredVariables(stmt).some(variable =>
        variable.references.some(reference => {
          const referenceRange = reference.identifier.range;
          return referenceRange == null || referenceRange[0] < statementStart || referenceRange[1] > statementEnd;
        })
      );

      return !hasReferenceOutsideStatement;
    }

    /**
     * Finds the write reference to the variable bound to `node` that actually reaches
     * this read — i.e. the last write occurring strictly before `node` in program order —
     * rather than treating "any write anywhere in scope" as relevant. Returns null if no
     * such write exists in the visible scope chain.
     */
    function findReachingWrite(node: TSESTree.Identifier): TSESTree.Expression | null {
      let scope: SourceCodeScope | null = sourceCode.getScope(node);
      while (scope) {
        const variable = scope.set.get(node.name);
        if (variable) {
          let reaching: TSESTree.Expression | null = null;
          let reachingPos = -Infinity;
          for (const ref of variable.references) {
            const writeExpr = ref.writeExpr;
            if (writeExpr == null) continue;
            const refPos = ref.identifier.range?.[0] ?? -Infinity;
            const nodePos = node.range?.[0] ?? Infinity;
            if (refPos < nodePos && refPos > reachingPos) {
              reaching = writeExpr as TSESTree.Expression;
              reachingPos = refPos;
            }
          }
          return reaching;
        }
        scope = scope.upper;
      }
      return null;
    }

    /**
     * Returns true when the identifier at `node` resolves — via the write reference that
     * actually reaches this read (the last write before it in program order), following a
     * single-hop `const` alias if needed — to a bare `await fetch(...)` call, i.e. a Response
     * obtained with no chained `.catch`/rejection handling of its own.
     */
    function resolvesToBareAwaitFetch(node: TSESTree.Identifier, aliasHopsRemaining = 1): boolean {
      const writeExpr = findReachingWrite(node);
      if (writeExpr == null) return false;
      if (isDirectAwaitFetchCall(writeExpr)) return true;
      if (aliasHopsRemaining > 0 && writeExpr.type === AST_NODE_TYPES.Identifier) {
        return resolvesToBareAwaitFetch(writeExpr, aliasHopsRemaining - 1);
      }
      return false;
    }

    return {
      AwaitExpression(node) {
        const argument = unwrapChain(node.argument);
        if (argument.type !== AST_NODE_TYPES.CallExpression) return;
        const callee = argument.callee;
        if (callee.type !== AST_NODE_TYPES.MemberExpression) return;

        const methodName = getMemberPropertyName(callee);
        if (methodName === null || !BODY_METHOD_NAMES.has(methodName)) return;

        const objectExpr = unwrapChain(callee.object);
        // Case 1: the call chain itself starts from a direct `fetch(...)` call, e.g.
        // `await fetch(url).json()`.
        const isDirectChain = objectExpr.type === AST_NODE_TYPES.CallExpression && objectExpr.callee.type === AST_NODE_TYPES.Identifier && objectExpr.callee.name === "fetch";
        // Case 2: the receiver is a variable whose value came from a bare `await fetch(...)`,
        // e.g. `const response = await fetch(url); ... await response.json();`.
        const isResolvedFetchVar = !isDirectChain && objectExpr.type === AST_NODE_TYPES.Identifier && resolvesToBareAwaitFetch(objectExpr);

        if (!isDirectChain && !isResolvedFetchVar) return;
        if (isInsideTryBlock(node)) return;

        const callText = sourceCode.getText(argument);
        const stmt = findEnclosingStatement(node);

        context.report({
          node,
          messageId: "requireTryCatch",
          data: { call: callText },
          suggest:
            stmt && canSuggestWrapStatement(stmt)
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
                          todoComment: "TODO: handle a malformed/errored fetch response body for this call.",
                          errorPrefix: `Failed to read fetch response ${methodName}(): `,
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
