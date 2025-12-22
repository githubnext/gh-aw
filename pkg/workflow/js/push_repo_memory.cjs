// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");
const core = require("@actions/core");

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

  /** @param {unknown} value */
  function isPlainObject(value) {
    return typeof value === "object" && value !== null && !Array.isArray(value);
  }

  /** @param {string} absPath */
  function tryParseJSONFile(absPath) {
    const raw = fs.readFileSync(absPath, "utf8");
    if (!raw.trim()) {
      throw new Error(`Empty JSON file: ${absPath}`);
    }
    try {
      return JSON.parse(raw);
    } catch (e) {
      throw new Error(`Invalid JSON in ${absPath}: ${e instanceof Error ? e.message : String(e)}`);
    }
  }

  /** @param {any} obj @param {string} campaignId @param {string} relPath */
  function validateCampaignCursor(obj, campaignId, relPath) {
    if (!isPlainObject(obj)) {
      throw new Error(`Cursor must be a JSON object: ${relPath}`);
    }

    // Cursor payload is intentionally treated as an opaque checkpoint.
    // We only enforce that it is valid JSON and (optionally) self-identifies the campaign.
    if (obj.campaign_id !== undefined) {
      if (typeof obj.campaign_id !== "string" || obj.campaign_id.trim() === "") {
        throw new Error(`Cursor 'campaign_id' must be a non-empty string when present: ${relPath}`);
      }
      if (obj.campaign_id !== campaignId) {
        throw new Error(`Cursor 'campaign_id' must match '${campaignId}' when present: ${relPath}`);
      }
    }

    // Allow optional date metadata if the cursor chooses to include it.
    if (obj.date !== undefined) {
      if (typeof obj.date !== "string" || obj.date.trim() === "") {
        throw new Error(`Cursor 'date' must be a non-empty string (YYYY-MM-DD) when present: ${relPath}`);
      }
      if (!/^\d{4}-\d{2}-\d{2}$/.test(obj.date)) {
        throw new Error(`Cursor 'date' must be YYYY-MM-DD when present: ${relPath}`);
      }
    }
  }

  /** @param {any} obj @param {string} campaignId @param {string} relPath */
  function validateCampaignMetricsSnapshot(obj, campaignId, relPath) {
    if (!isPlainObject(obj)) {
      throw new Error(`Metrics snapshot must be a JSON object: ${relPath}`);
    }
    if (typeof obj.campaign_id !== "string" || obj.campaign_id.trim() === "") {
      throw new Error(`Metrics snapshot must include non-empty 'campaign_id': ${relPath}`);
    }
    if (obj.campaign_id !== campaignId) {
      throw new Error(`Metrics snapshot 'campaign_id' must match '${campaignId}': ${relPath}`);
    }
    if (typeof obj.date !== "string" || obj.date.trim() === "") {
      throw new Error(`Metrics snapshot must include non-empty 'date' (YYYY-MM-DD): ${relPath}`);
    }
    if (!/^\d{4}-\d{2}-\d{2}$/.test(obj.date)) {
      throw new Error(`Metrics snapshot 'date' must be YYYY-MM-DD: ${relPath}`);
    }

    // Require these to be present and non-negative integers (aligns with CampaignMetricsSnapshot).
    const requiredIntFields = ["tasks_total", "tasks_completed"];
    for (const field of requiredIntFields) {
      if (!Number.isInteger(obj[field]) || obj[field] < 0) {
        throw new Error(`Metrics snapshot '${field}' must be a non-negative integer: ${relPath}`);
      }
    }

    // Optional numeric fields, if present.
    const optionalIntFields = ["tasks_in_progress", "tasks_blocked"];
    for (const field of optionalIntFields) {
      if (obj[field] !== undefined && (!Number.isInteger(obj[field]) || obj[field] < 0)) {
        throw new Error(`Metrics snapshot '${field}' must be a non-negative integer when present: ${relPath}`);
      }
    }
    if (obj.velocity_per_day !== undefined && (typeof obj.velocity_per_day !== "number" || obj.velocity_per_day < 0)) {
      throw new Error(`Metrics snapshot 'velocity_per_day' must be a non-negative number when present: ${relPath}`);
    }
    if (obj.estimated_completion !== undefined && typeof obj.estimated_completion !== "string") {
      throw new Error(`Metrics snapshot 'estimated_completion' must be a string when present: ${relPath}`);
    }
  }

  /** @param {string} ch */
  function escapeRegexChar(ch) {
    return ch.replace(/[\\^$+?.()|[\]{}]/g, "\\$&");
  }

  /** @param {string} glob */
  function globToRegExp(glob) {
    // Supports *, **, and ? globs. Matches against posix-style relative paths.
    let re = "^";
    for (let i = 0; i < glob.length; ) {
      const ch = glob[i];
      if (ch === "*") {
        if (glob[i + 1] === "*") {
          re += ".*";
          i += 2;
          continue;
        }
        re += "[^/]*";
        i += 1;
        continue;
      }
      if (ch === "?") {
        re += "[^/]";
        i += 1;
        continue;
      }
      re += escapeRegexChar(ch);
      i += 1;
    }
    re += "$";
    return new RegExp(re);
  }

  /** @param {string} rootDir */
  function listFilesRecursively(rootDir) {
    /** @type {{ relPath: string, absPath: string, size: number }[]} */
    const result = [];

    /** @param {string} currentDir */
    function walk(currentDir) {
      const entries = fs.readdirSync(currentDir, { withFileTypes: true });
      for (const entry of entries) {
        const absPath = path.join(currentDir, entry.name);
        if (entry.isSymbolicLink()) {
          throw new Error(`Symlinks are not allowed in repo-memory: ${absPath}`);
        }
        if (entry.isDirectory()) {
          walk(absPath);
          continue;
        }
        if (!entry.isFile()) {
          continue;
        }
        const relPath = path.posix.relative(rootDir, absPath).split(path.sep).join("/");
        const stats = fs.statSync(absPath);
        result.push({ relPath, absPath, size: stats.size });
      }
    }

    walk(rootDir);
    return result;
  }

  // Validate required environment variables
  if (!artifactDir || !memoryId || !targetRepo || !branchName || !ghToken) {
    core.setFailed("Missing required environment variables: ARTIFACT_DIR, MEMORY_ID, TARGET_REPO, BRANCH_NAME, GH_TOKEN");
    return;
  }

  // Source directory with memory files (artifact location)
  const sourceMemoryPath = path.join(artifactDir, "memory", memoryId);

  // Campaign mode enforcement (agentic campaigns):
  // We treat repo-memory ID "campaigns" with a single file-glob like "<campaign-id>/**" as a strong contract.
  // In this mode, cursor.json and at least one metrics snapshot are required.
  const singlePattern = fileGlobFilter.trim().split(/\s+/).filter(Boolean);
  const campaignPattern = singlePattern.length === 1 ? singlePattern[0] : "";
  const campaignMatch = memoryId === "campaigns" ? /^([^*?]+)\/\*\*$/.exec(campaignPattern) : null;
  const campaignId = campaignMatch ? campaignMatch[1].replace(/\/$/, "") : "";
  const isCampaignMode = Boolean(campaignId);

  // Check if artifact memory directory exists
  if (!fs.existsSync(sourceMemoryPath)) {
    if (isCampaignMode) {
      core.setFailed(`Campaign repo-memory is enabled but no campaign state was written. Expected to find cursor and metrics under: ${sourceMemoryPath}/${campaignId}/`);
      return;
    }
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
  } catch {
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
    } catch {
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
    const files = listFilesRecursively(sourceMemoryPath);
    const patterns = fileGlobFilter ? fileGlobFilter.split(/\s+/).filter(Boolean).map(globToRegExp) : [];

    if (isCampaignMode) {
      const expectedCursorRel = `${campaignId}/cursor.json`;
      const cursorFile = files.find(f => f.relPath === expectedCursorRel);
      if (!cursorFile) {
        core.error(`Missing required campaign cursor file: ${expectedCursorRel}`);
        core.setFailed("Campaign cursor validation failed");
        return;
      }

      const metricsFiles = files.filter(f => f.relPath.startsWith(`${campaignId}/metrics/`) && f.relPath.endsWith(".json"));
      if (metricsFiles.length === 0) {
        core.error(`Missing required campaign metrics snapshots under: ${campaignId}/metrics/*.json`);
        core.setFailed("Campaign metrics validation failed");
        return;
      }
    }

    for (const file of files) {
      // Validate file path patterns if filter is set
      if (patterns.length > 0) {
        if (!patterns.some(pattern => pattern.test(file.relPath))) {
          core.error(`File does not match allowed patterns: ${file.relPath}`);
          core.error(`Allowed patterns: ${fileGlobFilter}`);
          core.setFailed("File pattern validation failed");
          return;
        }
      }

      // Validate file size
      if (file.size > maxFileSize) {
        core.error(`File exceeds size limit: ${file.relPath} (${file.size} bytes > ${maxFileSize} bytes)`);
        core.setFailed("File size validation failed");
        return;
      }

      // Campaign JSON contract checks (only for the campaign subtree).
      if (isCampaignMode && file.relPath.startsWith(`${campaignId}/`)) {
        if (file.relPath === `${campaignId}/cursor.json`) {
          const obj = tryParseJSONFile(file.absPath);
          validateCampaignCursor(obj, campaignId, file.relPath);
        } else if (file.relPath.startsWith(`${campaignId}/metrics/`) && file.relPath.endsWith(".json")) {
          const obj = tryParseJSONFile(file.absPath);
          validateCampaignMetricsSnapshot(obj, campaignId, file.relPath);
        }
      }

      filesToCopy.push({ relPath: file.relPath, source: file.absPath, size: file.size });
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
    const destFilePath = path.join(destMemoryPath, file.relPath);
    try {
      const resolvedRoot = path.resolve(destMemoryPath) + path.sep;
      const resolvedDest = path.resolve(destFilePath);
      if (!resolvedDest.startsWith(resolvedRoot)) {
        core.setFailed(`Refusing to write outside repo-memory directory: ${file.relPath}`);
        return;
      }

      fs.mkdirSync(path.dirname(destFilePath), { recursive: true });
      fs.copyFileSync(file.source, destFilePath);
      core.info(`Copied: ${file.relPath} (${file.size} bytes)`);
    } catch (error) {
      core.setFailed(`Failed to copy file ${file.relPath}: ${error instanceof Error ? error.message : String(error)}`);
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
