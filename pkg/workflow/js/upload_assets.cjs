const fs = require("fs");
const path = require("path");
const crypto = require("crypto");

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";

  // Get the branch name from environment variable (required)
  const branchName = process.env.GITHUB_AW_ASSETS_BRANCH;
  if (!branchName || typeof branchName !== "string") {
    core.setFailed(
      "GITHUB_AW_ASSETS_BRANCH environment variable is required but not set"
    );
    return;
  }
  core.info(`Using assets branch: ${branchName}`);

  // Read the validated output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    core.setOutput("upload_count", "0");
    core.setOutput("branch_name", branchName);
    return;
  }
  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    core.setOutput("upload_count", "0");
    core.setOutput("branch_name", branchName);
    return;
  }

  core.info(`Agent output content length: ${outputContent.length}`);

  // Parse the validated output JSON
  let validatedOutput;
  try {
    validatedOutput = JSON.parse(outputContent);
  } catch (error) {
    core.setFailed(
      `Error parsing agent output JSON: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }

  if (!validatedOutput.items || !Array.isArray(validatedOutput.items)) {
    core.info("No valid items found in agent output");
    core.setOutput("upload_count", "0");
    core.setOutput("branch_name", branchName);
    return;
  }

  // Find all upload_asset items
  const uploadAssetItems = validatedOutput.items.filter(
    /** @param {any} item */ item => item.type === "upload_asset"
  );
  if (uploadAssetItems.length === 0) {
    core.info("No upload_asset items found in agent output");
    core.setOutput("upload_count", "0");
    core.setOutput("branch_name", branchName);
    return;
  }

  core.info(`Found ${uploadAssetItems.length} upload_asset item(s)`);

  let uploadCount = 0;
  let hasChanges = false;

  try {
    // Check if orphaned branch already exists, if not create it
    try {
      await exec.exec(`git rev-parse --verify origin/${branchName}`);
      await exec.exec(`git checkout -B ${branchName} origin/${branchName}`);
      core.info(`Checked out existing branch from origin: ${branchName}`);
    } catch (originError) {
      // Give an error if branch doesn't exist on origin
      core.info(`Creating new orphaned branch: ${branchName}`);
      await exec.exec(`git checkout --orphan ${branchName}`);
    }

    // Process each asset
    for (const asset of uploadAssetItems) {
      try {
        const { fileName, sha, size, targetFileName } = asset;

        if (!fileName || !sha || !targetFileName) {
          core.error(
            `Invalid asset entry missing required fields: ${JSON.stringify(asset)}`
          );
          continue;
        }

        // Check if file exists in artifacts
        const assetSourcePath = path.join("/tmp/safe-outputs/assets", fileName);
        if (!fs.existsSync(assetSourcePath)) {
          core.warning(`Asset file not found: ${assetSourcePath}`);
          continue;
        }

        // Verify SHA matches
        const fileContent = fs.readFileSync(assetSourcePath);
        const computedSha = crypto
          .createHash("sha256")
          .update(fileContent)
          .digest("hex");

        if (computedSha !== sha) {
          core.warning(
            `SHA mismatch for ${fileName}: expected ${sha}, got ${computedSha}`
          );
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
        core.warning(
          `Failed to process asset ${asset.fileName}: ${error instanceof Error ? error.message : String(error)}`
        );
      }
    }

    // Commit and push if there are changes (skip if staged)
    if (hasChanges) {
      const commitMessage = `[skip-ci] Add ${uploadCount} asset(s)`;
      await exec.exec(`git`, [`commit`, `-m`, `"${commitMessage}"`]);
      if (isStaged) {
        core.addRaw("## Staged Asset Publication");
      } else {
        core.summary
          .addRaw("## Assets")
          .addRaw(
            `Successfully uploaded **${uploadCount}** assets to branch \`${branchName}\``
          )
          .addRaw("");
        await exec.exec(`git push origin ${branchName}`);
        core.info(
          `Successfully uploaded ${uploadCount} assets to branch ${branchName}`
        );
      }

      for (const asset of uploadAssetItems) {
        if (asset.fileName && asset.sha && asset.size && asset.url) {
          core.summary.addRaw(
            `- [\`${asset.fileName}\`](${asset.url}) → \`${asset.targetFileName}\` (${asset.size} bytes)`
          );
        }
      }
      core.summary.write();
    } else {
      core.info("No new assets to upload");
    }
  } catch (error) {
    core.setFailed(
      `Failed to upload assets: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }

  core.setOutput("upload_count", uploadCount.toString());
  core.setOutput("branch_name", branchName);
}

await main();
