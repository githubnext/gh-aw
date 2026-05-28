#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const {
  InputError,
  LAMBDA,
  OVERLAP_THRESHOLD,
  RISKY_PERMISSION_LEVELS,
  TELEMETRY_KEYS,
  clamp,
  roundScore,
  roundTo,
  normalizeText,
  coerceBool,
  ensureRequired,
  writeJson,
  asList,
  parseFrontmatterText,
  readWorkflow,
  tokenize,
} = require("./aw_yield_shared.cjs");

function discoverWorkflowFiles(workflowsRoot) {
  const files = [];
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      const fullPath = path.join(dir, entry.name);
      if (entry.isDirectory()) {
        walk(fullPath);
        continue;
      }
      if (!entry.isFile() || !entry.name.endsWith(".md")) {
        continue;
      }
      const rel = path.relative(workflowsRoot, fullPath).split(path.sep);
      if (rel.includes("shared")) {
        continue;
      }
      files.push(fullPath);
    }
  };
  walk(workflowsRoot);
  return files.sort();
}

function getWorkflowsRoot(workflowPath) {
  let current = path.resolve(path.dirname(workflowPath));
  while (true) {
    const parent = path.dirname(current);
    if (parent === current) {
      return null;
    }
    if (path.basename(current) === "workflows" && path.basename(parent) === ".github") {
      return current;
    }
    current = parent;
  }
}

function pathIsWithin(targetPath, rootPath) {
  const resolvedTarget = path.resolve(targetPath);
  const resolvedRoot = path.resolve(rootPath);
  return resolvedTarget === resolvedRoot || resolvedTarget.startsWith(`${resolvedRoot}${path.sep}`);
}

function isAbsoluteImportPath(raw) {
  if (path.isAbsolute(raw)) {
    return true;
  }
  return /^[a-zA-Z]:[\\/]/.test(raw) || /^\\\\/.test(raw);
}

function normalizeImportPaths(workflowPath, frontmatter) {
  const workflowsRoot = getWorkflowsRoot(workflowPath);
  const imports = [];
  for (const item of asList(frontmatter.imports)) {
    let raw = null;
    if (typeof item === "string") {
      raw = item;
    } else if (item && typeof item === "object") {
      raw = normalizeText(item.uses) || normalizeText(item.path);
    }
    if (!raw) {
      continue;
    }
    if (
      !(raw.startsWith("shared/") || raw.startsWith("./") || raw.startsWith("../") || raw.startsWith("/") || isAbsoluteImportPath(raw)) &&
      (!raw.includes("/") || raw.includes("@"))
    ) {
      continue;
    }
    if (!workflowsRoot || isAbsoluteImportPath(raw)) {
      continue;
    }
    if (raw.startsWith("shared/")) {
      const importPath = path.resolve(workflowsRoot, raw);
      if (pathIsWithin(importPath, path.join(workflowsRoot, "shared"))) {
        imports.push(importPath);
      }
    } else if (raw.startsWith("./") || raw.startsWith("../")) {
      const importPath = path.resolve(path.dirname(workflowPath), raw);
      if (pathIsWithin(importPath, workflowsRoot)) {
        imports.push(importPath);
      }
    }
  }
  return imports;
}

function hasObservabilityConfig(frontmatter) {
  return Boolean(frontmatter?.observability && typeof frontmatter.observability === "object" && frontmatter.observability.otlp && typeof frontmatter.observability.otlp === "object");
}

function hasImportedObservability(workflowPath, frontmatter) {
  for (const importPath of normalizeImportPaths(workflowPath, frontmatter)) {
    if (!fs.existsSync(importPath)) {
      continue;
    }
    let importedFrontmatter;
    try {
      [importedFrontmatter] = readWorkflow(importPath);
    } catch {
      continue;
    }
    if (hasObservabilityConfig(importedFrontmatter)) {
      return true;
    }
    if (normalizeText(path.basename(importPath)).toLowerCase().includes("otel")) {
      return true;
    }
    const mcpServers = importedFrontmatter["mcp-servers"];
    if (mcpServers && typeof mcpServers === "object" && Object.prototype.hasOwnProperty.call(mcpServers, "otel")) {
      return true;
    }
  }
  return false;
}

function getMatchingLockfile(workflowPath) {
  return workflowPath.replace(/\.md$/u, ".lock.yml");
}

function detectLockfileStatus(workflowPath) {
  const lockfile = getMatchingLockfile(workflowPath);
  if (!fs.existsSync(lockfile)) {
    return [false, false];
  }
  try {
    const workflowMtime = fs.statSync(workflowPath).mtimeMs;
    const lockfileMtime = fs.statSync(lockfile).mtimeMs;
    return [true, lockfileMtime < workflowMtime];
  } catch {
    return [true, false];
  }
}

function countSteps(value) {
  if (Array.isArray(value)) {
    return value.length;
  }
  if (value && typeof value === "object") {
    return 1;
  }
  return 0;
}

function collectStepText(value) {
  return value ? JSON.stringify(value) : "";
}

function inferTimeoutMinutes(value) {
  if (typeof value === "number" && Number.isFinite(value)) {
    return Math.trunc(value);
  }
  if (typeof value === "string") {
    const match = value.match(/\d+/);
    if (match) {
      return Number.parseInt(match[0], 10);
    }
  }
  return null;
}

