// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");

/**
 * Push repo-memory changes to git branch
 * Environment variables:
 *   ARTIFACT_DIR: Path to the downloaded artifact directory containing memory files
 *   MEMORY_ID: Memory identifier (used for subdirectory path)
 *   TARGET_REPO: Target repository (owner/name)
 *   BRANCH_NAME: Branch name to push to
 *   MAX_FILE_SIZE: Maximum file size in bytes
 *   MAX_FILE_COUNT: Maximum number of files per commit
 *   FILE_GLOB_FILTER: Optional space-separated list of file patterns (e.g., "*.md *.txt")
 *   GH_TOKEN: GitHub token for authentication
 *   GITHUB_RUN_ID: Workflow run ID for commit messages
 */

async function main() {
  const artifactDir = process.env.ARTIFACT_DIR;
  const memoryId = process.env.MEMORY_ID;
  const targetRepo = process.env.TARGET_REPO;
  const branchName = process.env.BRANCH_NAME;
  const maxFileSize = parseInt(process.env.MAX_FILE_SIZE || "10240", 10);
  const maxFileCount = parseInt(process.env.MAX_FILE_COUNT || "100", 10);
  const fileGlobFilter = process.env.FILE_GLOB_FILTER || "";
  const ghToken = process.env.GH_TOKEN;
  const githubRunId = process.env.GITHUB_RUN_ID || "unknown";

  // Validate required environment variables
  if (!artifactDir || !memoryId || !targetRepo || !branchName || !ghToken) {
    core.setFailed("Missing required environment variables: ARTIFACT_DIR, MEMORY_ID, TARGET_REPO, BRANCH_NAME, GH_TOKEN");
    return;
  }

  // Source directory with memory files (artifact location)
  const sourceMemoryPath = path.join(artifactDir, "memory", memoryId);

  // Check if artifact memory directory exists
  if (!fs.existsSync(sourceMemoryPath)) {
    core.info(`Memory directory not found in artifact: ${sourceMemoryPath}`);
    return;
  }

  // We're already in the checked out repository (from checkout step)
  const workspaceDir = process.env.GITHUB_WORKSPACE || process.cwd();
  core.info(`Working in repository: ${workspaceDir}`);

  // Disable sparse checkout to work with full branch content
  // This is necessary because checkout was configured with sparse-checkout
  core.info(`Disabling sparse checkout...`);
  try {
    execSync("git sparse-checkout disable", { stdio: "pipe" });
  } catch (error) {
    // Ignore if sparse checkout wasn't enabled
    core.info("Sparse checkout was not enabled or already disabled");
  }

  // Checkout or create the memory branch
  core.info(`Checking out branch: ${branchName}...`);
  try {
    const repoUrl = `https://x-access-token:${ghToken}@github.com/${targetRepo}.git`;

    // Try to fetch the branch
    try {
      execSync(`git fetch "${repoUrl}" "${branchName}:${branchName}"`, { stdio: "pipe" });
      execSync(`git checkout "${branchName}"`, { stdio: "inherit" });
      core.info(`Checked out existing branch: ${branchName}`);
    } catch (fetchError) {
      // Branch doesn't exist, create orphan branch
      core.info(`Branch ${branchName} does not exist, creating orphan branch...`);
      execSync(`git checkout --orphan "${branchName}"`, { stdio: "inherit" });
      execSync("git rm -rf . || true", { stdio: "pipe" });
      core.info(`Created orphan branch: ${branchName}`);
    }
  } catch (error) {
    core.setFailed(`Failed to checkout branch: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  // Create destination directory in repo
  const destMemoryPath = path.join(workspaceDir, "memory", memoryId);
  fs.mkdirSync(destMemoryPath, { recursive: true });
  core.info(`Destination directory: ${destMemoryPath}`);

  // Read files from artifact directory and validate before copying
  let filesToCopy = [];
  try {
    const files = fs.readdirSync(sourceMemoryPath, { withFileTypes: true });

    for (const file of files) {
      if (!file.isFile()) {
        continue; // Skip directories
      }

      const fileName = file.name;
      const sourceFilePath = path.join(sourceMemoryPath, fileName);
      const stats = fs.statSync(sourceFilePath);

      // Validate file name patterns if filter is set
      if (fileGlobFilter) {
        const patterns = fileGlobFilter.split(/\s+/).map(pattern => {
          const regexPattern = pattern.replace(/\./g, "\\.").replace(/\*/g, "[^/]*");
          return new RegExp(`^${regexPattern}$`);
        });

        if (!patterns.some(pattern => pattern.test(fileName))) {
          core.error(`File does not match allowed patterns: ${fileName}`);
          core.error(`Allowed patterns: ${fileGlobFilter}`);
          core.setFailed("File pattern validation failed");
          return;
        }
      }

      // Validate file size
      if (stats.size > maxFileSize) {
        core.error(`File exceeds size limit: ${fileName} (${stats.size} bytes > ${maxFileSize} bytes)`);
        core.setFailed("File size validation failed");
        return;
      }

      filesToCopy.push({ name: fileName, source: sourceFilePath, size: stats.size });
    }
  } catch (error) {
    core.setFailed(`Failed to read artifact directory: ${error instanceof Error ? error.message : String(error)}`);
    return;
  }

  // Validate file count
  if (filesToCopy.length > maxFileCount) {
    core.setFailed(`Too many files (${filesToCopy.length} > ${maxFileCount})`);
    return;
  }

  if (filesToCopy.length === 0) {
    core.info("No files to copy from artifact");
    return;
  }

  core.info(`Copying ${filesToCopy.length} validated file(s)...`);

  // Copy files to destination
  for (const file of filesToCopy) {
    const destFilePath = path.join(destMemoryPath, file.name);
    try {
      fs.copyFileSync(file.source, destFilePath);
      core.info(`Copied: ${file.name} (${file.size} bytes)`);
    } catch (error) {
      core.setFailed(`Failed to copy file ${file.name}: ${error instanceof Error ? error.message : String(error)}`);
      return;
    }
  }

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
    core.info("No changes detected after copying files");
    return;
  }

  core.info("Changes detected, committing and pushing...");

  // Stage all changes
  try {
    execSync("git add .", { stdio: "inherit" });
  } catch (error) {
    core.setFailed(`Failed to stage changes: ${error instanceof Error ? error.message : String(error)}`);
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
