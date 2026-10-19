// @ts-check

const { ERR_VALIDATION } = require("./error_codes.cjs");

const WEBHOOK_PAYLOAD_POLICY = "resource-url-types/v1";
const ISSUE_TYPES = ["opened", "edited", "closed", "reopened", "labeled", "unlabeled", "deleted"];
const WEBHOOK_TYPES = {
  projects_v2_item: ["archived", "converted", "created", "deleted", "edited", "reordered", "restored"],
  code_scanning_alert: ["appeared_in_branch", "closed_by_user", "created", "fixed", "reopened", "reopened_by_user", "updated_assignment"],
  dependabot_alert: ["assignees_changed", "auto_dismissed", "auto_reopened", "created", "dismissed", "fixed", "reintroduced", "reopened", "withdrawn"],
  jira_issue: ISSUE_TYPES,
  linear_issue: ISSUE_TYPES,
  azure_devops_issue: [...ISSUE_TYPES, "restored"],
};

/** @returns {never} */
function reject(message) {
  throw new Error(`${ERR_VALIDATION}: Webhook validation failed: ${message}`);
}

function isWebhookEvent(event) {
  return event === "webhook" || (typeof event === "string" && Object.hasOwn(WEBHOOK_TYPES, event));
}

function isGenericWebhookType(type) {
  return typeof type === "string" && /^[A-Za-z0-9][A-Za-z0-9._:/-]{0,99}$/.test(type);
}

function resourceURL(value, message) {
  let url;
  try {
    url = new URL(value);
  } catch {
    reject(message);
  }
  if (url.protocol !== "https:" || url.username || url.password || url.search || url.hash || url.href !== value) {
    reject(message);
  }
  return url;
}

function resolveGitHubResource(event, url, env) {
  let base;
  try {
    base = new URL(env.GITHUB_API_URL || "https://api.github.com");
  } catch {
    reject("GITHUB_API_URL must be a valid URL");
  }
  const prefix = base.pathname.replace(/\/$/, "");
  const message = event === "projects_v2_item" ? "Projects payload requires a canonical GitHub API project item URL" : "security alert payload requires a canonical GitHub API alert URL";
  if (url.origin !== base.origin || !url.pathname.startsWith(`${prefix}/`)) reject(message);
  const path = url.pathname.slice(prefix.length);
  if (event === "projects_v2_item") {
    const match = path.match(/^\/(orgs|users)\/([a-zA-Z0-9_-]+)\/projectsV2\/([1-9][0-9]*)\/items\/([1-9][0-9]*)$/);
    if (!match) reject(message);
    return { item_type: event, item_number: match[4], repo: "", source: `${base.origin}${prefix}/${match[1]}/${match[2].toLowerCase()}/projectsV2/${match[3]}`, url: url.href };
  }
  const match = path.match(/^\/repos\/([a-zA-Z0-9_.-]+)\/([a-zA-Z0-9_.-]+)\/(code-scanning|dependabot)\/alerts\/([1-9][0-9]*)$/);
  if (!match || match[3] !== (event === "code_scanning_alert" ? "code-scanning" : "dependabot")) reject(message);
  return { item_type: event, item_number: match[4], repo: `${match[1]}/${match[2]}`, url: url.href };
}

function resolveProviderResource(event, url) {
  if (event === "jira_issue") {
    const match = url.pathname.match(/^\/rest\/api\/[23]\/issue\/([1-9][0-9]*)$/);
    if (!match) reject("Jira payload requires a canonical REST issue URL with a decimal issue ID");
    return { item_type: event, item_number: match[1], repo: "", source: url.origin, url: url.href };
  }
  if (event === "linear_issue") {
    const match = url.pathname.match(/^\/([a-zA-Z0-9_-]+)\/issue\/([A-Z][A-Z0-9_-]*-[1-9][0-9]*)(?:\/(?:[a-zA-Z0-9_-]|%[0-9a-fA-F]{2})+)?$/);
    if (url.origin !== "https://linear.app" || !match || match[1] === "issue") {
      reject("Linear payload requires a canonical workspace-qualified Linear issue URL");
    }
    return { item_type: event, item_number: match[2], repo: "", source: `${url.origin}/${match[1]}`, url: url.href };
  }
  // Work item IDs are collection-scoped, not update/revision IDs. The optional
  // project segment does not change that identity.
  const match = url.pathname.match(/^\/([a-zA-Z0-9_-]+)(?:\/(?:[a-zA-Z0-9_.-]|%[0-9a-fA-F]{2})+)?\/_apis\/wit\/work[Ii]tems\/([1-9][0-9]*)$/);
  if (!match || (url.hostname !== "dev.azure.com" && !/^[a-zA-Z0-9-]+\.visualstudio\.com$/.test(url.hostname))) {
    reject("Azure DevOps payload requires a canonical collection work item API URL");
  }
  return { item_type: event, item_number: match[2], repo: "", source: `${url.origin}/${match[1]}`, url: url.href };
}

function resolveWebhookResource(event, payload, env = process.env) {
  if (Object.keys(payload).some(key => !["url", "type"].includes(key))) {
    reject("webhook payload accepts only a canonical resource URL and type array; fetch resource details with an authenticated provider client");
  }
  if (event === "webhook") {
    if (!Array.isArray(payload.type) || payload.type.length !== 1 || !isGenericWebhookType(payload.type[0])) {
      reject("generic webhook payload type must contain exactly one literal event identifier");
    }
    const url = resourceURL(payload.url, "generic webhook payload requires a canonical HTTPS resource URL");
    return { item_type: "webhook", item_number: "", repo: "", source: url.href, url: url.href };
  }
  const supported = isWebhookEvent(event) ? WEBHOOK_TYPES[event] : undefined;
  if (!supported || !Array.isArray(payload.type) || payload.type.length === 0 || payload.type.some(type => !supported.includes(type)) || new Set(payload.type).size !== payload.type.length) {
    reject("webhook payload type must be a non-empty array of distinct supported event activities");
  }
  if (!event.endsWith("_issue") && payload.type.length !== 1) {
    reject("GitHub webhook payload type must contain exactly one event activity");
  }
  const message =
    event === "projects_v2_item"
      ? "Projects payload requires a canonical GitHub API project item URL"
      : event.endsWith("_alert")
        ? "security alert payload requires a canonical GitHub API alert URL"
        : "provider payload requires a canonical HTTPS resource URL";
  const url = resourceURL(payload.url, message);
  return event.endsWith("_issue") ? resolveProviderResource(event, url) : resolveGitHubResource(event, url, env);
}

module.exports = { WEBHOOK_PAYLOAD_POLICY, WEBHOOK_TYPES, isWebhookEvent, isGenericWebhookType, resolveWebhookResource };
