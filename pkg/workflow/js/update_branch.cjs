/** @type {typeof import("@actions/exec")} */
const exec = require("@actions/exec");

async function main() {
  try {
    core.info("Starting Update Branch workflow");

    // Get current branch name
    let currentBranch = "";
    const currentBranchRes = await exec.exec("git", ["branch", "--show-current"], {
      listeners: { stdout: data => (currentBranch += data.toString()) },
      silent: true,
    });
    if (currentBranchRes !== 0) {
      core.setFailed("Failed to get current branch name");
      return;
    }
    currentBranch = currentBranch.trim();
    core.info(`Current branch: ${currentBranch}`);

    // Get pull request information for the current branch
    let prInfo = "";
    try {
      const prInfoRes = await exec.exec(
        "gh",
        ["pr", "view", currentBranch, "--json", "baseRefName,number,title", "--jq", "{base: .baseRefName, number: .number, title: .title}"],
        {
          listeners: { stdout: data => (prInfo += data.toString()) },
          silent: true,
        }
      );

      if (prInfoRes !== 0) {
        core.setFailed(`No pull request found for branch ${currentBranch}`);
        return;
      }
    } catch (error) {
      core.setFailed(`Failed to get pull request information: ${error instanceof Error ? error.message : String(error)}`);
      return;
    }

    let prData;
    try {
      prData = JSON.parse(prInfo.trim());
    } catch (error) {
      core.setFailed(`Failed to parse pull request information: ${error instanceof Error ? error.message : String(error)}`);
      return;
    }

    const baseBranch = prData.base;
    const prNumber = prData.number;
    const prTitle = prData.title;

    core.info(`Pull Request #${prNumber}: ${prTitle}`);
    core.info(`Base branch: ${baseBranch}`);

    // Configure Git with GitHub Actions identity
    core.info("Configuring Git credentials");
    await exec.exec("git", ["config", "user.email", "github-actions[bot]@users.noreply.github.com"]);
    await exec.exec("git", ["config", "user.name", "github-actions[bot]"]);

    // Configure git to use fast-forward only merges
    core.info("Configuring Git to use fast-forward only merges");
    await exec.exec("git", ["config", "merge.ff", "only"]);

    // Fetch latest changes from origin
    core.info("Fetching latest changes from origin");
    await exec.exec("git", ["fetch", "origin"]);

    // Ensure we're on the current branch and it's up to date
    await exec.exec("git", ["checkout", currentBranch]);

    // Try to merge the base branch
    core.info(`Attempting to merge ${baseBranch} into ${currentBranch}`);
    try {
      const mergeRes = await exec.exec("git", ["merge", `origin/${baseBranch}`], {
        ignoreReturnCode: true,
      });

      if (mergeRes !== 0) {
        // Check if it's a merge conflict or other issue
        let statusOutput = "";
        await exec.exec("git", ["status", "--porcelain"], {
          listeners: { stdout: data => (statusOutput += data.toString()) },
          silent: true,
        });

        if (statusOutput.includes("UU") || statusOutput.includes("AA")) {
          core.setFailed(`Merge conflict detected when merging ${baseBranch} into ${currentBranch}. Manual resolution required.`);
        } else {
          core.setFailed(`Failed to merge ${baseBranch} into ${currentBranch}. Exit code: ${mergeRes}`);
        }
        return;
      }

      core.info("Merge completed successfully");

      // Check if there are any changes to push
      let hasChanges = false;
      try {
        const statusRes = await exec.exec("git", ["status", "--porcelain"], {
          listeners: {
            stdout: data => {
              if (data.toString().trim()) {
                hasChanges = true;
              }
            },
          },
          silent: true,
        });

        // Also check if we're ahead of origin
        let branchStatus = "";
        await exec.exec("git", ["status", "-b", "--porcelain"], {
          listeners: { stdout: data => (branchStatus += data.toString()) },
          silent: true,
        });

        if (branchStatus.includes("ahead")) {
          hasChanges = true;
        }
      } catch (error) {
        core.warning(`Failed to check git status: ${error instanceof Error ? error.message : String(error)}`);
      }

      if (hasChanges) {
        // Push the changes
        core.info(`Pushing updated ${currentBranch} to origin`);
        const pushRes = await exec.exec("git", ["push", "origin", currentBranch], {
          ignoreReturnCode: true,
        });

        if (pushRes !== 0) {
          core.setFailed(`Failed to push changes to ${currentBranch}`);
          return;
        }

        core.info("Successfully pushed updated branch");

        // Set outputs
        core.setOutput("updated", "true");
        core.setOutput("branch", currentBranch);
        core.setOutput("base_branch", baseBranch);
        core.setOutput("pr_number", prNumber.toString());

        // Add summary
        await core.summary
          .addRaw(`## ✅ Branch Update Successful\n\n`)
          .addRaw(`- **Branch**: \`${currentBranch}\`\n`)
          .addRaw(`- **Base Branch**: \`${baseBranch}\`\n`)
          .addRaw(`- **Pull Request**: #${prNumber}\n`)
          .addRaw(`\nSuccessfully merged latest changes from \`${baseBranch}\` and pushed to \`${currentBranch}\`.\n`)
          .write();
      } else {
        core.info("No changes to push - branch is already up to date");

        // Set outputs
        core.setOutput("updated", "false");
        core.setOutput("branch", currentBranch);
        core.setOutput("base_branch", baseBranch);
        core.setOutput("pr_number", prNumber.toString());

        // Add summary
        await core.summary
          .addRaw(`## ℹ️ Branch Already Up to Date\n\n`)
          .addRaw(`- **Branch**: \`${currentBranch}\`\n`)
          .addRaw(`- **Base Branch**: \`${baseBranch}\`\n`)
          .addRaw(`- **Pull Request**: #${prNumber}\n`)
          .addRaw(`\nNo changes needed - branch is already up to date with \`${baseBranch}\`.\n`)
          .write();
      }
    } catch (error) {
      core.setFailed(`Merge operation failed: ${error instanceof Error ? error.message : String(error)}`);
      return;
    }
  } catch (error) {
    core.setFailed(`Update branch workflow failed: ${error instanceof Error ? error.message : String(error)}`);
  }
}

await main();
