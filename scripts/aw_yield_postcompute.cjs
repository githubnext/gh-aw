#!/usr/bin/env node

const fs = require("fs");
const path = require("path");
const {
  FinalizeError,
  ALLOWED_RECOMMENDATIONS,
  LAMBDA,
  clamp,
  normalizeText,
  ensureRequired,
  ensureParentDir,
  writeJson,
  loadJson,
} = require("./aw_yield_shared.cjs");
const pre = require("./aw_yield_precompute.cjs");

const ALLOWED_BUCKETS = Object.fromEntries(ALLOWED_RECOMMENDATIONS.map((name) => [name.toLowerCase(), name]));
const AGENT_SUMMARY_CANDIDATES = ["portfolio-yield-agent.json", "aw-portfolio-yield-agent.json"];
const MAX_REPORT_LENGTH = 45000;

function clampWorkflowScores(workflow) {
  const bounded = { ...workflow };
  for (const key of [
    "permissions_risk",
    "agentic_fraction",
    "deterministic_fraction",
    "usefulness",
    "adoption",
    "trust",
    "cost",
    "risk",
    "maintenance_drag",
    "overlap_drag",
    "yield",
  ]) {
    bounded[key] = Math.round(clamp(workflow[key] ?? 0) * 10000) / 10000;
  }
  bounded.deterministic_fraction = Math.round(clamp(1 - bounded.agentic_fraction) * 10000) / 10000;
  bounded.notes = [...new Set(Array.isArray(workflow.notes) ? workflow.notes : [])];
  return bounded;
}

function recommendationBuckets(seed, workflows) {
  const buckets = Object.fromEntries(Object.keys(ALLOWED_BUCKETS).map((bucket) => [bucket, []]));
  for (const [bucket, entries] of Object.entries(seed || {})) {
    const lower = bucket.toLowerCase();
    if (!Object.prototype.hasOwnProperty.call(buckets, lower)) {
      continue;
    }
    for (const entry of entries || []) {
      let candidatePaths;
      if (entry && typeof entry === "object") {
        if (Array.isArray(entry.paths)) {
          candidatePaths = entry.paths.map((value) => normalizeText(value));
        } else {
          const pathValue = entry.path;
          candidatePaths = pathValue === undefined || pathValue === null ? [] : [normalizeText(pathValue)];
        }
      } else {
        candidatePaths = [normalizeText(entry)];
      }
      for (const p of candidatePaths) {
        if (workflows[p] && !buckets[lower].includes(p)) {
          buckets[lower].push(p);
        }
      }
    }
  }
  return buckets;
}

function readAgentSummary(agentDir) {
  for (const candidate of AGENT_SUMMARY_CANDIDATES) {
    const candidatePath = path.join(agentDir, candidate);
    if (!fs.existsSync(candidatePath)) {
      continue;
    }
    const payload = loadJson(candidatePath, FinalizeError);
    if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
      throw new FinalizeError(`Agent summary must be an object: ${candidatePath}`);
    }
    return payload;
  }
  return {};
}

