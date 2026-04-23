// @ts-check
/// <reference types="@actions/github-script" />

const { getErrorMessage } = require("./error_helpers.cjs");
const { withRetry, isTransientError, sleep } = require("./error_recovery.cjs");
const { fetchAndLogRateLimit } = require("./github_rate_limit_logger.cjs");

const ACTIVE_SESSION_STATES = new Set(["open", "active", "in_progress", "queued"]);
const LIST_PULL_REQUESTS_PER_PAGE = 100;
const SESSION_LIST_LIMIT = 1000;
const SESSION_PAGE_SIZE = 100;
const UPDATE_DELAY_MS = 1000;

/**
 * @param {unknown} value
 * @returns {number | null}
 */
function parsePullRequestNumber(value) {
  if (typeof value === "number" && Number.isInteger(value) && value > 0) return value;
  if (typeof value !== "string") return null;
  const trimmed = value.trim();
  if (!trimmed) return null;
  const parsed = Number.parseInt(trimmed, 10);
  return Number.isInteger(parsed) && parsed > 0 ? parsed : null;
}

/**
 * @param {unknown} value
 * @returns {boolean}
 */
function isActiveSessionState(value) {
  return typeof value === "string" && ACTIVE_SESSION_STATES.has(value.trim().toLowerCase());
}

/**
 * @returns {Promise<Set<number>>}
 */
async function listPullRequestsWithActiveSessions() {
  core.info("Listing agent sessions to identify PRs with active sessions");
  const copilotApiURL = await getCopilotAPIURL();
  core.info(`Resolved Copilot API endpoint for sessions: ${copilotApiURL}`);
  core.info(`Fetching up to ${SESSION_LIST_LIMIT} sessions (page_size=${SESSION_PAGE_SIZE})`);

  /** @type {Array<{resource_id?: number | string, state?: string, resource_type?: string}>} */
  const sessions = [];
  for (let pageNumber = 1; sessions.length < SESSION_LIST_LIMIT; pageNumber++) {
    const pageSessions = await listAgentSessionsPage(copilotApiURL, pageNumber, SESSION_PAGE_SIZE);
    core.info(`Fetched ${pageSessions.length} session(s) from page ${pageNumber}`);
    if (pageSessions.length === 0) break;
    sessions.push(...pageSessions);
    if (pageSessions.length < SESSION_PAGE_SIZE) break;
  }
  if (sessions.length >= SESSION_LIST_LIMIT) {
    core.warning(`Session list reached limit (${SESSION_LIST_LIMIT}); newer sessions may have been truncated`);
  }
  core.info(`Fetched ${sessions.length} total session record(s) for filtering`);

  const prNumbers = new Set();
  for (const session of sessions) {
    if (session?.resource_type !== "pull") continue;
    if (!isActiveSessionState(session?.state)) continue;
    const prNumber = parsePullRequestNumber(session?.resource_id);
    if (prNumber !== null) prNumbers.add(prNumber);
  }

  core.info(`Found ${prNumbers.size} pull request(s) with active agent sessions`);
  return prNumbers;
}

/**
 * @returns {Promise<string>}
 */
async function getCopilotAPIURL() {
  core.info("Resolving Copilot API endpoint from GraphQL viewer.copilotEndpoints.api");
  const response = await github.graphql(`
    query CopilotEndpointsForSessionListing {
      viewer {
        copilotEndpoints {
          api
        }
      }
    }
  `);
  const apiURL = response?.viewer?.copilotEndpoints?.api;
  if (typeof apiURL !== "string" || !apiURL.trim()) {
    throw new Error("Unable to resolve Copilot API URL for session listing");
  }
  const normalizedAPIURL = apiURL.replace(/\/+$/, "");
  core.info(`Copilot API endpoint resolved: ${normalizedAPIURL}`);
  return normalizedAPIURL;
}

/**
 * @param {string} copilotApiURL
 * @param {number} pageNumber
 * @param {number} pageSize
 * @returns {Promise<Array<{resource_id?: number | string, state?: string, resource_type?: string}>>}
 */
async function listAgentSessionsPage(copilotApiURL, pageNumber, pageSize) {
  const token = process.env.GH_TOKEN || process.env.GITHUB_TOKEN;
  if (!token) throw new Error("Missing GH_TOKEN/GITHUB_TOKEN for Copilot session listing");

  const sessionsURL = new URL(`${copilotApiURL}/agents/sessions`);
  sessionsURL.searchParams.set("page_size", String(pageSize));
  sessionsURL.searchParams.set("page_number", String(pageNumber));
  sessionsURL.searchParams.set("sort", "last_updated_at,desc");
  core.debug(`Requesting Copilot sessions page ${pageNumber}: ${sessionsURL.toString()}`);

  const response = await fetch(sessionsURL.toString(), {
    method: "GET",
    headers: {
      Accept: "application/json",
      Authorization: `Bearer ${token}`,
      "User-Agent": "gh-aw-update-pull-request-branches",
    },
  });

  if (!response.ok) {
    const responseBody = await response.text();
    const truncatedBody = responseBody.slice(0, 500);
    core.error(`Failed to list agent sessions page ${pageNumber}: HTTP ${response.status} ${response.statusText}`);
    if (truncatedBody) {
      core.error(`Copilot sessions error response (truncated): ${truncatedBody}`);
    }
    throw new Error(`Failed to list agent sessions: HTTP ${response.status}`);
  }

  const body = /** @type {any} */ await response.json();
  return Array.isArray(body?.sessions) ? body.sessions : [];
}

