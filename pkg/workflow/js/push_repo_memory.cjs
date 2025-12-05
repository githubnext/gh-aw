// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

/**
 * Push repo-memory changes to git branch
 * Environment variables:
 *   MEMORY_DIR: Path to the repo-memory directory
 *   TARGET_REPO: Target repository (owner/name)
 *   BRANCH_NAME: Branch name to push to
 *   MAX_FILE_SIZE: Maximum file size in bytes
 *   MAX_FILE_COUNT: Maximum number of files per commit
 *   FILE_GLOB_FILTER: Optional space-separated list of file patterns (e.g., "*.md *.txt")
 *   GH_TOKEN: GitHub token for authentication
 *   GITHUB_RUN_ID: Workflow run ID for commit messages
 */

async function main() {
  const memoryDir = process.env.MEMORY_DIR;
  const targetRepo = process.env.TARGET_REPO;
  const branchName = process.env.BRANCH_NAME;
  const maxFileSize = parseInt(process.env.MAX_FILE_SIZE || "10240", 10);
  const maxFileCount = parseInt(process.env.MAX_FILE_COUNT || "100", 10);
  const fileGlobFilter = process.env.FILE_GLOB_FILTER || "";
  const ghToken = process.env.GH_TOKEN;
  const githubRunId = process.env.GITHUB_RUN_ID || "unknown";

  // Validate required environment variables
  if (!memoryDir || !targetRepo || !branchName || !ghToken) {
    core.setFailed("Missing required environment variables: MEMORY_DIR, TARGET_REPO, BRANCH_NAME, GH_TOKEN");
    return;
  }

  // Check if memory directory exists
  if (!fs.existsSync(memoryDir)) {
    core.info(`Memory directory not found: ${memoryDir}`);
    return;
  }

  // Change to memory directory
  process.chdir(memoryDir);
  core.info(`Working directory: ${memoryDir}`);

  // Check if we have any changes to commit
  let hasChanges = false;
  try {
    const status = execSync("git status --porcelain", { encoding: "utf8" });
    hasChanges = status.trim().length > 0;
  } catch (error) {
    core.setFailed(`Failed to check git status: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  if (!hasChanges) {
    core.info("No changes detected in repo memory");
    return;
  }

  core.info("Changes detected in repo memory, committing and pushing...");

  // Stage all changes
  try {
    execSync("git add .", { stdio: "inherit" });
  } catch (error) {
    core.setFailed(`Failed to stage changes: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  // Validate file patterns if filter is set
  if (fileGlobFilter) {
    core.info(`Validating file patterns: ${fileGlobFilter}`);
    try {
      const stagedFiles = execSync("git diff --cached --name-only", { encoding: "utf8" })
        .trim()
        .split("\n")
        .filter(f => f);

      // Convert glob patterns to regex
      const patterns = fileGlobFilter.split(/\s+/).map(pattern => {
        // Convert glob pattern to regex
        // *.md -> ^[^/]*\.md$
        // *.txt -> ^[^/]*\.txt$
        const regexPattern = pattern.replace(/\./g, "\\.").replace(/\*/g, "[^/]*");
        return new RegExp(`^${regexPattern}$`);
      });

      const invalidFiles = stagedFiles.filter(file => {
        return !patterns.some(pattern => pattern.test(file));
      });

      if (invalidFiles.length > 0) {
        core.error("Files not matching allowed patterns detected:");
        invalidFiles.forEach(file => core.error(`  ${file}`));
        core.error(`Allowed patterns: ${fileGlobFilter}`);
        core.setFailed("File pattern validation failed");
        return;
      }
    } catch (error) {
      core.setFailed(`Failed to validate file patterns: ${error instanceof Error ? error.message : String(error)}`);
      return;
    }
  }

  // Check file sizes
  core.info(`Checking file sizes (max: ${maxFileSize} bytes)...`);
  try {
    const stagedFiles = execSync("git diff --cached --name-only", { encoding: "utf8" })
      .trim()
      .split("\n")
      .filter(f => f);
    const tooLarge = [];

    for (const file of stagedFiles) {
      if (fs.existsSync(file)) {
        const stats = fs.statSync(file);
        if (stats.size > maxFileSize) {
          tooLarge.push(`${file} (${stats.size} bytes)`);
        }
      }
    }

    if (tooLarge.length > 0) {
      core.error("Files exceeding size limit detected:");
      tooLarge.forEach(file => core.error(`  ${file}`));
      core.setFailed("File size validation failed");
      return;
    }
  } catch (error) {
    core.setFailed(`Failed to check file sizes: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  // Check file count
  core.info(`Checking file count (max: ${maxFileCount} files)...`);
  try {
    const stagedFiles = execSync("git diff --cached --name-only", { encoding: "utf8" })
      .trim()
      .split("\n")
      .filter(f => f);
    const fileCount = stagedFiles.length;

    if (fileCount > maxFileCount) {
      core.setFailed(`Too many files (${fileCount} > ${maxFileCount})`);
      return;
    }

    core.info(`Committing ${fileCount} file(s)...`);
  } catch (error) {
    core.setFailed(`Failed to check file count: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  // Commit changes
  try {
    execSync(`git commit -m "Update repo memory from workflow run ${githubRunId}"`, { stdio: "inherit" });
  } catch (error) {
    core.setFailed(`Failed to commit changes: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  // Pull with merge strategy (ours wins on conflicts)
  core.info(`Pulling latest changes from ${branchName}...`);
  try {
    const repoUrl = `https://x-access-token:${ghToken}@github.com/${targetRepo}.git`;
    execSync(`git pull --no-rebase -X ours "${repoUrl}" "${branchName}"`, { stdio: "inherit" });
  } catch (error) {
    // Pull might fail if branch doesn't exist yet or on conflicts - this is acceptable
    core.warning(`Pull failed (this may be expected): ${error instanceof Error ? error.message : String(error)}`);
  }

  // Push changes
  core.info(`Pushing changes to ${branchName}...`);
  try {
    const repoUrl = `https://x-access-token:${ghToken}@github.com/${targetRepo}.git`;
    execSync(`git push "${repoUrl}" HEAD:"${branchName}"`, { stdio: "inherit" });
    core.info(`Successfully pushed changes to ${branchName} branch`);
  } catch (error) {
    core.setFailed(`Failed to push changes: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }
}

main().catch(error => {
  core.setFailed(`Unexpected error: ${error instanceof Error ? error.message : String(error)}`);
});