function permissionsRisk(permissions) {
  if (!permissions || typeof permissions !== "object" || Array.isArray(permissions) || Object.keys(permissions).length === 0) {
    return 0.45;
  }
  let readScopes = 0;
  let elevated = 0;
  let idToken = 0;
  for (const [scope, level] of Object.entries(permissions)) {
    const normalizedScope = normalizeText(scope).toLowerCase();
    const normalizedLevel = normalizeText(level).toLowerCase();
    if (normalizedScope === "id-token" && RISKY_PERMISSION_LEVELS.has(normalizedLevel)) {
      idToken += 1;
    }
    if (normalizedLevel === "read") {
      readScopes += 1;
    } else if (RISKY_PERMISSION_LEVELS.has(normalizedLevel)) {
      elevated += 1;
    }
  }
  const breadth = clamp(readScopes / 6);
  return roundScore(0.2 + breadth * 0.35 + elevated * 0.45 + idToken * 0.1);
}

function countTools(frontmatter) {
  let toolCount = Object.keys(frontmatter.tools || {}).length;
  const mcpServers = frontmatter["mcp-servers"];
  if (mcpServers && typeof mcpServers === "object" && !Array.isArray(mcpServers)) {
    toolCount += Object.keys(mcpServers).length;
  }
  return toolCount;
}

function extractTriggerTokens(onValue) {
  if (typeof onValue === "string") {
    return tokenize(onValue);
  }
  if (Array.isArray(onValue)) {
    return onValue.flatMap((item) => tokenize(normalizeText(item)));
  }
  if (onValue && typeof onValue === "object") {
    const tokens = [];
    for (const [key, value] of Object.entries(onValue)) {
      tokens.push(...tokenize(key));
      tokens.push(...tokenize(JSON.stringify(value)));
    }
    return tokens;
  }
  return [];
}

