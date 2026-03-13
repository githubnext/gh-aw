// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Normalizes a branch name to be a valid git branch name.
 *
 * IMPORTANT: Keep this function in sync with the normalizeBranchName function in upload_assets.cjs
 *
 * Valid characters: alphanumeric (a-z, A-Z, 0-9), dash (-), underscore (_), forward slash (/), dot (.)
 * Max length: 128 characters (before salt is appended)
 *
 * The normalization process:
 * 1. Replaces invalid characters with a single dash
 * 2. Collapses multiple consecutive dashes to a single dash
 * 3. Removes leading and trailing dashes
 * 4. Truncates to 128 characters
 * 5. Removes trailing dashes after truncation
 * 6. Optionally converts to lowercase (opt-in via `options.lowercase`)
 * 7. Optionally appends a salt suffix (opt-in via `options.salt`)
 *
 * @param {string} branchName - The branch name to normalize
 * @param {{ lowercase?: boolean, salt?: string | null }} [options] - Normalization options
 * @param {boolean} [options.lowercase=false] - When true, converts the branch name to lowercase
 * @param {string | null} [options.salt=null] - When set, appends `-<salt>` to the branch name
 * @returns {string} The normalized branch name
 */
function normalizeBranchName(branchName, { lowercase = false, salt = null } = {}) {
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

  // Optionally convert to lowercase
  if (lowercase) {
    normalized = normalized.toLowerCase();
  }

  // Optionally append a salt suffix
  if (salt) {
    normalized = `${normalized}-${salt}`;
  }

  return normalized;
}

module.exports = {
  normalizeBranchName,
};
