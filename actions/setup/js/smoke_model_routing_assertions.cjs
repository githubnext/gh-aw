// @ts-check
"use strict";

/**
 * Deterministic model-routing smoke assertions.
 *
 * Smoke workflows call this from a `post-steps` github-script step. Every check
 * reads runner- and proxy-written evidence (the unified session and the
 * api-proxy token-usage log), never the agent's own reply. The agent-writable
 * `agent/awf-routing-outcome.json` (`model_routing.outcome`) is never evidence.
 *
 * Request attribution follows what AWF records: a token-usage record with
 * `purpose: "routing_classification"` is the classifier; every other record is
 * agent traffic (AWF writes no `purpose` for agent requests, and `x_initiator`
 * is only the billing class). Agent traffic is attributed to a declared
 * sub-agent when the session correlates it (request IDs on `subagent.*` events,
 * or sub-agent IDs on proxy records); otherwise by model: requests on a declared
 * sub-agent's model belong to that sub-agent, and all other agent traffic
 * belongs to the main agent. Declared sub-agent models must therefore be
 * distinct from each other and disjoint from `allowedModels`.
 *
 * Check IDs:
 * - C1 configuration: declared sub-agent models are distinct and not in
 *   allowed-models
 * - E1 agent execution outcome; E2 evidence files present and parseable
 * - R1 routing selected; R2 one classifier call; R3 main-agent traffic on the
 *   selected model and a supported endpoint; R4 main-agent requests exist and
 *   use only allowed models
 * - M1 main-agent requests exist and use the expected endpoint (when configured)
 * - S1 sub-agent ran exactly once; S2 completed without failing; S3 status 200 on
 *   the declared model and endpoint; S4 events carry the declared agent name
 *   (when configured)
 *
 * Every check that iterates over requests fails when there are none to check.
 *
 * Threat-detection checks (`detectionMain`, run from
 * `safe-outputs.threat-detection.post-steps` in the detection job) read the
 * detection run's own evidence under `/tmp/gh-aw/threat-detection`:
 * - D1 detection execution outcome is success and the structured result file
 *   parses with the verdict keys (`execution.json` is not completion evidence)
 * - D2 detection requests exist, all on the configured detection model, which is
 *   disjoint from allowed-models
 * - D3 no routing_classification request, no model-routing selection, and no
 *   model-routing block in the detection's resolved AWF config
 */

const fs = require("fs");
const path = require("path");
const { getErrorMessage } = require("./error_helpers.cjs");

const DEFAULT_ROOT_DIR = "/tmp/gh-aw";
const SESSION_PATH = "usage/aw_session.jsonl";
const TOKEN_USAGE_PATHS = ["sandbox/firewall/logs/api-proxy-logs/token-usage.jsonl", "sandbox/firewall/audit/api-proxy-logs/token-usage.jsonl", "sandbox/firewall-audit-logs/api-proxy-logs/token-usage.jsonl"];
const REFLECT_PATHS = ["sandbox/firewall/awf-reflect.json", "agent/awf-reflect.json"];
const CLASSIFIER_PURPOSE = "routing_classification";
const MAX_OBSERVED = 12;

/**
 * @typedef {{ name: string, model: string, endpoint: string | string[] }} SubAgentExpectation
 * @typedef {{
 *   allowedModels: string[],
 *   mainEndpoint?: string,
 *   subAgents?: SubAgentExpectation[],
 *   requireDeclaredAgentNames?: boolean,
 *   executionOutcome?: string,
 * }} RoutingExpectations
 * @typedef {{ passes: string[], failures: string[] }} RoutingAssertionResults
 */

/**
 * Normalize provider-qualified and dated model IDs to the served model name.
 * @param {unknown} model
 * @returns {string}
 */
