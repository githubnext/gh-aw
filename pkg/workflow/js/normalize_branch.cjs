/**
 * Normalizes the GITHUB_AW_ASSETS_BRANCH environment variable to be a valid git branch name.
 *
 * Valid characters: alphanumeric (a-z, A-Z, 0-9), dash (-), underscore (_), forward slash (/), dot (.)
 * Max length: 128 characters
 *
 * The normalization process:
 * 1. Replaces invalid characters with a single dash
 * 2. Removes leading and trailing dashes
 * 3. Truncates to 128 characters
 * 4. Removes trailing dashes after truncation
 * 5. Exports the normalized value to GITHUB_ENV
 */
async function main() {
  const branchName = process.env.GITHUB_AW_ASSETS_BRANCH;

  if (!branchName || branchName.trim() === "") {
    core.warning("⚠ GITHUB_AW_ASSETS_BRANCH not set, skipping normalization");
    return;
  }

  core.info(`Normalizing branch name: ${branchName}`);

  // Replace any sequence of invalid characters with a single dash
  // Valid characters are: a-z, A-Z, 0-9, -, _, /, .
  let normalized = branchName.replace(/[^a-zA-Z0-9\-_/.]+/g, "-");

  // Remove leading and trailing dashes
  normalized = normalized.replace(/^-+|-+$/g, "");

  // Truncate to max 128 characters
  if (normalized.length > 128) {
    normalized = normalized.substring(0, 128);
  }

  // Ensure it doesn't end with a dash after truncation
  normalized = normalized.replace(/-+$/, "");

  // Export the normalized value to GITHUB_ENV
  core.exportVariable("GITHUB_AW_ASSETS_BRANCH", normalized);

  core.info(`✓ Normalized GITHUB_AW_ASSETS_BRANCH: ${normalized}`);

  // Also set as output for potential use by subsequent steps
  core.setOutput("normalized_branch", normalized);
}

await main();
