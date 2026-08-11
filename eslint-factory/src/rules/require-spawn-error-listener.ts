import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { isChildProcessImportBinding, isChildProcessObjectBinding, isRequireChildProcess } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

const SPAWN_NAME = "spawn";

function findVariableByName(sourceCode: Readonly<TSESLint.SourceCode>, node: TSESTree.Node, varName: string): TSESLint.Scope.Variable | undefined {
  let scope: ReturnType<typeof sourceCode.getScope> | null = sourceCode.getScope(node);
  while (scope) {
    const variable = scope.set.get(varName);
    if (variable) return variable;
    scope = scope.upper;
  }
  return undefined;
}

function isSpawnBinding(identifierName: string, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode, seenVariables = new Set<TSESLint.Scope.Variable>()): boolean {
  let scope: ReturnType<typeof sourceCode.getScope> | null = sourceCode.getScope(scopeNode);
  while (scope) {
    const variable = scope.set.get(identifierName);
    if (variable && variable.defs.length > 0) {
      if (seenVariables.has(variable)) return false;
      seenVariables.add(variable);

      for (const def of variable.defs) {
        if (isChildProcessImportBinding(def) && def.node.type === AST_NODE_TYPES.ImportSpecifier) {
          const specifier = def.node as TSESTree.ImportSpecifier;
          const importedName = specifier.imported.type === AST_NODE_TYPES.Identifier ? specifier.imported.name : null;
          if (importedName === SPAWN_NAME) return true;
        }

        if (def.type !== "Variable") continue;
        const declarator = def.node as TSESTree.VariableDeclarator;

        if (declarator.id.type === AST_NODE_TYPES.ObjectPattern && isRequireChildProcess(declarator.init)) {
          for (const prop of declarator.id.properties) {
            if (prop.type !== AST_NODE_TYPES.Property) continue;
            if (prop.key.type !== AST_NODE_TYPES.Identifier || prop.key.name !== SPAWN_NAME) continue;
            const boundName = prop.value.type === AST_NODE_TYPES.Identifier ? prop.value.name : null;
            if (boundName === identifierName) return true;
          }
        }

        if (
          declarator.id.type === AST_NODE_TYPES.Identifier &&
          declarator.init?.type === AST_NODE_TYPES.LogicalExpression &&
          (declarator.init.operator === "??" || declarator.init.operator === "||") &&
          declarator.init.right.type === AST_NODE_TYPES.Identifier &&
          isSpawnBinding(declarator.init.right.name, declarator.init.right, sourceCode, seenVariables)
        ) {
          return true;
        }
      }
      return false;
    }
    scope = scope.upper;
  }
  return false;
}

/**
 * Returns true when the expression is a call to the async `spawn` sourced from
 * the `child_process` module. Does not match `spawnSync` or unrelated local
 * functions/imports named `spawn`.
 */
function isSpawnCall(node: TSESTree.Expression, sourceCode: TSESLint.SourceCode): boolean {
  if (node.type !== AST_NODE_TYPES.CallExpression) return false;
  const callee = node.callee;

  if (callee.type === AST_NODE_TYPES.Identifier) {
    return isSpawnBinding(callee.name, callee, sourceCode);
  }

  if (callee.type === AST_NODE_TYPES.MemberExpression && !callee.computed && callee.object.type === AST_NODE_TYPES.Identifier && callee.property.type === AST_NODE_TYPES.Identifier && callee.property.name === SPAWN_NAME) {
    return isChildProcessObjectBinding(callee.object.name, callee.object, sourceCode);
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
        '`child.on("error", ...)` / `child.once("error", ...)` call anywhere in the enclosing function; it does not analyze ' +
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
        if (!isSpawnCall(node.init, sourceCode)) return;
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
