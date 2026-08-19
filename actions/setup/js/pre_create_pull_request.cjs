// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { getBaseBranch } = require("./get_base_branch.cjs");
const { getPromptPath, renderTemplateFromFile } = require("./messages_core.cjs");
const { normalizeBranchName } = require("./normalize_branch_name.cjs");
const { applyTitlePrefix } = require("./sanitize_title.cjs");

/**
 * Best-effort deletion of a pre-allocated branch so a failed allocation does not
 * leave an orphaned branch behind.
 * @param {string} branch - Branch name to delete
 * @returns {Promise<void>}
 */
async function deleteBranch(branch) {
  try {
    await github.rest.git.deleteRef({
      owner: context.repo.owner,
      repo: context.repo.repo,
      ref: `heads/${branch}`,
    });
  } catch (error) {
    core.warning(`Failed to delete pre-allocated branch ${branch}: ${getErrorMessage(error)}`);
  }
}

async function main() {
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || context.workflow || "Agentic workflow";
  const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;
  const branch = `gh-aw/pre-created/${context.runId}-${process.env.GITHUB_RUN_ATTEMPT || "1"}`;

  // Resolve the base branch the eventual pull request will target (configured base-branch,
  // otherwise the event-derived branch) so the pre-created branch is forked from the same
  // commit the safe-outputs job will later rebase onto.
  const resolvedBaseBranch = await getBaseBranch();
  const baseBranch = normalizeBranchName(resolvedBaseBranch);
  if (!baseBranch || baseBranch !== resolvedBaseBranch) {
    throw new Error(`Invalid base branch for pre-created pull request: "${resolvedBaseBranch}"`);
  }

  const { data: baseCommit } = await github.rest.repos.getCommit({
    owner: context.repo.owner,
    repo: context.repo.repo,
    ref: `heads/${baseBranch}`,
  });

  const { data: commit } = await github.rest.git.createCommit({
    owner: context.repo.owner,
    repo: context.repo.repo,
    message: `Initialize pull request for ${workflowName}`,
    tree: baseCommit.commit.tree.sha,
    parents: [baseCommit.sha],
  });

  try {
    await github.rest.git.createRef({
      owner: context.repo.owner,
      repo: context.repo.repo,
      ref: `refs/heads/${branch}`,
      sha: commit.sha,
    });
  } catch (error) {
    throw new Error(`Failed to create pre-allocated pull request branch: ${getErrorMessage(error)}`, { cause: error });
  }

  let pullRequest;
  let checkRun;
  const titlePrefix = process.env.GH_AW_PR_TITLE_PREFIX || "";
  // "[WIP]" first so the in-progress state is visible even when a title prefix is configured.
  const title = `[WIP] ${applyTitlePrefix(`${workflowName}: work in progress`, titlePrefix)}`;
  const body = renderTemplateFromFile(getPromptPath("pre_created_pull_request_body.md"), {
    run_url: runUrl,
    workflow_name: workflowName,
  }).trimEnd();
  try {
    ({ data: pullRequest } = await github.rest.pulls.create({
      owner: context.repo.owner,
      repo: context.repo.repo,
      title,
      body,
      head: branch,
      base: baseBranch,
      draft: true,
    }));

    ({ data: checkRun } = await github.rest.checks.create({
      owner: context.repo.owner,
      repo: context.repo.repo,
      name: workflowName,
      head_sha: commit.sha,
      details_url: runUrl,
      status: "in_progress",
      output: {
        title: workflowName,
        summary: `Follow the [workflow run](${runUrl}) for progress.`,
      },
    }));
  } catch (error) {
    // Avoid leaving an orphaned branch behind when the pull request or check cannot be created.
    await deleteBranch(branch);
    throw error;
  }

  core.setOutput("pull_request_number", pullRequest.number);
  core.setOutput("pull_request_url", pullRequest.html_url);
  core.setOutput("branch", branch);
  core.setOutput("check_run_id", checkRun.id);
}

module.exports = { main };
