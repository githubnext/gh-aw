import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Unqualified function name used when spawnSync is destructured from child_process.
const SPAWNSYNC_NAME = "spawnSync";

// Known namespace aliases for the child_process module.
const CHILD_PROCESS_OBJECTS = new Set(["childProcess", "child_process"]);
const CONDITIONAL_TEST_PARENTS = new Set([AST_NODE_TYPES.IfStatement, AST_NODE_TYPES.WhileStatement, AST_NODE_TYPES.DoWhileStatement, AST_NODE_TYPES.ForStatement]);

function isConditionalTestParent(node: TSESTree.Node): node is TSESTree.IfStatement | TSESTree.WhileStatement | TSESTree.DoWhileStatement | TSESTree.ForStatement {
  return CONDITIONAL_TEST_PARENTS.has(node.type);
}

type ScopeType = ReturnType<TSESLint.SourceCode["getScope"]>;
type ScopeVariable = ScopeType["variables"][number];

function isGuardingErrorUsage(node: TSESTree.Expression): boolean {
  let current: TSESTree.Node = node;

  while (current.parent) {
    const parent: TSESTree.Node = current.parent;

    if (isConditionalTestParent(parent) && parent.test === current) {
      return true;
    }

    if (parent.type === AST_NODE_TYPES.ConditionalExpression && parent.test === current) {
      return true;
    }

    if (parent.type === AST_NODE_TYPES.LogicalExpression) {
      // Guarding intent depends on how the full logical expression is used:
      // climb through the left operand of either operator and the right operand of `||`,
      // but reject the right operand of `&&` because it executes conditionally and
      // does not establish an independent guard.
      if (parent.left === current || (parent.operator === "||" && parent.right === current)) {
        current = parent;
        continue;
      }

      break;
    }

    if ((parent.type === AST_NODE_TYPES.ThrowStatement || parent.type === AST_NODE_TYPES.ReturnStatement) && parent.argument === current) {
      return true;
    }

    if (parent.type === AST_NODE_TYPES.UnaryExpression && parent.operator === "!" && parent.argument === current) {
      current = parent;
      continue;
    }

    if (parent.type === AST_NODE_TYPES.BinaryExpression && (parent.left === current || parent.right === current)) {
      current = parent;
      continue;
    }

    break;
  }

  return false;
}

function findVariableByName(sourceCode: Readonly<TSESLint.SourceCode>, node: TSESTree.Node, varName: string): ScopeVariable | undefined {
  let scope: ReturnType<typeof sourceCode.getScope> | null = sourceCode.getScope(node);
  while (scope) {
    const variable = scope.set.get(varName);
    if (variable) return variable;
    scope = scope.upper;
  }
  return undefined;
}

function isErrorKey(node: TSESTree.PropertyName): boolean {
  return (node.type === AST_NODE_TYPES.Identifier && node.name === "error") || (node.type === AST_NODE_TYPES.Literal && node.value === "error");
}

function getErrorBindingNames(node: TSESTree.ObjectPattern): string[] {
  const names: string[] = [];

  for (const property of node.properties) {
    // Intentionally only support static `error` keys (`{ error }` / `{ error: err }`).
    // Computed keys are ignored.
    if (property.type !== AST_NODE_TYPES.Property || property.computed) {
      continue;
    }

    if (!isErrorKey(property.key)) {
      continue;
    }

    const value = property.value;
    if (value.type === AST_NODE_TYPES.Identifier) {
      names.push(value.name);
    } else if (value.type === AST_NODE_TYPES.AssignmentPattern && value.left.type === AST_NODE_TYPES.Identifier) {
      // Support `{ error = defaultValue }` by tracking the bound identifier name.
      names.push(value.left.name);
    }
  }

  return names;
}

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
        "checking only result.status silently swallows spawn-level failures or reports a misleading 'exit null' message. " +
        "Scope: this rule checks variable declarator initializers (including object destructuring) and does not analyze AssignmentExpression forms (`result = spawnSync(...)`) or inline chains (`spawnSync(...).status`).",
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

        if (node.id.type === AST_NODE_TYPES.ObjectPattern) {
          const errorBindingNames = getErrorBindingNames(node.id);
          if (errorBindingNames.length === 0) {
            context.report({ node: node.init, messageId: "missingErrorCheck" });
            return;
          }

          const hasGuardingErrorCheck = errorBindingNames.some(varName => {
            const variable = findVariableByName(sourceCode, node, varName);
            if (!variable) return false;
            return variable.references.some(ref => ref.identifier.type === AST_NODE_TYPES.Identifier && isGuardingErrorUsage(ref.identifier));
          });

          if (!hasGuardingErrorCheck) {
            context.report({ node: node.init, messageId: "missingErrorCheck" });
          }
          return;
        }

        // Only handle simple identifier bindings: const result = spawnSync(...)
        if (node.id.type !== AST_NODE_TYPES.Identifier) return;

        const variable = findVariableByName(sourceCode, node, node.id.name);
        if (!variable) return;

        // Check whether any reference to this variable uses .error in a guarding position.
        const hasGuardingErrorCheck = variable.references.some(ref => {
          const id = ref.identifier;
          const parent = id.parent;
          return (
            parent !== undefined &&
            parent.type === AST_NODE_TYPES.MemberExpression &&
            !parent.computed &&
            parent.object === id &&
            parent.property.type === AST_NODE_TYPES.Identifier &&
            parent.property.name === "error" &&
            isGuardingErrorUsage(parent)
          );
        });

        if (!hasGuardingErrorCheck) {
          context.report({ node: node.init, messageId: "missingErrorCheck" });
        }
      },
    };
  },
});
