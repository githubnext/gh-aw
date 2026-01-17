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
 * Extract a concise error message from filesystem errors.
 * Node.js filesystem errors include the file path in the error message,
 * which can be redundant when the file path is already in the log message.
 * This function extracts just the error code and description.
 *
 * Examples:
 *   "EACCES: permission denied, open '/path/file'" -> "EACCES: permission denied"
 *   "ENOENT: no such file or directory, open '/path/file'" -> "ENOENT: no such file or directory"
 *
 * @param {unknown} error - The error value to extract a message from
 * @returns {string} The concise error message without file path
 */
function getErrorMessageWithoutPath(error) {
  const message = getErrorMessage(error);

  // Match Node.js filesystem error pattern: "CODE: description, operation 'path'"
  // We want to extract just "CODE: description"
  const match = message.match(/^([A-Z]+:\s+[^,]+)/);
  if (match) {
    return match[1];
  }

  // If no match, return the full message
  return message;
}

module.exports = { getErrorMessage, getErrorMessageWithoutPath };
