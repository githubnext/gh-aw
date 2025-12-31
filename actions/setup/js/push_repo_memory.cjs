// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const path = require("path");
const { execSync } = require("child_process");
const { getErrorMessage } = require("./error_helpers.cjs");
const { globPatternToRegex } = require("./glob_pattern_helpers.cjs");

/**
 * Push repo-memory changes to git branch
 * Environment variables:
 *   ARTIFACT_DIR: Path to the downloaded artifact directory containing memory files
 *   MEMORY_ID: Memory identifier (used for subdirectory path)
 *   TARGET_REPO: Target repository (owner/name)
 *   BRANCH_NAME: Branch name to push to
 *   MAX_FILE_SIZE: Maximum file size in bytes
 *   MAX_FILE_COUNT: Maximum number of files per commit
 *   FILE_GLOB_FILTER: Optional space-separated list of file patterns (e.g., "*.md metrics/** data/**")
 *                     Supports * (matches any chars except /) and ** (matches any chars including /)
 *
 *                     IMPORTANT: Patterns are matched against the RELATIVE FILE PATH from the artifact directory,
 *                     NOT against the branch path. Do NOT include the branch name in the patterns.
 *
 *                     Example:
 *                       BRANCH_NAME: memory/code-metrics
 *                       Artifact file: /tmp/gh-aw/repo-memory/default/history.jsonl
 *                       Relative path tested: "history.jsonl"
 *                       CORRECT pattern: "*.jsonl"
 *                       INCORRECT pattern: "memory/code-metrics/*.jsonl"  (includes branch name)
 *
 *                     The branch name is used for git operations (checkout, push) but not for pattern matching.
 *   GH_AW_CAMPAIGN_ID: Optional campaign ID override. When set with MEMORY_ID=campaigns,
 *                      enforces all FILE_GLOB_FILTER patterns are under <campaign-id>/...
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

  // Log environment variable configuration for debugging
  core.info("Environment configuration:");
  core.info(`  MEMORY_ID: ${memoryId}`);
  core.info(`  MAX_FILE_SIZE: ${maxFileSize}`);
  core.info(`  MAX_FILE_COUNT: ${maxFileCount}`);
  core.info(`  FILE_GLOB_FILTER: ${fileGlobFilter ? `"${fileGlobFilter}"` : "(empty - all files accepted)"}`);
  core.info(`  FILE_GLOB_FILTER length: ${fileGlobFilter.length}`);

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

  // ============================================================================
  // CAMPAIGN-SPECIFIC VALIDATION FUNCTIONS
  // ============================================================================
  // The following functions implement validation for the campaign convention:
  // When memoryId is "campaigns" and file-glob matches "<campaign-id>/**",
  // enforce specific JSON schemas for cursor.json and metrics/*.json files.
  //
  // This is a domain-specific convention used by Campaign Workflows to maintain
  // durable state in repo-memory. See docs/guides/campaigns/ for details.
  // ============================================================================

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
      const value = obj[field];
      if (value === null || value === undefined) {
        throw new Error(`Metrics snapshot '${field}' is required but was ${value === null ? "null" : "undefined"}: ${relPath}`);
      }
      if (typeof value !== "number") {
        throw new Error(`Metrics snapshot '${field}' must be a number, got ${typeof value} (value: ${JSON.stringify(value)}): ${relPath}`);
      }
      if (!Number.isInteger(value)) {
        throw new Error(`Metrics snapshot '${field}' must be an integer, got ${value}: ${relPath}`);
      }
      if (value < 0) {
        throw new Error(`Metrics snapshot '${field}' must be non-negative, got ${value}: ${relPath}`);
      }
    }

    // Optional numeric fields, if present.
    const optionalIntFields = ["tasks_in_progress", "tasks_blocked"];
    for (const field of optionalIntFields) {
      const value = obj[field];
      if (value !== undefined && value !== null) {
        if (typeof value !== "number") {
          throw new Error(`Metrics snapshot '${field}' must be a number when present, got ${typeof value} (value: ${JSON.stringify(value)}): ${relPath}`);
        }
        if (!Number.isInteger(value)) {
          throw new Error(`Metrics snapshot '${field}' must be an integer when present, got ${value}: ${relPath}`);
        }
        if (value < 0) {
          throw new Error(`Metrics snapshot '${field}' must be non-negative when present, got ${value}: ${relPath}`);
        }
      }
    }
    if (obj.velocity_per_day !== undefined && obj.velocity_per_day !== null) {
      const value = obj.velocity_per_day;
      if (typeof value !== "number") {
        throw new Error(`Metrics snapshot 'velocity_per_day' must be a number when present, got ${typeof value} (value: ${JSON.stringify(value)}): ${relPath}`);
      }
      if (value < 0) {
        throw new Error(`Metrics snapshot 'velocity_per_day' must be non-negative when present, got ${value}: ${relPath}`);
      }
    }
    if (obj.estimated_completion !== undefined && obj.estimated_completion !== null) {
      const value = obj.estimated_completion;
      if (typeof value !== "string") {
        throw new Error(`Metrics snapshot 'estimated_completion' must be a string when present, got ${typeof value} (value: ${JSON.stringify(value)}): ${relPath}`);
      }
    }
  }

  // Validate required environment variables
  if (!artifactDir || !memoryId || !targetRepo || !branchName || !ghToken) {
    core.setFailed("Missing required environment variables: ARTIFACT_DIR, MEMORY_ID, TARGET_REPO, BRANCH_NAME, GH_TOKEN");
    return;
  }

  // Source directory with memory files (artifact location)
  // The artifactDir IS the memory directory (no nested structure needed)
  const sourceMemoryPath = artifactDir;

  // ============================================================================
  // CAMPAIGN MODE DETECTION
  // ============================================================================
  // Campaign Workflows use a convention-based pattern in repo-memory:
  //   - memoryId: "campaigns"
  //   - file-glob: one or more patterns like "<campaign-id>/**" or "<campaign-id>/metrics/**"
  //   - Optional: GH_AW_CAMPAIGN_ID environment variable to explicitly set campaign ID
  //
  // When this pattern is detected, we enforce campaign-specific validation:
  //   1. All patterns must be under "<campaign-id>/..." subdirectory
  //   2. cursor.json must exist and follow the cursor schema
  //   3. At least one metrics/*.json file must exist and follow the metrics schema
  //
  // This ensures campaigns maintain durable state consistency across workflow runs.
  // Non-campaign repo-memory configurations bypass this validation entirely.
  // ============================================================================

  // Allow explicit campaign ID override via environment variable
  const explicitCampaignId = process.env.GH_AW_CAMPAIGN_ID || "";

  // Parse file glob patterns (can be space-separated)
  const patterns = fileGlobFilter.trim().split(/\s+/).filter(Boolean);

  // Determine campaign ID from patterns or explicit override
  let campaignId = explicitCampaignId;

  // If no explicit campaign ID, try to extract from patterns when memoryId is "campaigns"
  if (!campaignId && memoryId === "campaigns" && patterns.length > 0) {
    // Try to extract campaign ID from first pattern matching "<campaign-id>/**"
    // This only works for simple patterns without wildcards in the campaign ID portion
    // For patterns like "campaign-id-*/**", use GH_AW_CAMPAIGN_ID environment variable
    const campaignMatch = /^([^*?/]+)\/\*\*/.exec(patterns[0]);
    if (campaignMatch) {
      campaignId = campaignMatch[1];
    }
  }

  const isCampaignMode = Boolean(campaignId);

  // Validate all patterns are under campaign-id when in campaign mode
  if (isCampaignMode && patterns.length > 0) {
    for (const pattern of patterns) {
      if (!pattern.startsWith(`${campaignId}/`)) {
        core.setFailed(`Campaign mode requires all file patterns to be under '${campaignId}/' subdirectory. Invalid pattern: ${pattern}`);
        return;
      }
    }
  }

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
    core.setFailed(`Failed to checkout branch: ${getErrorMessage(error)}`);
    return;
  }

  // Create destination directory in repo
  // Files are copied to the root of the checked-out branch (workspaceDir)
  // The branch name (e.g., "memory/campaigns") identifies the branch,
  // but files go at the branch root, not in a nested subdirectory
  const destMemoryPath = workspaceDir;
  core.info(`Destination directory: ${destMemoryPath}`);

  // Recursively scan and collect files from artifact directory
  let filesToCopy = [];
  // Track campaign files for validation
  let campaignCursorFound = false;
  let campaignMetricsCount = 0;

  // Log the file glob filter configuration
  if (fileGlobFilter) {
    core.info(`File glob filter enabled: ${fileGlobFilter}`);
    const patternCount = fileGlobFilter.trim().split(/\s+/).filter(Boolean).length;
    core.info(`Number of patterns: ${patternCount}`);
  } else {
    core.info("No file glob filter - all files will be accepted");
  }

  /**
   * Recursively scan directory and collect files
   * @param {string} dirPath - Directory to scan
   * @param {string} relativePath - Relative path from sourceMemoryPath (for nested files)
   */
  function scanDirectory(dirPath, relativePath = "") {
    const entries = fs.readdirSync(dirPath, { withFileTypes: true });

    for (const entry of entries) {
      const fullPath = path.join(dirPath, entry.name);
      const relativeFilePath = relativePath ? path.join(relativePath, entry.name) : entry.name;

      if (entry.isDirectory()) {
        // Recursively scan subdirectory
        scanDirectory(fullPath, relativeFilePath);
      } else if (entry.isFile()) {
        const stats = fs.statSync(fullPath);

        // Validate file name patterns if filter is set
        if (fileGlobFilter) {
          const patterns = fileGlobFilter.trim().split(/\s+/).filter(Boolean).map(globPatternToRegex);

          // Test patterns against the relative file path within the memory directory
          // Patterns are specified relative to the memory artifact directory, not the branch path
          const normalizedRelPath = relativeFilePath.replace(/\\/g, "/");

          // Enhanced logging: Show what we're testing (use info for first file to aid debugging)
          core.debug(`Testing file: ${normalizedRelPath}`);
          core.debug(`File glob filter: ${fileGlobFilter}`);
          core.debug(`Number of patterns: ${patterns.length}`);

          const matchResults = patterns.map((pattern, idx) => {
            const matches = pattern.test(normalizedRelPath);
            const patternStr = fileGlobFilter.trim().split(/\s+/).filter(Boolean)[idx];
            core.debug(`  Pattern ${idx + 1}: "${patternStr}" -> ${pattern.source} -> ${matches ? "✓ MATCH" : "✗ NO MATCH"}`);
            return matches;
          });

          if (!matchResults.some(m => m)) {
            // Enhanced warning with more context about the filtering issue
            core.warning(`Skipping file that does not match allowed patterns: ${normalizedRelPath}`);
            core.info(`  File path being tested (relative to artifact): ${normalizedRelPath}`);
            core.info(`  Configured patterns: ${fileGlobFilter}`);
            const patternStrs = fileGlobFilter.trim().split(/\s+/).filter(Boolean);
            patterns.forEach((pattern, idx) => {
              core.info(`    Pattern: "${patternStrs[idx]}" -> Regex: ${pattern.source} -> ${matchResults[idx] ? "✅ MATCH" : "❌ NO MATCH"}`);
            });
            core.info(`  Note: Patterns are matched against the full relative file path from the artifact directory.`);
            core.info(`  If patterns include directory prefixes (like 'branch-name/'), ensure files are organized that way in the artifact.`);
            // Skip this file instead of failing - it may be from a previous run with different patterns
            return;
          }
        }

        // Validate file size
        if (stats.size > maxFileSize) {
          core.error(`File exceeds size limit: ${relativeFilePath} (${stats.size} bytes > ${maxFileSize} bytes)`);
          core.setFailed("File size validation failed");
          throw new Error("File size validation failed");
        }

        // Campaign-specific JSON validation (only when campaign mode is active)
        // This enforces the campaign state file schemas for cursor and metrics
        if (isCampaignMode && relativeFilePath.startsWith(`${campaignId}/`)) {
          if (relativeFilePath === `${campaignId}/cursor.json`) {
            const obj = tryParseJSONFile(fullPath);
            validateCampaignCursor(obj, campaignId, relativeFilePath);
            campaignCursorFound = true;
          } else if (relativeFilePath.startsWith(`${campaignId}/metrics/`) && relativeFilePath.endsWith(".json")) {
            const obj = tryParseJSONFile(fullPath);
            validateCampaignMetricsSnapshot(obj, campaignId, relativeFilePath);
            campaignMetricsCount++;
          }
        }

        filesToCopy.push({
          relativePath: relativeFilePath,
          source: fullPath,
          size: stats.size,
        });
      }
    }
  }

  try {
    scanDirectory(sourceMemoryPath);
    core.info(`Scan complete: Found ${filesToCopy.length} file(s) to copy`);
    if (filesToCopy.length > 0 && filesToCopy.length <= 10) {
      core.info("Files found:");
      filesToCopy.forEach(f => core.info(`  - ${f.relativePath} (${f.size} bytes)`));
    } else if (filesToCopy.length > 10) {
      core.info(`First 10 files:`);
      filesToCopy.slice(0, 10).forEach(f => core.info(`  - ${f.relativePath} (${f.size} bytes)`));
      core.info(`  ... and ${filesToCopy.length - 10} more`);
    }
  } catch (error) {
    core.setFailed(`Failed to scan artifact directory: ${getErrorMessage(error)}`);
    return;
  }

  // Campaign mode validation: ensure required state files were found
  // This enforcement is only active when campaign mode is detected
  if (isCampaignMode) {
    if (!campaignCursorFound) {
      core.error(`Missing required campaign cursor file: ${campaignId}/cursor.json`);
      core.setFailed("Campaign cursor validation failed");
      return;
    }

    if (campaignMetricsCount === 0) {
      core.error(`Missing required campaign metrics snapshots under: ${campaignId}/metrics/*.json`);
      core.setFailed("Campaign metrics validation failed");
      return;
    }
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

  // Copy files to destination (preserving directory structure)
  for (const file of filesToCopy) {
    const destFilePath = path.join(destMemoryPath, file.relativePath);
    const destDir = path.dirname(destFilePath);

    try {
      // Path traversal protection
      const resolvedRoot = path.resolve(destMemoryPath) + path.sep;
      const resolvedDest = path.resolve(destFilePath);
      if (!resolvedDest.startsWith(resolvedRoot)) {
        core.setFailed(`Refusing to write outside repo-memory directory: ${file.relativePath}`);
        return;
      }

      // Ensure destination directory exists
      fs.mkdirSync(destDir, { recursive: true });

      // Copy file
      fs.copyFileSync(file.source, destFilePath);
      core.info(`Copied: ${file.relativePath} (${file.size} bytes)`);
    } catch (error) {
      core.setFailed(`Failed to copy file ${file.relativePath}: ${getErrorMessage(error)}`);
      return;
    }
  }

  // Check if we have any changes to commit
  let hasChanges = false;
  try {
    const status = execSync("git status --porcelain", { encoding: "utf8" });
    hasChanges = status.trim().length > 0;
  } catch (error) {
    core.setFailed(`Failed to check git status: ${getErrorMessage(error)}`);
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
    core.setFailed(`Failed to stage changes: ${getErrorMessage(error)}`);
    return;
  }

  // Commit changes
  try {
    execSync(`git commit -m "Update repo memory from workflow run ${githubRunId}"`, { stdio: "inherit" });
  } catch (error) {
    core.setFailed(`Failed to commit changes: ${getErrorMessage(error)}`);
    return;
  }

  // Pull with merge strategy (ours wins on conflicts)
  core.info(`Pulling latest changes from ${branchName}...`);
  try {
    const repoUrl = `https://x-access-token:${ghToken}@github.com/${targetRepo}.git`;
    execSync(`git pull --no-rebase -X ours "${repoUrl}" "${branchName}"`, { stdio: "inherit" });
  } catch (error) {
    // Pull might fail if branch doesn't exist yet or on conflicts - this is acceptable
    core.warning(`Pull failed (this may be expected): ${getErrorMessage(error)}`);
  }

  // Push changes
  core.info(`Pushing changes to ${branchName}...`);
  try {
    const repoUrl = `https://x-access-token:${ghToken}@github.com/${targetRepo}.git`;
    execSync(`git push "${repoUrl}" HEAD:"${branchName}"`, { stdio: "inherit" });
    core.info(`Successfully pushed changes to ${branchName} branch`);
  } catch (error) {
    core.setFailed(`Failed to push changes: ${getErrorMessage(error)}`);
    return;
  }
}

module.exports = { main };
