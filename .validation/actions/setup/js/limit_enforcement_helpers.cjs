// @ts-check

/**
 * Helper functions for enforcing maximum limits on safe output parameters.
 * These functions align with SEC-003 requirements for preventing resource exhaustion.
 */

/**
 * Enforces a maximum limit on an array parameter.
 * Throws E003 error when the limit is exceeded.
 *
 * @param {Array<any>|undefined|null} array - The array to check
 * @param {number} maxLimit - Maximum allowed length
 * @param {string} parameterName - Name of the parameter for error messages
 * @throws {Error} When array length exceeds maxLimit, with E003 error code
 */
function enforceArrayLimit(array, maxLimit, parameterName) {
  if (array && Array.isArray(array) && array.length > maxLimit) {
    throw new Error(`E003: Cannot add more than ${maxLimit} ${parameterName} (received ${array.length})`);
  }
}

/**
 * Safely enforces array limit and catches exceptions.
 * Returns an error result object instead of throwing.
 *
 * @param {Array<any>|undefined|null} array - The array to check
 * @param {number} maxLimit - Maximum allowed length
 * @param {string} parameterName - Name of the parameter for error messages
 * @returns {{success: true} | {success: false, error: string}} Result object
 */
function tryEnforceArrayLimit(array, maxLimit, parameterName) {
  try {
    enforceArrayLimit(array, maxLimit, parameterName);
    return { success: true };
  } catch (error) {
    const { getErrorMessage } = require("./error_helpers.cjs");
    return {
      success: false,
      error: getErrorMessage(error),
    };
  }
}

module.exports = {
  enforceArrayLimit,
  tryEnforceArrayLimit,
};
