// @ts-check
/**
 * Determines if a value is truthy according to template logic
 * @param {string} expr - The expression to evaluate
 * @returns {boolean} - Whether the expression is truthy
 */
function isTruthy(expr) {
  const v = expr.trim().toLowerCase();
  return !(v === "" || v === "false" || v === "0" || v === "null" || v === "undefined");
}

module.exports = { isTruthy };
