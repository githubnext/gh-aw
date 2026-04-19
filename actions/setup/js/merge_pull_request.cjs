// @ts-check
/// <reference types="@actions/github-script" />

const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { globPatternToRegex, simpleGlobToRegex } = require("./glob_pattern_helpers.cjs");
const { isStagedMode } = require("./safe_output_helpers.cjs");
const { selectLatestRelevantChecks } = require("./check_runs_helpers.cjs");
const { withRetry, isTransientError } = require("./error_recovery.cjs");

/**
 * @typedef {import('./types/handler-factory').HandlerFactoryFunction} HandlerFactoryFunction
 */

/**
 * @param {string[]} patterns
 * @returns {RegExp[]}
 */
function compilePathGlobs(patterns) {
  return patterns.map(p => globPatternToRegex(p, { pathMode: true, caseSensitive: true }));
}

/**
 * @param {string[]} patterns
 * @returns {RegExp[]}
 */
function compileLabelGlobs(patterns) {
  return patterns.map(p => simpleGlobToRegex(p, false));
}

/**
 * @param {string[]} changedFiles
 * @param {RegExp[]} patterns
 * @returns {string[]}
 */
function findNonMatchingFiles(changedFiles, patterns) {
  return changedFiles.filter(file => !patterns.some(re => re.test(file)));
}

/**
 * @param {string[]} changedFiles
 * @param {RegExp[]} patterns
 * @returns {string[]}
 */
function findMatchingFiles(changedFiles, patterns) {
  return changedFiles.filter(file => patterns.some(re => re.test(file)));
}

/**
 * @param {any} githubClient
 * @param {string} owner
 * @param {string} repo
 * @param {number} pullNumber
 * @returns {Promise<any>}
 */
async function getPullRequestWithMergeability(githubClient, owner, repo, pullNumber) {
  core.info(`Fetching PR #${pullNumber} in ${owner}/${repo} with mergeability retry`);
  return withRetry(
    async () => {
      const { data } = await githubClient.rest.pulls.get({
        owner,
        repo,
        pull_number: pullNumber,
      });
      if (data && data.mergeable === null) {
        throw new Error("pull request mergeability is still being computed");
      }
      return data;
    },
    {
      maxRetries: 3,
      initialDelayMs: 1000,
      shouldRetry: error => {
        const msg = getErrorMessage(error).toLowerCase();
        return isTransientError(error) || msg.includes("mergeability is still being computed");
      },
    },
    `fetch pull request #${pullNumber}`
  ).catch(async error => {
    const fallback = await githubClient.rest.pulls.get({
      owner,
      repo,
      pull_number: pullNumber,
    });
    if (fallback?.data) {
      core.warning(`Mergeability remained unknown after retries for PR #${pullNumber}, continuing with latest state`);
      return fallback.data;
    }
    throw error;
  });
}

/**
 * @param {any} githubClient
 * @param {string} owner
 * @param {string} repo
 * @param {number} pullNumber
 * @returns {Promise<{reviewDecision: string|null, unresolvedThreadCount: number}>}
 */
async function getReviewSummary(githubClient, owner, repo, pullNumber) {
  core.info(`Collecting review summary for PR #${pullNumber}`);
  let unresolvedThreadCount = 0;
  let reviewDecision = null;
  let cursor = null;
  let hasNextPage = true;
  let page = 0;
  while (hasNextPage) {
    page++;
    const result = await githubClient.graphql(
      `
        query($owner: String!, $repo: String!, $number: Int!, $after: String) {
          repository(owner: $owner, name: $repo) {
            pullRequest(number: $number) {
              reviewDecision
              reviewThreads(first: 100, after: $after) {
                pageInfo { hasNextPage endCursor }
                nodes { isResolved }
              }
            }
          }
        }
      `,
      { owner, repo, number: pullNumber, after: cursor }
    );

    const pr = result?.repository?.pullRequest;
    if (!pr) {
      core.warning(`No pull request data returned while reading review summary for PR #${pullNumber}`);
      break;
    }
    reviewDecision = pr.reviewDecision || null;
    const threads = pr.reviewThreads?.nodes || [];
    core.info(`Review page ${page}: ${threads.length} thread(s)`);
    unresolvedThreadCount += threads.filter(t => !t.isResolved).length;
    hasNextPage = pr.reviewThreads?.pageInfo?.hasNextPage === true;
    cursor = pr.reviewThreads?.pageInfo?.endCursor || null;
  }

  core.info(`Review summary: decision=${reviewDecision || "null"}, unresolvedThreads=${unresolvedThreadCount}`);
  return { reviewDecision, unresolvedThreadCount };
}

