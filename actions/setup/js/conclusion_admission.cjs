// @ts-check
"use strict";

const fs = require("node:fs");
const { AdmissionError, readAdmissionConfig, buildAdmissionIdentity } = require("./admission_protocol.cjs");
const { requestAuthority } = require("./check_admission.cjs");

const SESSION_FILE = "/tmp/gh-aw/usage/aw_session.jsonl";
const MAX_REQUEST_BYTES = 64 * 1024 * 1024;
const FAILURE = "ERR_ADMISSION: Unable to deliver the unified session to the conclusion service; completed workflow results are unchanged.";

/**
 * @param {{env?: NodeJS.ProcessEnv, fetchImpl?: typeof fetch, now?: () => number, mask?: (token: string) => void, sessionFile?: string}} [options]
 * @returns {Promise<{result: "skipped" | "delivered" | "unavailable"}>}
 */
async function sendConclusion({ env = process.env, sessionFile = SESSION_FILE, ...options } = {}) {
  const config = readAdmissionConfig(env);
  if (!config) return { result: "skipped" };
  const identity = JSON.parse(buildAdmissionIdentity(env));
  const fd = fs.openSync(sessionFile, fs.constants.O_RDONLY | fs.constants.O_NOFOLLOW | fs.constants.O_NONBLOCK);
  let session;
  try {
    const stat = fs.fstatSync(fd);
    if (!stat.isFile() || stat.size === 0 || stat.size > MAX_REQUEST_BYTES) throw new AdmissionError();
    const chunks = [];
    let size = 0;
    while (true) {
      const chunk = Buffer.alloc(65536);
      const read = fs.readSync(fd, chunk, 0, chunk.length, null);
      if (read === 0) break;
      size += read;
      if (size > MAX_REQUEST_BYTES) throw new AdmissionError();
      chunks.push(chunk.subarray(0, read));
    }
    if (size === 0) throw new AdmissionError();
    session = new TextDecoder("utf-8", { fatal: true, ignoreBOM: true }).decode(Buffer.concat(chunks));
  } finally {
    fs.closeSync(fd);
  }
  const body = JSON.stringify({ ...identity, session });
  if (Buffer.byteLength(body) > MAX_REQUEST_BYTES) throw new AdmissionError();
  const endpoint = new URL(config.endpoint);
  endpoint.pathname = "/api/conclusion";
  const { decision } = await requestAuthority({
    ...options,
    env,
    config: { ...config, endpoint: endpoint.href },
    body,
    acceptedStatuses: [202, 204],
    parseResponse: async response => {
      void response.body?.cancel().catch(() => {});
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
    else if (result.result === "delivered") core.info("Admission conclusion: unified session accepted for processing.");
  } catch {
    core.warning(FAILURE);
  }
}

module.exports = { sendConclusion, main, SESSION_FILE, MAX_REQUEST_BYTES, FAILURE };
