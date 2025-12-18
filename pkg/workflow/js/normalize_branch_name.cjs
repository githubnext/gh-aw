// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Normalizes a branch name to be a valid git branch name.
 *
 * IMPORTANT: Keep this function in sync with the normalizeBranchName function in upload_assets.cjs
 *
 * Valid characters: alphanumeric (a-z, A-Z, 0-9), dash (-), underscore (_), forward slash (/), dot (.)
 * Max length: 128 characters
 *
 * The normalization process:
 * 1. Replaces invalid characters with a single dash
 * 2. Collapses multiple consecutive dashes to a single dash
 * 3. Removes leading and trailing dashes
 * 4. Truncates to 128 characters
 * 5. Removes trailing dashes after truncation
 * 6. Converts to lowercase
 *
 * @param {string} branchName - The branch name to normalize
 * @returns {string} The normalized branch name
 */
function normalizeBranchName(branchName) {
  if (!branchName || typeof branchName !== "string" || branchName.trim() === "") {
    return branchName;
  }

  // Replace any sequence of invalid characters with a single dash
  // Valid characters are: a-z, A-Z, 0-9, -, _, /, .
  let normalized = branchName.replace(/[^a-zA-Z0-9\-_/.]+/g, "-");

  // Collapse multiple consecutive dashes to a single dash
  normalized = normalized.replace(/-+/g, "-");

  // Remove leading and trailing dashes
  normalized = normalized.replace(/^-+|-+$/g, "");

  // Truncate to max 128 characters
  if (normalized.length > 128) {
    normalized = normalized.substring(0, 128);
  }

  // Ensure it doesn't end with a dash after truncation
  normalized = normalized.replace(/-+$/, "");

  // Convert to lowercase
  normalized = normalized.toLowerCase();

  return normalized;
}

module.exports = {
  normalizeBranchName,
};
