// @ts-check
/// <reference types="@actions/github-script" />

const STEERING_ISSUE_TITLE_PREFIX = "[WIP] ";
const STEERING_ISSUE_MAX_AGE_MS = 24 * 60 * 60 * 1000;
const CLEANUP_COMMENT_MARKER = "<!-- gh-aw:stale-steering-issue-cleanup -->";
const CLEANUP_COMMENT = `This steering issue was left open after its linked workflow run completed successfully. Closing it during scheduled agentic maintenance.

${CLEANUP_COMMENT_MARKER}`;

const OPEN_ISSUES_QUERY = `
  query($owner: String!, $repo: String!, $cursor: String) {
    repository(owner: $owner, name: $repo) {
      issues(first: 100, after: $cursor, states: [OPEN], orderBy: { field: CREATED_AT, direction: ASC }) {
        nodes {
          number
          title
          createdAt
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
`;

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

async function getStaleSteeringIssues(owner, repo, cutoff) {
  const candidates = [];
  let cursor = null;
  let reachedCutoff = false;

  do {
    const result = await github.graphql(OPEN_ISSUES_QUERY, { owner, repo, cursor });
    const connection = result?.repository?.issues;
    if (!connection) {
      throw new Error("GraphQL query returned no repository issues connection");
    }

    for (const issue of connection.nodes || []) {
      const createdAt = Date.parse(issue.createdAt);
      if (!Number.isFinite(createdAt)) {
        continue;
      }
      if (createdAt > cutoff) {
        reachedCutoff = true;
        break;
      }
      if (issue.title?.startsWith(STEERING_ISSUE_TITLE_PREFIX) && createdAt <= cutoff) {
        candidates.push(issue);
      }
    }

    const pageInfo = connection.pageInfo || {};
    if (reachedCutoff || !pageInfo.hasNextPage) {
      break;
    }
    if (!pageInfo.endCursor || pageInfo.endCursor === cursor) {
      throw new Error("GraphQL issue pagination did not return a new cursor");
    }
    cursor = pageInfo.endCursor;
  } while (true);

  return candidates;
}

async function main() {
  const { owner, repo } = context.repo;
  const cutoff = Date.now() - STEERING_ISSUE_MAX_AGE_MS;
  const candidates = await getStaleSteeringIssues(owner, repo, cutoff);

  for (const candidate of candidates) {
    const { data: issue } = await github.rest.issues.get({ owner, repo, issue_number: candidate.number });
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
      issue_number: candidate.number,
      per_page: 100,
    });
    if (!comments.some(comment => comment.body?.includes(CLEANUP_COMMENT_MARKER))) {
      await github.rest.issues.createComment({
        owner,
        repo,
        issue_number: candidate.number,
        body: CLEANUP_COMMENT,
      });
    }
    await github.rest.issues.update({
      owner,
      repo,
      issue_number: candidate.number,
      state: "closed",
      state_reason: "completed",
    });
    core.info(`Closed stale steering issue #${candidate.number} for successful workflow run ${runID}`);
  }
}

module.exports = { main, getWorkflowRunId, getStaleSteeringIssues, CLEANUP_COMMENT, CLEANUP_COMMENT_MARKER, STEERING_ISSUE_MAX_AGE_MS };
