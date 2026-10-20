// @ts-check
"use strict";

const { AdmissionError, readAdmissionConfig, buildAdmissionIdentity } = require("./admission_protocol.cjs");
const { requestAuthority } = require("./check_admission.cjs");
const { readCompactUnifiedSession } = require("./compact_unified_session.cjs");

const SESSION_FILE = "/tmp/gh-aw/usage/aw_session.jsonl";
const MAX_REQUEST_BYTES = 64 * 1024;
const FAILURE = "ERR_ADMISSION: Unable to deliver the unified session to the conclusion service; completed workflow results are unchanged.";

/** @param {NodeJS.ProcessEnv} env */
function readUsageArtifactIds(env) {
  const ids = [env.GH_AW_USAGE_ARTIFACT_RETRY_ID || "", env.GH_AW_USAGE_ARTIFACT_ID || ""].filter(Boolean);
  if (ids.some(id => !/^[1-9][0-9]{0,19}$/.test(id))) throw new AdmissionError();
  return [...new Set(ids)];
}

/**
 * @param {{env?: NodeJS.ProcessEnv, fetchImpl?: typeof fetch, now?: () => number, mask?: (token: string) => void, sessionFile?: string}} [options]
 * @returns {Promise<{result: "skipped" | "delivered" | "unavailable"}>}
 */
async function sendConclusion({ env = process.env, sessionFile = SESSION_FILE, ...options } = {}) {
  const config = readAdmissionConfig(env);
  if (!config) return { result: "skipped" };
  const identity = JSON.parse(buildAdmissionIdentity(env));
  const usageArtifactIds = readUsageArtifactIds(env);
  const { version, session, includedEvents, omittedEvents, oversizedLines, scanTruncated, truncated } = readCompactUnifiedSession(sessionFile);
  const body = JSON.stringify({
    ...identity,
    session,
    session_compaction: { version, included_events: includedEvents, omitted_events: omittedEvents, oversized_lines: oversizedLines, scan_truncated: scanTruncated, truncated },
    usage_artifact_ids: usageArtifactIds,
  });
  if (Buffer.byteLength(body) > MAX_REQUEST_BYTES) throw new AdmissionError();
  const endpoint = config.endpoint.replace(/\/api\/admission$/, "/api/conclusion");
  const { decision } = await requestAuthority({
    ...options,
    env,
    config: { ...config, endpoint },
    body,
    acceptedStatuses: [202, 204],
    parseResponse: async response => {
      void response.body?.cancel().catch(() => {
        // Discarded response bytes cannot change accepted delivery.
      });
      return true;
    },
  });
  return { result: decision ? "delivered" : "unavailable" };
}

/** @param {typeof import("@actions/core")} core */
async function main(core) {
  try {
    const result = await sendConclusion({ mask: token => core.setSecret(token) });
    if (result.result === "unavailable") core.warning(FAILURE);
    else if (result.result === "delivered") core.info("Admission conclusion: compact unified session accepted for processing.");
  } catch {
    core.warning(FAILURE);
  }
}

module.exports = { sendConclusion, main, SESSION_FILE, MAX_REQUEST_BYTES, FAILURE };