function normalizeAgentBuckets(agentSummary, workflows) {
  const notes = [];
  const rawBuckets =
    agentSummary.recommendations && typeof agentSummary.recommendations === "object" && !Array.isArray(agentSummary.recommendations)
      ? agentSummary.recommendations
      : agentSummary;
  const buckets = Object.fromEntries(Object.keys(ALLOWED_BUCKETS).map((bucket) => [bucket, []]));
  const seen = {};

  for (const bucket of Object.keys(ALLOWED_BUCKETS)) {
    const entries = rawBuckets && typeof rawBuckets === "object" ? rawBuckets[bucket] ?? rawBuckets[ALLOWED_BUCKETS[bucket]] ?? [] : [];
    if (entries === null || entries === undefined) {
      continue;
    }
    if (!Array.isArray(entries)) {
      throw new FinalizeError(`Recommendation bucket '${bucket}' must be a list`);
    }
    for (const entry of entries) {
      const p = normalizeText(entry && typeof entry === "object" ? entry.path : entry);
      if (!p) {
        continue;
      }
      if (!workflows[p]) {
        notes.push(`Ignored unknown workflow in recommendations: ${p}`);
        continue;
      }
      const other = seen[p];
      if (other && other !== bucket) {
        notes.push(`Ignored conflicting recommendation for '${p}' in bucket '${bucket}' (already in '${other}').`);
        continue;
      }
      seen[p] = bucket;
      if (!buckets[bucket].includes(p)) {
        buckets[bucket].push(p);
      }
    }
  }

  const telemetryClaims = Array.isArray(agentSummary.telemetry_claims) ? agentSummary.telemetry_claims : [];
  for (const claim of telemetryClaims) {
    if (!claim || typeof claim !== "object" || Array.isArray(claim)) {
      notes.push("Ignored malformed telemetry claim from agent output.");
      continue;
    }
    const p = normalizeText(claim.path || claim.workflow);
    const metric = normalizeText(claim.metric);
    if (!p || !workflows[p] || !Object.prototype.hasOwnProperty.call(workflows[p].telemetry_metrics || {}, metric)) {
      notes.push(`Ignored invented telemetry claim: ${p || "unknown"}::${metric || "unknown"}`);
    }
  }
  return [buckets, notes];
}

function fillMissingRecommendations(current, seeds, workflows) {
  const assigned = new Set(Object.values(current).flat());
  for (const [bucket, entries] of Object.entries(seeds)) {
    for (const p of entries) {
      if (workflows[p] && !assigned.has(p)) {
        current[bucket] = current[bucket] || [];
        current[bucket].push(p);
        assigned.add(p);
      }
    }
  }
  for (const p of Object.keys(workflows)) {
    if (!assigned.has(p)) {
      current.revise = current.revise || [];
      current.revise.push(p);
    }
  }
  return Object.fromEntries(Object.entries(current).map(([bucket, entries]) => [bucket, [...new Set(entries)].sort()]));
}

function recomputeOverlapDrag(payload) {
  if (!Array.isArray(payload.overlap_pairs)) {
    return 0;
  }
  let drag = 0;
  for (const pair of payload.overlap_pairs) {
    if (!pair || typeof pair !== "object") {
      continue;
    }
    const score = clamp(pair.score ?? 0);
    drag += score ** 2 * 2;
  }
  return Math.round(drag * 10000) / 10000;
}

function deriveEvidenceQuality(workflows, baseQuality) {
  const observedCoverage = workflows.filter((workflow) => workflow.telemetry_observed).length / Math.max(1, workflows.length);
  const validatedCoverage = workflows.filter((workflow) => workflow.telemetry_validated).length / Math.max(1, workflows.length);
  const derived = pre.portfolioEvidenceQuality(workflows, observedCoverage, validatedCoverage);
  const order = { low: 0, medium: 1, high: 2 };
  return order[derived] <= (order[baseQuality] ?? 0) ? derived : baseQuality;
}

function topActions(finalPayload) {
  const actions = [];
  const { instrument = [], merge = [], retire = [], revise = [] } = finalPayload;
  if (instrument.length > 0) {
    actions.push(`Instrument ${instrument[0]} with stable OTel evidence and safe-output validation.`);
  }
  if (merge.length > 0) {
    actions.push(`Consolidate overlap around ${merge[0]} to reduce portfolio drag.`);
  }
  if (revise.length > 0) {
    actions.push(`Revise ${revise[0]} to shift deterministic work out of the agent path.`);
  }
  if (retire.length > 0 && actions.length < 3) {
    actions.push(`Retire or quarantine ${retire[0]} if trust does not improve.`);
  }
  return actions.slice(0, 3);
}

