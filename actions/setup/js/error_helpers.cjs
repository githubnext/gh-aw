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

/**
 * Check if an error is due to a locked issue/PR/discussion.
 * GitHub API returns 403 with specific messages for locked resources.
 * This helper is used to determine if an operation should be silently ignored.
 *
 * @param {unknown} error - The error value to check
 * @returns {boolean} True if error is due to locked resource, false otherwise
 */
function isLockedError(error) {
  // Check if the error has a 403 status code
  const is403Error = error && typeof error === "object" && "status" in error && error.status === 403;
  if (!is403Error) {
    return false;
  }

  // Check if the error message mentions "locked"
  const errorMessage = getErrorMessage(error);
  const hasLockedMessage = Boolean(errorMessage && (errorMessage.includes("locked") || errorMessage.includes("Lock conversation")));

  // Only return true if it's BOTH a 403 status code AND mentions locked
  return hasLockedMessage;
}

module.exports = { getErrorMessage, isLockedError };
