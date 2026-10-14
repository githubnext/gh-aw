// @ts-check

/** @typedef {import("./types/agent_session").SessionEvent} SessionEvent */
/** @typedef {Record<string, string[]>} Fields */
const { computeInferenceAIC, findModelPricing } = require("./model_costs.cjs");
const { COPILOT_WORKFLOW_EVENT_FIELDS } = require("./copilot_workflow_events.cjs");
const { DYNAMIC_WORKFLOW_EVENT_TYPES } = require("./dynamic_workflow_session.cjs");
const { sessionContext } = require("./agent_session.cjs");
const { normalizeMcpConfigurationData } = require("./mcp_configuration.cjs");

const SCOPED_AGENT_TYPES = new Set([
  "session.init",
  "session.start",
  "session.error",
  "session.info",
  "session.shutdown",
  "session.task_complete",
  "user.message",
  "assistant.message",
  "assistant.reasoning",
  "assistant.message_update",
  "assistant.refusal",
  "tool.execution_start",
  "tool.execution_update",
  "tool.execution_complete",
  "usage.report",
  "session.result",
  "model.call_final_result",
]);

/** @type {Fields} */
const USAGE_FIELDS = {
  totalTokens: ["totalTokens", "total_tokens"],
  inputTokens: ["inputTokens", "input_tokens"],
  outputTokens: ["outputTokens", "output_tokens"],
  reasoningOutputTokens: ["reasoningOutputTokens", "reasoning_output_tokens", "reasoning_tokens"],
  cacheReadInputTokens: ["cacheReadInputTokens", "cache_read_input_tokens", "cache_read_tokens"],
  cacheCreationInputTokens: ["cacheCreationInputTokens", "cache_creation_input_tokens", "cache_write_tokens"],
  inputTokensIncludeCache: ["inputTokensIncludeCache", "input_tokens_include_cache"],
  overflowedTokens: ["overflowedTokens", "overflowed_tokens"],
};
const AIC_RESOLVABLE_PHASES = new Set(["detection", "evals"]);

/** @type {Fields} */
const TOOL_FIELDS = {
  toolCallId: ["toolCallId", "tool_call_id", "tool_use_id"],
  toolName: ["toolName", "tool_name"],
  mcpServerName: ["mcpServerName"],
  parentToolCallId: ["parentToolCallId"],
  stepIndex: ["stepIndex", "step_index"],
};
/** @type {Fields} */
const RUNTIME_FIELDS = {
  event: ["event", "type", "event_name", "eventName"],
  level: ["level"],
  status: ["status"],
  message: ["message"],
  reason: ["reason"],
  requestId: ["requestId", "request_id", "rid"],
};
/** @type {Fields} */
const STEERING_FIELDS = { type: ["type"], threshold: ["threshold"] };
const STEERING_TYPES = new Set(["ai_credit", "timeout", "token"]);
const STEERING_THRESHOLDS = new Set([80, 90, 95, 99]);
/** @type {Fields} */
const MCP_FIELDS = {
  serverName: ["serverName", "server_name", "server_id"],
  direction: ["direction"],
  rpcId: ["rpcId"],
  method: ["method"],
  toolName: ["toolName", "tool_name"],
  toolCallId: ["toolCallId", "tool_call_id"],
  requestId: ["requestId", "request_id"],
  durationMs: ["durationMs", "duration_ms", "duration"],
  inputSize: ["inputSize", "input_size"],
  outputSize: ["outputSize", "output_size"],
  status: ["status"],
  reason: ["reason"],
  error: ["error"],
};
/** @type {Fields} */
const GRADER_FIELDS = {
  id: ["id"],
  name: ["name"],
  status: ["status"],
  value: ["value"],
  score: ["score"],
  unit: ["unit"],
  passed: ["passed"],
  direction: ["direction"],
  threshold: ["threshold"],
  error: ["error"],
  source: ["source"],
  enabled: ["enabled"],
  min: ["min"],
  max: ["max"],
  digest: ["digest"],
  severity: ["severity"],
  message: ["message"],
};
const MODEL_ROUTING_FIELDS = {
  status: ["status"],
  source: ["source"],
  provider: ["provider"],
  wireModel: ["wire_model", "wireModel"],
  model: ["model"],
  effort: ["effort"],
  appliedEffort: ["applied_effort", "appliedEffort"],
  effectiveEndpoint: ["effective_endpoint", "effectiveEndpoint", "endpoint"],
  selectedEndpoint: ["selected_endpoint", "selectedEndpoint"],
  mode: ["mode"],
  selectedId: ["selected_id", "selectedId"],
  routerVersion: ["router_version", "routerVersion"],
  failureCode: ["failure_code", "failureCode"],
};
/** @type {Fields} */
const EPISODE_LINEAGE_FIELDS = {
  episodeId: ["episodeId", "episode_id"],
  hopId: ["hopId", "hop_id"],
  parentHopId: ["parentHopId", "parent_hop_id"],
  originEvent: ["originEvent", "origin_event"],
  rootRepo: ["rootRepo", "root_repo"],
  rootWorkflowId: ["rootWorkflowId", "root_workflow_id"],
  rootRunId: ["rootRunId", "root_run_id"],
};
/** @type {Fields} */
const SAFE_OUTPUT_FIELDS = {
  type: ["type"],
  repo: ["repo", "repository"],
  number: ["number"],
  provider: ["provider"],
  identifier: ["identifier"],
  url: ["url"],
  status: ["status"],
  message: ["message"],
  error: ["error"],
  errorCode: ["errorCode", "error_code"],
};

