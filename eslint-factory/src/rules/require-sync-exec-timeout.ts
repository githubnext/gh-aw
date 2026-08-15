import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { isChildProcessImportBinding, isChildProcessObjectBinding, isRequireChildProcess } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCodeScope = ReturnType<TSESLint.SourceCode["getScope"]>;

// child_process synchronous APIs that block the event loop until the child exits.
// Without an explicit `timeout`, a hung or runaway child process (e.g. a `git`
// command against an unreachable remote, or a misbehaving CLI tool) blocks
// indefinitely and can only be killed by the surrounding CI job's own hard
// timeout — often minutes later, with no actionable diagnostic.
type SyncExecMethod = "execSync" | "execFileSync" | "spawnSync";
const SYNC_EXEC_METHODS: ReadonlySet<SyncExecMethod> = new Set(["execSync", "execFileSync", "spawnSync"]);
// Index of the options-object argument for each method when all parameters are supplied:
// execSync(cmd, opts), execFileSync(cmd, args?, opts), spawnSync(cmd, args?, opts).
const OPTIONS_ARG_INDEX: Record<SyncExecMethod, number> = { execSync: 1, execFileSync: 2, spawnSync: 2 };

function getOptionsArgument(node: TSESTree.CallExpression, method: SyncExecMethod): TSESTree.CallExpressionArgument | undefined {
  if (method === "execSync") return node.arguments[OPTIONS_ARG_INDEX.execSync];

  // execFileSync/spawnSync overload:
  //   method(cmd, args, options)
  //   method(cmd, options)
  const thirdArg = node.arguments[2];
  if (thirdArg) return thirdArg;

  const secondArg = node.arguments[1];
  if (!secondArg) return undefined;
  if (secondArg.type === AST_NODE_TYPES.ArrayExpression) return undefined;
  return secondArg;
}

/**
 * Walks the scope chain to decide whether `identifierName` resolves to one of the
 * SYNC_EXEC_METHODS imported/required from `child_process`.
 */
function resolveSyncExecBinding(identifierName: string, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode): SyncExecMethod | null {
  let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
  while (scope) {
    const variable = scope.set.get(identifierName);
    if (variable && variable.defs.length > 0) {
      for (const def of variable.defs) {
        // ESM: import { execSync } from "child_process"
        if (isChildProcessImportBinding(def) && def.node.type === AST_NODE_TYPES.ImportSpecifier) {
          const specifier = def.node as TSESTree.ImportSpecifier;
          const importedName = specifier.imported.type === AST_NODE_TYPES.Identifier ? specifier.imported.name : null;
          if (importedName && SYNC_EXEC_METHODS.has(importedName as SyncExecMethod)) return importedName as SyncExecMethod;
        }
        // CJS: const { execSync } = require("child_process")
        if (def.type === "Variable") {
          const declarator = def.node as TSESTree.VariableDeclarator;
          if (declarator.id.type === AST_NODE_TYPES.ObjectPattern && isRequireChildProcess(declarator.init)) {
            for (const prop of declarator.id.properties) {
              if (prop.type !== AST_NODE_TYPES.Property) continue;
              if (prop.key.type !== AST_NODE_TYPES.Identifier) continue;
              if (!SYNC_EXEC_METHODS.has(prop.key.name as SyncExecMethod)) continue;
              const boundName = prop.value.type === AST_NODE_TYPES.Identifier ? prop.value.name : null;
              if (boundName === identifierName) return prop.key.name as SyncExecMethod;
            }
          }
          // const execSync = childProcess.execSync
          if (declarator.id.type === AST_NODE_TYPES.Identifier && declarator.init?.type === AST_NODE_TYPES.MemberExpression) {
            const init = declarator.init;
            if (
              !init.computed &&
              init.object.type === AST_NODE_TYPES.Identifier &&
              isChildProcessObjectBinding(init.object.name, init.object, sourceCode) &&
              init.property.type === AST_NODE_TYPES.Identifier &&
              SYNC_EXEC_METHODS.has(init.property.name as SyncExecMethod)
            ) {
              return init.property.name as SyncExecMethod;
            }
          }
        }
      }
      return null;
    }
    scope = scope.upper;
  }
  return null;
}

