import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { isChildProcessImportBinding, isChildProcessObjectBinding, isRequireChildProcess } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCodeScope = ReturnType<TSESLint.SourceCode["getScope"]>;

/**
 * `child_process` methods that run a command to completion and return/capture its output —
 * the exact use case covered by `@actions/exec`'s `exec()` / `getExecOutput()`. These files run
 * as `actions/github-script` steps (loaded via `require()` and executed with `core`, `github`,
 * `context`, `exec`, `io`, and `getOctokit` already in scope — see
 * `generateGitHubScriptWithRequire` in `pkg/workflow/compiler_github_actions_steps.go`), so the
 * `@actions/exec` toolkit module is always available without an extra dependency.
 *
 * `spawn` / `spawnSync` are intentionally excluded: they cover long-running, detached, or
 * interactively-streamed child processes (background servers, sidecars, and similar) for which
 * `@actions/exec` has no equivalent, since `exec()` / `getExecOutput()` always wait for the
 * command to finish before resolving.
 */
const OUTPUT_CAPTURING_METHODS = new Set(["exec", "execSync", "execFile", "execFileSync"]);

function getImportSpecifierName(node: TSESTree.ImportSpecifier): string | null {
  if (node.imported.type === AST_NODE_TYPES.Identifier) return node.imported.name;
  if (node.imported.type === AST_NODE_TYPES.Literal && typeof node.imported.value === "string") return node.imported.value;
  return null;
}

/**
 * Resolves whether `identifierName` is bound (directly or via destructuring/require) to one of
 * `OUTPUT_CAPTURING_METHODS` from the `child_process` module, and returns the bound method name.
 */
function resolveChildProcessOutputMethodBinding(identifierName: string, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode): string | null {
  let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
  while (scope) {
    const variable = scope.set.get(identifierName);
    if (variable && variable.defs.length > 0) {
      for (const def of variable.defs) {
        // ESM: import { execSync } from "child_process"
        if (isChildProcessImportBinding(def) && def.node.type === AST_NODE_TYPES.ImportSpecifier) {
          const importedName = getImportSpecifierName(def.node);
          if (importedName && OUTPUT_CAPTURING_METHODS.has(importedName)) return importedName;
        }

        if (def.type !== "Variable") continue;
        const declarator = def.node as TSESTree.VariableDeclarator;

        // CJS: const { execSync } = require("child_process")
        if (declarator.id.type === AST_NODE_TYPES.ObjectPattern && isRequireChildProcess(declarator.init)) {
          for (const prop of declarator.id.properties) {
            if (prop.type !== AST_NODE_TYPES.Property || prop.computed) continue;
            if (prop.key.type !== AST_NODE_TYPES.Identifier || !OUTPUT_CAPTURING_METHODS.has(prop.key.name)) continue;
            const boundName = prop.value.type === AST_NODE_TYPES.Identifier ? prop.value.name : null;
            if (boundName === identifierName) return prop.key.name;
          }
        }

        // const execSync = childProcess.execSync (or cp.execSync, or require("child_process").execSync)
        if (declarator.id.type === AST_NODE_TYPES.Identifier && declarator.init?.type === AST_NODE_TYPES.MemberExpression) {
          const init = declarator.init;
          if (!init.computed && init.property.type === AST_NODE_TYPES.Identifier && OUTPUT_CAPTURING_METHODS.has(init.property.name)) {
            const isDirectChildProcessRequire = init.object.type === AST_NODE_TYPES.CallExpression && isRequireChildProcess(init.object);
            const isChildProcessNamespace = init.object.type === AST_NODE_TYPES.Identifier && isChildProcessObjectBinding(init.object.name, init.object, sourceCode);
            if (isDirectChildProcessRequire || isChildProcessNamespace) return init.property.name;
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
 * Returns the resolved `child_process` output-capturing method name (e.g. `"execSync"`) for a
 * `CallExpression`, or `null` if the call isn't one of `OUTPUT_CAPTURING_METHODS` sourced from
 * `child_process`.
 */
function resolveChildProcessOutputMethod(node: TSESTree.CallExpression, sourceCode: TSESLint.SourceCode): string | null {
  const callee = node.callee;

  // execSync(...) / exec(...) / execFile(...) / execFileSync(...) — destructured or aliased
  if (callee.type === AST_NODE_TYPES.Identifier) {
    return resolveChildProcessOutputMethodBinding(callee.name, callee, sourceCode);
  }

  if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed || callee.property.type !== AST_NODE_TYPES.Identifier) return null;
  if (!OUTPUT_CAPTURING_METHODS.has(callee.property.name)) return null;

  // require("child_process").execSync(...)
  if (callee.object.type === AST_NODE_TYPES.CallExpression && isRequireChildProcess(callee.object)) {
    return callee.property.name;
  }

  // childProcess.execSync(...) / cp.execSync(...)
  if (callee.object.type === AST_NODE_TYPES.Identifier && isChildProcessObjectBinding(callee.object.name, callee.object, sourceCode)) {
    return callee.property.name;
  }

  return null;
}

export const preferActionsExecOverChildProcessRule = createRule({
  name: "prefer-actions-exec-over-child-process",
  meta: {
    type: "suggestion",
    docs: {
      description:
        "Prefer @actions/exec's exec()/getExecOutput() over child_process's exec()/execSync()/execFile()/execFileSync() to spawn processes. " +
        "actions/setup/js scripts run as actions/github-script steps with the @actions/exec toolkit already available as `exec` (and `execApi`/`execImpl` " +
        "in some scripts), so blocking calls that run a command to completion and capture its output should go through it instead of child_process. " +
        "spawn()/spawnSync() are not flagged: they're used for long-running, detached, or interactively-streamed processes that @actions/exec has no equivalent for.",
    },
    schema: [],
    messages: {
      preferActionsExec:
        "Prefer @actions/exec's exec()/getExecOutput() over child_process.{{method}}() to spawn processes in actions/github-script scripts. child_process.{{method}}() duplicates functionality already provided by the @actions/exec toolkit available in this context.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node) {
        const method = resolveChildProcessOutputMethod(node, sourceCode);
        if (!method) return;

        context.report({
          node,
          messageId: "preferActionsExec",
          data: { method },
        });
      },
    };
  },
});
