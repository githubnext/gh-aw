// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check workflow lock file integrity using frontmatter hash validation.
 * This script verifies that the stored frontmatter hash in the lock file
 * matches the recomputed hash from the source .md file, regardless of
 * commit timestamps.
 */

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

  core.info(`Checking workflow lock file integrity using frontmatter hash:`);
  core.info(`  Source: ${workflowMdPath}`);
  core.info(`  Lock file: ${lockFilePath}`);

  const { owner, repo } = context.repo;
  const ref = context.sha;

  // Helper function to compute and compare frontmatter hashes
  // Returns: { match: boolean, storedHash: string, recomputedHash: string } or null on error
  async function compareFrontmatterHashes() {
    try {
      // Fetch lock file content to extract stored hash
      const lockFileContent = await getFileContent(github, owner, repo, lockFilePath, ref);
      if (!lockFileContent) {
        core.info("Unable to fetch lock file content for hash comparison");
        return null;
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
      core.info(`Could not compute frontmatter hash: ${errorMessage}`);
      return null;
    }
  }

  const hashComparison = await compareFrontmatterHashes();

  if (!hashComparison) {
    // Could not compute hash - be conservative and fail
    core.warning("Could not compare frontmatter hashes - assuming lock file is outdated");
    const warningMessage = `Lock file '${lockFilePath}' integrity check failed! Could not verify frontmatter hash for '${workflowMdPath}'. Run 'gh aw compile' to regenerate the lock file.`;

    let summary = core.summary
      .addRaw("### ⚠️ Workflow Lock File Warning\n\n")
      .addRaw("**WARNING**: Lock file integrity check failed. Could not verify frontmatter hash.\n\n")
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
