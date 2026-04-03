// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check for a stale workflow lock file using frontmatter hash comparison.
 * This script verifies that the stored frontmatter hash in the lock file
 * matches the recomputed hash from the source .md file, detecting cases where
 * the workflow was edited without recompiling the lock file. It does not
 * provide tamper protection — use code review to guard against intentional
 * modifications.
 *
 * Supports both same-repo and cross-repo reusable workflow scenarios:
 * - Primary: GitHub API (uses GITHUB_WORKFLOW_REF to identify source repo)
 * - Fallback: local filesystem ($GITHUB_WORKSPACE) when API access is unavailable
 *   (e.g., cross-org reusable workflows where the caller token can't read the source repo)
 */

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");
const { extractHashFromLockFile, computeFrontmatterHash, createGitHubFileReader } = require("./frontmatter_hash_pure.cjs");
const { getFileContent } = require("./github_api_helpers.cjs");
const { ERR_CONFIG } = require("./error_codes.cjs");

async function main() {
  const workflowFile = process.env.GH_AW_WORKFLOW_FILE;

  if (!workflowFile) {
    core.setFailed(`${ERR_CONFIG}: Configuration error: GH_AW_WORKFLOW_FILE not available.`);
    return;
  }

  // Construct file paths
  const workflowBasename = workflowFile.replace(".lock.yml", "");
  const workflowMdPath = `.github/workflows/${workflowBasename}.md`;
  const lockFilePath = `.github/workflows/${workflowFile}`;

  core.info(`Checking for stale lock file using frontmatter hash:`);
  core.info(`  Source: ${workflowMdPath}`);
  core.info(`  Lock file: ${lockFilePath}`);

  // Determine workflow source repository from GITHUB_WORKFLOW_REF for cross-repo support.
  // GITHUB_WORKFLOW_REF format: owner/repo/.github/workflows/file.yml@ref
  // This env var always reflects the repo where the workflow file is defined,
  // not the repo where the triggering event occurred (context.repo).
  // When running cross-repo via org rulesets, context.repo points to the target
  // repository, not the repository that defines the workflow files.
  const workflowEnvRef = process.env.GITHUB_WORKFLOW_REF || "";
  const currentRepo = process.env.GITHUB_REPOSITORY || `${context.repo.owner}/${context.repo.repo}`;

  // Parse owner, repo, and optional ref from GITHUB_WORKFLOW_REF as a single unit so that
  // repo and ref are always consistent with each other.  The @ref segment may be absent (e.g.
  // when the env var was set without a ref suffix), so treat it as optional.
  const workflowRefMatch = workflowEnvRef.match(/^([^/]+)\/([^/]+)\/.+?(?:@(.+))?$/);

  // Use the workflow source repo if parseable, otherwise fall back to context.repo
  const owner = workflowRefMatch ? workflowRefMatch[1] : context.repo.owner;
  const repo = workflowRefMatch ? workflowRefMatch[2] : context.repo.repo;
  const workflowRepo = `${owner}/${repo}`;

  // Determine ref in a way that keeps repo+ref consistent:
  //   - If a ref is present in GITHUB_WORKFLOW_REF, use it.
  //   - For same-repo runs without a parsed ref, fall back to context.sha (existing behavior).
  //   - For cross-repo runs without a parsed ref, omit ref so the API uses the default branch
  //     (avoids mixing source repo owner/name with a SHA that only exists in the triggering repo).
  let ref;
  if (workflowRefMatch && workflowRefMatch[3]) {
    ref = workflowRefMatch[3];
  } else if (workflowRepo === currentRepo) {
    ref = context.sha;
  } else {
    ref = undefined;
  }

  core.info(`GITHUB_WORKFLOW_REF: ${workflowEnvRef || "(not set)"}`);
  core.info(`GITHUB_REPOSITORY: ${currentRepo}`);
  core.info(`Resolved source repo: ${owner}/${repo} @ ${ref || "(default branch)"}`);

  if (workflowRepo !== currentRepo) {
    core.info(`Cross-repo invocation detected: workflow source is "${workflowRepo}", current repo is "${currentRepo}"`);
  } else {
    core.info(`Same-repo invocation: checking out ${workflowRepo} @ ${ref}`);
  }

  // Fallback: compare frontmatter hashes using local filesystem files.
  // Used when the GitHub API is inaccessible (e.g., cross-org reusable workflow where
  // the caller's GITHUB_TOKEN cannot read the source repo).
  // The activation job's "Checkout .github and .agents folders" step always runs before
  // this check and places the workflow source files in $GITHUB_WORKSPACE, so the local
  // files are always available at this point.
  async function compareFrontmatterHashesFromLocalFiles() {
    const workspace = process.env.GITHUB_WORKSPACE;
    if (!workspace) {
      core.info("GITHUB_WORKSPACE not available for local filesystem fallback");
      return null;
    }

    // Resolve and validate both paths to prevent path traversal attacks.
    // GH_AW_WORKFLOW_FILE could theoretically contain "../" segments; reject any
    // resolved path that escapes the workspace/.github/workflows directory.
    const allowedDir = path.resolve(workspace, ".github", "workflows");
    const localLockFilePath = path.resolve(workspace, lockFilePath);
    const localMdFilePath = path.resolve(workspace, workflowMdPath);

    if (!localLockFilePath.startsWith(allowedDir + path.sep) && localLockFilePath !== allowedDir) {
      core.info(`Resolved lock file path escapes workspace: ${localLockFilePath}`);
      return null;
    }
    if (!localMdFilePath.startsWith(allowedDir + path.sep) && localMdFilePath !== allowedDir) {
      core.info(`Resolved source file path escapes workspace: ${localMdFilePath}`);
      return null;
    }

    core.info(`Attempting local filesystem fallback for hash comparison:`);
    core.info(`  Lock file: ${localLockFilePath}`);
    core.info(`  Source: ${localMdFilePath}`);

    if (!fs.existsSync(localLockFilePath)) {
      core.info(`Local lock file not found: ${localLockFilePath}`);
      return null;
    }

    if (!fs.existsSync(localMdFilePath)) {
      core.info(`Local source file not found: ${localMdFilePath}`);
      return null;
    }

    try {
      const localLockContent = fs.readFileSync(localLockFilePath, "utf8");
      const storedHash = extractHashFromLockFile(localLockContent);
      if (!storedHash) {
        core.info("No frontmatter hash found in local lock file");
        return null;
      }

      // computeFrontmatterHash uses the local filesystem reader by default
      const recomputedHash = await computeFrontmatterHash(localMdFilePath);

      const match = storedHash === recomputedHash;

      core.info(`Frontmatter hash comparison (local filesystem fallback):`);
      core.info(`  Lock file hash:    ${storedHash}`);
      core.info(`  Recomputed hash:   ${recomputedHash}`);
      core.info(`  Status: ${match ? "✅ Hashes match" : "⚠️  Hashes differ"}`);

      return { match, storedHash, recomputedHash };
    } catch (error) {
      core.info(`Could not compute frontmatter hash from local files: ${getErrorMessage(error)}`);
      return null;
    }
  }

  // Primary: compare frontmatter hashes using the GitHub API.
  // Falls back to local filesystem if the API is inaccessible.
  async function compareFrontmatterHashes() {
    try {
      // Fetch lock file content to extract stored hash
      const lockFileContent = await getFileContent(github, owner, repo, lockFilePath, ref);
      if (!lockFileContent) {
        core.info("Unable to fetch lock file content for hash comparison via API, trying local filesystem fallback");
        return await compareFrontmatterHashesFromLocalFiles();
      }

      const storedHash = extractHashFromLockFile(lockFileContent);
      if (!storedHash) {
        core.info("No frontmatter hash found in lock file");
        return null;
      }

      // Compute hash using pure JavaScript implementation
      // Create a GitHub file reader for fetching workflow files via API
      const fileReader = createGitHubFileReader(github, owner, repo, ref);
      const recomputedHash = await computeFrontmatterHash(workflowMdPath, { fileReader });

      const match = storedHash === recomputedHash;

      // Log hash comparison
      core.info(`Frontmatter hash comparison:`);
      core.info(`  Lock file hash:    ${storedHash}`);
      core.info(`  Recomputed hash:   ${recomputedHash}`);
      core.info(`  Status: ${match ? "✅ Hashes match" : "⚠️  Hashes differ"}`);

      return { match, storedHash, recomputedHash };
    } catch (error) {
      const errorMessage = getErrorMessage(error);
      core.info(`Could not compute frontmatter hash via API: ${errorMessage}`);
      // Fall back to local filesystem when API is unavailable
      // (e.g., cross-org reusable workflow where caller token lacks source repo access)
      return await compareFrontmatterHashesFromLocalFiles();
    }
  }

  const hashComparison = await compareFrontmatterHashes();

  if (!hashComparison) {
    // Could not compute hash - be conservative and fail
    core.warning("Could not compare frontmatter hashes - assuming lock file is outdated");
    const warningMessage = `Lock file '${lockFilePath}' is outdated or unverifiable! Could not verify frontmatter hash for '${workflowMdPath}'. Run 'gh aw compile' to regenerate the lock file.`;

    let summary = core.summary
      .addRaw("### ⚠️ Workflow Lock File Warning\n\n")
      .addRaw("**WARNING**: Could not verify whether lock file is up to date. Frontmatter hash check failed.\n\n")
      .addRaw("**Files:**\n")
      .addRaw(`- Source: \`${workflowMdPath}\`\n`)
      .addRaw(`- Lock: \`${lockFilePath}\`\n\n`)
      .addRaw("**Action Required:** Run `gh aw compile` to regenerate the lock file.\n\n");

    await summary.write();

    core.setFailed(`${ERR_CONFIG}: ${warningMessage}`);
  } else if (hashComparison.match) {
    // Hashes match - lock file is up to date
    core.info("✅ Lock file is up to date (hashes match)");
  } else {
    // Hashes differ - lock file needs recompilation
    const warningMessage = `Lock file '${lockFilePath}' is outdated! The workflow file '${workflowMdPath}' frontmatter has changed. Run 'gh aw compile' to regenerate the lock file.`;

    let summary = core.summary
      .addRaw("### ⚠️ Workflow Lock File Warning\n\n")
      .addRaw("**WARNING**: Lock file is outdated (frontmatter hash mismatch).\n\n")
      .addRaw("**Files:**\n")
      .addRaw(`- Source: \`${workflowMdPath}\`\n`)
      .addRaw(`  - Frontmatter hash: \`${hashComparison.recomputedHash.substring(0, 12)}...\`\n`)
      .addRaw(`- Lock: \`${lockFilePath}\`\n`)
      .addRaw(`  - Stored hash: \`${hashComparison.storedHash.substring(0, 12)}...\`\n\n`)
      .addRaw("**Action Required:** Run `gh aw compile` to regenerate the lock file.\n\n");

    await summary.write();

    // Fail the step to prevent workflow from running with outdated configuration
    core.setFailed(`${ERR_CONFIG}: ${warningMessage}`);
  }
}

module.exports = { main };
