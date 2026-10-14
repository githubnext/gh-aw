// @ts-check
"use strict";

const { createLogger } = require("./mcp_logger.cjs");
const { performance } = require("node:perf_hooks");

/** @type {Map<string, number>} */
const lastEmittedAt = new Map();

/** @param {string} namespace */
function debugEnabled(namespace) {
  if (process.env.ACTIONS_RUNNER_DEBUG === "true" || process.env.RUNNER_DEBUG === "1") return true;
  let enabled = false;
  for (const token of (process.env.DEBUG || "").split(/[\s,]+/).filter(Boolean)) {
    const excluded = token.startsWith("-");
    const pattern = excluded ? token.slice(1) : token;
    const expression = pattern
      .split("*")
      .map(part => part.replace(/[.*+?^${}()|[\]\\]/g, "\\$&"))
      .join(".*");
    if (!new RegExp("^" + expression + "$").test(namespace)) continue;
    if (excluded) return false;
    enabled = true;
  }
  return enabled;
}

/**
 * Events and field names must be static developer-authored labels. Values are
 * counts/flags only: never pass identities, payloads, credentials or raw errors.
 * @param {string} component
 */
function createWorkQueueLogger(component) {
  const namespace = `work-queue:${component}`;
  const logger = createLogger(namespace);
  /** @param {string} event @param {Record<string, unknown>} [fields] */
  function diagnostic(event, fields = {}) {
    const metadata = Object.entries(fields).filter(([, value]) => typeof value === "boolean" || (typeof value === "number" && Number.isSafeInteger(value) && value >= 0));
    const now = performance.now();
    const previous = lastEmittedAt.get(namespace) ?? now;
    const elapsed = Math.max(0, Math.round(now - previous));
    logger.debug([event, ...metadata.map(([key, value]) => `${key}=${value}`), `+${elapsed}ms`].join(" "));
    lastEmittedAt.set(namespace, now);
  }
  /** @param {string} event @param {Record<string, unknown>} [fields] */
  function debug(event, fields = {}) {
    if (!debugEnabled(namespace)) return;
    diagnostic(event, fields);
  }
  /** @param {unknown} value @param {string} key */
  function ownValue(value, key) {
    try {
      return value && typeof value === "object" ? Object.getOwnPropertyDescriptor(value, key)?.value : undefined;
    } catch {
      return undefined;
    }
  }
  /** @param {string} event @param {unknown} error @param {boolean} [always] */
  function failure(event, error, always = false) {
    if (!always && !debugEnabled(namespace)) return;
    // Do not use debugError: API errors may contain request bodies and tokens.
    const response = ownValue(error, "response");
    const status = ownValue(error, "status") ?? ownValue(response, "status");
    const message = ownValue(ownValue(response, "data"), "message");
    const boundedMessage = typeof message === "string" && message.length <= 8192 ? message : "";
    diagnostic(event, {
      failed: true,
      ...(Number.isInteger(status) && status >= 100 && status <= 599 ? { http_status: status } : {}),
      ...(status === 422
        ? {
            ref_not_found: boundedMessage.startsWith("No ref found for:"),
            unexpected_inputs: boundedMessage.startsWith("Unexpected inputs provided:"),
            dispatch_trigger_missing: boundedMessage === "Workflow does not have 'workflow_dispatch' trigger",
          }
        : {}),
    });
  }
  return { debug, diagnostic, failure };
}

module.exports = { createWorkQueueLogger, debugEnabled };
