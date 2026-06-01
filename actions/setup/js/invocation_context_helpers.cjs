// @ts-check
/// <reference types="@actions/github-script" />

const { parseRepoSlug: parseSharedRepoSlug, parseAllowedRepos, validateTargetRepo } = require("./repo_helpers.cjs");
const { ERR_VALIDATION } = require("./error_codes.cjs");

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
 * Build a minimal event payload from aw_context metadata for centralized slash-command dispatches.
 * @param {Record<string, any>} awContext
 * @param {string} eventName
 * @returns {Record<string, any>|null}
 */
function buildPayloadFromAwContext(awContext, eventName) {
  const itemType = typeof awContext.item_type === "string" ? awContext.item_type : "";
  const itemNumberRaw = awContext.item_number;
  const commentIdRaw = awContext.comment_id;
  const commentNodeId = typeof awContext.comment_node_id === "string" ? awContext.comment_node_id : "";
  const itemNumber = Number(itemNumberRaw);
  const commentId = Number(commentIdRaw);

  if (!itemType || !Number.isFinite(itemNumber) || itemNumber <= 0) {
    return null;
  }

  /** @type {Record<string, any>} */
  const payload = {};

  if (itemType === "issue") {
    payload.issue = { number: itemNumber };
  } else if (itemType === "pull_request") {
    payload.pull_request = { number: itemNumber };
    if (eventName === "issue_comment") {
      payload.issue = { number: itemNumber, pull_request: {} };
    }
  } else if (itemType === "discussion") {
    payload.discussion = { number: itemNumber };
  }

  if (Number.isFinite(commentId) && commentId > 0) {
    payload.comment = { id: commentId };
    if (commentNodeId) {
      payload.comment.node_id = commentNodeId;
    }
  } else if (commentNodeId) {
    payload.comment = { node_id: commentNodeId };
  }

  return Object.keys(payload).length > 0 ? payload : null;
}

/**
 * Validate workflow_dispatch target repository against allowlist configuration.
 * Enforces SEC-005 by rejecting disallowed cross-repository target overrides.
 * @param {RepoRef} workflowRepo
 * @param {RepoRef} targetRepo
 */
function checkAllowedRepo(workflowRepo, targetRepo) {
  const defaultRepo = `${workflowRepo.owner}/${workflowRepo.repo}`;
  const targetRepoSlug = `${targetRepo.owner}/${targetRepo.repo}`;
  const allowedRepos = parseAllowedRepos(process.env.GH_AW_ALLOWED_REPOS);

  // Fall back to per-handler safe-output allowlists when a global allowlist is
  // not provided. This prevents post-action context resolution from rejecting
  // a repository that was already allowed by the active safe-output handler.
  if (allowedRepos.size === 0) {
    const handlerConfigRaw = process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG;
    if (typeof handlerConfigRaw === "string" && handlerConfigRaw.trim() !== "") {
      try {
        const handlerConfig = JSON.parse(handlerConfigRaw);
        if (handlerConfig && typeof handlerConfig === "object" && !Array.isArray(handlerConfig)) {
          for (const value of Object.values(handlerConfig)) {
            if (!value || typeof value !== "object" || Array.isArray(value)) {
              continue;
            }
            const parsed = parseAllowedRepos(value.allowed_repos);
            for (const repo of parsed) {
              allowedRepos.add(repo);
            }
          }
        }
      } catch (_error) {
        // Best-effort only. If the handler config cannot be parsed, continue
        // with the global allowlist (if any).
      }
    }
  }

  const validation = validateTargetRepo(targetRepoSlug, defaultRepo, allowedRepos);
  if (!validation.valid) {
    throw new Error(`${ERR_VALIDATION}: ${validation.error}`);
  }
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
      const awContext = parseJSONPayload(inputs.aw_context) || parseJSONPayload(inputs.awContext);
      const targetRepo = parseRepoSlug(inputs.target_repo) || parseRepoSlug(inputs.targetRepo);
      if (targetRepo) {
        checkAllowedRepo(workflowRepo, targetRepo);
      }
      if (inputsEventName) {
        eventName = inputsEventName;
      } else if (typeof awContext?.event_type === "string" && awContext.event_type.trim()) {
        eventName = awContext.event_type.trim();
      }
      if (parsedPayload) {
        eventPayload = parsedPayload;
      } else if (awContext && typeof awContext === "object") {
        const awPayload = buildPayloadFromAwContext(awContext, eventName);
        if (awPayload) {
          eventPayload = awPayload;
        }
      }
      eventRepo = eventRepo || parseRepoSlug(inputs.event_repo) || parseRepoSlug(inputs.eventRepo) || parseRepoSlug(typeof awContext?.repo === "string" ? awContext.repo : undefined) || targetRepo;
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
