const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const { execSync } = require("child_process");

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";
  
  // Get the branch name from environment variable (required)
  const branchName = process.env.GITHUB_AW_ASSETS_BRANCH;
  if (!branchName) {
    core.setFailed("GITHUB_AW_ASSETS_BRANCH environment variable is required but not set");
    return;
  }

  core.info("Starting asset publishing process");

  // Read the validated output content from environment variable
  const outputContent = process.env.GITHUB_AW_AGENT_OUTPUT;
  if (!outputContent) {
    core.info("No GITHUB_AW_AGENT_OUTPUT environment variable found");
    core.setOutput("published_count", "0");
    core.setOutput("branch_name", branchName);
    return;
  }
  if (outputContent.trim() === "") {
    core.info("Agent output content is empty");
    core.setOutput("published_count", "0");
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
    core.setOutput("published_count", "0");
    core.setOutput("branch_name", branchName);
    return;
  }

  // Find all publish-assets items
  const publishAssetItems = validatedOutput.items.filter(
    /** @param {any} item */ item => item.type === "publish-assets"
  );
  if (publishAssetItems.length === 0) {
    core.info("No publish-assets items found in agent output");
    core.setOutput("published_count", "0");
    core.setOutput("branch_name", branchName);
    return;
  }

  core.info(`Found ${publishAssetItems.length} publish-assets item(s)`);

  // If in staged mode, process files but don't push
  if (isStaged) {
    let summaryContent = "## 🎭 Staged Mode: Asset Publishing Preview\n\n";
    summaryContent += "The following assets would be published if staged mode was disabled:\n\n";
    
    let processedCount = 0;
    for (const asset of publishAssetItems) {
      try {
        const { fileName, filePath, sha, size, targetFileName } = asset;
        
        if (!fileName || !filePath || !sha || !targetFileName) {
          core.warning(
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
        const computedSha = crypto.createHash("sha256").update(fileContent).digest("hex");
        
        if (computedSha !== sha) {
          core.warning(
            `SHA mismatch for ${fileName}: expected ${sha}, got ${computedSha}`
          );
          continue;
        }

        summaryContent += `- **${fileName}** → \`${targetFileName}\` (${size} bytes, SHA: ${sha.substring(0, 16)}...)\n`;
        processedCount++;
        
      } catch (error) {
        core.warning(
          `Failed to process asset ${asset.fileName}: ${error instanceof Error ? error.message : String(error)}`
        );
      }
    }
    
    summaryContent += `\n**Total assets staged for publishing:** ${processedCount}\n`;
    summaryContent += `**Target branch:** ${branchName}\n`;
    
    core.summary.addRaw(summaryContent).write();
    core.setOutput("published_count", processedCount.toString());
    core.setOutput("branch_name", branchName);
    return;
  }

  let publishedCount = 0;
  let hasChanges = false;

  try {
    // Check if orphaned branch already exists, if not create it
    let branchExists = false;
    try {
      execSync(`git ls-remote --heads origin ${branchName}`, { stdio: "pipe" });
      branchExists = true;
    } catch {
      // Branch doesn't exist, will create it
    }

    if (branchExists) {
      core.info(`Checking out existing orphaned branch: ${branchName}`);
      execSync(`git fetch origin ${branchName}`, { stdio: "inherit" });
      execSync(`git checkout ${branchName}`, { stdio: "inherit" });
    } else {
      core.info(`Creating new orphaned branch: ${branchName}`);
      execSync(`git checkout --orphan ${branchName}`, { stdio: "inherit" });
    }

    // Process each asset
    for (const asset of publishAssetItems) {
      try {
        const { fileName, filePath, sha, size, targetFileName } = asset;

        if (!fileName || !filePath || !sha || !targetFileName) {
          core.warning(
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
        execSync(`git add "${targetFileName}"`, { stdio: "inherit" });

        publishedCount++;
        hasChanges = true;

        core.info(`Added asset: ${targetFileName} (${size} bytes)`);
      } catch (error) {
        core.warning(
          `Failed to process asset ${asset.fileName}: ${error instanceof Error ? error.message : String(error)}`
        );
      }
    }

    // Commit and push if there are changes
    if (hasChanges) {
      const commitMessage = `Add ${publishedCount} asset(s) via GitHub Actions`;
      execSync(`git commit -m "${commitMessage}"`, { stdio: "inherit" });
      execSync(`git push origin ${branchName}`, { stdio: "inherit" });

      core.info(
        `Successfully published ${publishedCount} assets to branch ${branchName}`
      );

      // Add summary
      core.summary
        .addRaw("## Assets Published")
        .addRaw(
          `Successfully published **${publishedCount}** assets to branch \`${branchName}\``
        )
        .addRaw("")
        .addRaw("### Published Assets:")
        .addRaw("");

      for (const asset of publishAssetItems) {
        if (asset.fileName && asset.sha && asset.size && asset.url) {
          core.summary.addRaw(
            `- [\`${asset.fileName}\`](${asset.url}) → \`${asset.targetFileName}\` (${asset.size} bytes)`
          );
        }
      }

      core.summary.write();
    } else {
      core.info("No new assets to publish");
    }
  } catch (error) {
    core.setFailed(
      `Failed to publish assets: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }

  core.setOutput("published_count", publishedCount.toString());
  core.setOutput("branch_name", branchName);
}

await main();