/** @type {Fields} */
const MESSAGE_FIELDS = {
  content: ["content"],
  delta: ["delta"],
  partial: ["partial"],
  messageId: ["messageId", "message_id"],
  contentIndex: ["contentIndex"],
  channel: ["channel"],
  sessionId: ["sessionId", "session_id"],
  agentId: ["agentId"],
  parentToolUseId: ["parentToolUseId", "parent_tool_use_id"],
  model: ["model"],
  apiCallId: ["apiCallId"],
  interactionId: ["interactionId"],
  turnId: ["turnId"],
  parentToolCallId: ["parentToolCallId"],
};

/** @type {Fields} */
const TASK_FIELDS = {
  taskId: ["taskId", "task_id"],
  toolCallId: ["toolCallId", "tool_use_id"],
  taskType: ["taskType", "task_type"],
  workflowName: ["workflowName", "workflow_name"],
  status: ["status"],
  sessionId: ["sessionId", "session_id"],
  parentToolUseId: ["parentToolUseId", "parent_tool_use_id"],
};

/** @type {Record<string, Fields>} */
const EVENT_FIELDS = {
  "model.inventory": {},
  "plugin.installation": {
    plugin: ["plugin"],
    repository: ["repository"],
    ref: ["ref"],
    path: ["path"],
    status: ["status"],
  },
  ...COPILOT_WORKFLOW_EVENT_FIELDS,
  "session.format": { version: ["version"] },
  "agent.execution": { categories: ["categories"], errorCodes: ["errorCodes"], errorTypes: ["errorTypes"], exitCode: ["exitCode", "exit_code"] },
  "skill.invoked": { name: ["name"], path: ["path"], source: ["source"], trigger: ["trigger"], model: ["model"], invokedAtTurn: ["invokedAtTurn"] },
  "skill.context_delivered_ref": { interactionId: ["interactionId"], source: ["source"], contentId: ["contentId"] },
  "skill.inventory": { skills: ["skills"] },
  "skill.conflict": { name: ["name"], kind: ["kind"], sources: ["sources"] },
  "session.init": { sourceEngine: ["sourceEngine"], model: ["model", "selectedModel"], sessionId: ["sessionId", "session_id"], cwd: ["cwd"], reasoningEffort: ["reasoningEffort"] },
  "session.shutdown": {
    shutdownType: ["shutdownType"],
    errorReason: ["errorReason"],
    premiumRequests: ["premiumRequests", "totalPremiumRequests"],
    totalNanoAiu: ["totalNanoAiu"],
    totalApiDurationMs: ["totalApiDurationMs"],
    sessionStartTime: ["sessionStartTime"],
    currentModel: ["currentModel"],
  },
  "session.task_complete": { success: ["success"] },
  "user.message": MESSAGE_FIELDS,
  "prompt.system": { content: ["content"] },
  "prompt.user": { content: ["content"] },
  "assistant.message": MESSAGE_FIELDS,
  "assistant.refusal": { ...MESSAGE_FIELDS, reason: ["reason"], policyCategory: ["policyCategory"], explanation: ["explanation"] },
  "assistant.reasoning": MESSAGE_FIELDS,
  "assistant.message_update": { ...MESSAGE_FIELDS, sourceType: ["sourceType"], type: ["type"], usageSnapshot: ["usageSnapshot", "usage"] },
  "tool.execution_start": { ...TOOL_FIELDS, input: ["input", "parameters", "arguments", "argumentText"], inputTruncated: ["inputTruncated"], command: ["command"] },
  "tool.execution_update": {
    ...TOOL_FIELDS,
    input: ["input", "parameters", "arguments"],
    output: ["output", "partialResult"],
    partial: ["partial"],
    delta: ["delta"],
    contentIndex: ["contentIndex"],
    sourceType: ["sourceType"],
    usageSnapshot: ["usageSnapshot"],
    durationMs: ["durationMs"],
    status: ["status"],
    command: ["command"],
    taskId: ["taskId", "task_id"],
    taskType: ["taskType", "task_type"],
    error: ["error"],
  },
  "tool.execution_complete": {
    ...TOOL_FIELDS,
    success: ["success"],
    output: ["output"],
    result: ["result"],
    error: ["error"],
    durationMs: ["durationMs", "duration_ms"],
    exitCode: ["exitCode", "exit_code"],
    status: ["status"],
    isError: ["isError", "is_error"],
    taskId: ["taskId"],
    taskType: ["taskType"],
    workflowName: ["workflowName"],
    workflowRunId: ["workflowRunId"],
  },
  "session.result": {
    sourceEngine: ["sourceEngine"],
    numTurns: ["numTurns", "num_turns"],
    durationMs: ["durationMs", "duration_ms"],
    totalCostUsd: ["totalCostUsd", "total_cost_usd"],
    status: ["status"],
    sourceType: ["sourceType"],
    errors: ["errors"],
    permissionDenials: ["permissionDenials", "permission_denials"],
    agentMetrics: ["agentMetrics"],
  },
  "session.error": {
    error: ["error"],
    message: ["message"],
    content: ["content"],
    code: ["code"],
    statusCode: ["statusCode"],
    errorType: ["errorType"],
    model: ["model"],
    severity: ["severity"],
    status: ["status"],
    sourceType: ["sourceType"],
    exitCode: ["exitCode", "exit_code"],
  },
  "session.info": {
    message: ["message"],
    content: ["content"],
    model: ["model"],
    status: ["status"],
    sourceType: ["sourceType"],
    role: ["role"],
    messageId: ["messageId"],
    toolCallId: ["toolCallId"],
    toolName: ["toolName"],
    usageSnapshot: ["usageSnapshot"],
    estimatedTokens: ["estimatedTokens"],
    estimatedTokensDelta: ["estimatedTokensDelta"],
    cwd: ["cwd"],
    branch: ["branch"],
    kind: ["kind"],
    rateLimitInfo: ["rateLimitInfo", "rate_limit_info"],
    isAuthenticating: ["isAuthenticating", "is_authenticating"],
    output: ["output"],
  },
  "mcp.rpc.request": MCP_FIELDS,
  "mcp.rpc.response": MCP_FIELDS,
  "mcp.difc.filtered": MCP_FIELDS,
  "mcp.guard.blocked": MCP_FIELDS,
  "mcp.tool_call": MCP_FIELDS,
  "mcp.event": { ...RUNTIME_FIELDS, serverName: ["serverName", "server_name"] },
  "firewall.http_access": {
    host: ["host", "domain"],
    method: ["method"],
    status: ["status", "http_status"],
    decision: ["decision", "squid_request_status"],
    bytes: ["bytes"],
    durationMs: ["durationMs", "duration_ms"],
  },
  "firewall.steering": { ...RUNTIME_FIELDS, threshold: ["threshold"] },
  "firewall.event": RUNTIME_FIELDS,
  "firewall.token_usage": {
    provider: ["provider"],
    model: ["model"],
    purpose: ["purpose"],
    path: ["path", "endpoint"],
    xInitiator: ["x_initiator", "xInitiator"],
    requestId: ["requestId", "request_id"],
    status: ["status"],
    aic: ["aic", "ai_credits_this_response", "aiCreditsThisResponse"],
    totalAic: ["totalAic", "aiCreditsTotal", "ai_credits_total", "ai_credits", "aiCredits"],
    premiumRequests: ["premiumRequests", "premium_requests"],
    durationMs: ["durationMs", "duration_ms"],
    steering: ["steering"],
  },
  "firewall.model_routing": {
    schema: ["_schema", "schema"],
    stage: ["stage"],
    purpose: ["purpose"],
    attempt: ["attempt"],
    classifierModel: ["classifier_model", "classifierModel"],
    classifierEffort: ["classifier_effort", "classifierEffort"],
    objective: ["objective"],
    provider: ["provider"],
    labels: ["labels"],
    mode: ["mode"],
    classifierAttempts: ["classifier_attempts", "classifierAttempts"],
    degradedClassification: ["degraded_classification", "degradedClassification"],
    degradedReason: ["degraded_reason", "degradedReason"],
    selectedId: ["selected_id", "selectedId"],
    selectedProvider: ["selected_provider", "selectedProvider"],
    selectedModel: ["selected_model", "selectedModel"],
    selectedEffort: ["selected_effort", "selectedEffort"],
    wireModel: ["wire_model", "wireModel"],
    endpoint: ["endpoint"],
    router: ["router"],
    latencyMs: ["latency_ms", "latencyMs"],
    code: ["code"],
    requestId: ["request_id", "requestId"],
    routed: ["routed"],
    deviations: ["deviations"],
    requestedModel: ["requested_model", "requestedModel"],
    requestedEffort: ["requested_effort", "requestedEffort"],
    outcome: ["outcome"],
  },
  "safe_output.request": SAFE_OUTPUT_FIELDS,
  "safe_output.result": SAFE_OUTPUT_FIELDS,
  "safe_output.error": SAFE_OUTPUT_FIELDS,
  "experiment.state": { runId: ["runId", "run_id"], assignments: ["assignments"], counts: ["counts"] },
  "grader.manifest": { version: ["version"] },
  "grader.result": GRADER_FIELDS,
  "eval.result": { id: ["id"], answer: ["answer"], model: ["model"], error: ["error"], runId: ["runId", "runid", "run_id"], assignments: ["assignments", "experiments"] },
  "github_api.rate_limit": {
    source: ["source"],
    credentialSource: ["credentialSource", "credential_source"],
    operation: ["operation"],
    resource: ["resource"],
    limit: ["limit"],
    remaining: ["remaining"],
    used: ["used"],
    reset: ["reset"],
    attempt: ["attempt"],
    delayMs: ["delayMs", "delay_ms"],
    status: ["status"],
  },
  "execution.result": { outcome: ["outcome"], conclusion: ["conclusion"], exitCode: ["exitCode", "exit_code"], durationMs: ["durationMs", "duration_ms"], startedAt: ["startedAt", "started_at"], finishedAt: ["finishedAt", "finished_at"] },
  "detection.result": {
    jobResult: ["jobResult", "job_result"],
    conclusion: ["conclusion"],
    reason: ["reason"],
    promptInjection: ["promptInjection", "prompt_injection"],
    secretLeak: ["secretLeak", "secret_leak"],
    maliciousPatch: ["maliciousPatch", "malicious_patch"],
  },
  "guardrail.daily_aic": {
    status: ["status"],
    exceeded: ["exceeded"],
    total: ["total"],
    estimated: ["estimated"],
    threshold: ["threshold"],
  },
  "workflow.info": {
    dryRun: ["dryRun", "dry_run"],
    engineId: ["engineId", "engine_id", "engine"],
    agentVersion: ["agentVersion", "agent_version"],
    cliVersion: ["cliVersion", "cli_version"],
    awfVersion: ["awfVersion", "awf_version"],
    mcpgVersion: ["mcpgVersion", "mcpg_version", "awmg_version"],
    model: ["model"],
    requestedModel: ["requested_model", "requestedModel", "model"],
    modelRouting: ["modelRouting", "model_routing"],
    triggerType: ["triggerType", "event_name"],
    workflow: ["workflow", "workflow_name"],
    repository: ["repository"],
    runId: ["runId", "run_id"],
    episode: ["episode"],
  },
  "model_routing.outcome": {
    status: ["status"],
    wireModel: ["wire_model", "wireModel"],
    effectiveEndpoint: ["effective_endpoint", "effectiveEndpoint", "endpoint"],
    selectedEndpoint: ["selected_endpoint", "selectedEndpoint"],
    effort: ["effort"],
    appliedEffort: ["applied_effort", "appliedEffort"],
    failureCode: ["failure_code", "failureCode"],
  },
};
EVENT_FIELDS["session.start"] = EVENT_FIELDS["session.init"];
EVENT_FIELDS["usage.report"] = { ...EVENT_FIELDS["firewall.token_usage"], stepIndex: ["stepIndex", "step_index"] };
for (const type of Object.values(DYNAMIC_WORKFLOW_EVENT_TYPES)) {
  EVENT_FIELDS[type] = TASK_FIELDS;
}