function extractHeadings(body) {
  return [...body.matchAll(/^#+\s+(.+)$/gm)].map((match) => match[1].trim());
}

function buildIntentText(workflowPath, frontmatter, body) {
  const safeOutputs = frontmatter["safe-outputs"];
  const tools = frontmatter.tools;
  const parts = [
    path.basename(workflowPath, ".md").replace(/-/g, " "),
    normalizeText(frontmatter.name),
    normalizeText(frontmatter.description),
    extractTriggerTokens(frontmatter.on).join(" "),
    safeOutputs && typeof safeOutputs === "object" ? Object.keys(safeOutputs).join(" ") : "",
    tools && typeof tools === "object" ? Object.keys(tools).join(" ") : "",
    extractHeadings(body).join(" "),
    body.replace(/\s+/g, " ").slice(0, 1500),
  ];
  return parts.filter(Boolean).join(" ").trim();
}

function estimateAgenticFraction(frontmatter, body) {
  const preText = collectStepText(frontmatter["pre-agent-steps"]);
  const postText = collectStepText(frontmatter["post-steps"]);
  const bodyWords = body.trim() ? body.trim().split(/\s+/).length : 0;
  let preWeight = countSteps(frontmatter["pre-agent-steps"]) * 1.3;
  let postWeight = countSteps(frontmatter["post-steps"]) * 1.1;
  preWeight += 0.8 * (preText.match(/\b(python3?|jq|grep|awk|sed|sort|uniq|cat|find)\b/g) || []).length;
  postWeight += 0.8 * (postText.match(/\b(python3?|jq|grep|awk|sed|sort|uniq|cat|find)\b/g) || []).length;
  const toolWeight = countTools(frontmatter) * 0.15;
  const agentWeight = Math.max(0.25, bodyWords / 220 + toolWeight);
  const total = preWeight + postWeight + agentWeight;
  if (total <= 0) {
    return [0.5, 0.5];
  }
  const agenticFraction = roundScore(agentWeight / total);
  return [agenticFraction, roundScore(1 - agenticFraction)];
}

function scoreObservability(hasDirect, hasImported, telemetryMetrics) {
  let score = 0;
  if (hasDirect) {
    score += 0.6;
  }
  if (hasImported) {
    score += 0.3;
  }
  if (telemetryMetrics && Object.keys(telemetryMetrics).length > 0) {
    score += 0.4;
  }
  return roundScore(score);
}

function scoreSafeOutputs(safeOutputs) {
  if (!safeOutputs || typeof safeOutputs !== "object" || Array.isArray(safeOutputs) || Object.keys(safeOutputs).length === 0) {
    return 0;
  }
  let score = 0.3;
  if (Object.prototype.hasOwnProperty.call(safeOutputs, "create-issue")) {
    score += 0.3;
  }
  if (["mentions", "allowed-github-references", "max-bot-mentions"].some((key) => Object.prototype.hasOwnProperty.call(safeOutputs, key))) {
    score += 0.2;
  }
  for (const value of Object.values(safeOutputs)) {
    if (value && typeof value === "object" && value.max !== undefined) {
      score += 0.2;
      break;
    }
  }
  return roundScore(score);
}

function scoreCost(frontmatter, body, telemetryMetrics, agenticFraction) {
  const timeout = inferTimeoutMinutes(frontmatter["timeout-minutes"]) ?? 20;
  let base = 0.15 + clamp(timeout / 60) * 0.2 + agenticFraction * 0.25 + clamp(countTools(frontmatter) / 8) * 0.15;
  base += clamp(body.length / 6000) * 0.1;
  if (telemetryMetrics && Object.keys(telemetryMetrics).length > 0) {
    base += clamp(((telemetryMetrics.input_tokens || 0) + (telemetryMetrics.output_tokens || 0)) / 250000) * 0.3;
    base += clamp((telemetryMetrics.runtime_duration || 0) / 1800) * 0.25;
    base += clamp((telemetryMetrics.tool_calls || 0) / 150) * 0.2;
    base += clamp((telemetryMetrics.retries || 0) / 6) * 0.2;
  }
  return roundScore(base);
}

function scoreTrust(strict, timeoutMinutes, hasLockfile, lockfileStale, safeOutputScore, observabilityScore, telemetryMetrics) {
  let score = 0.2;
  if (strict) {
    score += 0.2;
  }
  if (timeoutMinutes !== null && timeoutMinutes !== undefined) {
    score += 0.1;
  }
  if (hasLockfile && !lockfileStale) {
    score += 0.15;
  }
  score += safeOutputScore * 0.2;
  score += observabilityScore * 0.1;
  if (telemetryMetrics && Object.keys(telemetryMetrics).length > 0) {
    score += clamp(telemetryMetrics.success_rate ?? 0.5) * 0.35;
    score += clamp(1 - (telemetryMetrics.retries || 0) / 6) * 0.15;
    score += clamp(telemetryMetrics.safe_output_success || 0) * 0.2;
  }
  return roundScore(score);
}

function scoreUsefulness(frontmatter, body, safeOutputScore, telemetryMetrics) {
  let score = 0.1 + safeOutputScore * 0.25 + clamp(extractHeadings(body).length / 8) * 0.1;
  if (extractTriggerTokens(frontmatter.on).length > 0) {
    score += 0.1;
  }
  if (telemetryMetrics && Object.keys(telemetryMetrics).length > 0) {
    score += clamp(telemetryMetrics.outputs_acted_upon ?? telemetryMetrics.accepted_outputs ?? 0) * 0.35;
    score += clamp((telemetryMetrics.issues_resolved || 0) / 10) * 0.15;
    score += clamp((telemetryMetrics.manual_minutes_saved || 0) / 180) * 0.2;
    score += clamp((telemetryMetrics.actionable_comments || 0) / 10) * 0.1;
  } else if (safeOutputScore > 0) {
    score += 0.15;
  }
  return roundScore(score);
}

function scoreAdoption(frontmatter, telemetryMetrics) {
  let score = 0.05;
  const onValue = frontmatter.on;
  if (onValue && typeof onValue === "object" && Object.prototype.hasOwnProperty.call(onValue, "workflow_dispatch")) {
    score += 0.1;
  }
  if (onValue && typeof onValue === "object" && Object.prototype.hasOwnProperty.call(onValue, "schedule")) {
    score += 0.1;
  }
  score += clamp(asList(frontmatter.imports).length / 4) * 0.1;
  if (telemetryMetrics && Object.keys(telemetryMetrics).length > 0) {
    score += clamp((telemetryMetrics.workflow_invocation_count || 0) / 50) * 0.45;
    const interactions = (telemetryMetrics.user_interaction_count || 0) + (telemetryMetrics.reviewer_interaction_count || 0);
    score += clamp(interactions / 25) * 0.2;
  }
  return roundScore(score);
}

function scoreMaintenance(frontmatter, body, overlapHint, agenticFraction, hasPrecompute, hasPostcompute) {
  const bodyLines = body.split(/\r?\n/).length;
  const imports = asList(frontmatter.imports).length;
  const toolCount = countTools(frontmatter);
  let score = 0.1 + clamp(bodyLines / 250) * 0.2 + clamp(toolCount / 8) * 0.15 + clamp(imports / 5) * 0.1;
  score += overlapHint * 0.2 + agenticFraction * 0.2;
  if (!hasPrecompute) {
    score += 0.1;
  }
  if (!hasPostcompute) {
    score += 0.1;
  }
  return roundScore(score);
}

function scoreRisk(permissionScore, strict, timeoutMinutes, hasLockfile, lockfileStale, safeOutputScore, observabilityScore, network, agenticFraction) {
  let score = permissionScore * 0.35 + agenticFraction * 0.15;
  if (!strict) {
    score += 0.2;
  }
  if (timeoutMinutes === null || timeoutMinutes === undefined) {
    score += 0.15;
  }
  if (!hasLockfile) {
    score += 0.15;
  }
  if (lockfileStale) {
    score += 0.15;
  }
  if (safeOutputScore === 0) {
    score += 0.2;
  }
  if (observabilityScore < 0.4) {
    score += 0.15;
  }
  if (!network || typeof network !== "object" || !Object.prototype.hasOwnProperty.call(network, "allowed")) {
    score += 0.1;
  }
  return roundScore(score);
}

function evidenceQualityForWorkflow(observabilityDeclared, telemetryObserved, telemetryValidated, telemetryMetrics) {
  if (observabilityDeclared && telemetryValidated && Object.keys(telemetryMetrics).length >= 3) {
    return "high";
  }
  if (telemetryValidated || (telemetryObserved && Object.keys(telemetryMetrics).length >= 2)) {
    return "medium";
  }
  return "low";
}

function normalizeTelemetryEntry(entry) {
  const normalized = {};
  const aliases = {
    runtime_duration: ["runtime_duration", "duration_seconds", "duration", "runtime_seconds"],
    tool_calls: ["tool_calls", "tool_call_count"],
    retries: ["retries", "retry_count"],
    success_rate: ["success_rate"],
    safe_output_success: ["safe_output_success", "safe_output_success_rate"],
    workflow_invocation_count: ["workflow_invocation_count", "invocation_count", "runs"],
    user_interaction_count: ["user_interaction_count", "user_interactions"],
    reviewer_interaction_count: ["reviewer_interaction_count", "reviewer_interactions"],
    input_tokens: ["input_tokens"],
    output_tokens: ["output_tokens"],
    accepted_outputs: ["accepted_outputs"],
    outputs_acted_upon: ["outputs_acted_upon", "acted_upon_rate"],
    actionable_comments: ["actionable_comments"],
    pr_impact: ["pr_impact"],
    issues_resolved: ["issues_resolved"],
    bugs_found: ["bugs_found"],
    manual_minutes_saved: ["manual_minutes_saved", "minutes_saved"],
  };
  for (const [target, keys] of Object.entries(aliases)) {
    for (const key of keys) {
      if (Object.prototype.hasOwnProperty.call(entry, key)) {
        normalized[target] = entry[key];
        break;
      }
    }
  }
  const metrics = Object.fromEntries(Object.entries(normalized).filter(([key]) => TELEMETRY_KEYS.has(key)));
  const observed = coerceBool(entry.telemetry_observed ?? entry.observed, Object.keys(metrics).length > 0);
  const validated = coerceBool(entry.telemetry_validated ?? entry.validated, false);
  return {
    metrics,
    observed: observed && Object.keys(metrics).length > 0,
    validated: validated && Object.keys(metrics).length > 0,
    source: normalizeText(entry.source),
  };
}

function loadOtelSummary(filePath) {
  if (!filePath || !fs.existsSync(filePath)) {
    return {};
  }
  let payload;
  try {
    payload = JSON.parse(fs.readFileSync(filePath, "utf8"));
  } catch {
    return {};
  }
  let entries = [];
  if (payload && typeof payload === "object" && !Array.isArray(payload)) {
    if (Array.isArray(payload.workflows)) {
      entries = payload.workflows.filter((entry) => entry && typeof entry === "object" && !Array.isArray(entry));
    } else if (payload.workflow_metrics && typeof payload.workflow_metrics === "object") {
      entries = Object.entries(payload.workflow_metrics)
        .filter(([, value]) => value && typeof value === "object" && !Array.isArray(value))
        .map(([key, value]) => ({ ...value, name: value.name ?? key }));
    } else {
      entries = Object.entries(payload)
        .filter(([, value]) => value && typeof value === "object" && !Array.isArray(value))
        .map(([key, value]) => ({ ...value, name: value.name ?? key }));
    }
  } else if (Array.isArray(payload)) {
    entries = payload.filter((entry) => entry && typeof entry === "object" && !Array.isArray(entry));
  }
  const index = {};
  for (const entry of entries) {
    const normalized = normalizeTelemetryEntry(entry);
    if (!normalized.metrics || Object.keys(normalized.metrics).length === 0) {
      continue;
    }
    const keys = new Set([
      normalizeText(entry.path),
      normalizeText(entry.workflow_path),
      normalizeText(entry.name),
      normalizeText(entry.workflow),
      normalizeText(entry.workflow_name),
    ]);
    for (const key of keys) {
      if (!key) {
        continue;
      }
      index[key] = normalized;
      index[path.parse(key).name] = normalized;
    }
  }
  return index;
}

function telemetryForWorkflow(workflowPath, relativePath, frontmatter, telemetryIndex) {
  const candidates = [normalizeText(relativePath), path.basename(workflowPath), normalizeText(frontmatter.name)];
  for (const key of candidates) {
    if (key && telemetryIndex[key]) {
      return { ...telemetryIndex[key] };
    }
  }
  return { metrics: {}, observed: false, validated: false, source: "" };
}

function buildWorkflowRecord(workflowPath, workflowsRoot, telemetryIndex) {
  const [frontmatter, body] = readWorkflow(workflowPath);
  const relativePath = path.relative(path.dirname(path.dirname(workflowsRoot)), workflowPath).split(path.sep).join("/");
  const [hasLockfile, lockfileStale] = detectLockfileStatus(workflowPath);
  const strict = Boolean(frontmatter.strict ?? false);
  const timeoutMinutes = inferTimeoutMinutes(frontmatter["timeout-minutes"]);
  const safeOutputs = frontmatter["safe-outputs"] || {};
  const hasSafeOutputs = safeOutputs && typeof safeOutputs === "object" && !Array.isArray(safeOutputs) && Object.keys(safeOutputs).length > 0;
  const telemetryEntry = telemetryForWorkflow(workflowPath, relativePath, frontmatter, telemetryIndex);
  const telemetryMetrics = telemetryEntry.metrics || {};
  const telemetryObserved = coerceBool(telemetryEntry.observed, Object.keys(telemetryMetrics).length > 0) && Object.keys(telemetryMetrics).length > 0;
  const telemetryValidated = coerceBool(telemetryEntry.validated, false) && Object.keys(telemetryMetrics).length > 0;
  const hasDirectObservability = hasObservabilityConfig(frontmatter);
  const hasImported = hasImportedObservability(workflowPath, frontmatter);
  const observabilityDeclared = hasDirectObservability || hasImported;
  const observabilityScore = scoreObservability(hasDirectObservability, hasImported, telemetryMetrics);
  const safeOutputScore = scoreSafeOutputs(safeOutputs);
  const [agenticFraction, deterministicFraction] = estimateAgenticFraction(frontmatter, body);
  const permissionScore = permissionsRisk(frontmatter.permissions);
  const notes = [];
  if (!hasLockfile) {
    notes.push("missing lockfile");
  } else if (lockfileStale) {
    notes.push("stale lockfile");
  }
  if (!strict) {
    notes.push("strict mode disabled");
  }
  if (timeoutMinutes === null) {
    notes.push("missing timeout");
  }
  if (!hasSafeOutputs) {
    notes.push("missing safe outputs");
  }
  if (!observabilityDeclared) {
    notes.push("observability not declared");
  } else if (!telemetryObserved) {
    notes.push("telemetry not observed");
  } else if (!telemetryValidated) {
    notes.push("telemetry not validated");
  }

  const network = frontmatter.network;
  const usefulness = scoreUsefulness(frontmatter, body, safeOutputScore, telemetryMetrics);
  const adoption = scoreAdoption(frontmatter, telemetryMetrics);
  const trust = scoreTrust(strict, timeoutMinutes, hasLockfile, lockfileStale, safeOutputScore, observabilityScore, telemetryMetrics);
  const cost = scoreCost(frontmatter, body, telemetryMetrics, agenticFraction);
  const maintenanceDrag = scoreMaintenance(
    frontmatter,
    body,
    0,
    agenticFraction,
    countSteps(frontmatter["pre-agent-steps"]) > 0,
    countSteps(frontmatter["post-steps"]) > 0
  );
  const risk = scoreRisk(permissionScore, strict, timeoutMinutes, hasLockfile, lockfileStale, safeOutputScore, observabilityScore, network, agenticFraction);

  return {
    path: relativePath,
    name: normalizeText(frontmatter.name) || path.basename(workflowPath, ".md"),
    description: normalizeText(frontmatter.description),
    has_lockfile: hasLockfile,
    lockfile_stale: lockfileStale,
    has_safe_outputs: hasSafeOutputs,
    has_observability: hasDirectObservability,
    has_imported_observability: hasImported,
    observability_declared: observabilityDeclared,
    telemetry_observed: telemetryObserved,
    telemetry_validated: telemetryValidated,
    telemetry_source: normalizeText(telemetryEntry.source),
    strict,
    timeout_minutes: timeoutMinutes,
    permissions_risk: permissionScore,
    tool_count: countTools(frontmatter),
    pre_agent_steps_count: countSteps(frontmatter["pre-agent-steps"]),
    post_steps_count: countSteps(frontmatter["post-steps"]),
    agentic_fraction: agenticFraction,
    deterministic_fraction: deterministicFraction,
    usefulness,
    adoption,
    trust,
    cost,
    risk,
    maintenance_drag: maintenanceDrag,
    overlap_drag: 0,
    yield: 0,
    intent_text: buildIntentText(workflowPath, frontmatter, body),
    recommendation_seed: observabilityScore < 0.4 ? "Instrument" : "Revise",
    evidence_quality: evidenceQualityForWorkflow(observabilityDeclared, telemetryObserved, telemetryValidated, telemetryMetrics),
    notes,
    telemetry_metrics: telemetryMetrics,
  };
}

function pairKey(left, right) {
  return left < right ? `${left}\u0000${right}` : `${right}\u0000${left}`;
}

function getSimilarity(similarities, left, right) {
  return similarities.get(pairKey(left, right)) ?? 0;
}

function computeSimilarityMatrix(workflows) {
  const documents = {};
  const docFrequency = new Map();

  for (const workflow of workflows) {
    const counts = new Map();
    for (const token of tokenize(workflow.intent_text || "")) {
      counts.set(token, (counts.get(token) || 0) + 1);
    }
    documents[workflow.path] = counts;
    for (const token of counts.keys()) {
      docFrequency.set(token, (docFrequency.get(token) || 0) + 1);
    }
  }

  const totalDocs = Math.max(1, workflows.length);
  const vectors = {};
  const norms = {};
  for (const [docPath, counts] of Object.entries(documents)) {
    const vector = new Map();
    for (const [token, frequency] of counts.entries()) {
      const idf = Math.log((1 + totalDocs) / (1 + (docFrequency.get(token) || 0))) + 1;
      vector.set(token, frequency * idf);
    }
    vectors[docPath] = vector;
    norms[docPath] = Math.sqrt([...vector.values()].reduce((acc, value) => acc + value * value, 0)) || 1;
  }

  const similarities = new Map();
  const paths = workflows.map((workflow) => workflow.path);
  for (let i = 0; i < paths.length; i += 1) {
    const left = paths[i];
    for (const right of paths.slice(i + 1)) {
      let dot = 0;
      for (const token of vectors[left].keys()) {
        if (vectors[right].has(token)) {
          dot += vectors[left].get(token) * vectors[right].get(token);
        }
      }
      similarities.set(pairKey(left, right), roundScore(dot / (norms[left] * norms[right])));
    }
  }
  return [similarities, documents];
}

function buildOverlapClusters(workflows, similarities, documents) {
  const adjacency = new Map();
  for (const [key, similarity] of similarities.entries()) {
    if (similarity < OVERLAP_THRESHOLD) {
      continue;
    }
    const [left, right] = key.split("\u0000");
    if (!adjacency.has(left)) {
      adjacency.set(left, new Set());
    }
    if (!adjacency.has(right)) {
      adjacency.set(right, new Set());
    }
    adjacency.get(left).add(right);
    adjacency.get(right).add(left);
  }

  const seen = new Set();
  const clusters = [];
  for (const workflow of workflows) {
    const p = workflow.path;
    if (seen.has(p) || !adjacency.has(p)) {
      continue;
    }
    const stack = [p];
    const members = [];
    while (stack.length > 0) {
      const current = stack.pop();
      if (seen.has(current)) {
        continue;
      }
      seen.add(current);
      members.push(current);
      const peers = [...(adjacency.get(current) || [])].sort();
      for (const peer of peers) {
        if (!seen.has(peer)) {
          stack.push(peer);
        }
      }
    }
    members.sort();
    if (members.length < 2) {
      continue;
    }
    let maxOverlap = 0;
    const tokenCounts = new Map();
    for (let i = 0; i < members.length; i += 1) {
      for (const right of members.slice(i + 1)) {
        maxOverlap = Math.max(maxOverlap, getSimilarity(similarities, members[i], right));
      }
      const doc = documents[members[i]] || new Map();
      for (const [token, count] of doc.entries()) {
        tokenCounts.set(token, (tokenCounts.get(token) || 0) + count);
      }
    }
    const reason = [...tokenCounts.entries()]
      .sort((a, b) => b[1] - a[1])
      .slice(0, 4)
      .map(([token]) => token)
      .join(", ") || "shared operational intent";
    clusters.push({ workflows: members, max_overlap: roundScore(maxOverlap), reason });
  }
  return clusters;
}

function portfolioOverlapDrag(similarities) {
  let drag = 0;
  for (const similarity of similarities.values()) {
    drag += similarity * similarity * 2;
  }
  return roundTo(drag, 4);
}

function computeWorkflowYield(usefulness, adoption, trust, cost, risk, maintenanceDrag, overlapDrag) {
  const denominator = 1 + cost + risk + maintenanceDrag + overlapDrag;
  if (denominator <= 0) {
    return 0;
  }
  return roundTo((usefulness * adoption * trust) / denominator, 4);
}

function assignRecommendation(workflow, clusteredPaths) {
  if (!workflow.has_observability && !workflow.has_imported_observability) {
    return "Instrument";
  }
  if (workflow.yield < 0.08 && workflow.trust < 0.45 && (workflow.risk > 0.55 || workflow.cost > 0.55 || workflow.maintenance_drag > 0.55)) {
    return "Retire";
  }
  if (workflow.overlap_drag >= 0.45 || clusteredPaths.has(workflow.path)) {
    return "Merge";
  }
  if (
    workflow.usefulness >= 0.35 &&
    (workflow.cost > 0.55 || workflow.risk > 0.55 || workflow.maintenance_drag > 0.55 || workflow.agentic_fraction > 0.65)
  ) {
    return "Revise";
  }
  return "Keep";
}

function computeEpisodeMetrics(workflows, similarities) {
  const buckets = {
    "pr-pipeline": [],
    "issue-pipeline": [],
    "release-pipeline": [],
    "incident-pipeline": [],
  };
  for (const workflow of workflows) {
    const text = (workflow.intent_text || "").toLowerCase();
    if (text.includes("pull request") || text.includes("pr") || text.includes("review")) {
      buckets["pr-pipeline"].push(workflow);
    } else if (text.includes("issue") || text.includes("triage")) {
      buckets["issue-pipeline"].push(workflow);
    } else if (text.includes("release") || text.includes("deploy")) {
      buckets["release-pipeline"].push(workflow);
    } else if (text.includes("incident") || text.includes("security")) {
      buckets["incident-pipeline"].push(workflow);
    }
  }

  const episodes = [];
  for (const [label, members] of Object.entries(buckets)) {
    if (members.length < 2) {
      continue;
    }
    const paths = members.map((member) => member.path);
    const pairScores = [];
    for (let i = 0; i < paths.length; i += 1) {
      for (const right of paths.slice(i + 1)) {
        pairScores.push(getSimilarity(similarities, paths[i], right));
      }
    }
    if (pairScores.length === 0) {
      continue;
    }
    const avgOverlap = pairScores.reduce((acc, score) => acc + score, 0) / pairScores.length;
    const avgCost = members.reduce((acc, member) => acc + member.cost, 0) / members.length;
    const avgYield = members.reduce((acc, member) => acc + member.yield, 0) / members.length;
    const coordinationDrag = roundScore(avgOverlap * (0.5 + avgCost));
    const episodeYield = roundTo(Math.max(0, avgYield / (1 + coordinationDrag)), 4);
    if (avgOverlap < 0.2 && coordinationDrag < 0.2) {
      continue;
    }
    episodes.push({
      episode: label,
      workflows: paths,
      coordination_drag: coordinationDrag,
      episode_yield: episodeYield,
      evidence_quality: avgOverlap >= 0.35 ? "medium" : "low",
    });
  }
  return episodes;
}

function computeOrganizationalHealth(workflows, overlapDragValue) {
  if (workflows.length === 0) {
    return { fragmentation: 0, reuse: 0, trust_concentration: 0, governance_drag: 0, notes: [] };
  }
  const reuse = roundScore(
    workflows.filter((workflow) => workflow.has_imported_observability || (workflow.tool_count || 0) > 0).length / workflows.length
  );
  const averageTrust = workflows.reduce((acc, workflow) => acc + workflow.trust, 0) / workflows.length;
  const trustConcentration = roundScore(Math.max(...workflows.map((workflow) => workflow.trust)) - averageTrust);
  const governanceDrag = roundScore(workflows.reduce((acc, workflow) => acc + workflow.risk + workflow.agentic_fraction * 0.25, 0) / workflows.length);
  const fragmentation = roundScore(clamp((overlapDragValue / Math.max(1, workflows.length)) * 0.7 + (1 - reuse) * 0.3));
  const notes = [];
  if (fragmentation > 0.6 && averageTrust < 0.5) {
    notes.push("High overlap plus uneven trust suggests organizational fragmentation.");
  }
  if (reuse > 0.55 && fragmentation < 0.45 && averageTrust > 0.55) {
    notes.push("Shared imports and higher trust indicate improving operational coherence.");
  }
  if (governanceDrag > 0.55) {
    notes.push("Governance drag is elevated by broad scope, missing telemetry, or high agentic fractions.");
  }
  return {
    fragmentation,
    reuse,
    trust_concentration: trustConcentration,
    governance_drag: governanceDrag,
    notes,
  };
}

function portfolioEvidenceQuality(workflows, telemetryObservedCoverage, telemetryValidatedCoverage) {
  if (telemetryValidatedCoverage >= 0.75 && workflows.every((workflow) => workflow.evidence_quality !== "low")) {
    return "high";
  }
  if (telemetryValidatedCoverage >= 0.35 || (telemetryObservedCoverage >= 0.5 && telemetryValidatedCoverage > 0)) {
    return "medium";
  }
  return "low";
}

function buildRecommendationSeed(workflows) {
  const buckets = Object.fromEntries(["Keep", "Revise", "Merge", "Instrument", "Retire"].map((bucket) => [bucket.toLowerCase(), []]));
  for (const workflow of workflows) {
    buckets[workflow.recommendation_seed.toLowerCase()].push(workflow.path);
  }
  return buckets;
}

function computePortfolioMetrics(workflows, overlapDragValue) {
  if (workflows.length === 0) {
    return {
      workflow_count: 0,
      portfolio_yield: 0,
      portfolio_overlap_drag: 0,
      portfolio_cost: 0,
      portfolio_risk: 0,
      portfolio_maintenance_drag: 0,
      average_agentic_fraction: 0,
      average_deterministic_fraction: 0,
      observability_declared_coverage: 0,
      telemetry_observed_coverage: 0,
      telemetry_validated_coverage: 0,
      telemetry_coverage: 0,
      evidence_quality: "low",
    };
  }
  const averageYield = workflows.reduce((acc, workflow) => acc + workflow.yield, 0) / workflows.length;
  const observabilityDeclaredCoverage = workflows.filter((workflow) => workflow.observability_declared).length / workflows.length;
  const telemetryObservedCoverage = workflows.filter((workflow) => workflow.telemetry_observed).length / workflows.length;
  const telemetryValidatedCoverage = workflows.filter((workflow) => workflow.telemetry_validated).length / workflows.length;
  return {
    workflow_count: workflows.length,
    portfolio_yield: roundTo(averageYield - LAMBDA * overlapDragValue, 4),
    portfolio_overlap_drag: roundTo(overlapDragValue, 4),
    portfolio_cost: roundTo(workflows.reduce((acc, workflow) => acc + workflow.cost, 0) / workflows.length, 4),
    portfolio_risk: roundTo(workflows.reduce((acc, workflow) => acc + workflow.risk, 0) / workflows.length, 4),
    portfolio_maintenance_drag: roundTo(workflows.reduce((acc, workflow) => acc + workflow.maintenance_drag, 0) / workflows.length, 4),
    average_agentic_fraction: roundTo(workflows.reduce((acc, workflow) => acc + workflow.agentic_fraction, 0) / workflows.length, 4),
    average_deterministic_fraction: roundTo(workflows.reduce((acc, workflow) => acc + workflow.deterministic_fraction, 0) / workflows.length, 4),
    observability_declared_coverage: roundTo(observabilityDeclaredCoverage, 4),
    telemetry_observed_coverage: roundTo(telemetryObservedCoverage, 4),
    telemetry_validated_coverage: roundTo(telemetryValidatedCoverage, 4),
    telemetry_coverage: roundTo(telemetryValidatedCoverage, 4),
    evidence_quality: portfolioEvidenceQuality(workflows, telemetryObservedCoverage, telemetryValidatedCoverage),
  };
}

function precompute(workflowsRoot, otelSummaryPath = null) {
  const telemetryIndex = loadOtelSummary(otelSummaryPath);
  const workflowFiles = discoverWorkflowFiles(workflowsRoot);
  const workflows = workflowFiles.map((workflowPath) => buildWorkflowRecord(workflowPath, workflowsRoot, telemetryIndex));
  const [similarities, documents] = computeSimilarityMatrix(workflows);

  const overlapByPath = new Map();
  const overlapPeers = new Map();
  for (const [key, similarity] of similarities.entries()) {
    const [left, right] = key.split("\u0000");
    const squared = similarity * similarity;
    overlapByPath.set(left, (overlapByPath.get(left) || 0) + squared);
    overlapByPath.set(right, (overlapByPath.get(right) || 0) + squared);
    if (!overlapPeers.has(left)) {
      overlapPeers.set(left, {});
    }
    if (!overlapPeers.has(right)) {
      overlapPeers.set(right, {});
    }
    overlapPeers.get(left)[right] = similarity;
    overlapPeers.get(right)[left] = similarity;
  }

  for (const workflow of workflows) {
    workflow.overlap_drag = roundScore(overlapByPath.get(workflow.path) || 0);
    workflow.maintenance_drag = roundScore(workflow.maintenance_drag + workflow.overlap_drag * 0.2);
    workflow.yield = computeWorkflowYield(
      workflow.usefulness,
      workflow.adoption,
      workflow.trust,
      workflow.cost,
      workflow.risk,
      workflow.maintenance_drag,
      workflow.overlap_drag
    );
    workflow.overlap_peers = overlapPeers.get(workflow.path) || {};
  }

  const overlapClusters = buildOverlapClusters(workflows, similarities, documents);
  const clusteredPaths = new Set(overlapClusters.flatMap((cluster) => cluster.workflows));
  for (const workflow of workflows) {
    workflow.recommendation_seed = assignRecommendation(workflow, clusteredPaths);
  }

  const overlapDragValue = portfolioOverlapDrag(similarities);
  const portfolioMetrics = computePortfolioMetrics(workflows, overlapDragValue);
  const telemetryCoverage = {
    observability_declared_coverage: portfolioMetrics.observability_declared_coverage,
    telemetry_observed_coverage: portfolioMetrics.telemetry_observed_coverage,
    telemetry_validated_coverage: portfolioMetrics.telemetry_validated_coverage,
    observability_declared_workflows: workflows.filter((workflow) => workflow.observability_declared).map((workflow) => workflow.path),
    observed_workflows: workflows.filter((workflow) => workflow.telemetry_observed).map((workflow) => workflow.path),
    validated_workflows: workflows.filter((workflow) => workflow.telemetry_validated).map((workflow) => workflow.path),
    declared_without_observation: workflows
      .filter((workflow) => workflow.observability_declared && !workflow.telemetry_observed)
      .map((workflow) => workflow.path),
    observed_without_validation: workflows
      .filter((workflow) => workflow.telemetry_observed && !workflow.telemetry_validated)
      .map((workflow) => workflow.path),
    missing_workflows: workflows.filter((workflow) => !workflow.observability_declared).map((workflow) => workflow.path),
  };

  const episodeMetrics = computeEpisodeMetrics(workflows, similarities);
  const organizationalHealth = computeOrganizationalHealth(workflows, overlapDragValue);
  const overlapPairs = [...similarities.entries()]
    .map(([key, score]) => {
      const [left, right] = key.split("\u0000");
      return { left, right, score };
    })
    .sort((a, b) => `${a.left}:${a.right}`.localeCompare(`${b.left}:${b.right}`));

  return {
    portfolio_metrics: portfolioMetrics,
    workflows,
    overlap_clusters: overlapClusters,
    telemetry_coverage: telemetryCoverage,
    episode_metrics: episodeMetrics,
    organizational_health_signals: organizationalHealth,
    recommendations_seed: buildRecommendationSeed(workflows),
    overlap_pairs: overlapPairs,
  };
}

function parseArgs(argv = process.argv.slice(2)) {
  const parsed = {};
  for (let i = 0; i < argv.length; i += 2) {
    const key = argv[i];
    const value = argv[i + 1];
    if (!key?.startsWith("--") || value === undefined) {
      throw new InputError("Expected --workflows <path> --out <path>");
    }
    parsed[key.slice(2)] = value;
  }
  return parsed;
}

function runPrecompute({ workspace, workflows, out }) {
  ensureRequired({ workspace, workflows, out }, ["workspace", "workflows", "out"]);
  const workflowsRoot = path.resolve(workspace, workflows);
  const payload = precompute(workflowsRoot, process.env.AWY_OTEL_SUMMARY_JSON);
  writeJson(out, payload, { sortKeys: true });
  return payload;
}

function main(argv = process.argv.slice(2)) {
  const args = parseArgs(argv);
  const payload = precompute(path.resolve(args.workflows), process.env.AWY_OTEL_SUMMARY_JSON);
  writeJson(args.out, payload, { sortKeys: true });
  return 0;
}

if (require.main === module) {
  process.exitCode = main();
}

module.exports = {
  InputError,
  clamp,
  parseFrontmatterText,
  readWorkflow,
  discoverWorkflowFiles,
  normalizeImportPaths,
  hasImportedObservability,
  normalizeTelemetryEntry,
  loadOtelSummary,
  buildWorkflowRecord,
  computeSimilarityMatrix,
  buildOverlapClusters,
  portfolioOverlapDrag,
  computeWorkflowYield,
  estimateAgenticFraction,
  permissionsRisk,
  computePortfolioMetrics,
  portfolioEvidenceQuality,
  precompute,
  runPrecompute,
  main,
};
