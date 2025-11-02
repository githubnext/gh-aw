// @ts-check
/// <reference types="@actions/github-script" />

/**
 * Check workflow file timestamps to detect outdated lock files
 * This script compares the modification time of the source .md file
 * with the compiled .lock.yml file and warns if recompilation is needed
 */

const fs = require("fs");
const path = require("path");

async function main() {
  const workspace = process.env.GITHUB_WORKSPACE;
  const workflowFile = process.env.GH_AW_WORKFLOW_FILE;

  if (!workspace) {
    core.setFailed("Configuration error: GITHUB_WORKSPACE not available.");
    return;
  }

  if (!workflowFile) {
    core.setFailed("Configuration error: GH_AW_WORKFLOW_FILE not available.");
    return;
  }

  // Construct file paths
  const workflowBasename = path.basename(workflowFile, ".lock.yml");
  const workflowMdFile = path.join(workspace, ".github", "workflows", `${workflowBasename}.md`);
  const lockFile = path.join(workspace, ".github", "workflows", workflowFile);

  core.info(`Checking workflow timestamps:`);
  core.info(`  Source: ${workflowMdFile}`);
  core.info(`  Lock file: ${lockFile}`);

  // Check if both files exist
  let workflowExists = false;
  let lockExists = false;

  try {
    fs.accessSync(workflowMdFile, fs.constants.F_OK);
    workflowExists = true;
  } catch (error) {
    core.info(`Source file does not exist: ${workflowMdFile}`);
  }

  try {
    fs.accessSync(lockFile, fs.constants.F_OK);
    lockExists = true;
  } catch (error) {
    core.info(`Lock file does not exist: ${lockFile}`);
  }

  if (!workflowExists || !lockExists) {
    core.info("Skipping timestamp check - one or both files not found");
    return;
  }

  // Get file stats to compare modification times
  const workflowStat = fs.statSync(workflowMdFile);
  const lockStat = fs.statSync(lockFile);

  const workflowMtime = workflowStat.mtime.getTime();
  const lockMtime = lockStat.mtime.getTime();

  core.info(`  Source modified: ${workflowStat.mtime.toISOString()}`);
  core.info(`  Lock modified: ${lockStat.mtime.toISOString()}`);

  // Check if workflow file is newer than lock file
  if (workflowMtime > lockMtime) {
    const warningMessage = `🔴🔴🔴 WARNING: Lock file '${lockFile}' is outdated! The workflow file '${workflowMdFile}' has been modified more recently. Run 'gh aw compile' to regenerate the lock file.`;

    core.error(warningMessage);

    // Add summary to GitHub Step Summary
    await core.summary
      .addRaw("## ⚠️ Workflow Lock File Warning\n\n")
      .addRaw(`🔴🔴🔴 **WARNING**: Lock file \`${lockFile}\` is outdated!\n\n`)
      .addRaw(`The workflow file \`${workflowMdFile}\` has been modified more recently.\n\n`)
      .addRaw("Run `gh aw compile` to regenerate the lock file.\n\n")
      .write();
  } else {
    core.info("✅ Lock file is up to date");
  }
}

main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
