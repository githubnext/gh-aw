// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");

async function main() {
  const workflowName = process.env.GH_AW_WORKFLOW_NAME || context.workflow || "Agentic workflow";
  const runUrl = `${context.serverUrl}/${context.repo.owner}/${context.repo.repo}/actions/runs/${context.runId}`;
  const branch = `gh-aw/pre-created/${context.runId}-${process.env.GITHUB_RUN_ATTEMPT || "1"}`;
  const { stdout: checkoutHead } = await exec.getExecOutput("git", ["rev-parse", "HEAD"]);
  const headSha = checkoutHead.trim();

  const [{ data: repository }, { data: baseCommit }] = await Promise.all([
    github.rest.repos.get({
      owner: context.repo.owner,
      repo: context.repo.repo,
    }),
    github.rest.repos.getCommit({
      owner: context.repo.owner,
      repo: context.repo.repo,
      ref: headSha,
    }),
  ]);

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

  const { data: pullRequest } = await github.rest.pulls.create({
    owner: context.repo.owner,
    repo: context.repo.repo,
    title: `[${workflowName}] Work in progress`,
    body: `This draft pull request was pre-created for [the workflow run](${runUrl}).`,
    head: branch,
    base: repository.default_branch,
    draft: true,
  });

  const { data: checkRun } = await github.rest.checks.create({
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
  });

  core.setOutput("pull_request_number", pullRequest.number);
  core.setOutput("pull_request_url", pullRequest.html_url);
  core.setOutput("branch", branch);
  core.setOutput("check_run_id", checkRun.id);
}

module.exports = { main };