function buildReportMarkdown(finalPayload, precomputePayload, agentSummary, postNotes) {
  const metrics = {
    "Portfolio yield": finalPayload.portfolio_yield,
    "Workflow count": finalPayload.workflow_count,
    "Agentic fraction": finalPayload.average_agentic_fraction,
    "Deterministic fraction": Math.round((1 - finalPayload.average_agentic_fraction) * 10000) / 10000,
    "Observability declared coverage": precomputePayload.portfolio_metrics?.observability_declared_coverage || 0,
    "Telemetry observed coverage": precomputePayload.portfolio_metrics?.telemetry_observed_coverage || 0,
    "Telemetry validated coverage": precomputePayload.portfolio_metrics?.telemetry_validated_coverage || 0,
    "High-overlap clusters": (finalPayload.overlap_clusters || []).length,
    "Estimated governance drag": finalPayload.organizational_health_signals?.governance_drag || 0,
    "Estimated trust score":
      Math.round(
        ((precomputePayload.workflows || []).reduce((acc, workflow) => acc + (workflow.trust || 0), 0) /
          Math.max(1, (precomputePayload.workflows || []).length)) *
          10000
      ) / 10000,
  };

  const workflowRows = (precomputePayload.workflows || [])
    .slice()
    .sort((left, right) => (right.yield || 0) - (left.yield || 0))
    .map((workflow) => {
      const recommendation =
        Object.keys(ALLOWED_BUCKETS).find((bucket) => (finalPayload[bucket] || []).includes(workflow.path))?.replace(/^./u, (c) => c.toUpperCase()) ||
        workflow.recommendation_seed ||
        "Revise";
      const noteText = (workflow.notes || []).slice(0, 2).join("; ") || "-";
      return `| \`${workflow.path}\` | ${recommendation} | ${(workflow.yield || 0).toFixed(4)} | ${(workflow.trust || 0).toFixed(4)} | ${(workflow.cost || 0).toFixed(4)} | ${(workflow.risk || 0).toFixed(4)} | ${(workflow.overlap_drag || 0).toFixed(4)} | ${(workflow.adoption || 0).toFixed(4)} | ${(workflow.agentic_fraction || 0).toFixed(4)} | ${noteText} |`;
    });

  const overlapLines = (finalPayload.overlap_clusters || []).map(
    (cluster) => `- ${(cluster.workflows || []).join(", ")} (max overlap ${(cluster.max_overlap || 0).toFixed(4)}; ${cluster.reason || ""})`
  );
  if (overlapLines.length === 0) {
    overlapLines.push("- No high-overlap clusters detected.");
  }

  const episodeLines = (precomputePayload.episode_metrics || []).map(
    (episode) =>
      `- ${episode.episode}: workflows=${(episode.workflows || []).join(", ")}; coordination drag=${(episode.coordination_drag || 0).toFixed(4)}; episode yield=${(episode.episode_yield || 0).toFixed(4)}`
  );

  const org = finalPayload.organizational_health_signals || {};
  let deterministicFindings = Array.isArray(agentSummary.deterministic_vs_agentic_findings) ? agentSummary.deterministic_vs_agentic_findings : [];
  if (deterministicFindings.length === 0) {
    deterministicFindings = (precomputePayload.workflows || [])
      .slice()
      .sort((left, right) => (right.agentic_fraction || 0) - (left.agentic_fraction || 0))
      .slice(0, 3)
      .filter((workflow) => (workflow.agentic_fraction || 0) > 0.6)
      .map((workflow) => `${workflow.path} has agentic fraction ${(workflow.agentic_fraction || 0).toFixed(4)} despite limited deterministic scaffolding.`);
  }

  const highestValueActions = agentSummary.highest_value_actions || topActions(finalPayload);
  const retirementCandidates = agentSummary.retirement_candidates || finalPayload.retire || [];
  const consolidation = agentSummary.consolidation_opportunities || finalPayload.merge || [];
  const instrumentationGaps = agentSummary.instrumentation_gaps || finalPayload.instrument || [];

  let executiveSummary = normalizeText(agentSummary.executive_summary);
  if (!executiveSummary) {
    if (finalPayload.evidence_quality === "low") {
      executiveSummary =
        "The workflow ecosystem is under-instrumented, so the portfolio signal is directionally useful but not yet strong enough for confident optimization.";
    } else if ((org.fragmentation || 0) > 0.6) {
      executiveSummary = "The workflow ecosystem is fragmenting: overlap drag and governance drag are eroding portfolio yield.";
    } else if (finalPayload.portfolio_yield > 0.12) {
      executiveSummary = "The workflow ecosystem is producing positive value overall, with enough trust and reuse to justify continued investment.";
    } else {
      executiveSummary = "The workflow ecosystem is mixed: some workflows are valuable, but overlap, cost, or trust gaps are holding the portfolio back.";
    }
  }

  const compactJson = JSON.stringify(
    {
      portfolio_yield: finalPayload.portfolio_yield,
      workflow_count: finalPayload.workflow_count,
      observability_declared_coverage: finalPayload.observability_declared_coverage || 0,
      telemetry_observed_coverage: finalPayload.telemetry_observed_coverage || 0,
      telemetry_validated_coverage: finalPayload.telemetry_validated_coverage || 0,
      keep: finalPayload.keep || [],
      revise: finalPayload.revise || [],
      merge: finalPayload.merge || [],
      instrument: finalPayload.instrument || [],
      retire: finalPayload.retire || [],
      evidence_quality: finalPayload.evidence_quality,
    },
    null,
    0
  );

  const lines = [
    "# Agentic Workflow Portfolio Yield Report",
    "",
    "## Executive Summary",
    "",
    executiveSummary,
    "",
    "## Portfolio Health",
    "",
    "| Metric | Value |",
    "|---|---:|",
    ...Object.entries(metrics).map(([metric, value]) => `| ${metric} | ${value} |`),
    "",
    "## Workflow Portfolio",
    "",
    "| Workflow | Recommendation | Yield | Trust | Cost | Risk | Overlap | Adoption | Agentic Fraction | Notes |",
    "|---|---|---:|---:|---:|---:|---:|---:|---:|---|",
    ...workflowRows,
    "",
    "## Overlap Clusters",
    "",
    ...overlapLines,
  ];

  if (episodeLines.length > 0) {
    lines.push("", "## Episode-Level Observations", "", ...episodeLines);
  }

  lines.push(
    "",
    "## Organizational Health Signals",
    "",
    `- fragmentation: ${(org.fragmentation || 0).toFixed(4)}`,
    `- reuse: ${(org.reuse || 0).toFixed(4)}`,
    `- trust concentration: ${(org.trust_concentration || 0).toFixed(4)}`,
    `- governance drag: ${(org.governance_drag || 0).toFixed(4)}`,
    ...((org.notes || []).map((note) => `- ${note}`)),
    ...postNotes.map((note) => `- ${note}`),
    "",
    "## Deterministic vs Agentic Findings",
    "",
    ...(deterministicFindings.length > 0
      ? deterministicFindings.map((item) => `- ${item}`)
      : ["- No outsized agentic misuse detected from current evidence."]),
    "",
    "## Highest-Value Actions",
    "",
    ...(highestValueActions.length > 0 ? highestValueActions.slice(0, 3).map((item, idx) => `${idx + 1}. ${item}`) : ["1. Improve observability coverage."]),
    "",
    "## Retirement Candidates",
    "",
    ...(retirementCandidates.length > 0 ? retirementCandidates.map((item) => `- ${item}`) : ["- No immediate retirement candidates."]),
    "",
    "## Consolidation Opportunities",
    "",
    ...(consolidation.length > 0 ? consolidation.map((item) => `- ${item}`) : ["- No consolidation opportunities identified."]),
    "",
    "## Instrumentation Gaps",
    "",
    ...(instrumentationGaps.length > 0 ? instrumentationGaps.map((item) => `- ${item}`) : ["- No critical instrumentation gaps detected."]),
    "",
    "## Deterministic Portfolio JSON",
    "",
    "```json",
    compactJson,
    "```"
  );

  let report = `${lines.join("\n").trim()}\n`;
  if (report.length > MAX_REPORT_LENGTH) {
    report = `${report.slice(0, MAX_REPORT_LENGTH - 16).trimEnd()}\n\n[truncated]\n`;
  }
  return report;
}

