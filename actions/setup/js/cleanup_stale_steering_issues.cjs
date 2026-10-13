// @ts-check
/// <reference types="@actions/github-script" />

const STEERING_ISSUE_TITLE_PREFIX = "[WIP] ";
const STEERING_ISSUE_MAX_AGE_MS = 24 * 60 * 60 * 1000;
const CLEANUP_COMMENT_MARKER = "<!-- gh-aw:stale-steering-issue-cleanup -->";
const CLEANUP_COMMENT = `This steering issue was left open after its linked workflow run completed successfully. Closing it during scheduled agentic maintenance.

${CLEANUP_COMMENT_MARKER}`;

function getWorkflowRunId(issueBody, owner, repo) {
  const match = issueBody.match(/\[the workflow run\]\(([^)]+)\)/);
  if (!match) {
    return null;
  }

  try {
    const runURL = new URL(match[1]);
    const serverURL = new URL(context.serverUrl);
    const pathSegments = runURL.pathname.split("/").filter(Boolean).map(decodeURIComponent);
    const [runOwner, runRepo, actionsPath, runsPath, runID, ...extraSegments] = pathSegments;
    if (runURL.origin !== serverURL.origin || runOwner !== owner || runRepo !== repo || actionsPath !== "actions" || runsPath !== "runs" || extraSegments.length > 0 || !/^\d+$/.test(runID || "")) {
      return null;
    }
    const numericRunID = Number(runID);
    return Number.isSafeInteger(numericRunID) && numericRunID > 0 ? numericRunID : null;
  } catch {
    return null;
  }
}

async function main() {
  const { owner, repo } = context.repo;
  const cutoff = Date.now() - STEERING_ISSUE_MAX_AGE_MS;
  const issues = await github.paginate(github.rest.search.issuesAndPullRequests, {
    q: `repo:${owner}/${repo} is:issue is:open in:title "[WIP]"`,
    per_page: 100,
    sort: "created",
    order: "asc",
  });
  const candidates = issues.filter(issue => {
    const createdAt = Date.parse(issue.created_at);
    return !issue.pull_request && issue.title.startsWith(STEERING_ISSUE_TITLE_PREFIX) && Number.isFinite(createdAt) && createdAt <= cutoff;
  });

  for (const issue of candidates) {
    const runID = getWorkflowRunId(issue.body || "", owner, repo);
    if (!runID) {
      continue;
    }

    const { data: run } = await github.rest.actions.getWorkflowRun({ owner, repo, run_id: runID });
    if (run.status !== "completed" || run.conclusion !== "success") {
      continue;
    }

    const comments = await github.paginate(github.rest.issues.listComments, {
      owner,
      repo,
      issue_number: issue.number,
      per_page: 100,
    });
    if (!comments.some(comment => comment.body?.includes(CLEANUP_COMMENT_MARKER))) {
      await github.rest.issues.createComment({
        owner,
        repo,
        issue_number: issue.number,
        body: CLEANUP_COMMENT,
      });
    }
    await github.rest.issues.update({
      owner,
      repo,
      issue_number: issue.number,
      state: "closed",
      state_reason: "completed",
    });
    core.info(`Closed stale steering issue #${issue.number} for successful workflow run ${runID}`);
  }
}

module.exports = { main, getWorkflowRunId, CLEANUP_COMMENT, CLEANUP_COMMENT_MARKER, STEERING_ISSUE_MAX_AGE_MS };
