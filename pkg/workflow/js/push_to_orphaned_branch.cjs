const { execSync } = require("child_process");
const fs = require("fs");

// Get environment variables
const agentOutput = process.env.GITHUB_AW_AGENT_OUTPUT || "{}";
const maxCount = parseInt(
  process.env.GITHUB_AW_ORPHANED_BRANCH_MAX_COUNT || "1"
);
const branchName =
  process.env.GITHUB_AW_ORPHANED_BRANCH_NAME || "assets/workflow";
const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";

const repo = context.repo;
const owner = context.repo.owner;

core.info(`Processing agent output for orphaned branch upload`);
core.info(`Repository: ${owner}/${repo.repo}`);
core.info(`Max files allowed: ${maxCount}`);
core.info(`Branch name: ${branchName}`);

let parsedOutput;
try {
  parsedOutput = JSON.parse(agentOutput);
} catch (error) {
  core.setFailed(
    `Failed to parse agent output: ${error instanceof Error ? error.message : String(error)}`
  );
  return;
}

// Extract push-to-orphaned-branch items
const orphanedBranchItems = (parsedOutput.items || []).filter(
  item => item.type === "push-to-orphaned-branch"
);

if (orphanedBranchItems.length === 0) {
  core.info("No orphaned branch upload items found in agent output");
  return;
}

if (orphanedBranchItems.length > maxCount) {
  core.setFailed(
    `Too many files to upload: ${orphanedBranchItems.length} (max: ${maxCount})`
  );
  return;
}

core.info(
  `Found ${orphanedBranchItems.length} file(s) to upload to orphaned branch`
);

const uploadedFiles = [];
const fileUrls = [];
let commitSha = null;

if (isStaged) {
  // In staged mode, just show what would be uploaded
  core.summary.addHeading("Orphaned Branch File Upload (Staged Mode)", 2);
  core.summary.addRaw(
    "The following files would be uploaded to an orphaned branch:\n\n"
  );

  for (const item of orphanedBranchItems) {
    const originalFilename = item.original_filename || item.filename;
    const sha = item.sha || "unknown";
    core.summary.addRaw(
      `- **${item.filename}** (${Math.round(item.content.length * 0.75)} bytes) - SHA: ${sha} - Original: ${originalFilename}\n`
    );
    uploadedFiles.push(item.filename);
    fileUrls.push(
      `https://raw.githubusercontent.com/${owner}/${repo.repo}/${branchName}/staged/${item.filename}`
    );
  }

  await core.summary.write();
} else {
  // Actually upload files to orphaned branch
  try {
    // Create or switch to orphaned branch
    try {
      execSync(`git checkout ${branchName}`, { stdio: "inherit" });
      core.info(`Switched to existing orphaned branch: ${branchName}`);
    } catch (error) {
      // Branch doesn't exist, create orphaned branch
      execSync(`git checkout --orphan ${branchName}`, { stdio: "inherit" });
      execSync(`git rm -rf .`, { stdio: "inherit" });
      core.info(`Created new orphaned branch: ${branchName}`);
    }

    // Upload each file
    for (const item of orphanedBranchItems) {
      const { filename, original_filename, sha } = item;

      if (!filename) {
        core.warning(`Skipping invalid item: ${JSON.stringify(item)}`);
        continue;
      }

      // Find the file in the artifact files directory
      const filesDir =
        process.env.GITHUB_AW_SAFE_OUTPUTS_FILES_DIR ||
        `${process.env.GITHUB_AW_SAFE_OUTPUTS_DIR || "/tmp/gh-aw/safe-outputs"}/files`;
      const sourceFile = `${filesDir}/${filename}`;

      if (!fs.existsSync(sourceFile)) {
        core.setFailed(`File not found in artifact: ${sourceFile}`);
        return;
      }

      // Read the file and validate SHA
      const fileBuffer = fs.readFileSync(sourceFile);
      const crypto = require("crypto");
      const computedHash = crypto.createHash("sha256");
      computedHash.update(fileBuffer);
      const computedSha = computedHash.digest("hex");

      const fileSha = sha || "unknown";
      if (fileSha !== "unknown" && fileSha !== computedSha) {
        core.setFailed(
          `SHA validation failed for ${filename}. Expected: ${fileSha}, Computed: ${computedSha}`
        );
        return;
      }

      // Use the SHA-based filename directly (it already includes the extension)
      const safeFilename = filename.replace(/[^a-zA-Z0-9._-]/g, "_");

      // Copy file to working directory for git operations
      fs.copyFileSync(sourceFile, safeFilename);
      const originalName = original_filename || filename;
      core.info(
        `Created file: ${safeFilename} (${fileBuffer.length} bytes) - SHA: ${fileSha} - Original: ${originalName}`
      );

      // Add to git
      execSync(`git add ${safeFilename}`, { stdio: "inherit" });

      uploadedFiles.push(safeFilename);
    }

    // Commit files
    const fileList = uploadedFiles
      .map(filename => {
        const item = orphanedBranchItems.find(
          i => i.filename.replace(/[^a-zA-Z0-9._-]/g, "_") === filename
        );
        const originalName = item?.original_filename || filename;
        const sha = item?.sha || "unknown";
        return `${filename} (${originalName}, SHA: ${sha.substring(0, 8)})`;
      })
      .join(", ");
    const commitMessage = `Upload ${uploadedFiles.length} file(s) to orphaned branch\n\nFiles: ${fileList}`;
    execSync(`git commit -m "${commitMessage}"`, { stdio: "inherit" });

    // Push to remote
    execSync(`git push origin ${branchName}`, { stdio: "inherit" });

    // Get the commit SHA
    commitSha = execSync(`git rev-parse HEAD`, {
      encoding: "utf8",
    }).trim();
    core.info(`Pushed to orphaned branch with commit: ${commitSha}`);

    // Generate GitHub raw URLs using branch name
    for (const filename of uploadedFiles) {
      const rawUrl = `https://raw.githubusercontent.com/${owner}/${repo.repo}/${branchName}/${filename}`;
      fileUrls.push(rawUrl);
      core.info(`File URL: ${rawUrl}`);
    }

    // Add summary
    core.summary.addHeading("Files Uploaded to Orphaned Branch", 2);
    core.summary.addRaw(
      `Successfully uploaded ${uploadedFiles.length} file(s) to orphaned branch \`${branchName}\`\n\n`
    );
    core.summary.addRaw(`**Commit:** \`${commitSha}\`\n\n`);
    core.summary.addRaw("**Files:**\n");

    for (let i = 0; i < uploadedFiles.length; i++) {
      const filename = uploadedFiles[i];
      const item = orphanedBranchItems.find(
        item => item.filename.replace(/[^a-zA-Z0-9._-]/g, "_") === filename
      );
      const originalName = item?.original_filename || filename;
      const sha = item?.sha || "unknown";
      core.summary.addRaw(
        `- [${filename}](${fileUrls[i]}) - Original: ${originalName} - SHA: ${sha.substring(0, 8)}\n`
      );
    }

    await core.summary.write();
  } catch (error) {
    core.setFailed(
      `Failed to upload files to orphaned branch: ${error instanceof Error ? error.message : String(error)}`
    );
    return;
  }
}

// Set outputs
core.setOutput("uploaded_files", JSON.stringify(uploadedFiles));
core.setOutput("file_urls", JSON.stringify(fileUrls));
if (commitSha) {
  core.setOutput("commit_sha", commitSha);
}

core.info(
  `Successfully processed ${uploadedFiles.length} file(s) for orphaned branch upload`
);