/**
 * @param {any} githubClient
 * @param {string} owner
 * @param {string} repo
 * @param {string} baseBranch
 * @returns {Promise<{isProtected: boolean, requiredChecks: string[]}>}
 */
async function getBranchPolicy(githubClient, owner, repo, baseBranch) {
  core.info(`Checking target branch policy for ${owner}/${repo}@${baseBranch}`);
  const { data: branch } = await githubClient.rest.repos.getBranch({
    owner,
    repo,
    branch: baseBranch,
  });

  const isProtected = branch?.protected === true;
  if (isProtected) {
    core.warning(`Target branch ${baseBranch} is protected`);
    return { isProtected: true, requiredChecks: [] };
  }

  try {
    const { data } = await githubClient.rest.repos.getBranchProtection({
      owner,
      repo,
      branch: baseBranch,
    });
    const contexts = Array.isArray(data?.required_status_checks?.contexts) ? data.required_status_checks.contexts : [];
    const checks = Array.isArray(data?.required_status_checks?.checks) ? data.required_status_checks.checks.map(c => c?.context).filter(Boolean) : [];
    core.info(`Branch protection checks for ${baseBranch}: ${[...new Set([...contexts, ...checks])].join(", ") || "(none)"}`);
    return { isProtected: false, requiredChecks: [...new Set([...contexts, ...checks])] };
  } catch (error) {
    if (error && typeof error === "object" && "status" in error && error.status === 404) {
      core.info(`No branch protection rules found for ${baseBranch}`);
      return { isProtected: false, requiredChecks: [] };
    }
    core.error(`Failed to read branch protection for ${baseBranch}: ${getErrorMessage(error)}`);
    throw error;
  }
}

/**
 * @param {any} githubClient
 * @param {string} owner
 * @param {string} repo
 * @param {string} headSha
 * @param {string[]} requiredChecks
 * @returns {Promise<{missing: string[], failing: Array<{name: string, status: string, conclusion: string|null}>}>}
 */
async function evaluateRequiredChecks(githubClient, owner, repo, headSha, requiredChecks) {
  core.info(`Evaluating required checks on ${headSha}: ${requiredChecks.join(", ") || "(none)"}`);
  if (requiredChecks.length === 0) {
    return { missing: [], failing: [] };
  }

  const checkRuns = await githubClient.paginate(githubClient.rest.checks.listForRef, {
    owner,
    repo,
    ref: headSha,
    per_page: 100,
  });

  const { relevant } = selectLatestRelevantChecks(checkRuns, { includeList: requiredChecks });
  core.info(`Fetched ${checkRuns.length} check run(s), ${relevant.length} relevant latest check run(s)`);
  const byName = new Map(relevant.map(run => [run.name, run]));
  const missing = [];
  const failing = [];

  for (const checkName of requiredChecks) {
    const run = byName.get(checkName);
    if (!run) {
      core.warning(`Required check missing: ${checkName}`);
      missing.push(checkName);
      continue;
    }
    if (run.status !== "completed" || run.conclusion !== "success") {
      core.warning(`Required check not passing: ${checkName} status=${run.status} conclusion=${run.conclusion || "null"}`);
      failing.push({ name: checkName, status: run.status, conclusion: run.conclusion || null });
    }
  }

  return { missing, failing };
}

