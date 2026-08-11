import { AST_NODE_TYPES, ESLintUtils, TSESTree } from "@typescript-eslint/utils";

const createRule = ESLintUtils.RuleCreator(name => `https://github.com/github/gh-aw/tree/main/eslint-factory#${name}`);

/** Returns true when `expr` is a `new Set(...)` or `new Map(...)` construction. */
function isSetOrMapConstruction(expr: TSESTree.Node): "Set" | "Map" | null {
  if (expr.type !== AST_NODE_TYPES.NewExpression) return null;
  if (expr.callee.type !== AST_NODE_TYPES.Identifier) return null;
  if (expr.callee.name === "Set") return "Set";
  if (expr.callee.name === "Map") return "Map";
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
      jsonStringifySetOrMap:
        "JSON.stringify({{varName}}) serializes a {{kind}} instance to '{}' — {{kind}} entries are not own enumerable properties and are silently dropped. Convert first: {{suggestion}}.",
    },
  },
  defaultOptions: [],
  create(context) {
    const sourceCode = context.sourceCode;

    // Track variable names in scope whose declared initializer is `new Set(...)` / `new Map(...)`,
    // mapped to which kind they are. Reassignment to something else removes the binding from tracking
    // (handled conservatively: only VariableDeclarator initializers are tracked, not `let` reassignment).
    const trackedVars = new Map<string, "Set" | "Map">();

    function suggestionFor(kind: "Set" | "Map", varName: string): string {
      return kind === "Set" ? `Array.from(${varName})` : `Object.fromEntries(${varName})`;
    }

    return {
      VariableDeclarator(node: TSESTree.VariableDeclarator) {
        if (node.id.type !== AST_NODE_TYPES.Identifier) return;
        if (!node.init) return;
        const kind = isSetOrMapConstruction(node.init);
        if (!kind) return;

        // Only track when declared with `const` — `let`/`var` bindings could be reassigned
        // to a non-Set/Map value later, which this rule cannot statically follow.
        const declaration = node.parent;
        if (declaration.type !== AST_NODE_TYPES.VariableDeclaration || declaration.kind !== "const") return;

        trackedVars.set(node.id.name, kind);
      },

      CallExpression(node: TSESTree.CallExpression) {
        const callee = node.callee;
        if (callee.type !== AST_NODE_TYPES.MemberExpression || callee.computed) return;
        const obj = callee.object;
        const prop = callee.property;
        if (obj.type !== AST_NODE_TYPES.Identifier || obj.name !== "JSON") return;
        if (prop.type !== AST_NODE_TYPES.Identifier || prop.name !== "stringify") return;

        const firstArg = node.arguments[0];
        if (!firstArg) return;

        // Direct inline construction: JSON.stringify(new Set(...))
        const inlineKind = isSetOrMapConstruction(firstArg);
        if (inlineKind) {
          const suggestion = inlineKind === "Set" ? "Array.from(...)" : "Object.fromEntries(...)";
          context.report({
            node,
            messageId: "jsonStringifySetOrMap",
            data: { varName: sourceCode.getText(firstArg), kind: inlineKind, suggestion },
          });
          return;
        }

        // Reference to a tracked `const x = new Set(...)/new Map(...)` binding.
        if (firstArg.type !== AST_NODE_TYPES.Identifier) return;
        const kind = trackedVars.get(firstArg.name);
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