/**
 * Prefer an explicitly supplied canonical field, including false, zero, and null.
 * @param {any} value
 * @param {Fields} fields
 * @returns {Record<string, any>}
 */
function selectFields(value, fields) {
  const result = {};
  for (const [key, aliases] of Object.entries(fields)) {
    const alias = aliases.find(name => value != null && Object.hasOwn(value, name) && value[name] !== undefined);
    if (alias !== undefined) result[key] = structuredClone(value[alias]);
  }
  return result;
}

/** @param {Record<string, any>} record @returns {any} */
function evalResultRunId(record) {
  return EVENT_FIELDS["eval.result"].runId.reduce((runId, alias) => runId ?? record[alias], undefined);
}

/** @param {any} source @param {Record<string, any>} data @param {Record<string, any>} usage */
function resolveUsageAic(source, data, usage) {
  if (data.aic !== undefined || data.totalAic !== undefined) return;
  const provider = data.provider ?? source.usage?.provider;
  const model = data.model ?? source.usage?.model;
  if (typeof provider !== "string" || typeof model !== "string" || !findModelPricing(provider, model)) return;
  const tokens = [usage.inputTokens, usage.outputTokens, usage.cacheReadInputTokens, usage.cacheCreationInputTokens, usage.reasoningOutputTokens];
  if (!tokens.some(value => typeof value === "number" && value > 0) || tokens.some(value => value !== undefined && (typeof value !== "number" || !Number.isFinite(value) || value < 0))) return;
  const aic = computeInferenceAIC({
    provider,
    model,
    inputTokens: usage.inputTokens ?? 0,
    outputTokens: usage.outputTokens ?? 0,
    cacheReadTokens: usage.cacheReadInputTokens ?? 0,
    cacheWriteTokens: usage.cacheCreationInputTokens ?? 0,
    reasoningTokens: usage.reasoningOutputTokens ?? 0,
    ...(typeof usage.inputTokensIncludeCache === "boolean" ? { inputTokensIncludeCache: usage.inputTokensIncludeCache } : {}),
  });
  if (Number.isFinite(aic)) data.aic = aic;
}