function finalize(precomputePayload, agentDir) {
  const workflowsRaw = precomputePayload.workflows;
  if (!Array.isArray(workflowsRaw)) {
    throw new FinalizeError("Precompute JSON must contain a workflows array");
  }
  const workflows = workflowsRaw.filter((workflow) => workflow && typeof workflow === "object").map((workflow) => clampWorkflowScores(workflow));
  const workflowIndex = {};
  for (const workflow of workflows) {
    const p = workflow.path;
    if (!p || workflowIndex[p]) {
      throw new FinalizeError(`Duplicate or missing workflow path: ${p}`);
    }
    workflowIndex[p] = workflow;
  }

  for (const workflow of workflows) {
    workflow.yield = pre.computeWorkflowYield(
      workflow.usefulness,
      workflow.adoption,
      workflow.trust,
      workflow.cost,
      workflow.risk,
      workflow.maintenance_drag,
      workflow.overlap_drag
    );
  }

  const seeds = recommendationBuckets(precomputePayload.recommendations_seed || {}, workflowIndex);
  const agentSummary = readAgentSummary(agentDir);
  const postNotes = [];
  const [agentBuckets, notes] = normalizeAgentBuckets(agentSummary, workflowIndex);
  postNotes.push(...notes);
  const buckets = fillMissingRecommendations(agentBuckets, seeds, workflowIndex);

  const overlapDragValue = recomputeOverlapDrag(precomputePayload);
  const portfolioYield =
    Math.round(
      ((workflows.reduce((acc, workflow) => acc + workflow.yield, 0) / Math.max(1, workflows.length) - LAMBDA * overlapDragValue) * 10000)
    ) / 10000;

  const finalPayload = {
    portfolio_yield: portfolioYield,
    workflow_count: workflows.length,
    portfolio_cost: Math.round((workflows.reduce((acc, workflow) => acc + workflow.cost, 0) / Math.max(1, workflows.length)) * 10000) / 10000,
    portfolio_risk: Math.round((workflows.reduce((acc, workflow) => acc + workflow.risk, 0) / Math.max(1, workflows.length)) * 10000) / 10000,
    portfolio_maintenance_drag:
      Math.round((workflows.reduce((acc, workflow) => acc + workflow.maintenance_drag, 0) / Math.max(1, workflows.length)) * 10000) / 10000,
    portfolio_overlap_drag: overlapDragValue,
    average_agentic_fraction:
      Math.round((workflows.reduce((acc, workflow) => acc + workflow.agentic_fraction, 0) / Math.max(1, workflows.length)) * 10000) / 10000,
    observability_declared_coverage:
      Math.round((workflows.filter((workflow) => workflow.observability_declared).length / Math.max(1, workflows.length)) * 10000) / 10000,
    telemetry_observed_coverage:
      Math.round((workflows.filter((workflow) => workflow.telemetry_observed).length / Math.max(1, workflows.length)) * 10000) / 10000,
    telemetry_validated_coverage:
      Math.round((workflows.filter((workflow) => workflow.telemetry_validated).length / Math.max(1, workflows.length)) * 10000) / 10000,
    evidence_quality: deriveEvidenceQuality(workflows, precomputePayload.portfolio_metrics?.evidence_quality || "low"),
    keep: buckets.keep || [],
    revise: buckets.revise || [],
    merge: buckets.merge || [],
    instrument: buckets.instrument || [],
    retire: buckets.retire || [],
    overlap_clusters: precomputePayload.overlap_clusters || [],
    organizational_health_signals: precomputePayload.organizational_health_signals || {},
  };

  finalPayload.report_markdown = buildReportMarkdown(finalPayload, { ...precomputePayload, workflows }, agentSummary, postNotes);
  return [finalPayload, agentSummary, postNotes];
}

