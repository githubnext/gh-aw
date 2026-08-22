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
type FunctionWithParams = TSESTree.ArrowFunctionExpression | TSESTree.FunctionDeclaration | TSESTree.FunctionExpression;
type NodeWithParent = TSESTree.Node & { parent?: TSESTree.Node };

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

function resolveIdentifierVariable(identifierName: string, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode): TSESLint.Scope.Variable | null {
  let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
  while (scope) {
    const variable = scope.set.get(identifierName);
    if (variable) return variable;
    scope = scope.upper;
  }
  return null;
}

function getParent(node: TSESTree.Node): TSESTree.Node | undefined {
  return (node as NodeWithParent).parent;
}

function isFunctionWithParams(node: TSESTree.Node | undefined): node is FunctionWithParams {
  return node?.type === AST_NODE_TYPES.ArrowFunctionExpression || node?.type === AST_NODE_TYPES.FunctionDeclaration || node?.type === AST_NODE_TYPES.FunctionExpression;
}

function getContainingFunction(node: TSESTree.Node): FunctionWithParams | null {
  let current: TSESTree.Node | undefined = getParent(node);
  while (current) {
    if (isFunctionWithParams(current)) return current;
    current = getParent(current);
  }
  return null;
}

function isPositiveTimeoutProperty(prop: TSESTree.Property): boolean {
  const isTimeoutProp = (!prop.computed && prop.key.type === AST_NODE_TYPES.Identifier && prop.key.name === "timeout") || (!prop.computed && prop.key.type === AST_NODE_TYPES.Literal && prop.key.value === "timeout");
  if (!isTimeoutProp) return false;

  const value = prop.value;
  const isMissingTimeout =
    (value.type === AST_NODE_TYPES.Literal && (value.value == null || (typeof value.value === "number" && value.value <= 0))) ||
    (value.type === AST_NODE_TYPES.UnaryExpression && value.operator === "-" && value.argument.type === AST_NODE_TYPES.Literal && typeof value.argument.value === "number") ||
    (value.type === AST_NODE_TYPES.Identifier && value.name === "undefined");
  return !isMissingTimeout;
}

function hasPositiveTimeoutProperty(objectExpression: TSESTree.ObjectExpression): boolean {
  return objectExpression.properties.some(prop => prop.type === AST_NODE_TYPES.Property && isPositiveTimeoutProperty(prop));
}

function objectArgumentSuppliesTimeout(objectExpression: TSESTree.ObjectExpression): boolean {
  if (hasPositiveTimeoutProperty(objectExpression)) return true;

  // Call-site spread sources can be shared or externally-computed options. Keep
  // the rule conservative unless the local object is plainly missing timeout.
  return objectExpression.properties.some(prop => prop.type === AST_NODE_TYPES.SpreadElement);
}

function isReassignedBefore(variable: TSESLint.Scope.Variable, node: TSESTree.Node): boolean {
  const nodeStart = node.range?.[0];
  if (nodeStart == null) return variable.references.some(ref => ref.isWrite() && !ref.init);

  return variable.references.some(ref => {
    if (!ref.isWrite() || ref.init) return false;
    const refStart = ref.identifier.range?.[0];
    return refStart == null || refStart < nodeStart;
  });
}

function resolveStaticObjectInitializer(identifier: TSESTree.Identifier, sourceCode: TSESLint.SourceCode): { init: TSESTree.ObjectExpression; variable: TSESLint.Scope.Variable } | null {
  const variable = resolveIdentifierVariable(identifier.name, identifier, sourceCode);
  if (!variable || variable.defs.length !== 1) return null;

  const def = variable.defs[0];
  if (def.type !== "Variable") return null;

  const declarator = def.node as TSESTree.VariableDeclarator;
  if (declarator.init?.type !== AST_NODE_TYPES.ObjectExpression) return null;
  if (isReassignedBefore(variable, identifier)) return null;

  return { init: declarator.init, variable };
}

function callSiteArgumentSuppliesTimeout(argument: TSESTree.CallExpressionArgument | undefined, sourceCode: TSESLint.SourceCode): boolean {
  if (!argument) return false;
  if (argument.type === AST_NODE_TYPES.ObjectExpression) return objectArgumentSuppliesTimeout(argument);

  if (argument.type === AST_NODE_TYPES.Identifier) {
    const staticObject = resolveStaticObjectInitializer(argument, sourceCode);
    if (!staticObject) return true;
    return objectArgumentSuppliesTimeout(staticObject.init);
  }

  // Non-object expressions are not statically inspectable; preserve the prior
  // conservative behavior.
  return true;
}

