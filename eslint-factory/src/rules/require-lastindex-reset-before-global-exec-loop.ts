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

const LOOP_NODE_TYPES = new Set<string>([AST_NODE_TYPES.ForStatement, AST_NODE_TYPES.ForOfStatement, AST_NODE_TYPES.ForInStatement, AST_NODE_TYPES.WhileStatement, AST_NODE_TYPES.DoWhileStatement]);

/**
 * Returns true when `node` is nested inside another loop within the same enclosing
 * function/program (i.e. the exec loop could run again on a later sibling iteration,
 * reusing the same stateful regex across those iterations).
 */
function hasEnclosingLoop(node: TSESTree.Node): boolean {
  let current: TSESTree.Node | undefined = node.parent;
  while (
    current &&
    current.type !== AST_NODE_TYPES.FunctionDeclaration &&
    current.type !== AST_NODE_TYPES.FunctionExpression &&
    current.type !== AST_NODE_TYPES.ArrowFunctionExpression &&
    current.type !== AST_NODE_TYPES.Program
  ) {
    if (LOOP_NODE_TYPES.has(current.type)) return true;
    current = current.parent;
  }
  return false;
}

/**
 * Returns the set of labels that directly label `node` (e.g. `outer: while (...) {}`).
 */
function getOwnLabels(node: TSESTree.Node): Set<string> {
  const labels = new Set<string>();
  let current: TSESTree.Node = node;
  while (current.parent?.type === AST_NODE_TYPES.LabeledStatement && current.parent.body === current) {
    labels.add(current.parent.label.name);
    current = current.parent;
  }
  return labels;
}

/**
 * Walks `body` (without crossing into nested function scopes) looking for a
 * `break`/`return`/`throw`, or a labeled `continue` targeting a loop other than the one
 * labeled by `ownLabels`, any of which would let the loop stop before `.exec()` has a
 * chance to run out of matches and reset `lastIndex` to 0 naturally.
 */
function loopBodyCanExitEarly(body: TSESTree.Node, ownLabels: Set<string>): boolean {
  let exitsEarly = false;

  function visit(node: unknown, inNestedLoopOrSwitch: boolean): void {
    if (exitsEarly || !node || typeof node !== "object" || typeof (node as TSESTree.Node).type !== "string") return;
    const current = node as TSESTree.Node;

    switch (current.type) {
      case AST_NODE_TYPES.FunctionDeclaration:
      case AST_NODE_TYPES.FunctionExpression:
      case AST_NODE_TYPES.ArrowFunctionExpression:
        return; // Don't cross into nested function scopes; their control flow is independent.
      case AST_NODE_TYPES.ReturnStatement:
      case AST_NODE_TYPES.ThrowStatement:
        exitsEarly = true;
        return;
      case AST_NODE_TYPES.BreakStatement:
        // Unlike continue, break always terminates the loop rather than moving on to
        // the next iteration, so it can happen before .exec() naturally returns null,
        // regardless of whether it's labeled with this loop's own label. An unlabeled
        // break inside a nested loop/switch only exits that nested construct though.
        if (current.label || !inNestedLoopOrSwitch) exitsEarly = true;
        return;
      case AST_NODE_TYPES.ContinueStatement:
        // A labeled continue whose label isn't this loop's own label targets some
        // outer loop, ending this loop's iteration early.
        if (current.label && !ownLabels.has(current.label.name)) exitsEarly = true;
        return;
      default:
        break;
    }

    const isLoopOrSwitch = LOOP_NODE_TYPES.has(current.type) || current.type === AST_NODE_TYPES.SwitchStatement;
    for (const key of Object.keys(current)) {
      if (key === "parent") continue;
      const value = (current as unknown as Record<string, unknown>)[key];
      if (Array.isArray(value)) {
        for (const item of value) visit(item, inNestedLoopOrSwitch || isLoopOrSwitch);
      } else {
        visit(value, inNestedLoopOrSwitch || isLoopOrSwitch);
      }
    }
  }

  visit(body, false);
  return exitsEarly;
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
          // If this exec loop is nested inside another loop (reusing the same regex
          // across sibling iterations, e.g. one `while` per field of an outer `for`)
          // and its body has no `break`/`return`/`throw`/outer-`continue`, it is
          // guaranteed to run to natural exhaustion every time, at which point `.exec()`
          // resets `lastIndex` to 0 on its own. There's nothing to fix in that case.
          if (hasEnclosingLoop(node) && !loopBodyCanExitEarly(node.body, getOwnLabels(node))) {
            return;
          }

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