function writeSafeOutput(agentDir, reportMarkdown) {
  const title = `Agentic Workflow Portfolio Yield Report — ${new Date().toISOString().slice(0, 10)}`;
  const payload = {
    items: [{ type: "create_issue", title, body: reportMarkdown }],
    errors: [],
  };
  fs.writeFileSync(path.join(agentDir, "agent_output.json"), `${JSON.stringify(payload, null, 2)}\n`, "utf8");
}

function errorPayload(message) {
  return {
    error: message,
    portfolio_yield: 0,
    workflow_count: 0,
    portfolio_cost: 0,
    portfolio_risk: 0,
    portfolio_maintenance_drag: 0,
    portfolio_overlap_drag: 0,
    average_agentic_fraction: 0,
    observability_declared_coverage: 0,
    telemetry_observed_coverage: 0,
    telemetry_validated_coverage: 0,
    evidence_quality: "low",
    keep: [],
    revise: [],
    merge: [],
    instrument: [],
    retire: [],
    overlap_clusters: [],
    organizational_health_signals: { fragmentation: 0, reuse: 0, trust_concentration: 0, governance_drag: 0, notes: [message] },
    report_markdown: "# Agentic Workflow Portfolio Yield Report\n\n## Executive Summary\n\nPostcompute failed safely.\n",
  };
}