/**
 * Compact known payloads without truncating text or changing native parser traces.
 * Unknown extension data remains opaque for forward compatibility.
 * @param {SessionEvent} event
 * @param {string} [phase]
 * @returns {SessionEvent}
 */
function normalizeUnifiedSessionEvent(event, phase) {
  /** @type {any} */
  const source = event.data;
  const known = Object.hasOwn(EVENT_FIELDS, event.type);
  const data = event.type === "mcp.configuration" ? normalizeMcpConfigurationData(source) : known ? selectFields(source, EVENT_FIELDS[event.type]) : structuredClone(source);
  if (event.type === "model.inventory") {
    const endpoints = Array.isArray(source.endpoints)
      ? source.endpoints
      : source.providers && typeof source.providers === "object" && !Array.isArray(source.providers)
        ? Object.entries(source.providers).map(([provider, entry]) => ({ ...entry, provider }))
        : typeof source.provider === "string" && Array.isArray(source.models)
          ? [source]
          : [];
    data.endpoints = endpoints
      .filter(endpoint => endpoint && typeof endpoint === "object" && typeof endpoint.provider === "string")
      .map(endpoint => ({
        provider: endpoint.provider,
        models: (Array.isArray(endpoint.models ?? endpoint.available_models ?? endpoint.detected_models ?? endpoint.model_ids ?? endpoint.availableModels)
          ? (endpoint.models ?? endpoint.available_models ?? endpoint.detected_models ?? endpoint.model_ids ?? endpoint.availableModels)
          : []
        )
          .map(model => (typeof model === "string" ? model : (model?.id ?? model?.name ?? model?.model)))
          .filter(model => typeof model === "string"),
        ...(Array.isArray(endpoint.routingModels ?? endpoint.routing_models)
          ? {
              routingModels: (endpoint.routingModels ?? endpoint.routing_models)
                .filter(model => model && typeof (model.modelId ?? model.model_id) === "string")
                .map(model => ({
                  modelId: model.modelId ?? model.model_id,
                  ...(Array.isArray(model.supportedEndpoints ?? model.supported_endpoints) ? { supportedEndpoints: (model.supportedEndpoints ?? model.supported_endpoints).filter(endpoint => typeof endpoint === "string") } : {}),
                })),
            }
          : {}),
      }));
    if (typeof (source.modelsFetchComplete ?? source.models_fetch_complete) === "boolean") data.modelsFetchComplete = source.modelsFetchComplete ?? source.models_fetch_complete;
  }
  if (event.type === "workflow.info") {
    const awContext = Object.hasOwn(source, "awContext") ? source.awContext : source.context;
    if (awContext && typeof awContext === "object" && !Array.isArray(awContext)) {
      data.awContext = structuredClone(awContext);
    }
    const modelRouting = source.modelRouting ?? source.model_routing;
    if (modelRouting && typeof modelRouting === "object" && !Array.isArray(modelRouting)) {
      data.modelRouting = selectFields(modelRouting, MODEL_ROUTING_FIELDS);
    }
    const episode = Object.hasOwn(source, "episode") ? source.episode : source.context;
    if (episode && typeof episode === "object" && !Array.isArray(episode)) {
      const lineage = selectFields(episode, EPISODE_LINEAGE_FIELDS);
      if (Object.keys(lineage).length || Object.hasOwn(source, "episode")) data.episode = lineage;
    }
  }
  if (event.type === "skill.inventory" && Array.isArray(source.skills)) {
    data.skills = source.skills.map(skill =>
      selectFields(skill, {
        name: ["name"],
        source: ["source"],
        scope: ["scope"],
        digest: ["digest"],
        purpose: ["purpose"],
      })
    );
  }
  if (SCOPED_AGENT_TYPES.has(event.type) || (Object.hasOwn(COPILOT_WORKFLOW_EVENT_FIELDS, event.type) && event.type.startsWith("subagent."))) {
    const parentToolCallId = Object.hasOwn(source, "parentToolCallId") ? source.parentToolCallId : event.parentToolCallId;
    const hasParentToolUseId = ["parentToolUseId", "parent_tool_use_id"].some(key => Object.hasOwn(source, key) || Object.hasOwn(event, key));
    for (const [key, value] of Object.entries(sessionContext(event))) {
      if (key === "parentToolUseId" && parentToolCallId !== undefined && !hasParentToolUseId) continue;
      if (!Object.hasOwn(data, key)) data[key] = value;
    }
    if (parentToolCallId !== undefined && !Object.hasOwn(data, "parentToolCallId")) data.parentToolCallId = structuredClone(parentToolCallId);
  }
  if (event.type === "session.task_complete" && Object.hasOwn(source, "summary") && typeof source.summary !== "string") data.summary = structuredClone(source.summary);
  if (known && event.type.startsWith("dynamicWorkflows.")) {
    if (!Object.hasOwn(data, "status") && source.patch?.status !== undefined) data.status = structuredClone(source.patch.status);
    const usage = selectFields(source.usage, { totalTokens: ["totalTokens", "total_tokens"], toolUses: ["toolUses", "tool_uses"], durationMs: ["durationMs", "duration_ms"] });
    if (Object.keys(usage).length) data.usage = usage;
    if (Array.isArray(source.tasks)) data.tasks = source.tasks.map(task => selectFields(task, TASK_FIELDS));
    const progress = source.workflowProgress ?? source.workflow_progress;
    if (Array.isArray(progress)) {
      data.workflowProgress = progress.map(item =>
        selectFields(item, {
          type: ["type"],
          index: ["index"],
          phaseIndex: ["phaseIndex"],
          agentId: ["agentId"],
          model: ["model"],
          state: ["state"],
          attempt: ["attempt"],
          startedAt: ["startedAt"],
          queuedAt: ["queuedAt"],
          lastProgressAt: ["lastProgressAt"],
        })
      );
    }
  }
  if (event.type === "user.message" || event.type === "assistant.message" || event.type === "assistant.reasoning" || event.type === "assistant.refusal") {
    const metadata = selectFields(event, MESSAGE_FIELDS);
    delete metadata.content;
    for (const [key, value] of Object.entries(metadata)) if (!Object.hasOwn(data, key)) data[key] = value;
    const message = event.message;
    if (!Object.hasOwn(data, "messageId") && message && typeof message === "object" && !Array.isArray(message) && "id" in message && message.id !== undefined) data.messageId = structuredClone(message.id);
    if (event.copilotProjection === "assistant.message_delta") data.delta = true;
  }
  if (known && event.type.startsWith("mcp.") && event.type !== "mcp.event") {
    const rpc = source.payload;
    for (const [key, value] of Object.entries({ rpcId: rpc?.id, method: rpc?.method, toolName: rpc?.params?.name, error: rpc?.error })) {
      if (!Object.hasOwn(data, key) && value !== undefined) data[key] = structuredClone(value);
    }
    if (data.error && typeof data.error === "object") data.error = selectFields(data.error, { code: ["code"], message: ["message"] });
  }
  if (["session.result", "firewall.token_usage", "usage.report"].includes(event.type)) {
    const usage = selectFields(event.type === "session.result" ? source.usage : (source.usage ?? source), USAGE_FIELDS);
    if (Array.isArray(usage.overflowedTokens)) {
      const aliases = Object.entries(USAGE_FIELDS);
      usage.overflowedTokens = usage.overflowedTokens.map(name => aliases.find(([, names]) => names.includes(name))?.[0] ?? name);
    }
    if (Object.keys(usage).length || (source.usage && typeof source.usage === "object")) data.usage = usage;
    else if (source.usage === null) data.usage = null;
    if (event.type === "usage.report" && source.usage && typeof source.usage === "object") {
      const nested = selectFields(source.usage, EVENT_FIELDS["usage.report"]);
      for (const key of ["provider", "model", "aic", "totalAic", "premiumRequests"]) {
        if (data[key] === undefined && nested[key] !== undefined) data[key] = nested[key];
      }
    }
    if (event.type === "usage.report" && phase !== undefined && AIC_RESOLVABLE_PHASES.has(phase)) resolveUsageAic(source, data, usage);
    if (Object.hasOwn(data, "steering")) {
      // Keep only the AWF steering notice type and threshold; drop unrecognized notices.
      const steering = data.steering && typeof data.steering === "object" && !Array.isArray(data.steering) ? selectFields(data.steering, STEERING_FIELDS) : {};
      if (STEERING_TYPES.has(steering.type) && STEERING_THRESHOLDS.has(steering.threshold)) data.steering = steering;
      else delete data.steering;
    }
  }
  if (event.type === "tool.execution_complete" && (source.is_error === true || source.result?.isError === true || source.result?.is_error === true)) data.isError = true;
  if (event.type === "experiment.assignment") {
    for (const key of Object.keys(data)) delete data[key];
    data.assignments = structuredClone(source.assignments ?? source);
  }
  if (event.type === "grader.manifest" && Array.isArray(source.graders)) data.graders = source.graders.map(grader => selectFields(grader, GRADER_FIELDS));
  if (event.type === "grader.result") {
    if (source.version !== undefined) data.version = structuredClone(source.version);
    if (source.run !== undefined) data.run = source.run === null ? null : selectFields(source.run, { id: ["id"], attempt: ["attempt"], graderCount: ["graderCount"], passed: ["passed"], failed: ["failed"], errors: ["errors"] });
    const projectResult = result => ({
      ...selectFields(result, GRADER_FIELDS),
      ...(result?.implementation !== undefined ? { implementation: result.implementation === null ? null : selectFields(result.implementation, { id: ["id"], version: ["version"], digest: ["digest"] }) } : {}),
      ...(Array.isArray(result?.metrics) ? { metrics: result.metrics.map(metric => selectFields(metric, { id: ["id"], value: ["value"] })) } : {}),
    });
    Object.assign(data, projectResult(source));
    if (Array.isArray(source.results)) data.results = source.results.map(projectResult);
  }
  if (event.type === "safe_output.error") {
    const errors = source.errors ?? source.failures;
    if (Array.isArray(errors)) data.errors = errors.map(error => (error && typeof error === "object" && !Array.isArray(error) ? selectFields(error, SAFE_OUTPUT_FIELDS) : structuredClone(error)));
  }
  const metadata = selectFields(event, { id: ["id"], parentId: ["parentId"], timestamp: ["timestamp", "ts", "time", "created_at"], agentId: ["agentId"], ephemeral: ["ephemeral"] });
  /** @type {any} */
  const nativeMessage = event.message;
  if (!Object.hasOwn(metadata, "timestamp") && nativeMessage?.timestamp !== undefined) metadata.timestamp = structuredClone(nativeMessage.timestamp);
  return {
    type: event.type,
    ...metadata,
    data,
  };
}

module.exports = { normalizeUnifiedSessionEvent, evalResultRunId };
