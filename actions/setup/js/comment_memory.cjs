// @ts-check
/// <reference types="@actions/github-script" />
require("./shim.cjs");

const { sanitizeContent } = require("./sanitize_content.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { resolveTarget, isStagedMode } = require("./safe_output_helpers.cjs");
const { resolveTargetRepoConfig, resolveAndValidateRepo } = require("./repo_helpers.cjs");
const { createAuthenticatedGitHubClient } = require("./handler_auth.cjs");
const { generateFooterWithMessages, generateXMLMarker } = require("./messages_footer.cjs");
const { buildWorkflowRunUrl } = require("./workflow_metadata_helpers.cjs");
const { getTrackerID } = require("./get_tracker_id.cjs");
const { generateHistoryUrl } = require("./generate_history_link.cjs");
const { enforceCommentLimits } = require("./comment_limit_helpers.cjs");

const COMMENT_MEMORY_TAG = "gh-aw-comment-memory";
const COMMENT_MEMORY_MAX_SCAN_PAGES = 50;

function logInfo(message) {
  if (typeof core !== "undefined" && core && typeof core.info === "function") {
    core.info(message);
  }
}

function logWarning(message) {
  if (typeof core !== "undefined" && core && typeof core.warning === "function") {
    core.warning(message);
  }
}

function sanitizeMemoryID(memoryID) {
  const normalized = String(memoryID || "default").trim();
  if (!/^[a-zA-Z0-9_-]+$/.test(normalized)) {
    logInfo(`comment_memory: rejected invalid memory_id '${normalized}'`);
    return null;
  }
  return normalized;
}

function buildManagedMemoryBody(rawBody, memoryID, options) {
  const { includeFooter, runUrl, workflowName, workflowSource, workflowSourceURL, historyUrl, triggeringIssueNumber, triggeringPRNumber } = options;
  if (!/^[a-zA-Z0-9_-]+$/.test(memoryID)) {
    throw new Error("memory_id must contain only alphanumeric characters, hyphens, and underscores");
  }
  const openingTag = `<${COMMENT_MEMORY_TAG} id="${memoryID}">`;
  const closingTag = `</${COMMENT_MEMORY_TAG}>`;
  logInfo(`comment_memory: building managed body for memory_id='${memoryID}'`);
  let body = `${openingTag}\n${sanitizeContent(rawBody)}\n${closingTag}`;

  const tracker = getTrackerID("markdown");
  if (tracker) {
    body += `\n\n${tracker}`;
  }

  if (includeFooter) {
    logInfo(`comment_memory: footer enabled for memory_id='${memoryID}'`);
    body += "\n\n" + generateFooterWithMessages(workflowName, runUrl, workflowSource, workflowSourceURL, triggeringIssueNumber, triggeringPRNumber, undefined, historyUrl).trimEnd();
  } else {
    logInfo(`comment_memory: footer disabled for memory_id='${memoryID}', adding XML marker only`);
    body += "\n\n" + generateXMLMarker(workflowName, runUrl);
  }

  logInfo(`comment_memory: built body length=${body.length} for memory_id='${memoryID}'`);
  return body;
}

async function findManagedComment(github, owner, repo, itemNumber, memoryID) {
  const marker = `<${COMMENT_MEMORY_TAG} id="${memoryID}">`;
  logInfo(`comment_memory: scanning comments for marker='${marker}' on #${itemNumber} in ${owner}/${repo}`);
  let page = 1;
  const perPage = 100;
  while (page <= COMMENT_MEMORY_MAX_SCAN_PAGES) {
    logInfo(`comment_memory: scanning page ${page}/${COMMENT_MEMORY_MAX_SCAN_PAGES}`);
    const { data } = await github.rest.issues.listComments({
      owner,
      repo,
      issue_number: itemNumber,
      per_page: perPage,
      page,
    });
    if (!Array.isArray(data) || data.length === 0) {
      logInfo(`comment_memory: no comments found on page ${page}`);
      return null;
    }
    const match = data.find(comment => typeof comment.body === "string" && comment.body.includes(marker));
    if (match) {
      logInfo(`comment_memory: found existing managed comment id=${match.id} on page ${page}`);
      return match;
    }
    if (data.length < perPage) {
      logInfo(`comment_memory: reached final page ${page} without match`);
      return null;
    }
    page += 1;
  }
  logWarning(`comment_memory: reached scan limit (${COMMENT_MEMORY_MAX_SCAN_PAGES} pages) without match for memory_id='${memoryID}'`);
  return null;
}

async function main(config = {}) {
  const parsedMaxCount = parseInt(String(config.max ?? "1"), 10);
  const maxCount = Number.isInteger(parsedMaxCount) && parsedMaxCount > 0 ? parsedMaxCount : 1;
  const defaultMemoryID = sanitizeMemoryID(config.memory_id || "default") || "default";
  const includeFooter = String(config.footer ?? "true") !== "false";
  const target = config.target || "triggering";
  const githubClient = await createAuthenticatedGitHubClient(config);
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const staged = isStagedMode(config);
  logInfo(`comment_memory: initialized with max=${maxCount}, defaultMemoryID='${defaultMemoryID}', target='${target}', footer=${includeFooter}, staged=${staged}`);

  let processedCount = 0;

  return async message => {
    if (!message || message.type !== "comment_memory") {
      return null;
    }

    processedCount += 1;
    if (processedCount > maxCount) {
      logInfo(`comment_memory: skipping item because max count reached (${maxCount})`);
      return { success: true, skipped: true, warning: `Skipped comment_memory item: max ${maxCount} reached` };
    }
    logInfo(`comment_memory: processing item ${processedCount}/${maxCount}`);

    const targetResult = resolveTarget({
      targetConfig: target,
      item: message,
      context,
      itemType: "comment memory",
      // supportsPR=true means both issues and PRs in resolveTarget().
      supportsPR: true,
    });
    if (!targetResult.success) {
      logWarning(`comment_memory: target resolution failed: ${targetResult.error}`);
      return { success: false, error: targetResult.error };
    }
    logInfo(`comment_memory: resolved target item_number=${targetResult.number}`);

    const repoResolution = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "comment memory");
    if (!repoResolution.success) {
      logWarning(`comment_memory: repo resolution failed: ${repoResolution.error}`);
      return { success: false, error: repoResolution.error };
    }
    logInfo(`comment_memory: resolved target repo=${repoResolution.repo}`);

    const memoryID = sanitizeMemoryID(message.memory_id || defaultMemoryID);
    if (!memoryID) {
      return { success: false, error: "memory_id must contain only alphanumeric characters, hyphens, and underscores" };
    }
    logInfo(`comment_memory: using memory_id='${memoryID}'`);

    const runUrl = buildWorkflowRunUrl(context, context.repo);
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE ?? "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL ?? "";
    const triggeringIssueNumber = context.payload.issue?.number;
    const triggeringPRNumber = context.payload.pull_request?.number;
    const historyUrl =
      generateHistoryUrl({
        owner: repoResolution.repoParts.owner,
        repo: repoResolution.repoParts.repo,
        itemType: "comment",
        workflowCallId: process.env.GH_AW_CALLER_WORKFLOW_ID || "",
        workflowId: process.env.GH_AW_WORKFLOW_ID || "",
        serverUrl: context.serverUrl,
      }) || undefined;

    const managedBody = buildManagedMemoryBody(message.body || "", memoryID, {
      includeFooter,
      runUrl,
      workflowName,
      workflowSource,
      workflowSourceURL,
      historyUrl,
      triggeringIssueNumber,
      triggeringPRNumber,
    });
    try {
      enforceCommentLimits(managedBody);
    } catch (error) {
      logWarning(`comment_memory: body validation failed: ${getErrorMessage(error)}`);
      return { success: false, error: getErrorMessage(error) };
    }
    logInfo(`comment_memory: body validation passed for memory_id='${memoryID}'`);

    if (staged) {
      logInfo(`🎭 Staged Mode: would upsert comment-memory '${memoryID}' on #${targetResult.number} in ${repoResolution.repo}`);
      return { success: true, staged: true };
    }

    try {
      const existing = await findManagedComment(githubClient, repoResolution.repoParts.owner, repoResolution.repoParts.repo, targetResult.number, memoryID);
      if (existing) {
        logInfo(`comment_memory: updating existing managed comment id=${existing.id}`);
        const { data } = await githubClient.rest.issues.updateComment({
          owner: repoResolution.repoParts.owner,
          repo: repoResolution.repoParts.repo,
          comment_id: existing.id,
          body: managedBody,
        });
        logInfo(`comment_memory: updated comment url=${data.html_url}`);
        return {
          success: true,
          url: data.html_url,
          commentId: data.id,
          number: targetResult.number,
          repo: repoResolution.repo,
          managedBody,
        };
      }

      logInfo(`comment_memory: creating new managed comment`);
      const { data } = await githubClient.rest.issues.createComment({
        owner: repoResolution.repoParts.owner,
        repo: repoResolution.repoParts.repo,
        issue_number: targetResult.number,
        body: managedBody,
      });
      logInfo(`comment_memory: created comment id=${data.id} url=${data.html_url}`);
      return {
        success: true,
        url: data.html_url,
        commentId: data.id,
        number: targetResult.number,
        repo: repoResolution.repo,
        managedBody,
      };
    } catch (error) {
      logWarning(`comment_memory: upsert failed: ${getErrorMessage(error)}`);
      return { success: false, error: getErrorMessage(error) };
    }
  };
}

module.exports = { main, sanitizeMemoryID, findManagedComment, buildManagedMemoryBody };
