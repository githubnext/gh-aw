async function main() {
  const workflowFile = process.env.GH_AW_WORKFLOW_FILE;
  if (!workflowFile) return void core.setFailed("Configuration error: GH_AW_WORKFLOW_FILE not available.");
  const workflowMdPath = `.github/workflows/${workflowFile.replace(".lock.yml", "")}.md`,
    lockFilePath = `.github/workflows/${workflowFile}`;
  (core.info("Checking workflow timestamps using GitHub API:"), core.info(`  Source: ${workflowMdPath}`), core.info(`  Lock file: ${lockFilePath}`));
  const { owner, repo } = context.repo,
    ref = context.sha;
  async function getLastCommitForFile(path) {
    try {
      const response = await github.rest.repos.listCommits({ owner, repo, path, per_page: 1, sha: ref });
      if (response.data && response.data.length > 0) {
        const commit = response.data[0];
        return { sha: commit.sha, date: commit.commit.committer.date, message: commit.commit.message };
      }
      return null;
    } catch (error) {
      return (core.info(`Could not fetch commit for ${path}: ${error.message}`), null);
    }
  }
  const workflowCommit = await getLastCommitForFile(workflowMdPath),
    lockCommit = await getLastCommitForFile(lockFilePath);
  if ((workflowCommit || core.info(`Source file does not exist: ${workflowMdPath}`), lockCommit || core.info(`Lock file does not exist: ${lockFilePath}`), !workflowCommit || !lockCommit))
    return void core.info("Skipping timestamp check - one or both files not found");
  const workflowDate = new Date(workflowCommit.date),
    lockDate = new Date(lockCommit.date);
  if ((core.info(`  Source last commit: ${workflowDate.toISOString()} (${workflowCommit.sha.substring(0, 7)})`), core.info(`  Lock last commit: ${lockDate.toISOString()} (${lockCommit.sha.substring(0, 7)})`), workflowDate > lockDate)) {
    const warningMessage = `WARNING: Lock file '${lockFilePath}' is outdated! The workflow file '${workflowMdPath}' has been modified more recently. Run 'gh aw compile' to regenerate the lock file.`;
    core.error(warningMessage);
    const workflowTimestamp = workflowDate.toISOString(),
      lockTimestamp = lockDate.toISOString();
    let summary = core.summary
      .addRaw("### ⚠️ Workflow Lock File Warning\n\n")
      .addRaw("**WARNING**: Lock file is outdated and needs to be regenerated.\n\n")
      .addRaw("**Files:**\n")
      .addRaw(`- Source: \`${workflowMdPath}\`\n`)
      .addRaw(`  - Last commit: ${workflowTimestamp}\n`)
      .addRaw(`  - Commit SHA: [\`${workflowCommit.sha.substring(0, 7)}\`](https://github.com/${owner}/${repo}/commit/${workflowCommit.sha})\n`)
      .addRaw(`- Lock: \`${lockFilePath}\`\n`)
      .addRaw(`  - Last commit: ${lockTimestamp}\n`)
      .addRaw(`  - Commit SHA: [\`${lockCommit.sha.substring(0, 7)}\`](https://github.com/${owner}/${repo}/commit/${lockCommit.sha})\n\n`)
      .addRaw("**Action Required:** Run `gh aw compile` to regenerate the lock file.\n\n");
    await summary.write();
  } else workflowCommit.sha === lockCommit.sha ? core.info("✅ Lock file is up to date (same commit)") : core.info("✅ Lock file is up to date");
}
main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
