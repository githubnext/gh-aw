// @ts-check
/**
 * Determines if a value is truthy according to template logic.
 *
 * Supports:
 *   - Simple falsy string check: "", "false", "no", "0", "null", "undefined"
 *   - Equality helper: (eq VALUE "LITERAL") — returns true when VALUE === LITERAL
 *
 * @param {string} expr - The expression to evaluate
 * @returns {boolean} - Whether the expression is truthy
 */
function isTruthy(expr) {
  const trimmed = expr.trim();

  // Handle (eq VALUE "LITERAL") helper expression.
  // Used by experiment conditionals: {{#if (eq concise "concise")}}
  // after the experiment placeholder has been substituted into the condition.
  const eqMatch = trimmed.match(/^\(eq\s+(.+?)\s+"(.+?)"\)$/i);
  if (eqMatch) {
    return eqMatch[1].trim() === eqMatch[2].trim();
  }

  const v = trimmed.toLowerCase();
  return !(v === "" || v === "false" || v === "no" || v === "0" || v === "null" || v === "undefined");
}

module.exports = { isTruthy };
