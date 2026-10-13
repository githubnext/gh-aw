// @ts-check
"use strict";

const log = require("./work_queue_logging.cjs").createWorkQueueLogger("workflow-dispatch");
const { getErrorMessage } = require("./error_helpers.cjs");

async function dispatchWorkflow(githubClient, parameters, { allowLegacyFallback = false, coreApi = global.core } = {}) {
  const startedAt = Date.now();
  log.diagnostic("dispatch.post.start", { named_ref: parameters.ref.startsWith("refs/"), input_count: Object.keys(parameters.inputs || {}).length });
  let response;
  try {
    response = await githubClient.rest.actions.createWorkflowDispatch(parameters);
  } catch (error) {
    log.failure("dispatch.post.failed", error, true);
    const status = error && typeof error === "object" && "status" in error ? error.status : undefined;
    if (!allowLegacyFallback || parameters.return_run_details !== true || ![400, 422].includes(status)) throw error;
    const message = error?.response?.data?.message !== undefined ? String(error.response.data.message) : getErrorMessage(error);
    if (!message.toLowerCase().includes("return_run_details")) throw error;
    coreApi.info("Workflow dispatch failed due to unsupported 'return_run_details' parameter; retrying without it for GitHub Enterprise compatibility.");
    const { return_run_details, ...legacyParameters } = parameters;
    try {
      response = await githubClient.rest.actions.createWorkflowDispatch(legacyParameters);
    } catch (legacyError) {
      log.failure("dispatch.legacy_post.failed", legacyError, true);
      throw legacyError;
    }
  }
  log.diagnostic("dispatch.post.accepted", { elapsed_ms: Math.max(0, Date.now() - startedAt), ...(Number.isInteger(response.status) ? { http_status: response.status } : {}) });
  return response;
}

module.exports = { dispatchWorkflow };
