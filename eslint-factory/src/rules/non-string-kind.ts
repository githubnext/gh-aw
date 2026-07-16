import { AST_NODE_TYPES, TSESTree } from "@typescript-eslint/utils";

/**
 * Returns a description of the non-string value kind if the node is one of the
 * low-false-positive forms targeted by these rules, or null if the value may already
 * be a string or cannot be determined without type information.
 *
 * Targeted forms (low false-positive risk):
 *   - Numeric literal: 0, 42, 3.14
 *   - Boolean literal: true, false
 *   - Null literal
 *   - Identifier `undefined`
 *   - .length member access: commonly numeric in practice
 */
export const NUMERIC_LITERAL_KIND = "numeric literal" as const;
export const BOOLEAN_LITERAL_KIND = "boolean literal" as const;
export const NULL_KIND = "null" as const;
export const UNDEFINED_KIND = "undefined" as const;
export const LENGTH_KIND = ".length (number)" as const;

export type NonStringKind = typeof NUMERIC_LITERAL_KIND | typeof BOOLEAN_LITERAL_KIND | typeof NULL_KIND | typeof UNDEFINED_KIND | typeof LENGTH_KIND;

export function nonStringKind(node: TSESTree.Node): NonStringKind | null {
  if (node.type === AST_NODE_TYPES.Literal) {
    if (typeof node.value === "number") return NUMERIC_LITERAL_KIND;
    if (typeof node.value === "boolean") return BOOLEAN_LITERAL_KIND;
    if (node.value === null) return NULL_KIND;
  }

  if (node.type === AST_NODE_TYPES.Identifier && node.name === "undefined") {
    return UNDEFINED_KIND;
  }

  // expr.length — commonly numeric; computed access (expr["length"]) is intentionally
  // excluded because it is far less common and raises the FP risk slightly.
  if (node.type === AST_NODE_TYPES.MemberExpression && !node.computed && node.property.type === AST_NODE_TYPES.Identifier && node.property.name === "length") {
    return LENGTH_KIND;
  }

  return null;
}
