// @ts-check
/// <reference types="@actions/github-script" />

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
    discussion: context.payload?.discussion?.body ?? "",
    discussion_comment: context.payload?.comment?.body ?? "",
  };
  return bodyByEvent[context.eventName] ?? "";
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

async function main() {
  core.info("Starting centralized slash command routing.");
  core.info(`Incoming event name: '${context.eventName}'.`);

  const routeMap = JSON.parse(process.env.GH_AW_SLASH_ROUTING || "{}");
  core.info(`Configured centralized commands: ${Object.keys(routeMap).length}.`);

  const text = resolveBodyText();
  core.info(`Resolved payload text length: ${String(text).length}.`);
  const firstWord = String(text).trim().split(/\s+/)[0] ?? "";
  core.info(`First token in payload: '${firstWord || "<empty>"}'.`);
  if (!firstWord.startsWith("/")) {
    core.info("No slash command found at start of payload text; skipping dispatch.");
    return;
  }

  const commandName = firstWord.slice(1);
  const identifier = eventIdentifier();
  core.info(`Resolved command '/${commandName}' for event identifier '${identifier}'.`);
  const configuredRoutes = routeMap[commandName] ?? [];
  core.info(`Configured routes for '/${commandName}': ${configuredRoutes.length}.`);
  const routes = configuredRoutes.filter(route => Array.isArray(route.events) && route.events.includes(identifier));
  if (routes.length === 0) {
    core.info(`No centralized routes matched command '/${commandName}' for event '${identifier}'.`);
    return;
  }
  core.info(`Matched routes for '/${commandName}' on '${identifier}': ${routes.map(route => route.workflow).join(", ")}.`);

  const { buildAwContext } = require("./aw_context.cjs");

  const ref = resolveDispatchRef();
  core.info(`Dispatch ref resolved to '${ref}'.`);
  for (const route of routes) {
    const awContext = buildAwContext();
    awContext.command_name = commandName;
    core.info(`Dispatching workflow '${route.workflow}.lock.yml' for '/${commandName}'.`);
    await github.rest.actions.createWorkflowDispatch({
      owner: context.repo.owner,
      repo: context.repo.repo,
      workflow_id: `${route.workflow}.lock.yml`,
      ref,
      inputs: {
        aw_context: JSON.stringify(awContext),
      },
    });
    core.info(`Dispatched '${route.workflow}' for '/${commandName}'`);
  }
  core.info(`Completed centralized routing for '/${commandName}'.`);
}

module.exports = { main, eventIdentifier, resolveBodyText, resolveDispatchRef };