/**
 * @param {any} githubClient
 * @param {string} owner
 * @param {string} repo
 * @param {number} pullNumber
 * @returns {Promise<string[]>}
 */
async function listChangedFiles(githubClient, owner, repo, pullNumber) {
  core.info(`Listing changed files for PR #${pullNumber}`);
  const files = await githubClient.paginate(githubClient.rest.pulls.listFiles, {
    owner,
    repo,
    pull_number: pullNumber,
    per_page: 100,
  });
  const changed = files.map(f => f.filename).filter(Boolean);
  core.info(`PR #${pullNumber} changed ${changed.length} file(s)`);
  return changed;
}

/**
 * @returns {number|undefined}
 */
function resolveContextPullNumber() {
  if (context.payload?.pull_request?.number) {
    return context.payload.pull_request.number;
  }
  if (context.payload?.issue?.pull_request && context.payload?.issue?.number) {
    return context.payload.issue.number;
  }
  return undefined;
}

/**
 * Handler factory for merge_pull_request.
 * @type {HandlerFactoryFunction}
 */
async function main(config = {}) {
  const githubClient = await createAuthenticatedGitHubClient(config);
  const isStaged = isStagedMode(config);
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const maxCount = Number(config.max || 1);
  const requiredLabels = Array.isArray(config.required_labels) ? config.required_labels : [];
  const allowedLabels = Array.isArray(config.allowed_labels) ? config.allowed_labels : [];
  const allowedBranches = Array.isArray(config.allowed_branches) ? config.allowed_branches : [];
  const allowedFiles = Array.isArray(config.allowed_files) ? config.allowed_files : [];
  const protectedFiles = Array.isArray(config.protected_files) ? config.protected_files : [];

  const allowedLabelPatterns = compileLabelGlobs(allowedLabels);
  const allowedBranchPatterns = compilePathGlobs(allowedBranches);
  const allowedFilePatterns = compilePathGlobs(allowedFiles);
  const protectedFilePatterns = compilePathGlobs(protectedFiles);
  core.info(
    `merge_pull_request handler configured: max=${maxCount}, requiredLabels=${requiredLabels.length}, allowedLabels=${allowedLabels.length}, allowedBranches=${allowedBranches.length}, allowedFiles=${allowedFiles.length}, protectedFiles=${protectedFiles.length}, staged=${isStaged}`
  );

  let processedCount = 0;

  return async function handleMergePullRequest(message) {
    core.info(`Processing merge_pull_request message: ${JSON.stringify({ pull_request_number: message?.pull_request_number, repo: message?.repo, merge_method: message?.merge_method })}`);
    if (processedCount >= maxCount) {
      core.warning(`Skipping merge_pull_request: max count of ${maxCount} reached`);
      return { success: false, error: `Max count of ${maxCount} reached` };
    }
    processedCount++;

    const repoResult = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "merge pull request");
    if (!repoResult.success) {
      core.error(`Repository validation failed: ${repoResult.error}`);
      return { success: false, error: repoResult.error };
    }
    const { owner, repo } = repoResult.repoParts;
    core.info(`Resolved target repository: ${owner}/${repo}`);

    const pullNumberRaw = message.pull_request_number ?? resolveContextPullNumber();
    const pullNumber = parseInt(String(pullNumberRaw || ""), 10);
    if (!pullNumber || Number.isNaN(pullNumber)) {
      core.error("pull_request_number is required for merge_pull_request");
      return { success: false, error: "pull_request_number is required for merge_pull_request" };
    }
    core.info(`Target PR number: ${pullNumber}`);

    /** @type {Array<{code: string, message: string, details?: any}>} */
    const failureReasons = [];

    try {
      const pr = await getPullRequestWithMergeability(githubClient, owner, repo, pullNumber);
      if (!pr) {
        core.error(`Pull request #${pullNumber} not found`);
        return { success: false, error: `Pull request #${pullNumber} not found` };
      }
      core.info(`PR state: merged=${pr.merged}, draft=${pr.draft}, mergeable=${pr.mergeable}, mergeable_state=${pr.mergeable_state || "unknown"}, head=${pr.head?.ref}, base=${pr.base?.ref}`);
      if (pr.merged) {
        core.info(`PR #${pullNumber} is already merged, returning idempotent success`);
        return {
          success: true,
          merged: true,
          alreadyMerged: true,
          pull_request_number: pr.number,
          pull_request_url: pr.html_url,
          checks_evaluated: [],
        };
      }

      if (pr.draft) {
        failureReasons.push({ code: "pr_is_draft", message: "Pull request is still in draft state" });
      }
      if (pr.mergeable === false || pr.mergeable_state === "dirty") {
        failureReasons.push({ code: "merge_conflicts", message: "Pull request has unresolved merge conflicts" });
      }
      if (pr.mergeable !== true) {
        failureReasons.push({ code: "not_mergeable", message: `Pull request is not mergeable (mergeable=${String(pr.mergeable)}, state=${pr.mergeable_state || "unknown"})` });
      }

      const labels = (pr.labels || []).map(l => l.name).filter(Boolean);
      core.info(`PR labels (${labels.length}): ${labels.join(", ") || "(none)"}`);
      const missingRequiredLabels = requiredLabels.filter(label => !labels.includes(label));
      if (missingRequiredLabels.length > 0) {
        failureReasons.push({
          code: "missing_required_labels",
          message: "Required labels are missing",
          details: { missing: missingRequiredLabels, present: labels },
        });
      }

      if (allowedLabelPatterns.length > 0) {
        const matchedLabels = labels.filter(label => allowedLabelPatterns.some(re => re.test(label)));
        core.info(`Allowed label match count: ${matchedLabels.length}`);
        if (matchedLabels.length === 0) {
          failureReasons.push({
            code: "allowed_labels_no_match",
            message: "No pull request label matches allowed-labels patterns",
            details: { present: labels, patterns: allowedLabels },
          });
        }
      }

      if (allowedBranchPatterns.length > 0 && !allowedBranchPatterns.some(re => re.test(pr.head.ref))) {
        failureReasons.push({
          code: "branch_not_allowed",
          message: `Source branch "${pr.head.ref}" does not match allowed-branches`,
          details: { source_branch: pr.head.ref, patterns: allowedBranches },
        });
      }
      if (allowedBranchPatterns.length > 0) {
        core.info(`Allowed branch patterns: ${allowedBranches.join(", ")}`);
      }

      const branchPolicy = await getBranchPolicy(githubClient, owner, repo, pr.base.ref);
      if (branchPolicy.isProtected) {
        failureReasons.push({
          code: "target_branch_protected",
          message: `Target branch "${pr.base.ref}" is protected`,
        });
      }

      const checkSummary = await evaluateRequiredChecks(githubClient, owner, repo, pr.head.sha, branchPolicy.requiredChecks);
      core.info(`Required check summary: missing=${checkSummary.missing.length}, failing=${checkSummary.failing.length}`);
      if (checkSummary.missing.length > 0) {
        failureReasons.push({
          code: "required_checks_missing",
          message: "Required status checks are not completed",
          details: { missing: checkSummary.missing },
        });
      }
      if (checkSummary.failing.length > 0) {
        failureReasons.push({
          code: "required_checks_failing",
          message: "Required status checks are not passing",
          details: { failing: checkSummary.failing },
        });
      }

      if ((pr.requested_reviewers || []).length > 0 || (pr.requested_teams || []).length > 0) {
        failureReasons.push({
          code: "pending_reviewers",
          message: "All assigned reviewers have not approved yet",
          details: {
            requested_reviewers: (pr.requested_reviewers || []).map(r => r.login),
            requested_teams: (pr.requested_teams || []).map(t => t.slug),
          },
        });
      }

      const reviewSummary = await getReviewSummary(githubClient, owner, repo, pullNumber);
      if (reviewSummary.reviewDecision === "CHANGES_REQUESTED" || reviewSummary.reviewDecision === "REVIEW_REQUIRED") {
        failureReasons.push({
          code: "blocking_review_state",
          message: `Blocking review state remains active (${reviewSummary.reviewDecision})`,
        });
      }
      if (reviewSummary.unresolvedThreadCount > 0) {
        failureReasons.push({
          code: "unresolved_review_threads",
          message: "Pull request has unresolved review threads",
          details: { unresolved_count: reviewSummary.unresolvedThreadCount },
        });
      }

      const changedFiles = await listChangedFiles(githubClient, owner, repo, pullNumber);
      core.info(`Changed files sample: ${changedFiles.slice(0, 20).join(", ")}${changedFiles.length > 20 ? ", ..." : ""}`);

      if (protectedFilePatterns.length > 0) {
        const protectedMatches = findMatchingFiles(changedFiles, protectedFilePatterns);
        core.info(`Protected file match count: ${protectedMatches.length}`);
        if (protectedMatches.length > 0) {
          failureReasons.push({
            code: "protected_files_match",
            message: "Protected files were changed",
            details: { matched_files: protectedMatches, patterns: protectedFiles, protected_files_blocked: true },
          });
        }
      }

      if (allowedFilePatterns.length > 0) {
        const disallowedFiles = findNonMatchingFiles(changedFiles, allowedFilePatterns);
        core.info(`Allowed-file violations count: ${disallowedFiles.length}`);
        if (disallowedFiles.length > 0) {
          failureReasons.push({
            code: "allowed_files_violation",
            message: "Changed files outside allowed-files patterns",
            details: { disallowed_files: disallowedFiles, patterns: allowedFiles },
          });
        }
      }

      if (failureReasons.length > 0) {
        core.warning(`merge_pull_request blocked with ${failureReasons.length} gate failure(s): ${failureReasons.map(r => r.code).join(", ")}`);
        return {
          success: false,
          error: "merge_pull_request gate checks failed",
          failure_reasons: failureReasons,
          checks_evaluated: branchPolicy.requiredChecks,
        };
      }

      if (isStaged) {
        core.info(`Staged mode: merge for PR #${pullNumber} not executed`);
        return {
          success: true,
          staged: true,
          merged: false,
          pull_request_number: pr.number,
          pull_request_url: pr.html_url,
          checks_evaluated: branchPolicy.requiredChecks,
        };
      }

      const mergeResponse = await githubClient.rest.pulls.merge({
        owner,
        repo,
        pull_number: pullNumber,
        merge_method: message.merge_method || "merge",
        commit_title: message.commit_title,
        commit_message: message.commit_message,
      });

      if (mergeResponse.data?.merged !== true) {
        core.error(`Merge API returned merged=false for PR #${pullNumber}: ${mergeResponse.data?.message || "no message"}`);
        return {
          success: false,
          error: mergeResponse.data?.message || "Merge API returned merged=false",
          failure_reasons: [{ code: "merge_not_completed", message: mergeResponse.data?.message || "Merge was not completed" }],
          checks_evaluated: branchPolicy.requiredChecks,
        };
      }

      return {
        success: true,
        merged: true,
        pull_request_number: pr.number,
        pull_request_url: pr.html_url,
        sha: mergeResponse.data?.sha,
        message: mergeResponse.data?.message,
        checks_evaluated: branchPolicy.requiredChecks,
      };
    } catch (error) {
      core.error(`merge_pull_request failed for PR #${pullNumber}: ${getErrorMessage(error)}`);
      return {
        success: false,
        error: getErrorMessage(error),
        failure_reasons: [{ code: "merge_operation_error", message: getErrorMessage(error) }],
      };
    }
  };
}

module.exports = { main };