/**
 * Returns the resolved sync-exec method name for a CallExpression, or null if it
 * doesn't resolve to one of execSync/execFileSync/spawnSync from `child_process`.
 */
function resolveSyncExecMethod(node: TSESTree.CallExpression, sourceCode: TSESLint.SourceCode): SyncExecMethod | null {
  const callee = node.callee;

  // execSync(...) / execFileSync(...) / spawnSync(...) — destructured or aliased
  if (callee.type === AST_NODE_TYPES.Identifier) {
    return resolveSyncExecBinding(callee.name, callee, sourceCode);
  }

  // childProcess.execSync(...) / cp.spawnSync(...) etc.
  if (
    callee.type === AST_NODE_TYPES.MemberExpression &&
    !callee.computed &&
    callee.object.type === AST_NODE_TYPES.Identifier &&
    callee.property.type === AST_NODE_TYPES.Identifier &&
    SYNC_EXEC_METHODS.has(callee.property.name as SyncExecMethod)
  ) {
    if (isChildProcessObjectBinding(callee.object.name, callee.object, sourceCode)) {
      return callee.property.name as SyncExecMethod;
    }
  }

  return null;
}

/** Returns true when the options-object argument for the call statically carries a positive `timeout` property. */
function hasTimeoutOption(node: TSESTree.CallExpression, method: SyncExecMethod): boolean {
  const optionsArg = getOptionsArgument(node, method);
  if (!optionsArg) return false;

  // Spread arguments or non-object expressions (identifiers, shared config objects,
  // conditional expressions) can't be statically inspected; assume the caller may
  // have already included a timeout to avoid false positives.
  if (optionsArg.type !== AST_NODE_TYPES.ObjectExpression) return true;

  for (const prop of optionsArg.properties) {
    if (prop.type === AST_NODE_TYPES.SpreadElement) return true;
    if (prop.type !== AST_NODE_TYPES.Property) continue;

    const isTimeoutProp = (!prop.computed && prop.key.type === AST_NODE_TYPES.Identifier && prop.key.name === "timeout") || (!prop.computed && prop.key.type === AST_NODE_TYPES.Literal && prop.key.value === "timeout");
    if (!isTimeoutProp) continue;

    const value = prop.value;
    const isMissingTimeout =
      (value.type === AST_NODE_TYPES.Literal && (value.value == null || (typeof value.value === "number" && value.value <= 0))) ||
      (value.type === AST_NODE_TYPES.UnaryExpression && value.operator === "-" && value.argument.type === AST_NODE_TYPES.Literal && typeof value.argument.value === "number") ||
      (value.type === AST_NODE_TYPES.Identifier && value.name === "undefined");
    if (!isMissingTimeout) return true;
  }

  return false;
}

export const requireSyncExecTimeoutRule = createRule({
  name: "require-sync-exec-timeout",
  meta: {
    type: "problem",
    docs: {
      description:
        "Require execSync, execFileSync, and spawnSync calls in actions/setup/js scripts to pass a `timeout` option. " +
        "These synchronous child_process APIs block the Node.js event loop until the child process exits; " +
        "without an explicit timeout, a hung or runaway child process (network stall, interactive prompt, infinite loop) blocks indefinitely " +
        "with no actionable diagnostic until the surrounding CI job's own hard timeout eventually kills the whole run.",
    },
    schema: [],
    messages: {
      requireTimeout:
        "{{method}}({{arg}}) has no positive `timeout` option. `timeout: 0` disables the timeout; pass `{ timeout: <positive milliseconds>, ...otherOptions }` so a hung or runaway child process cannot block the job indefinitely.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node) {
        const method = resolveSyncExecMethod(node, sourceCode);
        if (!method) return;
        if (hasTimeoutOption(node, method)) return;

        const argText = node.arguments.length > 0 ? sourceCode.getText(node.arguments[0]) : "";

        context.report({
          node,
          messageId: "requireTimeout",
          data: { method, arg: argText },
        });
      },
    };
  },
});
