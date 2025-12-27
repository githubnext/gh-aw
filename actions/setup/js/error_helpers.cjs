// @ts-check

/**
 * Safely extract an error message from an unknown error value.
 * Handles Error instances, objects with message properties, and other values.
 *
 * @param {unknown} error - The error value to extract a message from
 * @returns {string} The error message as a string
 */
function getErrorMessage(error) {
  if (error instanceof Error) {
    return error.message;
  }
  if (error && typeof error === "object" && "message" in error && typeof error.message === "string") {
    return error.message;
  }
  return String(error);
}

module.exports = { getErrorMessage };
