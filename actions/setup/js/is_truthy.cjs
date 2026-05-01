// @ts-check
/**
 * Determines if a value is truthy according to template logic.
 *
 * Supports:
 *   - Simple falsy string check: "", "false", "no", "0", "null", "undefined"
 *   - GitHub Actions script style equality: lhs == "rhs" or lhs === "rhs"
 *     After experiment substitution the condition looks like: concise == "concise"
 *
 * @param {string} expr - The expression to evaluate
 * @returns {boolean} - Whether the expression is truthy
 */
function isTruthy(expr) {
  const trimmed = expr.trim();

  // Handle GitHub Actions script style equality expressions: lhs == "rhs" or lhs === "rhs"
  // Used by experiment conditionals after the experiment value has been substituted:
  //   {{#if experiments.prompt_style == "concise"}} becomes {{#if concise == "concise"}}
  // Note: (.*?) allows an empty LHS — if the experiment variable was not set the substituted
  // condition looks like ' == "concise"', which correctly returns false here rather than
  // falling through to the generic truthy check and incorrectly returning true.
  const eqMatch = trimmed.match(/^(.*?)\s*===?\s*"([^"]*)"\s*$/);
  if (eqMatch) {
    return eqMatch[1].trim() === eqMatch[2];
  }
  const neqMatch = trimmed.match(/^(.*?)\s*!==?\s*"([^"]*)"\s*$/);
  if (neqMatch) {
    return neqMatch[1].trim() !== neqMatch[2];
  }

  const v = trimmed.toLowerCase();
  return !(v === "" || v === "false" || v === "no" || v === "0" || v === "null" || v === "undefined");
}

module.exports = { isTruthy };