function getParamDefaultObject(functionNode: FunctionWithParams, paramName: string): { objectExpression: TSESTree.ObjectExpression; paramIndex: number } | null {
  for (let index = 0; index < functionNode.params.length; index += 1) {
    const param = functionNode.params[index];
    if (param.type !== AST_NODE_TYPES.AssignmentPattern) continue;
    if (param.left.type !== AST_NODE_TYPES.Identifier || param.left.name !== paramName) continue;
    if (param.right.type !== AST_NODE_TYPES.ObjectExpression) continue;
    return { objectExpression: param.right, paramIndex: index };
  }

  return null;
}

function getFunctionBindingVariable(functionNode: FunctionWithParams, sourceCode: TSESLint.SourceCode): TSESLint.Scope.Variable | null {
  if (functionNode.type === AST_NODE_TYPES.FunctionDeclaration && functionNode.id) {
    return resolveIdentifierVariable(functionNode.id.name, functionNode.id, sourceCode);
  }

  let current: TSESTree.Node = functionNode;
  let parent = getParent(current);
  while (parent) {
    if (parent.type === AST_NODE_TYPES.VariableDeclarator && parent.id.type === AST_NODE_TYPES.Identifier) {
      return resolveIdentifierVariable(parent.id.name, parent.id, sourceCode);
    }
    if (parent.type === AST_NODE_TYPES.AssignmentExpression && parent.left.type === AST_NODE_TYPES.Identifier) {
      return resolveIdentifierVariable(parent.left.name, parent.left, sourceCode);
    }
    current = parent;
    parent = getParent(current);
  }

  return null;
}

function getDirectCalls(binding: TSESLint.Scope.Variable): TSESTree.CallExpression[] {
  const calls: TSESTree.CallExpression[] = [];
  for (const reference of binding.references) {
    if (reference.isWrite()) continue;
    const identifier = reference.identifier;
    const parent = getParent(identifier);
    if (parent?.type === AST_NODE_TYPES.CallExpression && parent.callee === identifier) {
      calls.push(parent);
    }
  }
  return calls;
}

function characterizedParameterSpreadSuppliesTimeout(spread: TSESTree.SpreadElement, sourceCode: TSESLint.SourceCode): boolean | null {
  if (spread.argument.type !== AST_NODE_TYPES.Identifier) return null;

  const parameterVariable = resolveIdentifierVariable(spread.argument.name, spread.argument, sourceCode);
  if (!parameterVariable || !parameterVariable.defs.some(def => def.type === "Parameter")) return null;

  const containingFunction = getContainingFunction(spread);
  if (!containingFunction) return null;

  const defaultObject = getParamDefaultObject(containingFunction, spread.argument.name);
  if (!defaultObject) return null;
  if (hasPositiveTimeoutProperty(defaultObject.objectExpression)) return true;
  if (isReassignedBefore(parameterVariable, spread.argument)) return null;

  const binding = getFunctionBindingVariable(containingFunction, sourceCode);
  if (!binding) return null;

  const directCalls = getDirectCalls(binding);
  if (directCalls.length === 0) return null;

  return directCalls.every(call => callSiteArgumentSuppliesTimeout(call.arguments[defaultObject.paramIndex], sourceCode));
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
function hasTimeoutOption(node: TSESTree.CallExpression, method: SyncExecMethod, sourceCode: TSESLint.SourceCode): boolean {
  const optionsArg = getOptionsArgument(node, method);
  if (!optionsArg) return false;

  // Spread arguments or non-object expressions (identifiers, shared config objects,
  // conditional expressions) can't be statically inspected; assume the caller may
  // have already included a timeout to avoid false positives.
  if (optionsArg.type !== AST_NODE_TYPES.ObjectExpression) return true;

  for (const prop of optionsArg.properties) {
    if (prop.type === AST_NODE_TYPES.SpreadElement) {
      const spreadSuppliesTimeout = characterizedParameterSpreadSuppliesTimeout(prop, sourceCode);
      if (spreadSuppliesTimeout !== false) return true;
      continue;
    }
    if (prop.type !== AST_NODE_TYPES.Property) continue;

    if (isPositiveTimeoutProperty(prop)) return true;
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
        if (hasTimeoutOption(node, method, sourceCode)) return;

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
