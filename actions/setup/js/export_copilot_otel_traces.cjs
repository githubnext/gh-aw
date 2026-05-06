// @ts-check
/// <reference types="@actions/github-script" />
"use strict";

const fs = require("fs");
const readline = require("readline");

const { parseOTLPEndpoints, sendOTLPToAllEndpoints, sanitizeOTLPPayload, appendToOTLPJSONL } = require("./send_otlp_span.cjs");

/**
 * @param {unknown} payload
 * @returns {payload is { resourceSpans: unknown[] }}
 */
function isOTLPPayload(payload) {
  return !!payload && typeof payload === "object" && Array.isArray(payload.resourceSpans);
}

/**
 * @param {import("@actions/core")} core
 * @returns {Promise<void>}
 */
async function main(core) {
  const sourcePath = process.env.COPILOT_OTEL_FILE_EXPORTER_PATH || "";
  if (!sourcePath) {
    core.info("COPILOT_OTEL_FILE_EXPORTER_PATH is not set; skipping Copilot OTEL trace export");
    return;
  }

  if (!fs.existsSync(sourcePath)) {
    core.info(`Copilot OTEL trace file not found at ${sourcePath}; skipping export`);
    return;
  }

  const endpoints = parseOTLPEndpoints();
  if (endpoints.length === 0) {
    core.info("GH_AW_OTLP_ENDPOINTS is not configured; skipping Copilot OTEL endpoint export");
    return;
  }

  let forwarded = 0;
  let malformed = 0;
  let ignored = 0;

  const stream = fs.createReadStream(sourcePath, { encoding: "utf8" });
  const lines = readline.createInterface({ input: stream, crlfDelay: Infinity });
  for await (const raw of lines) {
    const line = raw.trim();
    if (!line) continue;

    let payload;
    try {
      payload = JSON.parse(line);
    } catch {
      malformed++;
      continue;
    }

    if (!isOTLPPayload(payload)) {
      ignored++;
      continue;
    }

    const sanitized = sanitizeOTLPPayload(payload);
    appendToOTLPJSONL(sanitized);

    await sendOTLPToAllEndpoints(endpoints, payload, { skipJSONL: true });
    forwarded++;
  }

  core.info(`Copilot OTEL trace export complete: forwarded=${forwarded}, malformed=${malformed}, ignored=${ignored}, source=${sourcePath}`);
}

module.exports = { main, isOTLPPayload };
