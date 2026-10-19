// @ts-check

const { isDeepStrictEqual } = require("node:util");
const { ERR_VALIDATION } = require("./error_codes.cjs");
const { isWebhookEvent, resolveWebhookResource } = require("./webhook_resource.cjs");

const WEBHOOK_SCHEMA_VERSION = 2;

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

/**
 * Preserve distinct updates while collapsing repeated deliveries and affected
 * items. Never select a single implicit target from a batch.
 *
 * @param {any} awContext
 * @param {NodeJS.ProcessEnv} [env]
 * @returns {{webhook: {event: string, deliveries: {delivery_id: string, payload: any}[]}, items: {item_type: string, item_number: string, repo: string, url: string, source?: string}[]} | null}
 */
function normalizeWebhookContext(awContext, env = process.env) {
  if (!isObject(awContext) || !Object.hasOwn(awContext, "webhook")) {
    return null;
  }
  const webhook = awContext.webhook;
  if (!isObject(webhook) || !isWebhookEvent(webhook.event)) {
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
    const prior = deliveries.get(delivery.delivery_id);
    if (prior && !isDeepStrictEqual(prior.payload, delivery.payload)) {
      reject("duplicate webhook delivery_id has conflicting payloads");
    }
    const item = resolveWebhookResource(webhook.event, delivery.payload, env);
    if (!prior) deliveries.set(delivery.delivery_id, { delivery_id: delivery.delivery_id, payload: delivery.payload });
    const source = "source" in item ? item.source : item.repo.toLowerCase();
    const identity = JSON.stringify([item.item_type, source, item.item_number]);
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
