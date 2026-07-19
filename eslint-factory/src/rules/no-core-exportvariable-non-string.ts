import { AST_NODE_TYPES, ESLintUtils, TSESLint } from "@typescript-eslint/utils";
import { CORE_ALIASES } from "./core-aliases";
import { isCoreAliasIdentifier, isDestructuredCoreMethodIdentifier } from "./core-method-resolve";
import { nonStringKind, NULL_KIND, UNDEFINED_KIND } from "./non-string-kind";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

export const noCoreExportVariableNonStringRule = createRule({
  name: "no-core-exportvariable-non-string",
  meta: {
    type: "problem",
    hasSuggestions: true,
    docs: {
      description:
        "Require core.exportVariable value arguments to be explicit strings; passing numbers, booleans, null, undefined, or .length can silently produce unexpected string representations (e.g. 'null', 'true') in downstream GitHub Actions steps that read the exported environment variable. Detects calls in the form core.exportVariable(name, value), aliased (const c = core; c.exportVariable(...)), and destructured (const { exportVariable } = core; exportVariable(...)).",
    },
    schema: [],
    messages: {
      nonStringValue:
        "The exportVariable value {{valueText}} is a {{kind}}. Implicit coercion may produce unexpected strings such as 'null' or 'true' when the environment variable is read by downstream steps. Use an explicit string conversion and choose the suggestion that matches the intended output semantics.",
      wrapWithString: "Wrap with String({{valueText}}) to make coercion explicit. For null/undefined, use an explicit default (for example '') when empty-string semantics are intended.",
      useEmptyString: "Replace with \"\" (empty string) — use this when the intended output is empty rather than the literal word 'null' or 'undefined'.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    return {
      CallExpression(node) {
        const callee = node.callee;

        if (callee.type === AST_NODE_TYPES.MemberExpression) {
          // Object must be a known @actions/core alias or a single-assignment alias (e.g. `const c = core`)
          if (callee.object.type !== AST_NODE_TYPES.Identifier) return;
          if (!CORE_ALIASES.has(callee.object.name) && !isCoreAliasIdentifier(callee.object, sourceCode)) return;

          // Property must be `exportVariable` (direct or computed string-literal access)
          const prop = callee.property;
          const isExportVariableProp = (!callee.computed && prop.type === AST_NODE_TYPES.Identifier && prop.name === "exportVariable") || (callee.computed && prop.type === AST_NODE_TYPES.Literal && prop.value === "exportVariable");
          if (!isExportVariableProp) return;
        } else if (callee.type === AST_NODE_TYPES.Identifier) {
          // Destructured: `const { exportVariable } = core; exportVariable(...)` or `const { exportVariable: alias } = core; alias(...)`
          if (!isDestructuredCoreMethodIdentifier(callee, "exportVariable", sourceCode)) return;
        } else {
          return;
        }

        // core.exportVariable expects exactly two arguments: (name, value)
        if (node.arguments.length !== 2) return;

        const valueArg = node.arguments[1];

        const kind = nonStringKind(valueArg);
        if (kind === null) return;

        const valueText = sourceCode.getText(valueArg);

        const isNullOrUndefined = kind === NULL_KIND || kind === UNDEFINED_KIND;

        context.report({
          node,
          messageId: "nonStringValue",
          data: { kind, valueText },
          suggest: [
            ...(isNullOrUndefined
              ? [
                  {
                    messageId: "useEmptyString" as const,
                    fix(fixer: TSESLint.RuleFixer) {
                      return fixer.replaceText(valueArg, `""`);
                    },
                  },
                ]
              : []),
            {
              messageId: "wrapWithString" as const,
              data: { valueText },
              fix(fixer: TSESLint.RuleFixer) {
                return fixer.replaceText(valueArg, `String(${valueText})`);
              },
            },
          ],
        });
      },
    };
  },
});
