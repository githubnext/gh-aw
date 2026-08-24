import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { isChildProcessImportBinding, isChildProcessObjectBinding, isRequireChildProcess } from "./try-catch-rule-utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCodeScope = ReturnType<TSESLint.SourceCode["getScope"]>;

/**
 * `child_process` methods that run a command to completion and return/capture its output —
 * the exact use case covered by `@actions/exec`'s `exec()` / `getExecOutput()`.
 *
 * `spawn` / `spawnSync` are intentionally excluded: they cover long-running, detached, or
 * interactively-streamed child processes (background servers, sidecars, and similar) for which
 * `@actions/exec` has no equivalent, since `exec()` / `getExecOutput()` always wait for the
 * command to finish before resolving.
 */
const OUTPUT_CAPTURING_METHODS = new Set(["exec", "execSync", "execFile", "execFileSync"]);

/**
 * Asynchronous `child_process` methods that return a `ChildProcess` handle. When the handle is
 * retained (assigned, returned, or member-accessed) the caller can stream stdin/stdout or manage
 * the process lifecycle — capabilities `@actions/exec` does not expose — so those calls are not
 * flagged. Only calls whose result is discarded (pure callback style) are reported.
 */
const HANDLE_RETURNING_METHODS = new Set(["exec", "execFile"]);

/**
 * Marker that identifies modules executed as `actions/github-script` steps (loaded via `require()`
 * and executed with `core`, `github`, `context`, `exec`, `io`, and `getOctokit` already in scope —
 * see `generateGitHubScriptWithRequire` in `pkg/workflow/compiler_github_actions_steps.go`). Only
 * those modules are guaranteed to have the `@actions/exec` toolkit available as the `exec` global;
 * standalone Node entry points (such as the mcp-scripts MCP server and the modules it loads) do
 * not, so this rule stays silent in files without the marker.
 */
const GITHUB_SCRIPT_REFERENCE_PATTERN = /<reference\s+types=["']@actions\/github-script["']\s*\/>/;

function isGitHubScriptModule(sourceCode: TSESLint.SourceCode): boolean {
  return sourceCode.getAllComments().some(comment => GITHUB_SCRIPT_REFERENCE_PATTERN.test(comment.value));
}

/** True when the result of `node` is used for anything beyond being discarded as a statement. */
function retainsCallResult(node: TSESTree.CallExpression): boolean {
  return node.parent != null && node.parent.type !== AST_NODE_TYPES.ExpressionStatement;
}

function getImportSpecifierName(node: TSESTree.ImportSpecifier): string | null {
  if (node.imported.type === AST_NODE_TYPES.Identifier) return node.imported.name;
  if (node.imported.type === AST_NODE_TYPES.Literal && typeof node.imported.value === "string") return node.imported.value;
  return null;
}

/**
 * True when `identifierName` refers to the whole `child_process` module: a `require("child_process")`
 * binding, an ESM namespace import (`import * as cp from "child_process"`), or an ESM default import
 * (`import childProcess from "child_process"`).
 */
function isChildProcessModuleBinding(identifierName: string, scopeNode: TSESTree.Node, sourceCode: TSESLint.SourceCode): boolean {
  if (isChildProcessObjectBinding(identifierName, scopeNode, sourceCode)) return true;

  let scope: SourceCodeScope | null = sourceCode.getScope(scopeNode);
  while (scope) {
    const variable = scope.set.get(identifierName);
    if (variable && variable.defs.length > 0) {
      return variable.defs.some(def => isChildProcessImportBinding(def) && def.node.type === AST_NODE_TYPES.ImportDefaultSpecifier);
    }
    scope = scope.upper;
  }
  return false;
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
            const isChildProcessNamespace = init.object.type === AST_NODE_TYPES.Identifier && isChildProcessModuleBinding(init.object.name, init.object, sourceCode);
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
  if (callee.object.type === AST_NODE_TYPES.Identifier && isChildProcessModuleBinding(callee.object.name, callee.object, sourceCode)) {
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
        'Only applies to modules marked with the `/// <reference types="@actions/github-script" />` triple-slash reference, which run as ' +
        "actions/github-script steps with the @actions/exec toolkit already available as `exec`; standalone Node entry points (and the modules they load) " +
        "have no such global and are left alone. spawn()/spawnSync() are never flagged, and exec()/execFile() calls whose returned ChildProcess handle is " +
        "retained (for stdin/stdout streaming or lifecycle management) are exempt, since @actions/exec has no equivalent for those.",
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
    if (!isGitHubScriptModule(sourceCode)) return {};

    return {
      CallExpression(node) {
        const method = resolveChildProcessOutputMethod(node, sourceCode);
        if (!method) return;
        if (HANDLE_RETURNING_METHODS.has(method) && retainsCallResult(node)) return;

        context.report({
          node,
          messageId: "preferActionsExec",
          data: { method },
        });
      },
    };
  },
});