function servedModel(model) {
  return String(model || "")
    .trim()
    .replace(/^.*\//, "")
    .replace(/-[0-9]{4}(?:-[0-9]{2}-[0-9]{2}|[0-9]{4})$/, "")
    .replace(/^(claude-.+)-([0-9]+)-([0-9]+)$/, "$1-$2.$3");
}

/**
 * Drop query strings (for example `/v1/messages?beta=true`) and trailing slashes.
 * @param {unknown} endpoint
 * @returns {string}
 */
function normalizeEndpoint(endpoint) {
  return String(endpoint || "")
    .split("?")[0]
    .replace(/(.)\/+$/, "$1");
}

/** @param {any} request @returns {string} */
function requestEndpoint(request) {
  return normalizeEndpoint(request?.path ?? request?.endpoint);
}

/** @param {any} request @returns {string} */
function describeRequest(request) {
  const status = request?.status ?? "<no status>";
  return `${servedModel(request?.model) || "<no model>"} ${requestEndpoint(request) || "<no endpoint>"} ${status}`;
}

/** @param {any[]} requests @returns {string} */
function describeRequests(requests) {
  if (!requests.length) return "none";
  const shown = requests.slice(0, MAX_OBSERVED).map(describeRequest).join(", ");
  return requests.length > MAX_OBSERVED ? `${shown}, … (${requests.length} total)` : shown;
}

/** @param {any} event @returns {string} */
function describeEvent(event) {
  const data = event?.data ?? {};
  const details = [`agentName=${data.agentName ?? "<none>"}`];
  for (const key of ["model", "resolvedModel", "outcome", "errorCode"]) {
    if (data[key] !== undefined) details.push(`${key}=${data[key]}`);
  }
  return `${event?.type ?? "<no type>"}(${details.join(" ")})`;
}

/** @param {any[]} events @returns {string} */
function describeEvents(events) {
  if (!events.length) return "none";
  const shown = events.slice(0, MAX_OBSERVED).map(describeEvent).join(", ");
  return events.length > MAX_OBSERVED ? `${shown}, … (${events.length} total)` : shown;
}

/** @param {any} event @returns {boolean} */
function isClassifierRequest(event) {
  return event?.purpose === CLASSIFIER_PURPOSE;
}

/**
 * @param {any} value
 * @returns {string}
 */
function idString(value) {
  return value === undefined || value === null ? "" : String(value);
}

/** @param {any} request @returns {string} */
function requestId(request) {
  return idString(request?.request_id ?? request?.requestId);
}

/** @param {any} request @returns {string} */
function requestAgentId(request) {
  return idString(request?.agent_id ?? request?.agentId);
}

/** @param {any} event @returns {string} */
function eventRequestId(event) {
  return idString(event?.data?.requestId ?? event?.data?.request_id ?? event?.requestId ?? event?.request_id);
}

/**
 * Supported endpoints for a model, according to the routing selection and the
 * AWF `/reflect` routing metadata.
 * @param {string} model
 * @param {any} selection
 * @param {any} reflect
 * @returns {Set<string>}
 */
function supportedEndpoints(model, selection, reflect) {
  const endpoints = new Set();
  const reflectEndpoints = Array.isArray(reflect?.endpoints) ? reflect.endpoints : [];
  for (const endpoint of reflectEndpoints) {
    for (const entry of Array.isArray(endpoint?.routing_models) ? endpoint.routing_models : []) {
      if (servedModel(entry?.model_id) !== model || !Array.isArray(entry?.supported_endpoints)) continue;
      for (const value of entry.supported_endpoints) endpoints.add(normalizeEndpoint(value));
    }
  }
  // The selection's endpoint is used only when /reflect has no metadata for the model.
  if (!endpoints.size && selection?.endpoint) endpoints.add(normalizeEndpoint(selection.endpoint));
  endpoints.delete("");
  return endpoints;
}

/**
 * Evaluate model-routing evidence against the smoke expectations.
 * @param {{ events: any[], requests: any[], reflect?: any, expectations: RoutingExpectations }} input
 * @returns {RoutingAssertionResults}
 */
function evaluateModelRoutingEvidence({ events, requests, reflect = null, expectations }) {
  /** @type {RoutingAssertionResults} */
  const results = { passes: [], failures: [] };
  /** @param {string} id @param {string} message */
  const pass = (id, message) => results.passes.push(`PASS ${id} ${message}`);
  /** @param {string} id @param {string} message */
  const fail = (id, message) => results.failures.push(`FAIL ${id} ${message}`);

  const allowedList = [...new Set((expectations.allowedModels ?? []).map(servedModel).filter(Boolean))];
  const allowed = new Set(allowedList);
  const subAgents = expectations.subAgents ?? [];

  // C1: sub-agent models must be disjoint from the routing candidates and from
  // each other, otherwise requests cannot be attributed by model.
  if (subAgents.length) {
    const c1Problems = [];
    const overlapping = subAgents.filter(agent => allowed.has(servedModel(agent.model)));
    if (overlapping.length)
      c1Problems.push(
        `declared sub-agent model(s) also in allowed-models [${allowedList.join(", ")}]: ${overlapping.map(agent => `${agent.name}=${servedModel(agent.model)}`).join(", ")}; sub-agent models must be disjoint from the routing candidates`
      );
    /** @type {Map<string, string[]>} */
    const namesByModel = new Map();
    for (const agent of subAgents) {
      const model = servedModel(agent.model);
      namesByModel.set(model, [...(namesByModel.get(model) ?? []), agent.name]);
    }
    const shared = [...namesByModel].filter(([, names]) => names.length > 1);
    if (shared.length) c1Problems.push(`declared sub-agents share a model: ${shared.map(([model, names]) => `${model} (${names.join(", ")})`).join(", ")}; each sub-agent must declare a distinct model`);
    if (c1Problems.length) fail("C1", c1Problems.join("; "));
    else pass("C1", `declared sub-agent models are distinct and disjoint from allowed-models [${allowedList.join(", ")}]`);
  }

  if (expectations.executionOutcome !== undefined) {
    if (expectations.executionOutcome === "success") pass("E1", "agent execution outcome is success");
    else fail("E1", `agent execution outcome is ${expectations.executionOutcome || "<empty>"}, expected success`);
  }

  // R1: runner-written routing status and a firewall selection record.
  const workflowInfo = events.filter(event => event?.type === "workflow.info" && event.provenance?.component === "workflow" && !String(event.provenance?.path ?? "").startsWith("agent/"));
  const runnerRouting = workflowInfo.map(event => event.data?.modelRouting).find(routing => routing && typeof routing.status === "string");
  const runnerStatus = runnerRouting?.status;
  const selections = events.filter(event => event?.type === "firewall.model_routing" && event.provenance?.component === "firewall" && event.data?.stage === "selection");
  const selection = selections.filter(event => event.data?.selectedModel).at(-1)?.data;
  const selectedModel = servedModel(selection?.selectedModel);
  const harnessOutcomes = events.filter(event => event?.type === "model_routing.outcome");
  const r1Problems = [];
  if (runnerStatus !== "selected") r1Problems.push(`runner-written routing status is ${runnerStatus ?? "missing"}, expected selected`);
  if (!selection) r1Problems.push(`no firewall model_routing selection record with a selected model (observed ${selections.length} selection record(s))`);
  else if (!allowed.has(selectedModel)) r1Problems.push(`selected model ${selectedModel} is outside allowed-models [${allowedList.join(", ")}]`);
  if (r1Problems.length) {
    const ignored = harnessOutcomes.length ? `; ignored agent-written model_routing.outcome (status=${harnessOutcomes.map(event => event.data?.status ?? "<none>").join(", ")})` : "";
    fail("R1", `routing not selected: ${r1Problems.join("; ")}${ignored}`);
  } else {
    pass("R1", `routing selected ${selectedModel} (runner status=${runnerStatus}, endpoint=${normalizeEndpoint(selection?.endpoint) || "<none>"})`);
  }

  // R2: exactly one classifier request.
  const classifierRequests = requests.filter(isClassifierRequest);
  if (classifierRequests.length === 1) pass("R2", `one routing_classification request: ${describeRequest(classifierRequests[0])}`);
  else fail("R2", `expected exactly 1 routing_classification request, observed ${classifierRequests.length}: ${describeRequests(classifierRequests)}`);

  // Every non-classifier request is agent traffic; AWF does not record which
  // agent made it.
  const agentTraffic = requests.filter(request => !isClassifierRequest(request));

  const subagentEvents = events.filter(event => typeof event?.type === "string" && event.type.startsWith("subagent."));
  const startedEvents = subagentEvents.filter(event => event.type === "subagent.started");
  const observedNames = [...new Set(startedEvents.map(event => event.data?.agentName ?? "<none>"))];

  // Attribute agent traffic to declared sub-agents: prefer session correlation
  // (request IDs on sub-agent events, or sub-agent IDs on proxy records), and
  // fall back to the declared model.
  const subAgentEvidence = subAgents.map(agent => {
    const started = startedEvents.filter(event => event.data?.agentName === agent.name);
    const ids = new Set(started.map(event => idString(event.data?.invocationId ?? event.agentId)).filter(Boolean));
    const activity = subagentEvents.filter(event => ids.has(idString(event.data?.invocationId)) || ids.has(idString(event.agentId)));
    const eventRequestIds = new Set(activity.map(eventRequestId).filter(Boolean));
    const correlated = agentTraffic.filter(request => eventRequestIds.has(requestId(request)) || ids.has(requestAgentId(request)));
    return { agent, model: servedModel(agent.model), started, activity, correlated };
  });
  const correlatedRequests = new Set(subAgentEvidence.flatMap(evidence => evidence.correlated));
  const attributed = subAgentEvidence.map(evidence => (evidence.correlated.length ? evidence.correlated : agentTraffic.filter(request => !correlatedRequests.has(request) && servedModel(request.model) === evidence.model)));
  const subagentRequests = new Set(attributed.flat());
  const mainRequests = agentTraffic.filter(request => !subagentRequests.has(request));
  const observedMain = mainRequests.length ? describeRequests(mainRequests) : `none (agent requests: ${describeRequests(agentTraffic)})`;

  // R3: main-agent traffic on the selected model and a supported endpoint.
  if (!selectedModel) {
    fail("R3", `no selected model to verify; observed main-agent requests: ${observedMain}`);
  } else {
    const endpoints = supportedEndpoints(selectedModel, selection, reflect);
    const onModel = mainRequests.filter(request => servedModel(request.model) === selectedModel);
    if (!endpoints.size) {
      fail("R3", `no supported-endpoint metadata for ${selectedModel} in the routing selection or /reflect; observed: ${onModel.length ? describeRequests(onModel) : observedMain}`);
    } else {
      const ok = onModel.find(request => Number(request.status) === 200 && endpoints.has(requestEndpoint(request)));
      if (ok) pass("R3", `main-agent request on selected model: ${describeRequest(ok)}`);
      else
        fail(
          "R3",
          `no 200 request for ${selectedModel} on a supported endpoint (${[...endpoints].join(", ")}); observed: ${onModel.length ? describeRequests(onModel) : `no ${selectedModel} requests; main-agent requests: ${observedMain}`}`
        );
    }
  }

  // R4: main-agent requests must use allowed-models; sub-agent models are scoped
  // to their own checks and cannot widen the main agent's model policy.
  const outOfPolicy = mainRequests.filter(request => !allowed.has(servedModel(request.model)));
  if (!mainRequests.length) fail("R4", `no main-agent requests to check against allowed-models [${allowedList.join(", ")}]; observed agent requests: ${describeRequests(agentTraffic)}`);
  else if (outOfPolicy.length) fail("R4", `main-agent request(s) on models outside allowed-models [${allowedList.join(", ")}]: ${describeRequests(outOfPolicy)}`);
  else pass("R4", `all ${mainRequests.length} main-agent request(s) used allowed models`);

  // M1: main-agent requests use the expected endpoint.
  if (expectations.mainEndpoint) {
    const expected = normalizeEndpoint(expectations.mainEndpoint);
    const wrong = mainRequests.filter(request => requestEndpoint(request) !== expected);
    if (!mainRequests.length) fail("M1", `no main-agent requests to check for ${expected}; observed agent requests: ${describeRequests(agentTraffic)}`);
    else if (wrong.length) fail("M1", `main-agent request(s) not on ${expected}; observed: ${describeRequests(wrong)}`);
    else pass("M1", `all ${mainRequests.length} main-agent request(s) used ${expected}`);
  }

  for (const [index, { agent, model, started, activity, correlated }] of subAgentEvidence.entries()) {
    const endpoints = (Array.isArray(agent.endpoint) ? agent.endpoint : [agent.endpoint]).map(normalizeEndpoint);
    const label = `sub-agent ${agent.name}`;

    // S1: ran exactly once.
    if (started.length === 1) pass("S1", `${label} started once`);
    else fail("S1", `${label}: expected exactly 1 subagent.started, observed ${started.length}; observed agent names: [${observedNames.join(", ")}]`);

    // S2: completed and did not fail.
    const completed = activity.filter(event => event.type === "subagent.completed");
    const failed = activity.filter(event => event.type === "subagent.failed");
    if (!started.length) fail("S2", `${label}: never started; observed events: ${describeEvents(subagentEvents)}`);
    else if (!completed.length || failed.length) fail("S2", `${label}: completed=${completed.length} failed=${failed.length}; observed events: ${describeEvents(activity)}`);
    else pass("S2", `${label} completed`);

    // S3: sub-agent traffic on the declared model and endpoint.
    const related = attributed[index];
    const onModel = related.filter(request => servedModel(request.model) === model);
    const ok = onModel.find(request => Number(request.status) === 200 && endpoints.includes(requestEndpoint(request)));
    const recorded = activity.flatMap(event => [event.data?.model, event.data?.selectedModel, event.data?.resolvedModel, event.data?.firstDispatchedModel]).filter(value => typeof value === "string" && value !== "");
    const mismatched = [...new Set(recorded.filter(value => servedModel(value) !== model))];
    const s3Problems = [];
    if (!ok) s3Problems.push(`no 200 request for ${model} on ${endpoints.join(" or ")}`);
    if (mismatched.length) s3Problems.push(`recorded model(s) [${mismatched.join(", ")}] differ from declared ${model}`);
    if (s3Problems.length) {
      const observed = onModel.length
        ? onModel.map(request => `${requestEndpoint(request) || "<no endpoint>"} ${request.status ?? "<no status>"}`).join(", ")
        : `no ${model} requests; ${correlated.length ? `requests correlated to ${label}: ${describeRequests(correlated)}` : `agent requests: ${describeRequests(agentTraffic)}`}`;
      fail("S3", `${label}: ${s3Problems.join("; ")}; observed: ${observed}`);
    } else {
      pass("S3", `${label} request ${describeRequest(ok)}`);
    }

    // S4: events carry the declared agent name.
    if (expectations.requireDeclaredAgentNames) {
      const renamed = activity.filter(event => event.data?.agentName !== undefined && event.data.agentName !== agent.name);
      if (!started.length) fail("S4", `${label}: no events carry the declared agent name; observed agent names: [${observedNames.join(", ")}]`);
      else if (renamed.length) fail("S4", `${label}: events carry agent name(s) other than the declared name; observed: ${describeEvents(renamed)}`);
      else pass("S4", `${label} events carry the declared agent name`);
    }
  }

  if (expectations.requireDeclaredAgentNames) {
    const declared = new Set(subAgents.map(agent => agent.name));
    const undeclared = startedEvents.filter(event => !declared.has(event.data?.agentName));
    if (undeclared.length) fail("S4", `sub-agent events carry undeclared agent name(s); declared: [${[...declared].join(", ")}]; observed: ${describeEvents(undeclared)}`);
  }

  return results;
}

/**
 * @param {string} file
 * @param {string[]} problems
 * @returns {any[]}
 */
function readJsonl(file, problems) {
  const records = [];
  const lines = fs.readFileSync(file, "utf8").split("\n");
  for (const [index, line] of lines.entries()) {
    if (!line.trim()) continue;
    try {
      records.push(JSON.parse(line));
    } catch (error) {
      problems.push(`${file}:${index + 1} is not valid JSON: ${getErrorMessage(error)}`);
    }
  }
  return records;
}

/**
 * Collect evidence from the gh-aw temp directory. Post-steps run before the
 * usage-artifact step writes aw_session.jsonl, so the unified session is built first.
 * @param {{ rootDir?: string, engine?: string, buildSession?: boolean }} [options]
 * @returns {{ events: any[], requests: any[], reflect: any, problems: string[] }}
 */
function collectModelRoutingEvidence({ rootDir = DEFAULT_ROOT_DIR, engine, buildSession = true } = {}) {
  /** @type {string[]} */
  const problems = [];
  if (buildSession) {
    try {
      const { writeUnifiedSession } = require("./unified_session.cjs");
      writeUnifiedSession({ rootDir, ...(engine ? { engine } : {}) });
    } catch (error) {
      problems.push(`failed to build the unified session: ${getErrorMessage(error)}`);
    }
  }

  let events = [];
  const sessionFile = path.join(rootDir, SESSION_PATH);
  if (fs.existsSync(sessionFile)) events = readJsonl(sessionFile, problems);
  else problems.push(`missing ${SESSION_PATH}`);

  let requests = [];
  const tokenFile = TOKEN_USAGE_PATHS.map(file => path.join(rootDir, file)).find(file => fs.existsSync(file));
  if (tokenFile) {
    requests = readJsonl(tokenFile, problems).filter(record => record && typeof record === "object" && !Array.isArray(record) && (record.event === undefined || record.event === "token_usage"));
  } else {
    problems.push(`missing api-proxy token-usage.jsonl (looked in ${TOKEN_USAGE_PATHS.join(", ")})`);
  }

  let reflect = null;
  const reflectFile = REFLECT_PATHS.map(file => path.join(rootDir, file)).find(file => fs.existsSync(file));
  if (reflectFile) {
    try {
      reflect = JSON.parse(fs.readFileSync(reflectFile, "utf8"));
    } catch (error) {
      problems.push(`${reflectFile} is not valid JSON: ${getErrorMessage(error)}`);
    }
  }
  return { events, requests, reflect, problems };
}

/**
 * Collect evidence and evaluate all checks.
 * @param {RoutingExpectations & { rootDir?: string, engine?: string, buildSession?: boolean }} options
 * @returns {RoutingAssertionResults}
 */
function checkModelRoutingEvidence(options) {
  const { rootDir, engine, buildSession, ...expectations } = options;
  const evidence = collectModelRoutingEvidence({ rootDir, engine, buildSession });
  const results = evaluateModelRoutingEvidence({ events: evidence.events, requests: evidence.requests, reflect: evidence.reflect, expectations });
  if (evidence.problems.length) results.failures.unshift(...evidence.problems.map(problem => `FAIL E2 evidence: ${problem}`));
  else results.passes.unshift("PASS E2 evidence: unified session and api-proxy token usage present");
  return results;
}

/**
 * Post-step entry point: logs every check and fails the step on any failure.
 * @param {RoutingExpectations & { rootDir?: string, engine?: string, buildSession?: boolean, core?: any }} options
 * @returns {Promise<RoutingAssertionResults>}
 */
async function main(options) {
  const { core: actionsCore = global.core, ...rest } = options;
  const results = checkModelRoutingEvidence(rest);
  for (const line of results.passes) actionsCore.info(line);
  for (const line of results.failures) actionsCore.error(line);
  if (results.failures.length) {
    actionsCore.setFailed(`Model-routing smoke assertions failed (${results.failures.length}):\n${results.failures.join("\n")}`);
  } else {
    actionsCore.info(`All ${results.passes.length} model-routing smoke assertions passed`);
  }
  return results;
}

const DETECTION_ROOT_DIR = "/tmp/gh-aw/threat-detection";
const DETECTION_RESULT_PATH = "detection_result.json";
const DETECTION_VERDICT_FLAGS = ["prompt_injection", "secret_leak", "malicious_patch"];
const DETECTION_VERDICT_LISTS = ["reasons", "warnings"];
const MODEL_ROUTING_LOG = "model-routing.jsonl";
const RESOLVED_CONFIG_PATH = "sandbox/firewall/audit/awf-resolved-config.json";

/**
 * @typedef {{ detectionModel: string, allowedModels: string[], executionOutcome?: string }} DetectionExpectations
 * @typedef {{ D1?: string[], D2?: string[], D3?: string[] }} DetectionEvidenceProblems
 */

/** @param {any} value @returns {boolean} */
function isPlainObject(value) {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

/**
 * Evaluate the threat-detection run's own evidence: detection must complete on
 * its configured model and must not be routed.
 * @param {{ result?: any, requests: any[], routingEvents?: any[], resolvedConfig?: any, problems?: DetectionEvidenceProblems, expectations: DetectionExpectations }} input
 * @returns {RoutingAssertionResults}
 */
function evaluateDetectionEvidence({ result, requests, routingEvents = [], resolvedConfig, problems = {}, expectations }) {
  /** @type {RoutingAssertionResults} */
  const results = { passes: [], failures: [] };
  /** @param {string} id @param {string[]} found @param {string} message */
  const report = (id, found, message) => {
    if (found.length) results.failures.push(`FAIL ${id} ${found.join("; ")}`);
    else results.passes.push(`PASS ${id} ${message}`);
  };

  const allowedList = [...new Set((expectations.allowedModels ?? []).map(servedModel).filter(Boolean))];
  const allowed = new Set(allowedList);
  const detectionModel = servedModel(expectations.detectionModel);

  // D1: the detection step succeeded and the structured result parses with the
  // verdict keys. execution.json stays "started" after success, so it is not used.
  const d1 = [...(problems.D1 ?? [])];
  if (expectations.executionOutcome !== "success") d1.push(`detection execution outcome is ${expectations.executionOutcome || "<empty>"}, expected success`);
  if (!isPlainObject(result)) {
    if (!problems.D1?.length) d1.push(`no parsed ${DETECTION_RESULT_PATH}`);
  } else {
    const badKeys = [...DETECTION_VERDICT_FLAGS.filter(key => typeof result[key] !== "boolean"), ...DETECTION_VERDICT_LISTS.filter(key => !Array.isArray(result[key]))];
    if (badKeys.length) d1.push(`${DETECTION_RESULT_PATH} is missing or has malformed verdict key(s) [${badKeys.join(", ")}]; observed keys: [${Object.keys(result).join(", ")}]`);
  }
  report("D1", d1, `detection completed and ${DETECTION_RESULT_PATH} has verdict keys [${[...DETECTION_VERDICT_FLAGS, ...DETECTION_VERDICT_LISTS].join(", ")}]`);

  // D2: every detection request used the configured detection model, which must
  // be disjoint from the routing candidates.
  const d2 = [...(problems.D2 ?? [])];
  if (!detectionModel) d2.push("no configured detection model to check");
  else if (allowed.has(detectionModel)) d2.push(`configured detection model ${detectionModel} is in allowed-models [${allowedList.join(", ")}]; it must be disjoint from the routing candidates`);
  if (!requests.length) {
    if (!problems.D2?.length) d2.push("no detection requests in the api-proxy token usage");
  } else {
    const offModel = requests.filter(request => servedModel(request?.model) !== detectionModel);
    const routed = requests.filter(request => allowed.has(servedModel(request?.model)));
    if (offModel.length) d2.push(`detection request(s) not on configured model ${detectionModel || "<none>"}: ${describeRequests(offModel)}`);
    if (routed.length) d2.push(`detection request(s) on allowed-models [${allowedList.join(", ")}]: ${describeRequests(routed)}`);
  }
  report("D2", d2, `all ${requests.length} detection request(s) used ${detectionModel}, outside allowed-models [${allowedList.join(", ")}]`);

  // D3: detection was not routed.
  const d3 = [...(problems.D3 ?? [])];
  const classifierRequests = requests.filter(isClassifierRequest);
  if (classifierRequests.length) d3.push(`detection made ${classifierRequests.length} ${CLASSIFIER_PURPOSE} request(s): ${describeRequests(classifierRequests)}`);
  const selections = routingEvents.filter(event => event?.stage === "selection");
  if (selections.length)
    d3.push(
      `detection ${MODEL_ROUTING_LOG} has ${selections.length} selection event(s): ${selections.map(event => `${servedModel(event.wire_model ?? event.selected_model) || "<no model>"} ${normalizeEndpoint(event.endpoint) || "<no endpoint>"}`).join(", ")}`
    );
  if (!isPlainObject(resolvedConfig)) {
    if (!problems.D3?.length) d3.push("no parsed detection AWF resolved config");
  } else {
    const routingKeys = [];
    if (resolvedConfig.modelRouting !== undefined && resolvedConfig.modelRouting !== null) routingKeys.push("modelRouting");
    if (resolvedConfig.experimentalModelRouting === true) routingKeys.push("experimentalModelRouting");
    if (routingKeys.length) d3.push(`detection AWF resolved config has model-routing setting(s) [${routingKeys.join(", ")}]`);
  }
  report("D3", d3, `detection made no ${CLASSIFIER_PURPOSE} request, has no model-routing selection, and its AWF config has no model routing`);

  return results;
}

/**
 * @param {string} file
 * @param {string[]} problems
 * @returns {any}
 */
function readJsonFile(file, problems) {
  if (!fs.existsSync(file)) {
    problems.push(`missing ${file}`);
    return undefined;
  }
  try {
    return JSON.parse(fs.readFileSync(file, "utf8"));
  } catch (error) {
    problems.push(`${file} is not valid JSON: ${getErrorMessage(error)}`);
    return undefined;
  }
}

/**
 * Collect the detection run's own evidence. The detection job copies its AWF
 * proxy and audit logs under the threat-detection directory before post-steps run.
 * @param {{ rootDir?: string }} [options]
 * @returns {{ result: any, requests: any[], routingEvents: any[], resolvedConfig: any, problems: Required<DetectionEvidenceProblems> }}
 */
function collectDetectionEvidence({ rootDir = DETECTION_ROOT_DIR } = {}) {
  /** @type {Required<DetectionEvidenceProblems>} */
  const problems = { D1: [], D2: [], D3: [] };

  const result = readJsonFile(path.join(rootDir, DETECTION_RESULT_PATH), problems.D1);

  /** @type {string[]} */
  const tokenProblems = [];
  let requests = [];
  const tokenFile = TOKEN_USAGE_PATHS.map(file => path.join(rootDir, file)).find(file => fs.existsSync(file));
  if (tokenFile) {
    requests = readJsonl(tokenFile, tokenProblems).filter(record => isPlainObject(record) && (record.event === undefined || record.event === "token_usage"));
  } else {
    tokenProblems.push(`missing api-proxy token-usage.jsonl (looked in ${TOKEN_USAGE_PATHS.join(", ")})`);
  }
  // D3's classifier check reads the same requests, so token-usage problems fail both.
  problems.D2.push(...tokenProblems);
  problems.D3.push(...tokenProblems);

  const routingEvents = [];
  for (const file of [...new Set(TOKEN_USAGE_PATHS.map(tokenPath => path.join(rootDir, path.dirname(tokenPath), MODEL_ROUTING_LOG)))]) {
    if (fs.existsSync(file)) routingEvents.push(...readJsonl(file, problems.D3));
  }

  const resolvedConfig = readJsonFile(path.join(rootDir, RESOLVED_CONFIG_PATH), problems.D3);
  return { result, requests, routingEvents, resolvedConfig, problems };
}

/**
 * Collect detection evidence and evaluate D1–D3.
 * @param {DetectionExpectations & { rootDir?: string }} options
 * @returns {RoutingAssertionResults}
 */
function checkDetectionEvidence(options) {
  const { rootDir, ...expectations } = options;
  const evidence = collectDetectionEvidence({ rootDir });
  return evaluateDetectionEvidence({ ...evidence, expectations });
}

/**
 * Detection-job post-step entry point: logs every check and fails the step on any failure.
 * @param {DetectionExpectations & { rootDir?: string, core?: any }} options
 * @returns {Promise<RoutingAssertionResults>}
 */
async function detectionMain(options) {
  const { core: actionsCore = global.core, ...rest } = options;
  const results = checkDetectionEvidence(rest);
  for (const line of results.passes) actionsCore.info(line);
  for (const line of results.failures) actionsCore.error(line);
  if (results.failures.length) {
    actionsCore.setFailed(`Threat-detection smoke assertions failed (${results.failures.length}):\n${results.failures.join("\n")}`);
  } else {
    actionsCore.info(`All ${results.passes.length} threat-detection smoke assertions passed`);
  }
  return results;
}

module.exports = {
  CLASSIFIER_PURPOSE,
  TOKEN_USAGE_PATHS,
  servedModel,
  normalizeEndpoint,
  supportedEndpoints,
  evaluateModelRoutingEvidence,
  collectModelRoutingEvidence,
  checkModelRoutingEvidence,
  main,
  evaluateDetectionEvidence,
  collectDetectionEvidence,
  checkDetectionEvidence,
  detectionMain,
};
