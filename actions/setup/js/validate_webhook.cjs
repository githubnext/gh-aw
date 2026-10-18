// @ts-check
/// <reference types="@actions/github-script" />

const fs = require("fs");
const { ERR_VALIDATION } = require("./error_codes.cjs");
const { getErrorMessage } = require("./error_helpers.cjs");
const { normalizeWebhookContext, WEBHOOK_SCHEMA_VERSION } = require("./webhook_context.cjs");

const MAX_CLIENT_PAYLOAD_BYTES = 65535;
const MAX_DISPATCH_REQUEST_BYTES = 65535;
const MAX_CLIENT_PAYLOAD_KEYS = 10;
const MAX_EVENT_TYPE_LENGTH = 100;

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

function parseConfiguration(value, name) {
  try {
    return JSON.parse(value || "");
  } catch {
    return reject(`${name} must be valid compiler-generated JSON`);
  }
}

function eventName(env) {
  return env.GITHUB_EVENT_NAME ?? env.GH_AW_WEBHOOK_EVENT_NAME;
}

/**
 * Validate the dispatch transport without replacing the GitHub authorization
 * context. Same-kind deliveries are carried in client_payload.aw_context.webhook.
 * The gateway may project only part of each source webhook payload.
 * Actor authorization uses the standard pre-activation membership check.
 * Subscription conditions are evaluated by the webhook receiver, not here.
 *
 * @param {any} event - Actual repository_dispatch event, never the inner webhook
 * @param {NodeJS.ProcessEnv} [env]
 * @returns {{skipped: boolean, event?: string, delivery_ids?: string[]}}
 */
function validateWebhook(event, env = process.env) {
  if (eventName(env) !== "repository_dispatch") {
    return { skipped: true };
  }

  const subscriptions = parseConfiguration(env.GH_AW_WEBHOOK_SUBSCRIPTIONS, "GH_AW_WEBHOOK_SUBSCRIPTIONS");
  if (
    !Array.isArray(subscriptions) ||
    subscriptions.length === 0 ||
    !subscriptions.every(
      subscription =>
        isObject(subscription) &&
        isNonemptyString(subscription.event) &&
        isObject(subscription.dispatch) &&
        isNonemptyString(subscription.dispatch.event_type) &&
        subscription.dispatch.event_type.length <= MAX_EVENT_TYPE_LENGTH &&
        subscription.schema_version === WEBHOOK_SCHEMA_VERSION &&
        Array.isArray(subscription.types) &&
        subscription.types.every(isNonemptyString)
    )
  ) {
    reject("GH_AW_WEBHOOK_SUBSCRIPTIONS must contain valid version 2 batch subscriptions");
  }

  if (!isObject(event) || !isNonemptyString(event.action) || event.action.length > MAX_EVENT_TYPE_LENGTH) {
    reject("repository_dispatch action must be a non-empty event_type of at most 100 characters");
  }
  const matching = subscriptions.filter(subscription => subscription.dispatch.event_type === event.action);
  if (matching.length === 0) {
    return { skipped: true };
  }
  const envelope = event.client_payload;
  if (!isObject(envelope)) {
    reject("client_payload must be an object");
  }
  if (Object.keys(envelope).length > MAX_CLIENT_PAYLOAD_KEYS) {
    reject("client_payload must have at most 10 root keys");
  }
  if (Buffer.byteLength(JSON.stringify(envelope), "utf8") > MAX_CLIENT_PAYLOAD_BYTES) {
    reject("client_payload exceeds the 65535-byte inline payload limit; oversized payloads are not fetched");
  }
  if (Buffer.byteLength(JSON.stringify({ event_type: event.action, client_payload: envelope }), "utf8") > MAX_DISPATCH_REQUEST_BYTES) {
    reject("repository_dispatch request must be less than 65536 bytes including event_type and client_payload");
  }
  if (envelope.schema_version !== WEBHOOK_SCHEMA_VERSION) {
    reject("client_payload.schema_version must match subscription version 2");
  }
  if (Object.keys(envelope).some(key => !["schema_version", "aw_context"].includes(key))) {
    reject("client_payload accepts only schema_version and aw_context; singleton webhook envelopes are unsupported");
  }
  const normalized = normalizeWebhookContext(envelope.aw_context, env);
  if (!normalized) {
    reject("client_payload.aw_context.webhook is required");
  }
  const eventSubscriptions = matching.filter(subscription => subscription.event === normalized.webhook.event);
  if (eventSubscriptions.length !== 1) {
    reject("aw_context.webhook.event must match the webhook subscription");
  }
  const subscription = eventSubscriptions[0];
  for (const delivery of normalized.webhook.deliveries) {
    if (delivery.payload.action !== undefined && subscription.types.length > 0 && !subscription.types.includes(delivery.payload.action)) {
      reject("webhook payload action does not match the subscription types");
    }
  }
  return { skipped: false, event: normalized.webhook.event, delivery_ids: normalized.webhook.deliveries.map(delivery => delivery.delivery_id) };
}

/**
 * Read the runner-owned event file, rather than interpolating webhook data into
 * shell commands or trusting a supplied payload.sender.
 */
async function main() {
  if (eventName(process.env) !== "repository_dispatch") {
    return { skipped: true };
  }
  let event;
  try {
    if (!process.env.GITHUB_EVENT_PATH) {
      reject("GITHUB_EVENT_PATH is required for repository_dispatch");
    }
    event = JSON.parse(fs.readFileSync(process.env.GITHUB_EVENT_PATH, "utf8"));
  } catch {
    reject("GITHUB_EVENT_PATH must contain a readable JSON event");
  }
  const result = validateWebhook(event);
  if (!result.skipped && typeof core !== "undefined") {
    core.info(`Webhook batch validated (${result.delivery_ids?.length} unique deliveries).`);
  }
  return result;
}

module.exports = { main, validateWebhook, MAX_CLIENT_PAYLOAD_BYTES, MAX_DISPATCH_REQUEST_BYTES, MAX_CLIENT_PAYLOAD_KEYS, MAX_EVENT_TYPE_LENGTH };

if (require.main === module) {
  main().catch(error => {
    console.error(getErrorMessage(error));
    process.exitCode = 1;
  });
}
