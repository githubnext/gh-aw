// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");

/**
 * Normalizes a branch name to be a valid git branch name.
 *
 * IMPORTANT: Keep this function in sync with the normalizeBranchName function in safe_outputs_mcp_server.cjs
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

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Get the branch name from environment variable (required)
  const branchName = process.env.GH_AW_ASSETS_BRANCH;
  if (!branchName || typeof branchName !== "string") {
    core.setFailed("GH_AW_ASSETS_BRANCH environment variable is required but not set");
    return;
  }

  // Normalize the branch name to ensure it's a valid git branch name
  const normalizedBranchName = normalizeBranchName(branchName);
  core.info(`Using assets branch: ${normalizedBranchName}`);

  const result = loadAgentOutput();
  if (!result.success) {
    core.setOutput("upload_count", "0");
    core.setOutput("branch_name", normalizedBranchName);
    return;
  }

  // Find all upload-asset items
  const uploadItems = result.items.filter(/** @param {any} item */ item => item.type === "upload_asset");

  if (uploadItems.length === 0) {
    core.info("No upload-asset items found in agent output");
    core.setOutput("upload_count", "0");
    core.setOutput("branch_name", normalizedBranchName);
    return;
  }

  core.info(`Found ${uploadItems.length} upload-asset item(s)`);

  let uploadCount = 0;
  let hasChanges = false;

  try {
    // Check if orphaned branch already exists, if not create it
    try {
      await exec.exec(`git rev-parse --verify origin/${normalizedBranchName}`);
      await exec.exec(`git checkout -B ${normalizedBranchName} origin/${normalizedBranchName}`);
      core.info(`Checked out existing branch from origin: ${normalizedBranchName}`);
    } catch (originError) {
      // Validate that branch starts with "assets/" prefix before creating orphaned branch
      if (!normalizedBranchName.startsWith("assets/")) {
        core.setFailed(
          `Branch '${normalizedBranchName}' does not start with the required 'assets/' prefix. ` +
            `Orphaned branches can only be automatically created under the 'assets/' prefix. ` +
            `Please create the branch manually first, or use a branch name starting with 'assets/'.`
        );
        return;
      }

      // Branch doesn't exist on origin and has valid prefix, create orphaned branch
      core.info(`Creating new orphaned branch: ${normalizedBranchName}`);
      await exec.exec(`git checkout --orphan ${normalizedBranchName}`);
      await exec.exec(`git rm -rf .`);
      await exec.exec(`git clean -fdx`);
    }

    // Process each asset
    for (const asset of uploadItems) {
      const { fileName, sha, size, targetFileName } = asset;

      if (!fileName || !sha || !targetFileName) {
        core.setFailed(`Invalid asset entry missing required fields: ${JSON.stringify(asset)}`);
        return;
      }

      // Check if file exists in artifacts
      const assetSourcePath = path.join("/tmp/gh-aw/safeoutputs/assets", fileName);
      if (!fs.existsSync(assetSourcePath)) {
        core.setFailed(`Asset file not found: ${assetSourcePath}`);
        return;
      }

      // Verify SHA matches
      const fileContent = fs.readFileSync(assetSourcePath);
      const computedSha = crypto.createHash("sha256").update(fileContent).digest("hex");

      if (computedSha !== sha) {
        core.setFailed(`SHA mismatch for ${fileName}: expected ${sha}, got ${computedSha}`);
        return;
      }

      // Check if file already exists in the branch
      if (fs.existsSync(targetFileName)) {
        core.info(`Asset ${targetFileName} already exists, skipping`);
        continue;
      }

      // Copy file to branch with target filename
      try {
        fs.copyFileSync(assetSourcePath, targetFileName);

        // Add to git
        await exec.exec(`git add "${targetFileName}"`);

        uploadCount++;
        hasChanges = true;

        core.info(`Added asset: ${targetFileName} (${size} bytes)`);
      } catch (error) {
        core.setFailed(`Failed to process asset ${fileName}: ${getErrorMessage(error)}`);
        return;
      }
    }

    // Commit and push if there are changes (skip if staged)
    if (hasChanges) {
      const commitMessage = `[skip-ci] Add ${uploadCount} asset(s)`;
      await exec.exec(`git`, [`commit`, `-m`, commitMessage]);
      if (isStaged) {
        core.summary.addRaw("## Staged Asset Publication");
      } else {
        await exec.exec(`git push origin ${normalizedBranchName}`);
        core.summary.addRaw("## Assets").addRaw(`Successfully uploaded **${uploadCount}** assets to branch \`${normalizedBranchName}\``).addRaw("");
        core.info(`Successfully uploaded ${uploadCount} assets to branch ${normalizedBranchName}`);
      }

      for (const asset of uploadItems) {
        if (asset.fileName && asset.sha && asset.size && asset.url) {
          core.summary.addRaw(`- [\`${asset.fileName}\`](${asset.url}) → \`${asset.targetFileName}\` (${asset.size} bytes)`);
        }
      }
      core.summary.write();
    } else {
      core.info("No new assets to upload");
    }
  } catch (error) {
    core.setFailed(`Failed to upload assets: ${getErrorMessage(error)}`);
    return;
  }

  core.setOutput("upload_count", uploadCount.toString());
  core.setOutput("branch_name", normalizedBranchName);
}

module.exports = { main };
