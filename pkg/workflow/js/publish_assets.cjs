const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const { execSync } = require("child_process");

async function main() {
  const branchPrefix = process.env.GITHUB_AW_BRANCH_PREFIX || "assets";
  const staged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";

  core.info("Starting asset publishing process");

  if (staged) {
    core.summary
      .addRaw("## Asset Publishing (Staged Mode)")
      .addRaw(
        "**Note**: Running in staged mode - no actual publishing performed"
      )
      .addRaw(
        "Assets would be published to orphaned branch with prefix: `" +
          branchPrefix +
          "`"
      )
      .write();

    core.setOutput("published_count", "0");
    core.setOutput("branch_name", branchPrefix + "-staged");
    return;
  }

  // Read safe outputs to find assets to publish
  const safeOutputsFile = "/tmp/safe-outputs/safe-outputs.jsonl";
  let assetsToPublish = [];

  if (fs.existsSync(safeOutputsFile)) {
    const lines = fs.readFileSync(safeOutputsFile, "utf8").trim().split("\n");
    for (const line of lines) {
      if (line.trim()) {
        try {
          const entry = JSON.parse(line);
          if (entry.type === "publish-asset") {
            assetsToPublish.push(entry);
          }
        } catch (error) {
          core.warning(`Failed to parse JSONL line: ${line}`);
        }
      }
    }
  }

  if (assetsToPublish.length === 0) {
    core.info("No assets found to publish");
    core.setOutput("published_count", "0");
    core.setOutput("branch_name", "");
    return;
  }

  core.info(`Found ${assetsToPublish.length} assets to publish`);

  // Configure git
  try {
    execSync("git config user.name 'github-actions[bot]'", {
      stdio: "inherit",
    });
    execSync(
      "git config user.email '41898282+github-actions[bot]@users.noreply.github.com'",
      { stdio: "inherit" }
    );
  } catch (error) {
    core.setFailed(
      `Failed to configure git: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }

  // Generate unique branch name
  const timestamp = new Date().toISOString().slice(0, 19).replace(/[:-]/g, "");
  const branchName = `${branchPrefix}-${timestamp}`;

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
      // Remove all files from the working directory to start clean
      try {
        execSync("git rm -rf .", { stdio: "pipe" });
      } catch {
        // Ignore errors if there are no files to remove
      }
    }

    // Process each asset
    for (const asset of assetsToPublish) {
      try {
        const { fileName, filePath, sha, size } = asset;

        if (!fileName || !filePath || !sha) {
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

        // Use SHA as filename to avoid conflicts
        const targetFileName = `${sha.substring(0, 8)}-${fileName}`;
        const targetPath = targetFileName;

        // Check if file already exists in the branch
        if (fs.existsSync(targetPath)) {
          core.info(`Asset ${targetFileName} already exists, skipping`);
          continue;
        }

        // Copy file to branch
        fs.copyFileSync(assetSourcePath, targetPath);

        // Add to git
        execSync(`git add "${targetPath}"`, { stdio: "inherit" });

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

      for (const asset of assetsToPublish) {
        if (asset.fileName && asset.sha && asset.size) {
          const targetFileName = `${asset.sha.substring(0, 8)}-${asset.fileName}`;
          const rawUrl = `https://raw.githubusercontent.com/${context.repo.owner}/${context.repo.repo}/${branchName}/${targetFileName}`;
          core.summary.addRaw(
            `- [\`${asset.fileName}\`](${rawUrl}) (${asset.size} bytes)`
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
