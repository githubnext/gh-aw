// @ts-check
/// <reference types="@actions/github-script" />

const { REACTION_MAP } = require("./add_reaction.cjs");
// Keep this aligned with the current default stable GitHub REST API version used by workflows.
// Update when GitHub advances the recommended version to avoid sunset/deprecation warnings.
const GITHUB_API_VERSION = "2022-11-28";

function eventIdentifier() {
  if (context.eventName !== "issue_comment") {
    return context.eventName;
  }
  return context.payload?.issue?.pull_request ? "pull_request_comment" : "issue_comment";
}

function resolveBodyText() {
  const bodyByEvent = {
    issues: context.payload?.issue?.body ?? "",
    pull_request: context.payload?.pull_request?.body ?? "",
    issue_comment: context.payload?.comment?.body ?? "",
    pull_request_review_comment: context.payload?.comment?.body ?? "",
    pull_request_review: context.payload?.review?.body ?? "",
    discussion: context.payload?.discussion?.body ?? "",
    discussion_comment: context.payload?.comment?.body ?? "",
  };
  return bodyByEvent[context.eventName] ?? "";
}

function isPRClosedAtStart() {
  const pullRequestState = context.payload?.pull_request?.state;
  if (pullRequestState === "closed") {
    return true;
  }
  const issueState = context.payload?.issue?.state;
  if (context.payload?.issue?.pull_request && issueState === "closed") {
    return true;
  }
  return false;
}

function resolveDispatchRef() {
  if (process.env.GITHUB_HEAD_REF) {
    return `refs/heads/${process.env.GITHUB_HEAD_REF}`;
  }

  const fallbackRef = process.env.GITHUB_REF || context.ref;
  if (fallbackRef) {
    return fallbackRef;
  }

  const defaultBranch = context.payload?.repository?.default_branch || "main";
  return `refs/heads/${defaultBranch}`;
}

function normalizeReaction(reaction) {
  if (typeof reaction !== "string") {
    return "";
  }
  const trimmed = reaction.trim();
  if (!trimmed || trimmed === "none" || !Object.hasOwn(REACTION_MAP, trimmed)) {
    return "";
  }
  return trimmed;
}

/**
 * Returns the first valid non-"none" ai_reaction configured on matching routes.
 * @param {Array<{ai_reaction?: unknown}>} routes
 * @returns {string}
 */
function resolveImmediateReaction(routes) {
  for (const route of routes) {
    const reaction = normalizeReaction(route?.ai_reaction);
    if (reaction) {
      return reaction;
    }
  }
  return "";
}

