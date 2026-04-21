// @ts-check
/// <reference types="@actions/github-script" />

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

function sanitizeMemoryID(memoryID) {
  const normalized = String(memoryID || "default")
    .trim()
    .toLowerCase();
  if (!/^[a-z0-9_-]+$/.test(normalized)) {
    return null;
  }
  return normalized;
}

function buildManagedMemoryBody(rawBody, memoryID, includeFooter, runUrl, workflowName, workflowSource, workflowSourceURL, historyUrl) {
  let body = `<comment-memory id="${memoryID}">\n${sanitizeContent(rawBody)}\n</comment-memory>`;

  const tracker = getTrackerID("markdown");
  if (tracker) {
    body += `\n\n${tracker}`;
  }

  if (includeFooter) {
    body += "\n\n" + generateFooterWithMessages(workflowName, runUrl, workflowSource, workflowSourceURL, context.payload.issue?.number, context.payload.pull_request?.number, undefined, historyUrl).trimEnd();
  } else {
    body += "\n\n" + generateXMLMarker(workflowName, runUrl);
  }

  return body;
}

async function findManagedComment(github, owner, repo, itemNumber, memoryID) {
  const marker = `<comment-memory id="${memoryID}">`;
  let page = 1;
  const perPage = 100;
  while (true) {
    const { data } = await github.rest.issues.listComments({
      owner,
      repo,
      issue_number: itemNumber,
      per_page: perPage,
      page,
    });
    if (!Array.isArray(data) || data.length === 0) {
      return null;
    }
    const match = data.find(comment => typeof comment.body === "string" && comment.body.includes(marker));
    if (match) {
      return match;
    }
    if (data.length < perPage) {
      return null;
    }
    page += 1;
  }
}

async function main(config = {}) {
  const maxCount = Number(config.max || 1);
  const defaultMemoryID = sanitizeMemoryID(config.memory_id || "default") || "default";
  const includeFooter = String(config.footer ?? "true") !== "false";
  const target = config.target || "triggering";
  const githubClient = await createAuthenticatedGitHubClient(config);
  const { defaultTargetRepo, allowedRepos } = resolveTargetRepoConfig(config);
  const staged = isStagedMode(config);

  let processedCount = 0;

  return async message => {
    if (!message || message.type !== "comment_memory") {
      return null;
    }

    processedCount += 1;
    if (processedCount > maxCount) {
      return { success: true, skipped: true, warning: `Skipped comment_memory item: max ${maxCount} reached` };
    }

    const targetResult = resolveTarget({
      targetConfig: target,
      item: message,
      context,
      itemType: "comment memory",
      supportsPR: true,
      supportsIssue: false,
    });
    if (!targetResult.success) {
      return { success: false, error: targetResult.error };
    }

    const repoResolution = resolveAndValidateRepo(message, defaultTargetRepo, allowedRepos, "comment memory");
    if (!repoResolution.success) {
      return { success: false, error: repoResolution.error };
    }

    const memoryID = sanitizeMemoryID(message.memory_id || defaultMemoryID);
    if (!memoryID) {
      return { success: false, error: "memory_id must contain only alphanumeric characters, hyphens, and underscores" };
    }

    const runUrl = buildWorkflowRunUrl(context, context.repo);
    const workflowName = process.env.GH_AW_WORKFLOW_NAME || "Workflow";
    const workflowSource = process.env.GH_AW_WORKFLOW_SOURCE ?? "";
    const workflowSourceURL = process.env.GH_AW_WORKFLOW_SOURCE_URL ?? "";
    const historyUrl =
      generateHistoryUrl({
        owner: repoResolution.repoParts.owner,
        repo: repoResolution.repoParts.repo,
        itemType: "comment",
        workflowCallId: process.env.GH_AW_CALLER_WORKFLOW_ID || "",
        workflowId: process.env.GH_AW_WORKFLOW_ID || "",
        serverUrl: context.serverUrl,
      }) || undefined;

    const managedBody = buildManagedMemoryBody(message.body || "", memoryID, includeFooter, runUrl, workflowName, workflowSource, workflowSourceURL, historyUrl);
    try {
      enforceCommentLimits(managedBody);
    } catch (error) {
      return { success: false, error: getErrorMessage(error) };
    }

    if (staged) {
      core.info(`🎭 Staged Mode: would upsert comment-memory '${memoryID}' on #${targetResult.number} in ${repoResolution.repo}`);
      return { success: true, staged: true };
    }

    try {
      const existing = await findManagedComment(githubClient, repoResolution.repoParts.owner, repoResolution.repoParts.repo, targetResult.number, memoryID);
      if (existing) {
        const { data } = await githubClient.rest.issues.updateComment({
          owner: repoResolution.repoParts.owner,
          repo: repoResolution.repoParts.repo,
          comment_id: existing.id,
          body: managedBody,
        });
        return {
          success: true,
          url: data.html_url,
          number: targetResult.number,
          repo: repoResolution.repo,
        };
      }

      const { data } = await githubClient.rest.issues.createComment({
        owner: repoResolution.repoParts.owner,
        repo: repoResolution.repoParts.repo,
        issue_number: targetResult.number,
        body: managedBody,
      });
      return {
        success: true,
        url: data.html_url,
        number: targetResult.number,
        repo: repoResolution.repo,
      };
    } catch (error) {
      return { success: false, error: getErrorMessage(error) };
    }
  };
}

module.exports = { main, sanitizeMemoryID, findManagedComment, buildManagedMemoryBody };
