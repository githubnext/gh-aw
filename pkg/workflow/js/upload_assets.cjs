const fs = require("fs");
const path = require("path");
const crypto = require("crypto");

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

  // Read the validated output content from environment variable
  const agentOutputFile = process.env.GH_AW_AGENT_OUTPUT;
  if (!agentOutputFile) {
    core.info("No GH_AW_AGENT_OUTPUT environment variable found");
    core.setOutput("upload_count", "0");
    core.setOutput("branch_name", normalizedBranchName);
    return;
  }

  // Read agent output from file
  let outputContent;
  try {
    outputContent = fs.readFileSync(agentOutputFile, "utf8");
  } catch (error) {
    core.setFailed(`Error reading agent output file: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    core.setOutput("upload_count", "0");
    core.setOutput("branch_name", normalizedBranchName);
    return;
  }

  core.info(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(`Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    core.setOutput("upload_count", "0");
    core.setOutput("branch_name", normalizedBranchName);
    return;
  }

  // Find all upload-asset items
  const uploadAssetItems = validatedOutput.items.filter(/** @param {any} item */ item => item.type === "upload_asset");
  if (uploadAssetItems.length === 0) {
    core.info("No upload-asset items found in agent output");
    core.setOutput("upload_count", "0");
    core.setOutput("branch_name", normalizedBranchName);
    return;
  }

  core.info(`Found ${uploadAssetItems.length} upload-asset item(s)`);

  let uploadCount = 0;
  let hasChanges = false;

  try {
    // Check if orphaned branch already exists, if not create it
    try {
      await exec.exec(`git rev-parse --verify origin/${normalizedBranchName}`);
      await exec.exec(`git checkout -B ${normalizedBranchName} origin/${normalizedBranchName}`);
      core.info(`Checked out existing branch from origin: ${normalizedBranchName}`);
    } catch (originError) {
      // Give an error if branch doesn't exist on origin
      core.info(`Creating new orphaned branch: ${normalizedBranchName}`);
      await exec.exec(`git checkout --orphan ${normalizedBranchName}`);
      await exec.exec(`git rm -rf .`);
      await exec.exec(`git clean -fdx`);
    }

    // Process each asset
    for (const asset of uploadAssetItems) {
      try {
        const { fileName, sha, size, targetFileName } = asset;

        if (!fileName || !sha || !targetFileName) {
          core.error(`Invalid asset entry missing required fields: ${JSON.stringify(asset)}`);
          continue;
        }

        // Check if file exists in artifacts
        const assetSourcePath = path.join("/tmp/gh-aw/safe-outputs/assets", fileName);
        if (!fs.existsSync(assetSourcePath)) {
          core.warning(`Asset file not found: ${assetSourcePath}`);
          continue;
        }

        // Verify SHA matches
        const fileContent = fs.readFileSync(assetSourcePath);
        const computedSha = crypto.createHash("sha256").update(fileContent).digest("hex");

        if (computedSha !== sha) {
          core.warning(`SHA mismatch for ${fileName}: expected ${sha}, got ${computedSha}`);
          continue;
        }

        // Check if file already exists in the branch
        if (fs.existsSync(targetFileName)) {
          core.info(`Asset ${targetFileName} already exists, skipping`);
          continue;
        }

        // Copy file to branch with target filename
        fs.copyFileSync(assetSourcePath, targetFileName);

        // Add to git
        await exec.exec(`git add "${targetFileName}"`);

        uploadCount++;
        hasChanges = true;

        core.info(`Added asset: ${targetFileName} (${size} bytes)`);
      } catch (error) {
        core.warning(`Failed to process asset ${asset.fileName}: ${error instanceof Error ? error.message : String(error)}`);
      }
    }

    // Commit and push if there are changes (skip if staged)
    if (hasChanges) {
      const commitMessage = `[skip-ci] Add ${uploadCount} asset(s)`;
      await exec.exec(`git`, [`commit`, `-m`, `"${commitMessage}"`]);
      if (isStaged) {
        core.summary.addRaw("## Staged Asset Publication");
      } else {
        await exec.exec(`git push origin ${normalizedBranchName}`);
        core.summary
          .addRaw("## Assets")
          .addRaw(`Successfully uploaded **${uploadCount}** assets to branch \`${normalizedBranchName}\``)
          .addRaw("");
        core.info(`Successfully uploaded ${uploadCount} assets to branch ${normalizedBranchName}`);
      }

      for (const asset of uploadAssetItems) {
        if (asset.fileName && asset.sha && asset.size && asset.url) {
          core.summary.addRaw(`- [\`${asset.fileName}\`](${asset.url}) → \`${asset.targetFileName}\` (${asset.size} bytes)`);
        }
      }
      core.summary.write();
    } else {
      core.info("No new assets to upload");
    }
  } catch (error) {
    core.setFailed(`Failed to upload assets: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  core.setOutput("upload_count", uploadCount.toString());
  core.setOutput("branch_name", normalizedBranchName);
}

await main();