async function addImmediateReaction(reaction) {
  const normalized = normalizeReaction(reaction);
  if (!normalized) {
    return;
  }

  const { owner, repo } = context.repo;
  try {
    switch (context.eventName) {
      case "issues": {
        const issueNumber = context.payload?.issue?.number;
        if (!issueNumber) {
          core.warning("Skipping immediate reaction: issue number was not found in payload.");
          return;
        }
        await github.request(`POST /repos/${owner}/${repo}/issues/${issueNumber}/reactions`, {
          content: normalized,
          headers: { Accept: "application/vnd.github+json" },
        });
        return;
      }
      case "issue_comment": {
        const commentId = context.payload?.comment?.id;
        if (!commentId) {
          core.warning("Skipping immediate reaction: comment id was not found in payload.");
          return;
        }
        await github.request(`POST /repos/${owner}/${repo}/issues/comments/${commentId}/reactions`, {
          content: normalized,
          headers: { Accept: "application/vnd.github+json" },
        });
        return;
      }
      case "pull_request": {
        const prNumber = context.payload?.pull_request?.number;
        if (!prNumber) {
          core.warning("Skipping immediate reaction: pull request number was not found in payload.");
          return;
        }
        await github.request(`POST /repos/${owner}/${repo}/issues/${prNumber}/reactions`, {
          content: normalized,
          headers: { Accept: "application/vnd.github+json" },
        });
        return;
      }
      case "pull_request_review_comment": {
        const reviewCommentId = context.payload?.comment?.id;
        if (!reviewCommentId) {
          core.warning("Skipping immediate reaction: review comment id was not found in payload.");
          return;
        }
        await github.request(`POST /repos/${owner}/${repo}/pulls/comments/${reviewCommentId}/reactions`, {
          content: normalized,
          headers: { Accept: "application/vnd.github+json" },
        });
        return;
      }
      case "discussion_comment": {
        const commentNodeId = context.payload?.comment?.node_id;
        if (!commentNodeId) {
          core.warning("Skipping immediate reaction: discussion comment node id was not found in payload.");
          return;
        }
        await github.graphql(
          `
            mutation($subjectId: ID!, $content: ReactionContent!) {
              addReaction(input: { subjectId: $subjectId, content: $content }) {
                reaction { id }
              }
            }`,
          { subjectId: commentNodeId, content: REACTION_MAP[normalized] }
        );
        return;
      }
      case "discussion": {
        const discussionNumber = context.payload?.discussion?.number;
        if (!discussionNumber) {
          core.warning("Skipping immediate reaction: discussion number was not found in payload.");
          return;
        }
        const { repository } = await github.graphql(
          `
            query($owner: String!, $repo: String!, $num: Int!) {
              repository(owner: $owner, name: $repo) {
                discussion(number: $num) { id }
              }
            }`,
          { owner, repo, num: discussionNumber }
        );
        const discussionNodeId = repository?.discussion?.id;
        if (!discussionNodeId) {
          core.warning("Skipping immediate reaction: discussion node id was not found.");
          return;
        }
        await github.graphql(
          `
            mutation($subjectId: ID!, $content: ReactionContent!) {
              addReaction(input: { subjectId: $subjectId, content: $content }) {
                reaction { id }
              }
            }`,
          { subjectId: discussionNodeId, content: REACTION_MAP[normalized] }
        );
        return;
      }
      default:
        core.warning(`Skipping immediate reaction: unsupported event type '${context.eventName}'.`);
        return;
    }
  } catch (error) {
    core.warning(`Immediate reaction '${normalized}' failed: ${String(error)}`);
  }
}

/**
 * Dispatches a workflow with the API version header required by GitHub REST.
 * @param {string} workflowId
 * @param {string} ref
 * @param {Record<string, string>} inputs
 * @returns {Promise<void>}
 */
async function dispatchWorkflow(workflowId, ref, inputs) {
  try {
    await github.rest.actions.createWorkflowDispatch({
      owner: context.repo.owner,
      repo: context.repo.repo,
      workflow_id: workflowId,
      ref,
      inputs,
      headers: {
        "X-GitHub-Api-Version": GITHUB_API_VERSION,
      },
    });
  } catch (error) {
    throw new Error(`Failed to dispatch workflow '${workflowId}' on ref '${ref}': ${String(error)}`);
  }
}

