const fs = require("fs"),
  path = require("path");
async function main() {
  const workspace = process.env.GITHUB_WORKSPACE,
    workflowFile = process.env.GH_AW_WORKFLOW_FILE;
  if (!workspace) return void core.setFailed("Configuration error: GITHUB_WORKSPACE not available.");
  if (!workflowFile) return void core.setFailed("Configuration error: GH_AW_WORKFLOW_FILE not available.");
  const workflowBasename = path.basename(workflowFile, ".lock.yml"),
    workflowMdFile = path.join(workspace, ".github", "workflows", `${workflowBasename}.md`),
    lockFile = path.join(workspace, ".github", "workflows", workflowFile);
  (core.info("Checking workflow timestamps:"), core.info(`  Source: ${workflowMdFile}`), core.info(`  Lock file: ${lockFile}`));
  let workflowExists = !1,
    lockExists = !1;
  try {
    (fs.accessSync(workflowMdFile, fs.constants.F_OK), (workflowExists = !0));
  } catch (error) {
    core.info(`Source file does not exist: ${workflowMdFile}`);
  }
  try {
    (fs.accessSync(lockFile, fs.constants.F_OK), (lockExists = !0));
  } catch (error) {
    core.info(`Lock file does not exist: ${lockFile}`);
  }
  if (!workflowExists || !lockExists) return void core.info("Skipping timestamp check - one or both files not found");
  const workflowStat = fs.statSync(workflowMdFile),
    lockStat = fs.statSync(lockFile),
    workflowMtime = workflowStat.mtime.getTime(),
    lockMtime = lockStat.mtime.getTime();
  if ((core.info(`  Source modified: ${workflowStat.mtime.toISOString()}`), core.info(`  Lock modified: ${lockStat.mtime.toISOString()}`), workflowMtime > lockMtime)) {
    const warningMessage = `WARNING: Lock file '${lockFile}' is outdated! The workflow file '${workflowMdFile}' has been modified more recently. Run 'gh aw compile' to regenerate the lock file.`;
    core.error(warningMessage);
    const workflowTimestamp = workflowStat.mtime.toISOString(),
      lockTimestamp = lockStat.mtime.toISOString(),
      gitSha = process.env.GITHUB_SHA;
    let summary = core.summary
      .addRaw("### ⚠️ Workflow Lock File Warning\n\n")
      .addRaw("**WARNING**: Lock file is outdated and needs to be regenerated.\n\n")
      .addRaw("**Files:**\n")
      .addRaw(`- Source: \`${workflowMdFile}\` (modified: ${workflowTimestamp})\n`)
      .addRaw(`- Lock: \`${lockFile}\` (modified: ${lockTimestamp})\n\n`);
    (gitSha && (summary = summary.addRaw(`**Git Commit:** \`${gitSha}\`\n\n`)), (summary = summary.addRaw("**Action Required:** Run `gh aw compile` to regenerate the lock file.\n\n")), await summary.write());
  } else core.info("✅ Lock file is up to date");
}
main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
