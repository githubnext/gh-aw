// @ts-check

/**
 * API Proxy Event Log Helpers
 *
 * This module provides functions to read and parse the API proxy event log
 * (api-proxy-logs/events.jsonl) to determine the cause of errors during agent execution.
 *
 * The API proxy event log is the authoritative source for rate-limiting and other API errors,
 * providing more reliable detection than pattern matching on agent stdout/stderr.
 *
 * Key responsibilities:
 *   - Locating the API proxy events.jsonl file in the firewall logs directory
 *   - Parsing events to detect rate-limiting errors (HTTP 429)
 *   - Detecting max-runs-exceeded errors from the proxy budget enforcement
 *   - Providing structured error classification for retry logic
 */

"use strict";

const fs = require("fs");
const path = require("path");

// Path to API proxy events log relative to firewall logs directory
const API_PROXY_EVENTS_PATH = "api-proxy-logs/events.jsonl";

// Default firewall logs directory
const DEFAULT_FIREWALL_LOGS_DIR = "/tmp/gh-aw/sandbox/firewall/logs";

/**
 * Locates the API proxy events.jsonl file.
 * Checks the standard location first, then falls back to searching for legacy artifact directories.
 * @param {string} [firewallLogsDir] - Optional firewall logs directory path
 * @returns {string} Path to events.jsonl or empty string if not found
 */
function findAPIProxyEventsFile(firewallLogsDir = DEFAULT_FIREWALL_LOGS_DIR) {
  // Try primary location
  const primaryPath = path.join(firewallLogsDir, API_PROXY_EVENTS_PATH);
  if (fs.existsSync(primaryPath)) {
    return primaryPath;
  }

  // Legacy fallback: search for firewall-audit-logs or firewall-logs directories
  const parentDir = path.dirname(firewallLogsDir);
  if (!fs.existsSync(parentDir)) {
    return "";
  }

  try {
    const entries = fs.readdirSync(parentDir);
    for (const entry of entries) {
      if (entry.startsWith("firewall-audit-logs") || entry.startsWith("firewall-logs")) {
        const candidatePath = path.join(parentDir, entry, API_PROXY_EVENTS_PATH);
        if (fs.existsSync(candidatePath)) {
          return candidatePath;
        }
      }
    }
  } catch (err) {
    // Ignore filesystem errors during fallback search
  }

  return "";
}

/**
 * Error classification result from API proxy event log analysis
 * @typedef {Object} ProxyErrorClassification
 * @property {boolean} hasRateLimitError - True if a rate-limit error (HTTP 429) was detected
 * @property {boolean} hasMaxRunsExceeded - True if max-runs budget was exhausted
 * @property {boolean} hasOverloadError - True if an overload error (HTTP 529) was detected
 * @property {number} totalErrors - Total number of error events in the log
 * @property {Array<string>} errorTypes - List of unique error types encountered
 */

/**
 * Parses the API proxy events.jsonl file and classifies errors.
 * @param {string} eventsPath - Path to the events.jsonl file
 * @param {(message: string) => void} [logger] - Optional logger function
 * @returns {ProxyErrorClassification} Error classification result
 */
function parseAPIProxyEvents(eventsPath, logger) {
  /** @type {ProxyErrorClassification} */
  const classification = {
    hasRateLimitError: false,
    hasMaxRunsExceeded: false,
    hasOverloadError: false,
    totalErrors: 0,
    errorTypes: [],
  };

  if (!eventsPath || !fs.existsSync(eventsPath)) {
    if (logger) {
      logger(`API proxy events file not found: ${eventsPath || "(no path)"}`);
    }
    return classification;
  }

  try {
    const content = fs.readFileSync(eventsPath, "utf8");
    const lines = content.split("\n");
    const errorTypesSet = new Set();

    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed) {
        continue;
      }

      let event;
      try {
        event = JSON.parse(trimmed);
      } catch (err) {
        // Skip malformed lines
        continue;
      }

      // Extract event type/name using multiple possible field names
      const eventType = (event.event || event.type || event.event_name || event.eventName || "").toString().toLowerCase();
      const message = (event.message || "").toString();
      const statusCode = event.status_code || event.statusCode || event.status;

      // Detect rate-limiting errors (HTTP 429)
      if (
        statusCode === 429 ||
        eventType.includes("rate_limit") ||
        message.toLowerCase().includes("rate limit") ||
        message.includes("429")
      ) {
        classification.hasRateLimitError = true;
        errorTypesSet.add("rate_limit");
        classification.totalErrors++;
      }
      // Detect overload errors (HTTP 529)
      else if (
        statusCode === 529 ||
        eventType.includes("overload") ||
        message.toLowerCase().includes("overload")
      ) {
        classification.hasOverloadError = true;
        errorTypesSet.add("overload");
        classification.totalErrors++;
      }
      // Detect max-runs-exceeded errors
      else if (
        eventType.includes("max_runs") ||
        eventType.includes("budget_exceeded") ||
        message.includes("max_runs_exceeded") ||
        message.includes("Maximum LLM invocations exceeded")
      ) {
        classification.hasMaxRunsExceeded = true;
        errorTypesSet.add("max_runs_exceeded");
        classification.totalErrors++;
      }
      // Count other error-level events
      else if (
        eventType.includes("error") ||
        event.level === "error" ||
        event.severity === "error"
      ) {
        classification.totalErrors++;
        if (eventType) {
          errorTypesSet.add(eventType);
        }
      }
    }

    classification.errorTypes = Array.from(errorTypesSet);

    if (logger) {
      logger(
        `Parsed API proxy events: hasRateLimitError=${classification.hasRateLimitError}, ` +
        `hasMaxRunsExceeded=${classification.hasMaxRunsExceeded}, ` +
        `hasOverloadError=${classification.hasOverloadError}, ` +
        `totalErrors=${classification.totalErrors}`
      );
    }
  } catch (err) {
    if (logger) {
      logger(`Failed to parse API proxy events file: ${err.message}`);
    }
  }

  return classification;
}

/**
 * Convenience function to find and parse API proxy events in one call.
 * @param {string} [firewallLogsDir] - Optional firewall logs directory path
 * @param {(message: string) => void} [logger] - Optional logger function
 * @returns {ProxyErrorClassification} Error classification result
 */
function checkAPIProxyErrors(firewallLogsDir, logger) {
  const eventsPath = findAPIProxyEventsFile(firewallLogsDir);
  if (!eventsPath) {
    if (logger) {
      logger("API proxy events file not found");
    }
    return {
      hasRateLimitError: false,
      hasMaxRunsExceeded: false,
      hasOverloadError: false,
      totalErrors: 0,
      errorTypes: [],
    };
  }

  if (logger) {
    logger(`Found API proxy events file: ${eventsPath}`);
  }

  return parseAPIProxyEvents(eventsPath, logger);
}

module.exports = {
  findAPIProxyEventsFile,
  parseAPIProxyEvents,
  checkAPIProxyErrors,
  DEFAULT_FIREWALL_LOGS_DIR,
  API_PROXY_EVENTS_PATH,
};
