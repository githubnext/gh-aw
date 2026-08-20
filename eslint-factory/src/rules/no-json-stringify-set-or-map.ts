import { AST_NODE_TYPES, ESLintUtils, TSESLint, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

type SourceCodeScope = ReturnType<TSESLint.SourceCode["getScope"]>;

/**
 * Returns true when `name` at `node` resolves to the global binding, i.e. no
 * enclosing scope declares a local variable, import, class or function with
 * that name shadowing the built-in.
 */
function isGlobalReference(sourceCode: TSESLint.SourceCode, node: TSESTree.Node, name: string): boolean {
  let scope: SourceCodeScope | null = sourceCode.getScope(node);

  while (scope) {
    const variable = scope.set.get(name);
    if (variable && variable.defs.length > 0) return false;
    scope = scope.upper;
  }

  return true;
}

/** Returns "Set"/"Map" when `expr` is a `new Set(...)` / `new Map(...)` construction using the global constructor. */
function isSetOrMapConstruction(sourceCode: TSESLint.SourceCode, expr: TSESTree.Node): "Set" | "Map" | null {
  if (expr.type !== AST_NODE_TYPES.NewExpression) return null;
  if (expr.callee.type !== AST_NODE_TYPES.Identifier) return null;
  const name = expr.callee.name;
  if (name !== "Set" && name !== "Map") return null;
  if (!isGlobalReference(sourceCode, expr.callee, name)) return null;
  return name;
}

function getConstSetOrMapBindingKind(sourceCode: TSESLint.SourceCode, identifier: TSESTree.Identifier): "Set" | "Map" | null {
  let scope: SourceCodeScope | null = sourceCode.getScope(identifier);

  while (scope) {
    const variable = scope.set.get(identifier.name);
    if (variable && variable.defs.length > 0) {
      for (const def of variable.defs) {
        if (def.type !== "Variable") continue;

        const declarator = def.node as TSESTree.VariableDeclarator;
        if (declarator.id.type !== AST_NODE_TYPES.Identifier || declarator.id.name !== identifier.name) continue;
        if (!declarator.init) continue;

        const declaration = declarator.parent;
        if (declaration.type !== AST_NODE_TYPES.VariableDeclaration || declaration.kind !== "const") continue;

        const kind = isSetOrMapConstruction(sourceCode, declarator.init);
        if (kind) return kind;
      }

      return null;
    }

    scope = scope.upper;
  }

  return null;
}

export const noJsonStringifySetOrMapRule = createRule({
  name: "no-json-stringify-set-or-map",
  meta: {
    type: "problem",
    docs: {
      description:
        "Disallow JSON.stringify() directly on a Set or Map instance — both serialize to '{}' with no own enumerable properties, silently dropping every entry. " +
        "Convert with Array.from(set) / [...set] for a Set, or Object.fromEntries(map) / Array.from(map) for a Map, before stringifying.",
    },
    schema: [],
    messages: {
      jsonStringifySetOrMap: "JSON.stringify({{varName}}) serializes a {{kind}} instance to '{}' — {{kind}} entries are not own enumerable properties and are silently dropped. Convert first: {{suggestion}}.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    function suggestionFor(kind: "Set" | "Map", varName: string): string {
      return kind === "Set" ? `Array.from(${varName})` : `Object.fromEntries(${varName})`;
    }

    return {
      CallExpression(node: TSESTree.CallExpression) {
        const callee = node.callee;
        if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return;
        const obj = callee.object;
        const prop = callee.property;
        if (obj.type !== AST_NODE_TYPES.Identifier || obj.name !== "JSON") return;
        if (!isGlobalReference(sourceCode, obj, "JSON")) return;
        if (prop.type !== AST_NODE_TYPES.Identifier || prop.name !== "stringify") return;

        const firstArg = node.arguments[0];
        if (!firstArg) return;

        // Direct inline construction: JSON.stringify(new Set(...))
        const inlineKind = isSetOrMapConstruction(sourceCode, firstArg);
        if (inlineKind) {
          const suggestion = inlineKind === "Set" ? "Array.from(...)" : "Object.fromEntries(...)";
          context.report({
            node,
            messageId: "jsonStringifySetOrMap",
            data: { varName: sourceCode.getText(firstArg), kind: inlineKind, suggestion },
          });
          return;
        }

        // Reference to a `const x = new Set(...)/new Map(...)` binding.
        if (firstArg.type !== AST_NODE_TYPES.Identifier) return;
        const kind = getConstSetOrMapBindingKind(sourceCode, firstArg);
        if (!kind) return;

        context.report({
          node,
          messageId: "jsonStringifySetOrMap",
          data: { varName: firstArg.name, kind, suggestion: suggestionFor(kind, firstArg.name) },
        });
      },
    };
  },
});
