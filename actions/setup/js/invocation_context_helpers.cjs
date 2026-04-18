// @ts-check
/// <reference types="@actions/github-script" />

const { parseRepoSlug: parseSharedRepoSlug } = require("./repo_helpers.cjs");

/**
 * @typedef {{ owner: string, repo: string }} RepoRef
 */

/**
 * Parse a repository slug in owner/repo format.
 * @param {unknown} value
 * @returns {RepoRef|null}
 */
function parseRepoSlug(value) {
  if (typeof value !== "string") {
    return null;
  }

  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }

  return parseSharedRepoSlug(trimmed);
}

/**
 * Normalize a repo object into { owner, repo } shape.
 * @param {unknown} repoValue
 * @returns {RepoRef|null}
 */
function normalizeRepo(repoValue) {
  if (!repoValue || typeof repoValue !== "object") {
    return null;
  }

  /** @type {{ owner?: unknown, repo?: unknown }} */
  const maybeRepo = repoValue;
  if (typeof maybeRepo.owner === "string" && typeof maybeRepo.repo === "string" && maybeRepo.owner && maybeRepo.repo) {
    return {
      owner: maybeRepo.owner,
      repo: maybeRepo.repo,
    };
  }

  return null;
}

/**
 * Extract a repository from event payload.repository.
 * Supports both REST event shape (owner.login + name) and
 * github-script context-style payload.repo style.
 * @param {unknown} payload
 * @returns {RepoRef|null}
 */
function extractRepoFromPayload(payload) {
  if (!payload || typeof payload !== "object") {
    return null;
  }

  /** @type {{ repository?: unknown }} */
  const payloadObject = payload;
  const repositoryValue = payloadObject.repository;
  if (!repositoryValue || typeof repositoryValue !== "object") {
    return null;
  }
  /** @type {{ owner?: unknown, name?: unknown, repo?: unknown }} */
  const repository = repositoryValue;

  let owner;
  const ownerValue = repository.owner;
  if (typeof ownerValue === "string") {
    owner = ownerValue;
  } else if (ownerValue && typeof ownerValue === "object" && "login" in ownerValue && typeof ownerValue.login === "string") {
    owner = ownerValue.login;
  }
  const repo = typeof repository.name === "string" ? repository.name : typeof repository.repo === "string" ? repository.repo : undefined;

  if (owner && repo) {
    return { owner, repo };
  }

  return null;
}

/**
 * Parse a JSON input string into object payload.
 * @param {unknown} value
 * @returns {Record<string, any>|null}
 */
function parseJSONPayload(value) {
  if (typeof value !== "string" || value.trim() === "") {
    return null;
  }

  try {
    const parsed = JSON.parse(value);
    if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
      return /** @type {Record<string, any>} */ parsed;
    }
  } catch (_error) {
    // Best-effort parsing only.
  }

  return null;
}

/**
 * Resolve workflow repo and effective event context across invocation styles:
 * - native events
 * - workflow_dispatch (optional explicit overrides in inputs)
 * - repository_dispatch (event wrapped in client_payload)
 *
 * @param {any} rawContext
 * @returns {{
 *   source: "native" | "workflow_dispatch" | "repository_dispatch",
 *   eventName: string,
 *   eventPayload: any,
 *   workflowRepo: RepoRef,
 *   eventRepo: RepoRef
 * }}
 */
function resolveInvocationContext(rawContext) {
  const contextRepo = normalizeRepo(rawContext?.repo) || { owner: "", repo: "" };
  const workflowRepo = normalizeRepo(rawContext?.workflowRepo) || contextRepo;

  /** @type {"native" | "workflow_dispatch" | "repository_dispatch"} */
  let source = "native";
  let eventName = rawContext?.eventName || "";
  let eventPayload = rawContext?.payload || {};
  let eventRepo = normalizeRepo(rawContext?.eventRepo);

  if (eventName === "repository_dispatch") {
    const clientPayload = rawContext?.payload?.client_payload;
    if (clientPayload && typeof clientPayload === "object") {
      source = "repository_dispatch";
      eventName = rawContext?.payload?.action || eventName;
      eventPayload = clientPayload;
      eventRepo = eventRepo || extractRepoFromPayload(clientPayload) || parseRepoSlug(clientPayload?.aw_context?.repo);
    }
  } else if (eventName === "workflow_dispatch") {
    source = "workflow_dispatch";
    const inputs = rawContext?.payload?.inputs;
    if (inputs && typeof inputs === "object") {
      const inputsEventName = typeof inputs.event_name === "string" ? inputs.event_name : typeof inputs.eventName === "string" ? inputs.eventName : "";
      const parsedPayload = parseJSONPayload(inputs.event_payload) || parseJSONPayload(inputs.eventPayload);
      if (inputsEventName) {
        eventName = inputsEventName;
      }
      if (parsedPayload) {
        eventPayload = parsedPayload;
      }
      eventRepo = eventRepo || parseRepoSlug(inputs.event_repo) || parseRepoSlug(inputs.eventRepo) || parseRepoSlug(inputs.target_repo) || parseRepoSlug(inputs.targetRepo);
    }
  }

  if (!eventRepo) {
    eventRepo = extractRepoFromPayload(eventPayload) || workflowRepo;
  }

  return {
    source,
    eventName,
    eventPayload,
    workflowRepo,
    eventRepo,
  };
}

module.exports = {
  resolveInvocationContext,
};
