// @ts-check

const { parseJsonlContent } = require("./jsonl_helpers.cjs");

/**
 * Pre-filter pattern: only parse lines that contain the word "steering".
 * This avoids JSON.parse on unrelated log entries.
 *
 * @type {RegExp}
 */
const STEERING_EVENT_PATTERN = /steering/i;

/**
 * Resolve an event name from a firewall proxy event entry.
 *
 * Supports three log schema variants:
 *   - Top-level `event` field: `{ event: "token_steering", ... }`
 *   - Top-level `type` field:  `{ type: "model_steering", ... }`
 *   - Nested payload:          `{ payload: { event: "steering" } }`
 *
 * @param {unknown} entry
 * @returns {string}
 */
function getApiProxyEventName(entry) {
  if (!entry || typeof entry !== "object" || Array.isArray(entry)) {
    return "";
  }
  if ("event" in entry && typeof entry.event === "string") {
    return entry.event;
  }
  if ("type" in entry && typeof entry.type === "string") {
    return entry.type;
  }
  if ("payload" in entry) {
    const payload = entry.payload;
    if (payload && typeof payload === "object" && !Array.isArray(payload)) {
      if ("event" in payload && typeof payload.event === "string") {
        return payload.event;
      }
      if ("type" in payload && typeof payload.type === "string") {
        return payload.type;
      }
    }
  }
  return "";
}

/**
 * Count steering events in proxy event-log JSONL content.
 *
 * Known steering event names: "steering", "token_steering", "model_steering".
 * Any event whose name is exactly "steering" or ends with "_steering" is counted.
 *
 * @param {string} jsonlContent
 * @returns {number}
 */
function countSteeringEventsInApiProxyJsonl(jsonlContent) {
  let count = 0;
  for (const parsed of parseJsonlContent(jsonlContent, line => STEERING_EVENT_PATTERN.test(line))) {
    const eventName = getApiProxyEventName(parsed).toLowerCase();
    // Known steering events: "steering", "token_steering", "model_steering".
    if (eventName === "steering" || eventName.endsWith("_steering")) {
      count += 1;
    }
  }
  return count;
}

module.exports = {
  STEERING_EVENT_PATTERN,
  getApiProxyEventName,
  countSteeringEventsInApiProxyJsonl,
};
