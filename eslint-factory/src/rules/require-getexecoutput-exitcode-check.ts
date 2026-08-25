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
 * Returns true when the options argument (last argument, if an object literal) resolves to a
 * statically-true `ignoreReturnCode`. Properties are evaluated in source order so that the last
 * write wins. A spread makes the value unresolvable from that point on, so options that end with
 * a spread (or that only contain spreads) are treated as out of scope to avoid false positives;
 * an explicit `ignoreReturnCode: true` written after a spread still counts, since it overrides it.
 */
function hasIgnoreReturnCodeTrue(node: TSESTree.CallExpression): boolean {
  const optionsArg = node.arguments[node.arguments.length - 1];
  if (!optionsArg || optionsArg.type !== AST_NODE_TYPES.ObjectExpression) return false;

  let ignoreReturnCode: boolean | "unresolved" = false;

  for (const prop of optionsArg.properties) {
    if (prop.type === AST_NODE_TYPES.SpreadElement) {
      // The spread may carry an `ignoreReturnCode` value we can't statically resolve.
      ignoreReturnCode = "unresolved";
      continue;
    }
    if (prop.type !== AST_NODE_TYPES.Property || prop.computed) continue;
    const isIgnoreReturnCodeKey = (prop.key.type === AST_NODE_TYPES.Identifier && prop.key.name === "ignoreReturnCode") || (prop.key.type === AST_NODE_TYPES.Literal && prop.key.value === "ignoreReturnCode");
    if (!isIgnoreReturnCodeKey) continue;
    ignoreReturnCode = prop.value.type === AST_NODE_TYPES.Literal && typeof prop.value.value === "boolean" ? prop.value.value : "unresolved";
  }

  return ignoreReturnCode === true;
}

function isExitCodeMemberAccess(memberExpression: TSESTree.MemberExpression, object: TSESTree.Node): boolean {
  if (memberExpression.object !== object) return false;

  if (!memberExpression.computed) {
    return memberExpression.property.type === AST_NODE_TYPES.Identifier && memberExpression.property.name === "exitCode";
  }

  return memberExpression.property.type === AST_NODE_TYPES.Literal && memberExpression.property.value === "exitCode";
}

function isReturnedFromFunctionExpression(node: TSESTree.Node): boolean {
  const parent = node.parent;
  if (!parent) return false;
  if (parent.type === AST_NODE_TYPES.ArrowFunctionExpression && parent.body === node) return true;
  if (parent.type !== AST_NODE_TYPES.ReturnStatement || parent.argument !== node) return false;

  let current: TSESTree.Node | undefined = parent.parent;
  while (current && current.type !== AST_NODE_TYPES.FunctionDeclaration && current.type !== AST_NODE_TYPES.FunctionExpression && current.type !== AST_NODE_TYPES.ArrowFunctionExpression) {
    current = current.parent;
  }
  return current?.type === AST_NODE_TYPES.FunctionExpression || current?.type === AST_NODE_TYPES.ArrowFunctionExpression;
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
            return idParent !== undefined && idParent.type === AST_NODE_TYPES.MemberExpression && isExitCodeMemberAccess(idParent, id);
          });
          const escapesViaReturn = variable?.references.some(ref => isReturnedFromFunctionExpression(ref.identifier));
          if (!usesExitCode) {
            // Cross-function return/value forwarding is not resolved by this rule.
            // If the binding is returned from this function, skip reporting here to
            // avoid false positives and let the caller-side check patterns handle it.
            if (escapesViaReturn) return;
            context.report({ node: call, messageId: "missingExitCodeCheck" });
          }
          return;
        }
        // Other binding shapes (array pattern, etc.) are out of scope; don't flag.
        return;
      }

      // let result; result = await getExecOutput(...); if (result.exitCode !== 0) ...
      if (parent.type === AST_NODE_TYPES.AssignmentExpression && parent.right === resultNode && parent.left.type === AST_NODE_TYPES.Identifier) {
        const variable = findInUpperScopes(context.sourceCode.getScope(parent), parent.left.name);
        const usesExitCode = variable?.references.some(ref => {
          const id = ref.identifier;
          const idParent = id.parent;
          return idParent !== undefined && idParent.type === AST_NODE_TYPES.MemberExpression && isExitCodeMemberAccess(idParent, id);
        });
        if (!usesExitCode) {
          context.report({ node: call, messageId: "missingExitCodeCheck" });
        }
        return;
      }

      // Direct member access: (await getExecOutput(...)).exitCode
      if (parent.type === AST_NODE_TYPES.MemberExpression && isExitCodeMemberAccess(parent, resultNode)) {
        return;
      }

      // Cross-function return/value forwarding isn't resolved at this callsite.
      // Skip to avoid false positives for helper/callback-return wrappers.
      if (isReturnedFromFunctionExpression(resultNode)) {
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