/**
 * @param {number[]} pullNumbers
 * @returns {Promise<number[]>}
 */
async function filterPullRequestsWithoutActiveSessions(pullNumbers) {
  const pullRequestsWithSessions = await listPullRequestsWithActiveSessions();
  const eligiblePullRequests = pullNumbers.filter(number => !pullRequestsWithSessions.has(number));
  core.info(`Found ${eligiblePullRequests.length} eligible pull request(s) without active sessions`);
  return eligiblePullRequests;
}

/**
 * @param {string} owner
 * @param {string} repo
 * @returns {Promise<number[]>}
 */
async function listOpenPullRequests(owner, repo) {
  const pulls = await github.paginate(github.rest.pulls.list, {
    owner,
    repo,
    state: "open",
    per_page: LIST_PULL_REQUESTS_PER_PAGE,
  });

  return pulls.map(pr => pr.number).filter(number => Number.isInteger(number));
}

/**
 * @param {string} owner
 * @param {string} repo
 * @param {number[]} pullNumbers
 * @returns {Promise<number[]>}
 */
async function filterMergeablePullRequests(owner, repo, pullNumbers) {
  const mergeable = [];

  for (const pullNumber of pullNumbers) {
    const { data: pull } = await withRetry(
      () =>
        github.rest.pulls.get({
          owner,
          repo,
          pull_number: pullNumber,
        }),
      {
        maxRetries: 2,
        initialDelayMs: 500,
        maxDelayMs: 2000,
        jitterMs: 0,
        shouldRetry: isTransientError,
      },
      `fetch pull request #${pullNumber}`
    );

    const isMergeable = pull?.state === "open" && pull?.mergeable === true && pull?.draft !== true;
    if (isMergeable) {
      mergeable.push(pullNumber);
      continue;
    }

    core.info(`Skipping PR #${pullNumber}: mergeable=${String(pull?.mergeable)}, state=${pull?.state || "unknown"}, draft=${String(Boolean(pull?.draft))}`);
  }

  return mergeable;
}

/**
 * @param {unknown} error
 * @returns {boolean}
 */
function isNonFatalUpdateBranchError(error) {
  if (typeof error === "object" && error !== null && "status" in error && error.status === 422) {
    return true;
  }

  const message = getErrorMessage(error).toLowerCase();
  return message.includes("update branch failed") || message.includes("head branch is not behind");
}

/**
 * @param {string} owner
 * @param {string} repo
 * @param {number} pullNumber
 * @returns {Promise<void>}
 */
async function updatePullRequestBranch(owner, repo, pullNumber) {
  await withRetry(
    () =>
      github.rest.pulls.updateBranch({
        owner,
        repo,
        pull_number: pullNumber,
      }),
    {
      maxRetries: 2,
      initialDelayMs: 1000,
      maxDelayMs: 10000,
      shouldRetry: isTransientError,
    },
    `update branch for pull request #${pullNumber}`
  );
}

/**
 * Update all mergeable PR branches that do not have active agent sessions.
 * @returns {Promise<void>}
 */
async function main() {
  const owner = context.repo.owner;
  const repo = context.repo.repo;

  core.info(`Updating pull request branches in ${owner}/${repo}`);
  await fetchAndLogRateLimit(github, "update_pull_request_branches_start");

  const openPullRequests = await listOpenPullRequests(owner, repo);
  core.info(`Found ${openPullRequests.length} open pull request(s)`);
  if (openPullRequests.length === 0) return;

  const mergeablePullRequests = await filterMergeablePullRequests(owner, repo, openPullRequests);
  core.info(`Found ${mergeablePullRequests.length} mergeable pull request(s)`);
  if (mergeablePullRequests.length === 0) return;

  const eligiblePullRequests = await filterPullRequestsWithoutActiveSessions(mergeablePullRequests);
  if (eligiblePullRequests.length === 0) return;

  let updatedCount = 0;
  let skippedCount = 0;
  let failedCount = 0;

  for (let i = 0; i < eligiblePullRequests.length; i++) {
    const pullNumber = eligiblePullRequests[i];
    try {
      core.info(`Updating branch for PR #${pullNumber}`);
      await updatePullRequestBranch(owner, repo, pullNumber);
      updatedCount++;
    } catch (error) {
      if (isNonFatalUpdateBranchError(error)) {
        skippedCount++;
        core.warning(`Skipping PR #${pullNumber}: ${getErrorMessage(error)}`);
      } else {
        failedCount++;
        core.error(`Failed to update branch for PR #${pullNumber}: ${getErrorMessage(error)}`);
      }
    }

    if (i < eligiblePullRequests.length - 1) {
      await sleep(UPDATE_DELAY_MS);
    }
  }

  await fetchAndLogRateLimit(github, "update_pull_request_branches_end");
  core.notice(`update_pull_request_branches completed: updated=${updatedCount}, skipped=${skippedCount}, failed=${failedCount}`);
}

module.exports = {
  main,
  parsePullRequestNumber,
  isActiveSessionState,
  listPullRequestsWithActiveSessions,
  filterPullRequestsWithoutActiveSessions,
  filterMergeablePullRequests,
  isNonFatalUpdateBranchError,
};