async function main() {
  core.info("Starting centralized command routing.");
  core.info(`Incoming event name: '${context.eventName}'.`);

  const slashRouteMap = JSON.parse(process.env.GH_AW_SLASH_ROUTING || "{}");
  const labelRouteMap = JSON.parse(process.env.GH_AW_LABEL_ROUTING || "{}");
  const reviewerRoutes = JSON.parse(process.env.GH_AW_REVIEWER_ROUTING || "[]");
  core.info(`Configured centralized slash commands: ${Object.keys(slashRouteMap).length}.`);
  core.info(`Configured decentralized label commands: ${Object.keys(labelRouteMap).length}.`);
  core.info(`Configured pull-request reviewer workflows: ${Array.isArray(reviewerRoutes) ? reviewerRoutes.length : 0}.`);

  const identifier = eventIdentifier();
  const { buildAwContext } = require("./aw_context.cjs");
  const ref = resolveDispatchRef();
  if (isPRClosedAtStart()) {
    core.info("Pull request is closed at workflow start; skipping centralized routing.");
    return;
  }

  if (context.payload?.action === "labeled") {
    const labelName = context.payload?.label?.name ?? "";
    if (!labelName) {
      core.info("Labeled event missing label name; skipping dispatch.");
      return;
    }
    const configuredRoutes = labelRouteMap[labelName] ?? [];
    core.info(`Configured routes for label '${labelName}': ${configuredRoutes.length}.`);
    const routes = configuredRoutes.filter(route => Array.isArray(route.events) && route.events.includes(identifier));
    if (routes.length === 0) {
      core.info(`No decentralized label routes matched label '${labelName}' for event '${identifier}'.`);
      return;
    }
    core.info(`Matched routes for label '${labelName}' on '${identifier}': ${routes.map(route => route.workflow).join(", ")}.`);
    const immediateReaction = resolveImmediateReaction(routes);
    if (immediateReaction) {
      core.info(`Adding immediate '${immediateReaction}' reaction for label '${labelName}'.`);
      await addImmediateReaction(immediateReaction);
    }
    for (const route of routes) {
      const routeReaction = normalizeReaction(route?.ai_reaction);
      const awContext = {
        ...buildAwContext(),
        command_name: "",
        ...(routeReaction ? { desired_ai_reaction: routeReaction } : {}),
      };
      core.info(`Dispatching workflow '${route.workflow}.lock.yml' for label '${labelName}'.`);
      await dispatchWorkflow(`${route.workflow}.lock.yml`, ref, {
        aw_context: JSON.stringify(awContext),
      });
      core.info(`Dispatched '${route.workflow}' for label '${labelName}'`);
    }
    core.info(`Completed decentralized label routing for '${labelName}'.`);
    return;
  }

  if (context.eventName === "pull_request" && Array.isArray(reviewerRoutes) && reviewerRoutes.length > 0) {
    const matches = reviewerRoutes.filter(route => Array.isArray(route.events) && route.events.includes(context.eventName));
    if (matches.length > 0) {
      let selected = matches;
      if (!["ready_for_review", "review_requested"].includes(context.payload?.action ?? "")) {
        selected = [];
      }
      if (selected.length > 0) {
        core.info(`Matched reviewer routes on '${context.eventName}': ${selected.map(route => route.workflow).join(", ")}.`);
        for (const route of selected) {
          const awContext = {
            ...buildAwContext(),
            command_name: "",
            reviewer_lifecycle_event: context.eventName,
          };
          core.info(`Dispatching reviewer workflow '${route.workflow}.lock.yml'.`);
          await dispatchWorkflow(`${route.workflow}.lock.yml`, ref, {
            aw_context: JSON.stringify(awContext),
          });
        }
        core.info("Completed reviewer lifecycle routing.");
      }
    }
  }

  const text = resolveBodyText();
  core.info(`Resolved payload text length: ${String(text).length}.`);
  const firstWord = String(text).trim().split(/\s+/)[0] ?? "";
  core.info(`First token in payload: '${firstWord || "<empty>"}'.`);
  if (!firstWord.startsWith("/")) {
    core.info("No slash command found at start of payload text; skipping dispatch.");
    return;
  }

  const commandName = firstWord.slice(1);
  core.info(`Resolved command '/${commandName}' for event identifier '${identifier}'.`);
  const configuredRoutes = slashRouteMap[commandName] ?? [];
  core.info(`Configured routes for '/${commandName}': ${configuredRoutes.length}.`);
  const routes = configuredRoutes.filter(route => Array.isArray(route.events) && route.events.includes(identifier));
  if (routes.length === 0) {
    core.info(`No centralized routes matched command '/${commandName}' for event '${identifier}'.`);
    return;
  }
  core.info(`Matched routes for '/${commandName}' on '${identifier}': ${routes.map(route => route.workflow).join(", ")}.`);
  const immediateReaction = resolveImmediateReaction(routes);
  if (immediateReaction) {
    core.info(`Adding immediate '${immediateReaction}' reaction for '/${commandName}'.`);
    await addImmediateReaction(immediateReaction);
  }

  core.info(`Dispatch ref resolved to '${ref}'.`);
  for (const route of routes) {
    const routeReaction = normalizeReaction(route?.ai_reaction);
    const awContext = {
      ...buildAwContext(),
      command_name: commandName,
      ...(routeReaction ? { desired_ai_reaction: routeReaction } : {}),
    };
    core.info(`Dispatching workflow '${route.workflow}.lock.yml' for '/${commandName}'.`);
    await dispatchWorkflow(`${route.workflow}.lock.yml`, ref, {
      aw_context: JSON.stringify(awContext),
    });
    core.info(`Dispatched '${route.workflow}' for '/${commandName}'`);
  }
  core.info(`Completed centralized routing for '/${commandName}'.`);
}

module.exports = { main, eventIdentifier, resolveBodyText, resolveDispatchRef, GITHUB_API_VERSION };
