import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Unqualified function name used when spawnSync is destructured from child_process.
const SPAWNSYNC_NAME = "spawnSync";

// Known namespace aliases for the child_process module.
const CHILD_PROCESS_OBJECTS = new Set(["childProcess", "child_process"]);

/**
 * Returns true when the expression is a call to spawnSync (either bare or namespaced).
 * Matched forms:
 *   spawnSync(cmd, args, opts)
 *   childProcess.spawnSync(cmd, args, opts)
 *   child_process.spawnSync(cmd, args, opts)
 */
function isSpawnSyncCall(node: TSESTree.Expression): boolean {
  if (node.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = node.callee;

  if (callee.type === AST_NODE_TYPES.Identifier && callee.name === SPAWNSYNC_NAME) {
    return true;
  }

  if (
    callee.type === AST_NODE_TYPES.MemberExpression &&
    !callee.computed &&
    callee.object.type === AST_NODE_TYPES.Identifier &&
    CHILD_PROCESS_OBJECTS.has(callee.object.name) &&
    callee.property.type === AST_NODE_TYPES.Identifier &&
    callee.property.name === SPAWNSYNC_NAME
  ) {
    return true;
  }

  return false;
}

export const requireSpawnSyncErrorCheckRule = createRule({
  name: "require-spawnsync-error-check",
  meta: {
    type: "problem",
    docs: {
      description:
        "Require spawnSync result variables in actions/setup/js scripts to check result.error in addition to result.status. " +
        "When spawnSync cannot spawn the child process (e.g. ENOENT, ETIMEDOUT), result.status is null and result.error holds the actual Error — " +
        "checking only result.status silently swallows spawn-level failures or reports a misleading 'exit null' message.",
    },
    schema: [],
    messages: {
      missingErrorCheck:
        "spawnSync result must have its .error property checked. " +
        "When the child process cannot be spawned (e.g. ENOENT, ETIMEDOUT), result.status is null and result.error contains the cause — " +
        "checking only result.status produces a misleading diagnostic. Add: if (result.error) { throw result.error; }",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      VariableDeclarator(node: TSESTree.VariableDeclarator) {
        if (!node.init) return;
        if (!isSpawnSyncCall(node.init)) return;

        // Only handle simple identifier bindings: const result = spawnSync(...)
        if (node.id.type !== AST_NODE_TYPES.Identifier) return;

        const varName = node.id.name;

        // Locate the variable in scope (may be in an outer scope).
        let scope: ReturnType<typeof sourceCode.getScope> | null = sourceCode.getScope(node);
        let variable: (typeof scope)["variables"][number] | undefined;
        while (scope) {
          variable = scope.set.get(varName);
          if (variable) break;
          scope = scope.upper;
        }

        if (!variable) return;

        // Check whether any reference to this variable reads the .error property.
        const hasErrorCheck = variable.references.some(ref => {
          const id = ref.identifier;
          const parent = id.parent;
          return parent !== undefined && parent.type === AST_NODE_TYPES.MemberExpression && !parent.computed && parent.object === id && parent.property.type === AST_NODE_TYPES.Identifier && parent.property.name === "error";
        });

        if (!hasErrorCheck) {
          context.report({ node: node.init, messageId: "missingErrorCheck" });
        }
      },
    };
  },
});
