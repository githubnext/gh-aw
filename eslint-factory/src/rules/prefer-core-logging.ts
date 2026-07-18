import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

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

export const preferCoreLoggingRule = createRule({
  name: "prefer-core-logging",
  meta: {
    type: "suggestion",
    hasSuggestions: true,
    docs: {
      description:
        "Prefer @actions/core logging methods (core.info, core.error, core.warning, core.debug) over console.* — " +
        "global.core is always available via shim.cjs in Node.js context and via github-script in Actions context. " +
        "core.* methods integrate with the Actions annotation system (errors and warnings appear as file annotations in the UI) and produce structured log output; console.* does not.",
    },
    schema: [],
    messages: {
      preferCoreLogging: "Use {{replacement}} instead of console.{{method}}() — @actions/core logging integrates with the Actions annotation system and structured output. global.core is always available via shim.cjs.",
      replaceWithCoreMethod: "Replace with {{replacement}}({{args}}).",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node) {
        const method = getConsoleMethod(node);
        if (!method) return;

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
