// @ts-check

import { describe, it, expect, afterEach, vi } from "vitest";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const { buildIssueClusteringHint, generateIssueClusteringHint } = require("./issue_clustering.cjs");
const { buildIssueReport } = require("./issue_report.cjs");
const { renderSizedReport } = require("./report_size.cjs");
const { redactReport } = require("./report_redaction.cjs");

function readHint(body) {
  return JSON.parse(body.match(/<!-- gh-aw-clustering-hint: (.+) -->/)[1]);
}

afterEach(() => vi.unstubAllEnvs());

describe("runtime issue clustering hints", () => {
  it.each(["agent-output", "runtime-failure", "tracking", "maintenance"])("emits a versioned, parseable %s hint", kind => {
    const hint = readHint(generateIssueClusteringHint(kind));
    expect(hint).toEqual({ version: 1, kind, categories: [], guidance: expect.any(String) });
    expect(hint.guidance.length).toBeGreaterThan(50);
    expect(buildIssueClusteringHint(kind).agentTainted).toBe(false);
  });

  it("distinguishes operational symptoms from proven duplicate findings", () => {
    const hint = readHint(generateIssueClusteringHint("runtime-failure", ["missing_tool", "agent_failure", "missing_tool"]));
    expect(hint.categories).toEqual(["agent_failure", "missing_tool"]);
    expect(hint.guidance).toContain("not an agent finding about the target project");
    expect(hint.guidance).toContain("matching categories or repeated titles alone do not prove duplicates");
    expect(readHint(generateIssueClusteringHint("tracking")).guidance).toContain("not an independent finding");
  });

  it("does not use workflow or execution provenance as a clustering key", () => {
    const before = generateIssueClusteringHint("agent-output");
    vi.stubEnv("GH_AW_WORKFLOW_NAME", "Different Workflow");
    vi.stubEnv("GH_AW_ENGINE_ID", "other-engine");
    vi.stubEnv("GITHUB_RUN_ID", "12345");
    expect(generateIssueClusteringHint("agent-output")).toBe(before);
    expect(readHint(before).guidance).toContain("not the generating workflow, engine, model, run, date, or title prefix");
  });

  it("keeps metadata inside one comment and redacts secret values", () => {
    vi.stubEnv("GH_AW_SECRET_NAMES", "CLUSTER_FIXTURE");
    vi.stubEnv("SECRET_CLUSTER_FIXTURE", "private-clustering-fixture");
    const body = generateIssueClusteringHint("runtime-failure", ['bad --> <img src=x> "category"', "private-clustering-fixture"]);
    expect(body.match(/<!--/g)).toHaveLength(1);
    expect(body.match(/-->/g)).toHaveLength(1);
    expect(body).not.toContain("<img");
    expect(body).not.toContain("private-clustering-fixture");
    expect(readHint(body).categories).toContain('bad --> <img src=x> "category"');
  });

  it("preserves framework hints through redaction, size fitting, and publication parsing", () => {
    const report = buildIssueReport("Finding", "x".repeat(100000));
    report.children.push(buildIssueClusteringHint("agent-output"));
    const { body } = renderSizedReport(redactReport(report, () => true));
    expect(body.length).toBeLessThanOrEqual(65536);
    expect(readHint(body).kind).toBe("agent-output");
    expect(readHint(renderSizedReport(buildIssueReport("Finding", body)).body)).toEqual(readHint(body));
  });

  it("rejects unknown runtime classifications", () => {
    // @ts-expect-error Exercise runtime validation of an invalid classification.
    expect(() => buildIssueClusteringHint("unknown")).toThrow("Unknown issue clustering kind");
  });
});
