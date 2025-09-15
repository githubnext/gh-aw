const { execSync } = require("child_process");
const fs = require("fs");

// Get environment variables
const agentOutput = process.env.GITHUB_AW_AGENT_OUTPUT || "{}";
const maxCount = parseInt(
  process.env.GITHUB_AW_ORPHANED_BRANCH_MAX_COUNT || "1"
);
const isStaged = process.env.GITHUB_AW_SAFE_OUTPUTS_STAGED === "true";

const repo = context.repo;
const owner = context.repo.owner;

core.info(`Processing agent output for orphaned branch upload`);
core.info(`Repository: ${owner}/${repo.repo}`);
core.info(`Max files allowed: ${maxCount}`);

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

if (isStaged) {
  // In staged mode, just show what would be uploaded
  core.summary.addHeading("Orphaned Branch File Upload (Staged Mode)", 2);
  core.summary.addRaw(
    "The following files would be uploaded to an orphaned branch:\n\n"
  );

  for (const item of orphanedBranchItems) {
    core.summary.addRaw(
      `- **${item.filename}** (${Math.round(item.content.length * 0.75)} bytes)\n`
    );
    uploadedFiles.push(item.filename);
    fileUrls.push(
      `https://raw.githubusercontent.com/${owner}/${repo.repo}/orphaned-uploads/staged/${item.filename}`
    );
  }

  await core.summary.write();
} else {
  // Actually upload files to orphaned branch
  const branchName = "orphaned-uploads";
  const timestamp = new Date().toISOString().replace(/[:.]/g, "-");

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
      const { filename, content } = item;

      if (!filename || !content) {
        core.warning(`Skipping invalid item: ${JSON.stringify(item)}`);
        continue;
      }

      // Decode base64 content and write file
      const fileBuffer = Buffer.from(content, "base64");
      const safeFilename = filename.replace(/[^a-zA-Z0-9._-]/g, "_");
      const timestampedFilename = `${timestamp}-${safeFilename}`;

      fs.writeFileSync(timestampedFilename, fileBuffer);
      core.info(
        `Created file: ${timestampedFilename} (${fileBuffer.length} bytes)`
      );

      // Add to git
      execSync(`git add ${timestampedFilename}`, { stdio: "inherit" });

      uploadedFiles.push(timestampedFilename);
    }

    // Commit files
    const commitMessage = `Upload ${uploadedFiles.length} file(s) to orphaned branch\n\nFiles: ${uploadedFiles.join(", ")}`;
    execSync(`git commit -m "${commitMessage}"`, { stdio: "inherit" });

    // Push to remote
    execSync(`git push origin ${branchName}`, { stdio: "inherit" });

    // Get the commit SHA
    const commitSha = execSync(`git rev-parse HEAD`, {
      encoding: "utf8",
    }).trim();
    core.info(`Pushed to orphaned branch with commit: ${commitSha}`);

    // Generate GitHub raw URLs
    for (const filename of uploadedFiles) {
      const rawUrl = `https://raw.githubusercontent.com/${owner}/${repo.repo}/${commitSha}/${filename}`;
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
      core.summary.addRaw(`- [${uploadedFiles[i]}](${fileUrls[i]})\n`);
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

core.info(
  `Successfully processed ${uploadedFiles.length} file(s) for orphaned branch upload`
);
