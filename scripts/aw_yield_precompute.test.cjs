#!/usr/bin/env node

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("fs");
const path = require("path");
const pre = require("./aw_yield_precompute.cjs");

function writeFile(filePath, content) {
  fs.mkdirSync(path.dirname(filePath), { recursive: true });
  fs.writeFileSync(filePath, content, "utf8");
}

test("discoverWorkflowFiles excludes shared", () => {
  const root = fs.mkdtempSync(path.join(process.cwd(), "aw-yield-pre-"));
  const workflows = path.join(root, ".github", "workflows");
  writeFile(path.join(workflows, "alpha.md"), "---\non: workflow_dispatch\n---\n# Alpha\n");
  writeFile(path.join(workflows, "shared", "helper.md"), "---\non: workflow_dispatch\n---\n# Helper\n");
  const files = pre.discoverWorkflowFiles(workflows).map((p) => path.basename(p));
  assert.deepEqual(files, ["alpha.md"]);
});

test("normalizeImportPaths resolves shared imports and blocks escape", () => {
  const root = fs.mkdtempSync(path.join(process.cwd(), "aw-yield-imports-"));
  const workflows = path.join(root, ".github", "workflows");
  const workflow = path.join(workflows, "alpha.md");
  writeFile(workflow, "---\nimports:\n  - shared/otel-observability.md\n---\n# Alpha\n");
  writeFile(path.join(workflows, "shared", "otel-observability.md"), "---\nobservability:\n  otlp:\n    endpoint:\n      url: x\n---\n");

  const [fm] = pre.readWorkflow(workflow);
  const imports = pre.normalizeImportPaths(workflow, fm);
  assert.equal(imports.length, 1);
  assert.equal(path.basename(imports[0]), "otel-observability.md");

  writeFile(workflow, "---\nimports:\n  - ../outside.md\n---\n# Alpha\n");
  const [escapedFm] = pre.readWorkflow(workflow);
  assert.equal(pre.normalizeImportPaths(workflow, escapedFm).length, 0);
});

test("telemetry selection prefers workflow_path over name collision", () => {
  const root = fs.mkdtempSync(path.join(process.cwd(), "aw-yield-telemetry-"));
  const workflows = path.join(root, ".github", "workflows");
  const first = path.join(workflows, "foo", "alpha.md");
  const second = path.join(workflows, "bar", "alpha.md");
  writeFile(first, "---\nname: Alpha One\n---\n# Alpha One\n");
  writeFile(second, "---\nname: Alpha Two\n---\n# Alpha Two\n");

  const telemetryPath = path.join(root, "summary.json");
  fs.writeFileSync(
    telemetryPath,
    JSON.stringify({
      workflows: [
        { workflow_path: ".github/workflows/foo/alpha.md", workflow_invocation_count: 1, observed: true, validated: true },
        { workflow_path: ".github/workflows/bar/alpha.md", workflow_invocation_count: 7, observed: true, validated: true },
      ],
    }),
    "utf8"
  );

  const telemetryIndex = pre.loadOtelSummary(telemetryPath);
  const record = pre.buildWorkflowRecord(second, workflows, telemetryIndex);
  assert.equal(record.telemetry_metrics.workflow_invocation_count, 7);
});

test("computePortfolioMetrics preserves coverage split", () => {
  const workflows = [
    {
      yield: 0.3,
      cost: 0.2,
      risk: 0.2,
      maintenance_drag: 0.2,
      agentic_fraction: 0.4,
      deterministic_fraction: 0.6,
      observability_declared: true,
      telemetry_observed: true,
      telemetry_validated: true,
      evidence_quality: "high",
    },
    {
      yield: 0.2,
      cost: 0.2,
      risk: 0.2,
      maintenance_drag: 0.2,
      agentic_fraction: 0.5,
      deterministic_fraction: 0.5,
      observability_declared: true,
      telemetry_observed: false,
      telemetry_validated: false,
      evidence_quality: "low",
    },
    {
      yield: 0.1,
      cost: 0.2,
      risk: 0.2,
      maintenance_drag: 0.2,
      agentic_fraction: 0.6,
      deterministic_fraction: 0.4,
      observability_declared: false,
      telemetry_observed: false,
      telemetry_validated: false,
      evidence_quality: "low",
    },
  ];

  const metrics = pre.computePortfolioMetrics(workflows, 0);
  assert.equal(metrics.observability_declared_coverage, 0.6667);
  assert.equal(metrics.telemetry_observed_coverage, 0.3333);
  assert.equal(metrics.telemetry_validated_coverage, 0.3333);
});
