// @ts-check

const { isDeepStrictEqual } = require("node:util");
const { ERR_VALIDATION } = require("./error_codes.cjs");

const WEBHOOK_SCHEMA_VERSION = 2;
const WEBHOOK_EVENTS = new Set(["projects_v2_item", "code_scanning_alert", "dependabot_alert"]);

function isObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

/** @returns {value is string} */
function isNonemptyString(value) {
  return typeof value === "string" && value.trim().length > 0;
}

/** @returns {never} */
function reject(message) {
  throw new Error(`${ERR_VALIDATION}: Webhook validation failed: ${message}`);
}

function resolveSecurityAlertItem(event, payload, env) {
  let baseURL;
  try {
    baseURL = new URL(env.GITHUB_API_URL || "https://api.github.com");
  } catch {
    reject("GITHUB_API_URL must be a valid URL");
  }
  if (Object.keys(payload).length !== 1 || !isNonemptyString(payload.url)) {
    reject("security alert payload must contain only a canonical GitHub API alert URL");
  }
  let alertURL;
  try {
    alertURL = new URL(payload.url);
  } catch {
    reject("security alert payload must contain only a canonical GitHub API alert URL");
  }
  const apiPath = `${baseURL.pathname.replace(/\/$/, "")}/repos/`;
  const match = alertURL.pathname.slice(apiPath.length).match(/^([a-zA-Z0-9_.-]+)\/([a-zA-Z0-9_.-]+)\/(code-scanning|dependabot)\/alerts\/([1-9][0-9]*)$/);
  const source = event === "code_scanning_alert" ? "code-scanning" : "dependabot";
  if (
    alertURL.protocol !== "https:" ||
    alertURL.origin !== baseURL.origin ||
    !alertURL.pathname.startsWith(apiPath) ||
    !match ||
    match[3] !== source ||
    alertURL.username !== "" ||
    alertURL.password !== "" ||
    alertURL.search !== "" ||
    alertURL.hash !== "" ||
    alertURL.href !== payload.url
  ) {
    reject("security alert payload must contain only a canonical GitHub API alert URL");
  }
  return { item_type: event, item_number: match[4], repo: `${match[1]}/${match[2]}`, url: payload.url };
}

function resolveWebhookItem(event, payload, env) {
  if (event === "projects_v2_item") {
    const item = payload.projects_v2_item;
    if (!isObject(item)) {
      reject("projects_v2_item webhook payload must contain a projects_v2_item object");
    }
    if (!Number.isSafeInteger(item.id) || item.id <= 0 || (item.node_id !== undefined && !isNonemptyString(item.node_id)) || (item.project_node_id !== undefined && !isNonemptyString(item.project_node_id))) {
      reject("projects_v2_item must contain a positive integer id; node_id and project_node_id must be non-empty strings when present");
    }
    if (payload.repository !== undefined && (!isObject(payload.repository) || !isNonemptyString(payload.repository.full_name))) {
      reject("webhook repository.full_name must be a non-empty string when repository is present");
    }
    return { item_type: event, item_number: String(item.id), repo: payload.repository?.full_name ?? "", project_id: item.project_node_id ?? "" };
  }
  return resolveSecurityAlertItem(event, payload, env);
}

/**
 * Preserve distinct updates while collapsing repeated deliveries and affected
 * items. Never select a single implicit target from a batch.
 *
 * @param {any} awContext
 * @param {NodeJS.ProcessEnv} [env]
 * @returns {{webhook: {event: string, deliveries: {delivery_id: string, payload: any}[]}, items: {item_type: string, item_number: string, repo: string, project_id?: string, url?: string}[]} | null}
 */
function normalizeWebhookContext(awContext, env = process.env) {
  if (!isObject(awContext) || !Object.hasOwn(awContext, "webhook")) {
    return null;
  }
  const webhook = awContext.webhook;
  if (!isObject(webhook) || !WEBHOOK_EVENTS.has(webhook.event)) {
    reject("aw_context.webhook.event must be a supported webhook event");
  }
  if (Object.keys(webhook).some(key => !["event", "deliveries"].includes(key))) {
    reject("aw_context.webhook accepts only event and deliveries");
  }
  if (!Array.isArray(webhook.deliveries) || webhook.deliveries.length === 0) {
    reject("aw_context.webhook.deliveries must be a non-empty array");
  }
  const deliveries = new Map();
  const items = new Map();
  for (const delivery of webhook.deliveries) {
    if (!isObject(delivery) || Object.keys(delivery).some(key => !["delivery_id", "payload"].includes(key)) || !isNonemptyString(delivery.delivery_id)) {
      reject("each webhook delivery must contain a non-empty delivery_id and an inline payload, without per-delivery event overrides");
    }
    if (!isObject(delivery.payload)) {
      reject("each webhook delivery payload must be an inline webhook object");
    }
    if (delivery.payload.action !== undefined && !isNonemptyString(delivery.payload.action)) {
      reject("webhook payload action must be a non-empty string when present");
    }
    const item = resolveWebhookItem(webhook.event, delivery.payload, env);
    const prior = deliveries.get(delivery.delivery_id);
    if (prior && !isDeepStrictEqual(prior.payload, delivery.payload)) {
      reject("duplicate webhook delivery_id has conflicting payloads");
    }
    if (!prior) deliveries.set(delivery.delivery_id, { delivery_id: delivery.delivery_id, payload: delivery.payload });
    const identity = JSON.stringify([item.item_type, item.item_type === "projects_v2_item" ? "" : item.repo.toLowerCase(), item.item_number]);
    if (!items.has(identity)) items.set(identity, item);
  }
  return { webhook: { event: webhook.event, deliveries: [...deliveries.values()] }, items: [...items.values()] };
}

function redactWebhookPayloads(awContext) {
  if (typeof awContext === "string") {
    let parsed;
    try {
      parsed = JSON.parse(awContext);
    } catch {
      return awContext;
    }
    const redacted = redactWebhookPayloads(parsed);
    return redacted === parsed ? awContext : JSON.stringify(redacted);
  }
  if (!isObject(awContext?.webhook) || !Array.isArray(awContext.webhook.deliveries)) return awContext;
  return { ...awContext, webhook: { event: awContext.webhook.event, deliveries: awContext.webhook.deliveries.map(delivery => ({ delivery_id: delivery?.delivery_id })) } };
}

module.exports = { normalizeWebhookContext, redactWebhookPayloads, WEBHOOK_SCHEMA_VERSION };
