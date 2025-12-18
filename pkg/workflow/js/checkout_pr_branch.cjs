async function main() {
  const eventName = context.eventName,
    pullRequest = context.payload.pull_request;
  if (pullRequest) {
    (core.info(`Event: ${eventName}`), core.info(`Pull Request #${pullRequest.number}`));
    try {
      if ("pull_request" === eventName) {
        const branchName = pullRequest.head.ref;
        (core.info(`Checking out PR branch: ${branchName}`), await exec.exec("git", ["fetch", "origin", branchName]), await exec.exec("git", ["checkout", branchName]), core.info(`✅ Successfully checked out branch: ${branchName}`));
      } else {
        const prNumber = pullRequest.number;
        (core.info(`Checking out PR #${prNumber} using gh pr checkout`), await exec.exec("gh", ["pr", "checkout", prNumber.toString()]), core.info(`✅ Successfully checked out PR #${prNumber}`));
      }
    } catch (error) {
      core.setFailed(`Failed to checkout PR branch: ${error instanceof Error ? error.message : String(error)}`);
    }
  } else core.info("No pull request context available, skipping checkout");
}
main().catch(error => {
  core.setFailed(error instanceof Error ? error.message : String(error));
});
