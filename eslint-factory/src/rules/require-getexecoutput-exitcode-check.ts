import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/**
 * Returns true when the call expression is `<obj>.getExecOutput(...)` — the `@actions/exec`
 * helper that returns `{ exitCode, stdout, stderr }`. Matches any receiver name (`exec`,
 * `execApi`, `this.exec`, etc.) since the module is imported under different local aliases
 * across actions/setup/js.
 */
function isGetExecOutputCall(node: TSESTree.Expression): node is TSESTree.CallExpression {
  if (node.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = node.callee;
  return callee.type === AST_NODE_TYPES.MemberExpression && !callee.computed && callee.property.type === AST_NODE_TYPES.Identifier && callee.property.name === "getExecOutput";
}

/**
 * Returns true when the options argument (last argument, if an object literal) contains a
 * statically-true `ignoreReturnCode` property. Only object literals are inspected; spreads
 * and identifiers are treated conservatively as "may set it" since they can't be statically
 * resolved, which would otherwise produce false positives.
 */
function hasIgnoreReturnCodeTrue(node: TSESTree.CallExpression): boolean {
  const optionsArg = node.arguments[node.arguments.length - 1];
  if (!optionsArg || optionsArg.type !== AST_NODE_TYPES.ObjectExpression) return false;

  for (const prop of optionsArg.properties) {
    if (prop.type === AST_NODE_TYPES.SpreadElement) {
      // Can't statically confirm the spread doesn't carry ignoreReturnCode: true;
      // assume it might to avoid false positives on options composed via `{ ...opts }`.
      return true;
    }
    if (prop.type !== AST_NODE_TYPES.Property || prop.computed) continue;
    const isIgnoreReturnCodeKey = (prop.key.type === AST_NODE_TYPES.Identifier && prop.key.name === "ignoreReturnCode") || (prop.key.type === AST_NODE_TYPES.Literal && prop.key.value === "ignoreReturnCode");
    if (!isIgnoreReturnCodeKey) continue;
    if (prop.value.type === AST_NODE_TYPES.Literal && prop.value.value === true) return true;
  }

  return false;
}

export const requireGetExecOutputExitCodeCheckRule = createRule({
  name: "require-getexecoutput-exitcode-check",
  meta: {
    type: "problem",
    docs: {
      description:
        "Require the exitCode from @actions/exec getExecOutput() to be read (destructured or accessed) when the call passes { ignoreReturnCode: true }. " +
        "ignoreReturnCode: true suppresses the automatic throw-on-nonzero-exit behavior, so the caller becomes solely responsible for detecting failure; " +
        "discarding exitCode (e.g. only destructuring { stdout }) silently swallows command failures and proceeds with empty or stale output. " +
        "Scope: this rule only inspects the immediate destructuring pattern or member-expression access on the awaited/returned call result; " +
        "results forwarded to a helper function that checks exitCode internally are out of scope and will not satisfy the rule.",
    },
    schema: [],
    messages: {
      missingExitCodeCheck:
        "getExecOutput() is called with ignoreReturnCode: true but its exitCode is never read. " +
        "Without the default throw-on-failure behavior, a non-zero exit code is silently ignored. " +
        "Destructure exitCode and check it (e.g. `const { stdout, exitCode } = await exec.getExecOutput(...); if (exitCode !== 0) { ... }`).",
    },
  },
  defaultOptions: [],
  create(context) {
    /** Returns true when an ObjectPattern includes an `exitCode` binding. */
    function objectPatternHasExitCode(pattern: TSESTree.ObjectPattern): boolean {
      return pattern.properties.some(prop => {
        if (prop.type === AST_NODE_TYPES.RestElement) return true; // `...rest` may capture exitCode
        if (prop.computed) return true; // can't statically rule out; avoid false positive
        return (prop.key.type === AST_NODE_TYPES.Identifier && prop.key.name === "exitCode") || (prop.key.type === AST_NODE_TYPES.Literal && prop.key.value === "exitCode");
      });
    }

    function reportIfMissing(call: TSESTree.CallExpression, resultNode: TSESTree.Node) {
      const parent = resultNode.parent;
      if (!parent) {
        context.report({ node: call, messageId: "missingExitCodeCheck" });
        return;
      }

      // const { stdout, exitCode } = await getExecOutput(...)
      if (parent.type === AST_NODE_TYPES.VariableDeclarator && parent.init === resultNode) {
        if (parent.id.type === AST_NODE_TYPES.ObjectPattern) {
          if (!objectPatternHasExitCode(parent.id)) {
            context.report({ node: call, messageId: "missingExitCodeCheck" });
          }
          return;
        }
        if (parent.id.type === AST_NODE_TYPES.Identifier) {
          // const result = await getExecOutput(...); look for result.exitCode usages.
          const variable = findInUpperScopes(context.sourceCode.getScope(parent), parent.id.name);
          const usesExitCode = variable?.references.some(ref => {
            const id = ref.identifier;
            const idParent = id.parent;
            return idParent !== undefined && idParent.type === AST_NODE_TYPES.MemberExpression && !idParent.computed && idParent.object === id && idParent.property.type === AST_NODE_TYPES.Identifier && idParent.property.name === "exitCode";
          });
          if (!usesExitCode) {
            context.report({ node: call, messageId: "missingExitCodeCheck" });
          }
          return;
        }
        // Other binding shapes (array pattern, etc.) are out of scope; don't flag.
        return;
      }

      // Direct member access: (await getExecOutput(...)).exitCode
      if (parent.type === AST_NODE_TYPES.MemberExpression && parent.object === resultNode && !parent.computed && parent.property.type === AST_NODE_TYPES.Identifier && parent.property.name === "exitCode") {
        return;
      }

      // Any other usage (passed as an argument, returned directly, etc.) can't be
      // statically verified to check exitCode; treat conservatively as reported since
      // ignoreReturnCode: true is the strongest failure-suppression signal we see.
      if (parent.type !== AST_NODE_TYPES.AwaitExpression) {
        context.report({ node: call, messageId: "missingExitCodeCheck" });
      }
    }

    // Fallback scope walk mirrors patterns used elsewhere in this rule set for resolving
    // a variable across nested function/block scopes.
    function findInUpperScopes(scope: ReturnType<typeof context.sourceCode.getScope> | null, name: string) {
      let current = scope;
      while (current) {
        const variable = current.set.get(name);
        if (variable) return variable;
        current = current.upper;
      }
      return undefined;
    }

    return {
      CallExpression(node: TSESTree.CallExpression) {
        if (!isGetExecOutputCall(node)) return;
        if (!hasIgnoreReturnCodeTrue(node)) return;

        // Walk up through an optional AwaitExpression wrapper to find the real usage site.
        const usageNode: TSESTree.Node = node.parent && node.parent.type === AST_NODE_TYPES.AwaitExpression && node.parent.argument === node ? node.parent : node;

        reportIfMissing(node, usageNode);
      },
    };
  },
});
