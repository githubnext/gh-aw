import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";
import { CORE_ALIASES } from "./core-aliases";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Maps console method → recommended core replacement
const CONSOLE_TO_CORE: Record<string, string> = {
  log: "core.info",
  info: "core.info",
  warn: "core.warning",
  error: "core.error",
  debug: "core.debug",
};

/**
 * Returns the `console` method name if the call expression is a `console.*`
 * call that has a known core replacement, otherwise null.
 */
function getConsoleMethod(node: TSESTree.CallExpression): string | null {
  const callee = node.callee;
  if (callee.type !== AST_NODE_TYPES.MemberExpression) return null;
  if (callee.computed) return null;
  const obj = callee.object;
  const prop = callee.property;
  if (obj.type !== AST_NODE_TYPES.Identifier || obj.name !== "console") return null;
  if (prop.type !== AST_NODE_TYPES.Identifier) return null;
  return prop.name in CONSOLE_TO_CORE ? prop.name : null;
}

/**
 * Walks the scope chain to determine whether any binding for a known
 * @actions/core alias (`core`, `coreObj`) is visible at the given node.
 *
 * Accepted patterns:
 *   - `const core = require("@actions/core")`
 *   - `import * as core from "@actions/core"`
 *   - A function parameter named `core` or `coreObj` (used in github-script style)
 */
function hasCoreInScope(node: TSESTree.Node, sourceCode: TSESLint.SourceCode): boolean {
  let scope: TSESLint.Scope.Scope | null = sourceCode.getScope(node);
  while (scope) {
    for (const variable of scope.variables) {
      if (CORE_ALIASES.has(variable.name) && variable.defs.length > 0) {
        return true;
      }
    }
    scope = scope.upper;
  }
  return false;
}

export const preferCoreLoggingRule = createRule({
  name: "prefer-core-logging",
  meta: {
    type: "suggestion",
    hasSuggestions: true,
    docs: {
      description:
        "Prefer @actions/core logging methods (core.info, core.error, core.warning, core.debug) over console.* in files that have access to @actions/core. " +
        "console.* bypasses GitHub Actions' built-in secret masking and structured annotation system; core logging ensures secrets in output are redacted and messages appear correctly in the Actions UI.",
    },
    schema: [],
    messages: {
      preferCoreLogging:
        "Use {{replacement}} instead of console.{{method}}() — @actions/core logging masks secrets and integrates with the Actions annotation system. console.* output is not masked.",
      replaceWithCoreMethod: "Replace with {{replacement}}({{args}}) — ensure a @actions/core alias (core / coreObj) is in scope.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node) {
        const method = getConsoleMethod(node);
        if (!method) return;

        // Only flag when @actions/core alias is demonstrably in scope
        if (!hasCoreInScope(node, sourceCode)) return;

        const replacement = CONSOLE_TO_CORE[method]!;

        // Build replacement argument text from original call
        const argsText = node.arguments.map(arg => sourceCode.getText(arg)).join(", ");

        context.report({
          node,
          messageId: "preferCoreLogging",
          data: { method, replacement },
          suggest: [
            {
              messageId: "replaceWithCoreMethod",
              data: { replacement, args: argsText },
              fix(fixer) {
                return fixer.replaceText(node, `${replacement}(${argsText})`);
              },
            },
          ],
        });
      },
    };
  },
});
