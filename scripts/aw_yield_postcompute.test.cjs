#!/usr/bin/env node

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("fs");
const path = require("path");
const post = require("./aw_yield_postcompute.cjs");

function samplePrecompute() {
  return {
    portfolio_metrics: {
      workflow_count: 2,
      portfolio_yield: 0.1,
      portfolio_overlap_drag: 0.4,
      portfolio_cost: 0.3,
      portfolio_risk: 0.2,
      portfolio_maintenance_drag: 0.3,
      average_agentic_fraction: 0.5,
      average_deterministic_fraction: 0.5,
      observability_declared_coverage: 0.5,
      telemetry_observed_coverage: 0.5,
      telemetry_validated_coverage: 0.5,
      telemetry_coverage: 0.5,
      evidence_quality: "low",
    },
    workflows: [
      {
        path: ".github/workflows/a.md",
        trust: 0.7,
        cost: 1.2,
        risk: -0.2,
        maintenance_drag: 0.4,
        overlap_drag: 0.5,
        usefulness: 0.8,
        adoption: 0.6,
        yield: 0,
        has_observability: false,
        has_imported_observability: false,
        observability_declared: false,
        telemetry_observed: false,
        telemetry_validated: false,
        permissions_risk: 1.2,
        agentic_fraction: 1.4,
        deterministic_fraction: -0.4,
        recommendation_seed: "Instrument",
        notes: ["observability not declared"],
        telemetry_metrics: {},
      },
      {
        path: ".github/workflows/b.md",
        trust: 0.8,
        cost: 0.3,
        risk: 0.2,
        maintenance_drag: 0.2,
        overlap_drag: 0.5,
        usefulness: 0.7,
        adoption: 0.5,
        yield: 0,
        has_observability: true,
        has_imported_observability: false,
        observability_declared: true,
        telemetry_observed: true,
        telemetry_validated: true,
        permissions_risk: 0.2,
        agentic_fraction: 0.4,
        deterministic_fraction: 0.6,
        recommendation_seed: "Keep",
        notes: [],
        telemetry_metrics: { success_rate: 0.9, workflow_invocation_count: 4, runtime_duration: 30 },
      },
    ],
    overlap_clusters: [{ workflows: [".github/workflows/a.md", ".github/workflows/b.md"], max_overlap: 0.9, reason: "review" }],
    episode_metrics: [],
    organizational_health_signals: { fragmentation: 0.6, reuse: 0.2, trust_concentration: 0.2, governance_drag: 0.7, notes: [] },
    recommendations_seed: {
      keep: [".github/workflows/b.md"],
      revise: [],
      merge: [],
      instrument: [".github/workflows/a.md"],
      retire: [],
    },
    overlap_pairs: [{ left: ".github/workflows/a.md", right: ".github/workflows/b.md", score: 0.9 }],
  };
}

test("clampWorkflowScores bounds invalid values", () => {
  const bounded = post.clampWorkflowScores(samplePrecompute().workflows[0]);
  assert.equal(bounded.permissions_risk, 1);
  assert.equal(bounded.cost, 1);
  assert.equal(bounded.risk, 0);
  assert.equal(bounded.agentic_fraction, 1);
  assert.equal(bounded.deterministic_fraction, 0);
});

test("finalize keeps buckets mutually exclusive and preserves coverage", () => {
  const dir = fs.mkdtempSync(path.join(process.cwd(), "aw-yield-post-"));
  fs.writeFileSync(
    path.join(dir, "portfolio-yield-agent.json"),
    JSON.stringify({ recommendations: { keep: [{ path: ".github/workflows/b.md" }], instrument: [{ path: ".github/workflows/a.md" }] } }),
    "utf8"
  );
  const [payload] = post.finalize(samplePrecompute(), dir);
  assert.deepEqual(payload.keep, [".github/workflows/b.md"]);
  assert.deepEqual(payload.instrument, [".github/workflows/a.md"]);
  assert.equal(payload.observability_declared_coverage, 0.5);
  assert.equal(payload.telemetry_observed_coverage, 0.5);
  assert.equal(payload.telemetry_validated_coverage, 0.5);
});

test("normalizeAgentBuckets ignores unknown/conflicting recommendations", () => {
  const workflows = Object.fromEntries(samplePrecompute().workflows.map((w) => [w.path, w]));
  const [buckets, notes] = post.normalizeAgentBuckets(
    {
      recommendations: {
        keep: [{ path: ".github/workflows/b.md" }],
        instrument: [{ path: ".github/workflows/a.md" }, { path: ".github/workflows/missing.md" }, { path: ".github/workflows/b.md" }],
      },
    },
    workflows
  );
  assert.deepEqual(buckets.keep, [".github/workflows/b.md"]);
  assert.deepEqual(buckets.instrument, [".github/workflows/a.md"]);
  assert.ok(notes.some((note) => note.toLowerCase().includes("unknown workflow")));
  assert.ok(notes.some((note) => note.toLowerCase().includes("conflicting recommendation")));
});

test("recommendationBuckets accepts merge paths list", () => {
  const workflows = Object.fromEntries(samplePrecompute().workflows.map((w) => [w.path, w]));
  const buckets = post.recommendationBuckets({ merge: [{ paths: [".github/workflows/a.md", ".github/workflows/b.md"] }] }, workflows);
  assert.deepEqual(buckets.merge, [".github/workflows/a.md", ".github/workflows/b.md"]);
});

test("finalize notes invented telemetry claims and keeps low evidence", () => {
  const dir = fs.mkdtempSync(path.join(process.cwd(), "aw-yield-post-claims-"));
  fs.writeFileSync(
    path.join(dir, "portfolio-yield-agent.json"),
    JSON.stringify({
      recommendations: { keep: [{ path: ".github/workflows/b.md" }], instrument: [{ path: ".github/workflows/a.md" }] },
      telemetry_claims: [{ path: ".github/workflows/a.md", metric: "success_rate" }],
      evidence_quality: "high",
    }),
    "utf8"
  );
  const [payload, , notes] = post.finalize(samplePrecompute(), dir);
  assert.equal(payload.evidence_quality, "low");
  assert.ok(notes.some((note) => note.toLowerCase().includes("invented telemetry")));
});
