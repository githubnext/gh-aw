import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

interface ConstantDeclaration {
  name: string;
  declaration: TSESTree.VariableDeclarator;
}

const MIN_NUMERIC_DUPLICATE_GROUP_SIZE = 3;

function getStaticValueKey(node: TSESTree.Expression): string | null {
  if (node.type === AST_NODE_TYPES.Literal) {
    if ("regex" in node && node.regex) {
      return `regexp:${node.regex.pattern}/${[...node.regex.flags].sort().join("")}`;
    }
    return `${typeof node.value}:${String(node.value)}`;
  }

  if (node.type === AST_NODE_TYPES.TemplateLiteral && node.expressions.length === 0) {
    return `string:${node.quasis[0].value.cooked ?? node.quasis[0].value.raw}`;
  }

  if (node.type === AST_NODE_TYPES.UnaryExpression && (node.operator === "+" || node.operator === "-") && node.argument.type === AST_NODE_TYPES.Literal && typeof node.argument.value === "number") {
    return `number:${String(node.operator === "-" ? -node.argument.value : node.argument.value)}`;
  }

  return null;
}

export const noDuplicateConstantValuesRule = createRule({
  name: "no-duplicate-constant-values",
  meta: {
    type: "suggestion",
    docs: {
      description: "List module-level constant declarations by their static primitive values and report later declarations that duplicate a value in the same file.",
    },
    schema: [],
    messages: {
      duplicateConstantValue: "Constant '{{name}}' duplicates the value of constant '{{originalName}}' ({{value}}).",
    },
  },
  defaultOptions: [],
  create(context) {
    const constantsByValue = new Map<string, ConstantDeclaration[]>();

    return {
      VariableDeclaration(node) {
        if (node.kind !== "const" || node.parent.type !== AST_NODE_TYPES.Program) {
          return;
        }

        for (const declaration of node.declarations) {
          if (declaration.id.type !== AST_NODE_TYPES.Identifier || !declaration.init) {
            continue;
          }

          const valueKey = getStaticValueKey(declaration.init);
          if (valueKey === null) {
            continue;
          }

          const declarationsForValue = constantsByValue.get(valueKey);
          if (!declarationsForValue) {
            constantsByValue.set(valueKey, [
              {
                name: declaration.id.name,
                declaration,
              },
            ]);
            continue;
          }

          declarationsForValue.push({
            name: declaration.id.name,
            declaration,
          });
        }
      },
      "Program:exit"() {
        for (const [valueKey, declarations] of constantsByValue) {
          const shouldReportDuplicates = declarations.length > 1 && (!valueKey.startsWith("number:") || declarations.length >= MIN_NUMERIC_DUPLICATE_GROUP_SIZE);
          if (!shouldReportDuplicates) continue;

          const original = declarations[0];
          for (const duplicate of declarations.slice(1)) {
            context.report({
              node: duplicate.declaration,
              messageId: "duplicateConstantValue",
              data: {
                name: duplicate.name,
                originalName: original.name,
                value: context.sourceCode.getText(duplicate.declaration.init!),
              },
            });
          }
        }
      },
    };
  },
});
