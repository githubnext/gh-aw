// @ts-check
/// <reference types="@actions/github-script" />

const { buildWorkflowCallId } = require("./aw_context.cjs");
const { ERR_ADMISSION } = require("./error_codes.cjs");

const ADMISSION_TIMEOUT_MS = 10000;
const ADMISSION_MAX_ATTEMPTS = 3;
const ADMISSION_FAILURE = `${ERR_ADMISSION}: Workflow execution was not admitted. Contact the repository administrator.`;
const ADMISSION_OUTAGE = `${ERR_ADMISSION}: Admission check did not complete. Continuing under the configured outage policy.`;

function buildAdmissionURL() {
  const endpoint = process.env.GH_AW_ADMISSION_URL || "";
  let url;
  try {
    url = new URL(endpoint);
  } catch {
    throw new Error(ADMISSION_FAILURE);
  }
  if (!/^https:\/\/[^\s\\?#${}]+$/.test(endpoint) || !url.hostname || url.username || url.password) {
    throw new Error(ADMISSION_FAILURE);
  }
  const runId = String(context.runId ?? "");
  const runAttempt = process.env.GITHUB_RUN_ATTEMPT || "1";
  const workflowId = process.env.GH_AW_WORKFLOW_ID || "";
  // Never let caller-supplied aw_context replace the identity of this execution.
  const metadata = {
    repo: `${context.repo.owner}/${context.repo.repo}`,
    workflow_id: workflowId,
    workflow_call_id: buildWorkflowCallId(runId, runAttempt, workflowId),
    run_id: runId,
    run_attempt: runAttempt,
    ref: context.ref,
    event_type: context.eventName,
  };
  if (Object.values(metadata).some(value => typeof value !== "string" || !value)) {
    throw new Error(ADMISSION_FAILURE);
  }
  url.search = new URLSearchParams(metadata).toString();
  return url;
}

/**
 * @param {URL} url
 * @param {string} token
 * @returns {Promise<"admitted" | "denied" | "unavailable">}
 */
async function requestAdmission(url, token) {
  let response;
  try {
    response = await fetch(url, {
      method: "GET",
      headers: { Authorization: `Bearer ${token}`, "Cache-Control": "no-store" },
      redirect: "manual",
      cache: "no-store",
      signal: AbortSignal.timeout(ADMISSION_TIMEOUT_MS),
    });
  } catch {
    return "unavailable";
  }
  const result = response.status === 200 || response.status === 204 ? "admitted" : response.status === 408 || response.status === 429 || (response.status >= 500 && response.status <= 599) ? "unavailable" : "denied";
  try {
    // Discard bodies without parsing or logging service-controlled diagnostics.
    await response.body?.cancel();
  } catch {
    // A body cleanup failure must never turn an explicit denial into an outage.
    return result;
  }
  return result;
}

async function main() {
  try {
    const url = buildAdmissionURL();
    const token = process.env.GH_AW_ADMISSION_TOKEN;
    const onOutage = process.env.GH_AW_ADMISSION_ON_OUTAGE || "ignore";
    if (!token || !["ignore", "block"].includes(onOutage)) {
      throw new Error(ADMISSION_FAILURE);
    }
    for (let attempt = 0; attempt < ADMISSION_MAX_ATTEMPTS; attempt++) {
      const result = await requestAdmission(url, token);
      if (result === "admitted") return;
      if (result === "denied") throw new Error(ADMISSION_FAILURE);
      if (attempt < ADMISSION_MAX_ATTEMPTS - 1) {
        await new Promise(resolve => setTimeout(resolve, 250 * 2 ** attempt + Math.floor(Math.random() * 250)));
      }
    }
    if (onOutage === "ignore") {
      core.warning(ADMISSION_OUTAGE);
      return;
    }
    throw new Error(ADMISSION_FAILURE);
  } catch {
    // Policy denials and transport failures deliberately have the same public code.
    core.setFailed(ADMISSION_FAILURE);
  }
}

module.exports = { main };