function parseArgs(argv = process.argv.slice(2)) {
  const parsed = {};
  for (let i = 0; i < argv.length; i += 2) {
    const key = argv[i];
    const value = argv[i + 1];
    if (!key?.startsWith("--") || value === undefined) {
      throw new FinalizeError("Expected --precompute <path> --agent-output <path> --out <path>");
    }
    parsed[key.slice(2)] = value;
  }
  return parsed;
}

function runPostcompute({ workspace, precompute: precomputePath, agentOutput, out }) {
  ensureRequired({ workspace, precompute: precomputePath, agentOutput, out }, ["workspace", "precompute", "agentOutput", "out"]);
  const absolutePrecomputePath = path.resolve(workspace, precomputePath);
  const absoluteAgentDir = path.resolve(workspace, agentOutput);
  const absoluteOut = path.resolve(workspace, out);
  ensureParentDir(absoluteOut);
  ensureParentDir(path.join(absoluteAgentDir, "agent_output.json"));
  const precomputePayload = loadJson(absolutePrecomputePath, FinalizeError);
  if (!precomputePayload || typeof precomputePayload !== "object" || Array.isArray(precomputePayload)) {
    throw new FinalizeError("Precompute JSON must be an object");
  }
  const [finalPayload] = finalize(precomputePayload, absoluteAgentDir);
  writeSafeOutput(absoluteAgentDir, finalPayload.report_markdown);
  writeJson(absoluteOut, finalPayload, { sortKeys: true });
  return finalPayload;
}

function main(argv = process.argv.slice(2)) {
  const args = parseArgs(argv);
  const outPath = path.resolve(args.out);
  const agentDir = path.resolve(args["agent-output"]);
  ensureParentDir(outPath);
  fs.mkdirSync(agentDir, { recursive: true });
  try {
    const precomputePayload = loadJson(path.resolve(args.precompute), FinalizeError);
    if (!precomputePayload || typeof precomputePayload !== "object" || Array.isArray(precomputePayload)) {
      throw new FinalizeError("Precompute JSON must be an object");
    }
    const [finalPayload] = finalize(precomputePayload, agentDir);
    writeSafeOutput(agentDir, finalPayload.report_markdown);
    writeJson(outPath, finalPayload, { sortKeys: true });
    return 0;
  } catch (error) {
    const payload = errorPayload(error.message);
    writeJson(outPath, payload, { sortKeys: true });
    return 1;
  }
}

if (require.main === module) {
  process.exitCode = main();
}

module.exports = {
  FinalizeError,
  clampWorkflowScores,
  recommendationBuckets,
  normalizeAgentBuckets,
  fillMissingRecommendations,
  recomputeOverlapDrag,
  deriveEvidenceQuality,
  topActions,
  buildReportMarkdown,
  finalize,
  writeSafeOutput,
  errorPayload,
  runPostcompute,
  main,
};
