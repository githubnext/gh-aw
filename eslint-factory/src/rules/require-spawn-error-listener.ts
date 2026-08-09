import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Unqualified function name used when spawn is destructured from child_process.
const SPAWN_NAME = "spawn";

// Known namespace aliases for the child_process module.
const CHILD_PROCESS_OBJECTS = new Set(["childProcess", "child_process"]);

type ScopeType = ReturnType<TSESLint.SourceCode["getScope"]>;
type ScopeVariable = ScopeType["variables"][number];

function findVariableByName(sourceCode: Readonly<TSESLint.SourceCode>, node: TSESTree.Node, varName: string): ScopeVariable | undefined {
  let scope: ReturnType<typeof sourceCode.getScope> | null = sourceCode.getScope(node);
  while (scope) {
    const variable = scope.set.get(varName);
    if (variable) return variable;
    scope = scope.upper;
  }
  return undefined;
}

/**
 * Returns true when the expression is a call to the async `spawn` (either bare
 * or namespaced). Does not match `spawnSync`/`spawnImpl`-style aliases.
 * Matched forms:
 *   spawn(cmd, args, opts)
 *   childProcess.spawn(cmd, args, opts)
 *   child_process.spawn(cmd, args, opts)
 */
function isSpawnCall(node: TSESTree.Expression): boolean {
  if (node.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = node.callee;

  if (callee.type === AST_NODE_TYPES.Identifier && callee.name === SPAWN_NAME) {
    return true;
  }

  if (
    callee.type === AST_NODE_TYPES.MemberExpression &&
    !callee.computed &&
    callee.object.type === AST_NODE_TYPES.Identifier &&
    CHILD_PROCESS_OBJECTS.has(callee.object.name) &&
    callee.property.type === AST_NODE_TYPES.Identifier &&
    callee.property.name === SPAWN_NAME
  ) {
    return true;
  }

  return false;
}

/** Returns true when `call` is `<name>.on("error", ...)` / `.once("error", ...)`. */
function isErrorListenerCall(call: TSESTree.CallExpression, name: string): boolean {
  const callee = call.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return false;
  if (callee.object.type !== AST_NODE_TYPES.Identifier || callee.object.name !== name) return false;
  if (callee.property.type !== AST_NODE_TYPES.Identifier || (callee.property.name !== "on" && callee.property.name !== "once")) return false;

  const firstArg = call.arguments[0];
  return firstArg !== undefined && firstArg.type === AST_NODE_TYPES.Literal && firstArg.value === "error";
}

export const requireSpawnErrorListenerRule = createRule({
  name: "require-spawn-error-listener",
  meta: {
    type: "problem",
    docs: {
      description:
        "Require an 'error' event listener on child processes created with the async spawn() in actions/setup/js scripts. " +
        "Unlike spawnSync, spawn() is asynchronous: when the executable cannot be launched (e.g. ENOENT, EACCES), " +
        "Node emits an 'error' event on the returned ChildProcess instead of throwing synchronously or rejecting a promise. " +
        "With no 'error' listener registered, that event becomes an uncaught exception that crashes the process. " +
        "Scope: this rule only checks variable declarator initializers (`const child = spawn(...)`) and looks for a " +
        "`child.on(\"error\", ...)` / `child.once(\"error\", ...)` call anywhere in the enclosing function; it does not analyze " +
        "assignment expressions (`child = spawn(...)`) or inline chains (`spawn(...).on(...)`).",
    },
    schema: [],
    messages: {
      missingErrorListener:
        "spawn() result must have an 'error' event listener attached (e.g. `child.on(\"error\", err => { ... })`). " +
        "Without it, a failure to launch the process (ENOENT, EACCES, etc.) is an unhandled 'error' event that crashes the action.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      VariableDeclarator(node: TSESTree.VariableDeclarator) {
        if (!node.init) return;
        if (!isSpawnCall(node.init)) return;
        if (node.id.type !== AST_NODE_TYPES.Identifier) return;

        const varName = node.id.name;
        const variable = findVariableByName(sourceCode, node, varName);
        if (!variable) return;

        const hasErrorListener = variable.references.some(ref => {
          const id = ref.identifier;
          const parent = id.parent;
          if (!parent || parent.type !== AST_NODE_TYPES.MemberExpression || parent.object !== id) return false;
          const grandparent = parent.parent;
          return grandparent !== undefined && grandparent.type === AST_NODE_TYPES.CallExpression && grandparent.callee === parent && isErrorListenerCall(grandparent, varName);
        });

        if (!hasErrorListener) {
          context.report({ node: node.init, messageId: "missingErrorListener" });
        }
      },
    };
  },
});
