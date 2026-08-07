// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const crypto = require("crypto");
const { loadAgentOutput } = require("./load_agent_output.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { ERR_API, ERR_CONFIG, ERR_SYSTEM, ERR_VALIDATION } = require("./error_codes.cjs");
const { normalizeBranchName } = require("./normalize_branch_name.cjs");

/**
 * @typedef {{ type: string, path?: string, fileName: string, sha: string, size: number, targetFileName: string, url?: string }} UploadAssetItem
 */

/**
 * @param {string} githubServer
 * @param {string} repo
 * @param {string} branchName
 * @param {string} targetFileName
 * @returns {string}
 */
function buildAssetUrl(githubServer, repo, branchName, targetFileName) {
  try {
    const serverHostname = new URL(githubServer).hostname;
    if (serverHostname === "github.com") {
      return `https://github.com/${repo}/blob/${branchName}/${targetFileName}?raw=true`;
    }
  } catch {
    // Fall through to the GHES-compatible raw URL.
  }
  return `${githubServer}/${repo}/raw/${branchName}/${targetFileName}`;
}

async function main() {
  // Check if we're in staged mode
  const isStaged = process.env.GH_AW_SAFE_OUTPUTS_STAGED === "true";

  // Get the branch name from environment variable (required)
  const branchName = process.env.GH_AW_ASSETS_BRANCH;
  if (!branchName || typeof branchName !== "string") {
    core.setFailed(`${ERR_CONFIG}: GH_AW_ASSETS_BRANCH environment variable is required but not set`);
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
  /** @type {UploadAssetItem[]} */
  const uploadItems = result.items.filter(item => item.type === "upload_asset");

  if (uploadItems.length === 0) {
    core.info("No upload-asset items found in agent output");
    core.setOutput("upload_count", "0");
    core.setOutput("branch_name", normalizedBranchName);
    return;
  }

  core.info(`Found ${uploadItems.length} upload-asset item(s)`);

  // Read the staged-assets directory directly. The upload_assets job's
  // download-artifact step writes the safe-outputs assets artifact to this exact
  // directory, and the Go generator passes the same path via GH_AW_ASSETS_DIR, so
  // producer and consumer can never disagree on the location. The literal fallback
  // matches constants.TmpGhAwAssetsDir for robustness if the env var is unset.
  const assetsDir = process.env.GH_AW_ASSETS_DIR || "/tmp/gh-aw/safeoutputs/assets";
  core.info(`Reading staged assets from: ${assetsDir}`);
  let uploadCount = 0;
  let missingAssetCount = 0;
  let hasChanges = false;
  const processedAssets = [];

  try {
    // Check if orphaned branch already exists, if not create it
    try {
      await exec.exec("git", ["rev-parse", "--verify", `origin/${normalizedBranchName}`]);
      await exec.exec("git", ["checkout", "-B", normalizedBranchName, `origin/${normalizedBranchName}`]);
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
      await exec.exec("git", ["checkout", "--orphan", normalizedBranchName]);
      await exec.exec("git", ["rm", "-rf", "."]);
      await exec.exec("git", ["clean", "-fdx"]);
    }

    // Process each asset
    for (const asset of uploadItems) {
      const declaredPath = typeof asset.path === "string" ? asset.path : "";
      const pathFileName = declaredPath ? path.basename(declaredPath) : "";
      const rawFileName = typeof asset.fileName === "string" ? asset.fileName : pathFileName;
      const fileName = path.basename(rawFileName);

      if (!fileName || (asset.fileName && asset.fileName !== fileName)) {
        core.setFailed(`${ERR_VALIDATION}: Invalid asset filename: ${JSON.stringify(asset)}`);
        return;
      }
      if (!pathFileName && (!asset.sha || !asset.targetFileName)) {
        core.setFailed(`${ERR_VALIDATION}: Invalid asset entry missing required fields: ${JSON.stringify(asset)}`);
        return;
      }

      // Check if file exists in the staged-assets directory
      const stagedFileName = pathFileName ? `${crypto.createHash("sha256").update(declaredPath).digest("hex")}${path.extname(fileName).toLowerCase()}` : fileName;
      const assetSourcePath = path.join(assetsDir, stagedFileName);
      if (!fs.existsSync(assetSourcePath)) {
        core.warning(`${ERR_SYSTEM}: Asset file not found: ${assetSourcePath} — skipping`);
        missingAssetCount++;
        continue;
      }

      // Verify SHA matches
      const fileContent = fs.readFileSync(assetSourcePath);
      const computedSha = crypto.createHash("sha256").update(fileContent).digest("hex");

      if (asset.sha && computedSha !== asset.sha) {
        core.setFailed(`${ERR_VALIDATION}: SHA mismatch for ${fileName}: expected ${asset.sha}, got ${computedSha}`);
        return;
      }

      const generatedTargetFileName = `${computedSha}${path.extname(fileName).toLowerCase()}`;
      const targetFileName = asset.targetFileName || generatedTargetFileName;
      if (targetFileName !== path.basename(targetFileName)) {
        core.setFailed(`${ERR_VALIDATION}: Invalid asset target filename: ${targetFileName}`);
        return;
      }

      const size = fileContent.length;
      const githubServer = process.env.GITHUB_SERVER_URL || "https://github.com";
      const repo = process.env.GITHUB_REPOSITORY || "owner/repo";
      const url = buildAssetUrl(githubServer, repo, normalizedBranchName, targetFileName);
      processedAssets.push({ fileName, sha: computedSha, size, targetFileName, url });

      // Check if file already exists in the branch
      if (fs.existsSync(targetFileName)) {
        core.info(`Asset ${targetFileName} already exists, skipping`);
        continue;
      }

      // Copy file to branch with target filename
      try {
        fs.copyFileSync(assetSourcePath, targetFileName);

        // Add to git
        await exec.exec("git", ["add", targetFileName]);

        uploadCount++;
        hasChanges = true;

        core.info(`Added asset: ${targetFileName} (${size} bytes)`);
      } catch (error) {
        core.setFailed(`${ERR_API}: Failed to process asset ${fileName}: ${getErrorMessage(error)}`);
        return;
      }
    }

    if (uploadCount === 0 && missingAssetCount > 0 && missingAssetCount === uploadItems.length) {
      core.setFailed(`All ${missingAssetCount} declared assets were missing; no assets published.`);
      return;
    }

    // Commit and push if there are changes (skip if staged)
    if (hasChanges) {
      const commitMessage = `[skip-ci] Add ${uploadCount} asset(s)`;
      await exec.exec(`git`, [`commit`, `-m`, commitMessage]);
      if (isStaged) {
        core.summary.addRaw("## 🎭 Staged Mode: Asset Publication Preview");
      } else {
        await exec.exec("git", ["push", "origin", normalizedBranchName]);
        core.summary.addRaw("## Assets").addRaw(`Successfully uploaded **${uploadCount}** assets to branch \`${normalizedBranchName}\``).addRaw("");
        core.info(`Successfully uploaded ${uploadCount} assets to branch ${normalizedBranchName}`);
      }

      for (const asset of processedAssets) {
        core.summary.addRaw(`- [\`${asset.fileName}\`](${asset.url}) → \`${asset.targetFileName}\` (${asset.size} bytes)`);
      }
      await core.summary.write();
    } else {
      core.info("No new assets to upload");
    }
  } catch (error) {
    core.setFailed(`${ERR_API}: Failed to upload assets: ${getErrorMessage(error)}`);
    return;
  }

  core.setOutput("upload_count", uploadCount.toString());
  core.setOutput("branch_name", normalizedBranchName);
}

module.exports = { main };
