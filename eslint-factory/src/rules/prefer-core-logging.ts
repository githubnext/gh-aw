import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

// Maps console method → recommended core replacement.
// NOTE: console.error and console.warn are intentionally excluded.
// Those methods write to process.stderr, while core.error / core.warning emit
// GitHub Actions workflow commands to process.stdout. For processes that own
// stdout as a data/protocol channel (e.g. stdio MCP servers), rewriting
// stderr logging to stdout would corrupt the stream. The substitution is not
// behavior-preserving and must not be auto-applied.
const CONSOLE_TO_CORE: Record<string, string> = {
  log: "core.info",
  info: "core.info",
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

function getStaticStringValue(node: TSESTree.CallExpressionArgument): string | null {
  if (node.type === AST_NODE_TYPES.Literal) {
    return typeof node.value === "string" ? node.value : null;
  }

  if (node.type !== AST_NODE_TYPES.TemplateLiteral) {
    return null;
  }

  if (node.expressions.length > 0) {
    return null;
  }

  const parts: string[] = [];
  for (const quasi of node.quasis) {
    if (quasi.value.cooked === null) {
      return null;
    }
    parts.push(quasi.value.cooked);
  }

  return parts.join("");
}

function hasConsoleFormatSpecifier(node: TSESTree.CallExpressionArgument | undefined): boolean {
  const value = node ? getStaticStringValue(node) : null;
  if (value === null) {
    return false;
  }

  for (let i = 0; i < value.length - 1; i++) {
    if (value[i] === "%" && value[i - 1] !== "%" && "sdifojO".includes(value[i + 1] ?? "")) {
      return true;
    }
  }

  return false;
}

function isInterpolatedTemplateLiteral(node: TSESTree.CallExpressionArgument): boolean {
  return node.type === AST_NODE_TYPES.TemplateLiteral && node.expressions.length > 0;
}

function canSuggestCoreReplacement(node: TSESTree.CallExpression): boolean {
  const arg = node.arguments[0];
  if (node.arguments.length !== 1 || !arg) {
    return false;
  }

  return !isInterpolatedTemplateLiteral(arg) && !hasConsoleFormatSpecifier(arg);
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
        const suggest = canSuggestCoreReplacement(node)
          ? [
              {
                messageId: "replaceWithCoreMethod" as const,
                data: { replacement, args: sourceCode.getText(node.arguments[0]) },
                fix(fixer: TSESLint.RuleFixer) {
                  return fixer.replaceText(node, `${replacement}(${sourceCode.getText(node.arguments[0])})`);
                },
              },
            ]
          : undefined;

        context.report({
          node,
          messageId: "preferCoreLogging",
          data: { method, replacement },
          suggest,
        });
      },
    };
  },
});
