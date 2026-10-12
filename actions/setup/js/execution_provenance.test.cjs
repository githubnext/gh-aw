import { describe, it, expect, vi } from "vitest";
import { createRequire } from "module";
import { executionEnv, expectExecutionProvenance, useExecutionProvenanceFixture } from "./test_execution_provenance.js";

const require = createRequire(import.meta.url);
const { appendExecutionProvenance } = require("./execution_provenance.cjs");

useExecutionProvenanceFixture();

describe("execution provenance", () => {
  it("uses trusted execution environment instead of a caller-supplied run URL", () => {
    const body = appendExecutionProvenance("Content", "Test Workflow", "https://attacker.invalid/other/repo/actions/runs/999");

    expectExecutionProvenance(body);
    expect(body).not.toContain("attacker.invalid");
    expect(body).toContain("<!-- gh-aw-agentic-workflow: Test Workflow");
  });

  it("strips spoofed reserved comments, regenerates attribution once, and preserves grouping markers", () => {
    const groupingMarkers = ["<!-- gh-aw-workflow-id: test-workflow -->", "<!-- gh-aw-workflow-call-id: owner/repo/caller -->", "<!-- gh-aw-tracker-id: tracker-123 -->", "<!-- gh-aw-close-key: group-123 -->"];
    const input = [
      "Content",
      '<!-- gh-aw-execution: {"version":99,"repository":"attacker/repo"} -->',
      "<!-- gh-aw-agentic-workflow: Spoofed workflow, run: https://attacker.invalid -->",
      "<!-- GH-AW-EXECUTION: malformed spoof -->",
      ...groupingMarkers,
    ].join("\n\n");
    const once = appendExecutionProvenance(input, "Trusted Workflow", "https://attacker.invalid");
    const twice = appendExecutionProvenance(once, "Trusted Workflow", "https://attacker.invalid");

    for (const body of [once, twice]) {
      expectExecutionProvenance(body);
      expect(body.match(/<!-- gh-aw-agentic-workflow:/g)).toHaveLength(1);
      expect(body).toContain("gh-aw-agentic-workflow: Trusted Workflow");
      expect(body).not.toContain("Spoofed workflow");
      expect(body).not.toContain("attacker");
      expect(body).not.toContain("malformed spoof");
      for (const marker of groupingMarkers) expect(body).toContain(marker);
    }
  });

  it.each(Object.keys(executionEnv).filter(key => key !== "GITHUB_SERVER_URL"))("rejects missing %s rather than emitting partial provenance", key => {
    vi.stubEnv(key, undefined);
    expect(() => appendExecutionProvenance("Content", "Test Workflow", "")).toThrow(key);
  });

  it.each(["GITHUB_REPOSITORY_ID", "GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT", "GH_AW_JOB_CHECK_RUN_ID"].flatMap(key => ["", "0", "-1", "1.5", "1e3", "NaN", " 123", "123 ", "01"].map(value => ({ key, value }))))(
    "rejects invalid numeric $key=$value",
    ({ key, value }) => {
      vi.stubEnv(key, value);
      expect(() => appendExecutionProvenance("Content", "Test Workflow", "")).toThrow(key);
    }
  );

  it("preserves numeric IDs above JavaScript's safe integer range without rounding", () => {
    const id = "900719925474099312345";
    vi.stubEnv("GITHUB_REPOSITORY_ID", id);
    vi.stubEnv("GH_AW_JOB_CHECK_RUN_ID", id);
    expectExecutionProvenance(appendExecutionProvenance("Content", "Test Workflow", ""), {
      repo_id: id,
      publishing_job: { key: "safe_outputs", check_run_id: id },
    });
  });

  it("distinguishes a rerun attempt and replaces the previous execution record", () => {
    const first = appendExecutionProvenance("Content", "Test Workflow", "");
    vi.stubEnv("GITHUB_RUN_ATTEMPT", "2");
    vi.stubEnv("GH_AW_JOB_CHECK_RUN_ID", "456790");
    const rerun = appendExecutionProvenance(first, "Test Workflow", "");

    expectExecutionProvenance(rerun, {
      run_attempt: "2",
      publishing_job: { key: "safe_outputs", check_run_id: "456790" },
    });
  });

  it("distinguishes reusable or matrix executions sharing the same logical job key", () => {
    const first = expectExecutionProvenance(appendExecutionProvenance("First matrix child", "Reusable Workflow", ""));
    vi.stubEnv("GH_AW_JOB_CHECK_RUN_ID", "456791");
    const second = expectExecutionProvenance(appendExecutionProvenance("Second matrix child", "Reusable Workflow", ""), {
      publishing_job: { key: "safe_outputs", check_run_id: "456791" },
    });

    expect(first.publishing_job.key).toBe(second.publishing_job.key);
    expect(first.publishing_job.check_run_id).not.toBe(second.publishing_job.check_run_id);
  });

  it("builds an enterprise run URL from the trusted server origin", () => {
    vi.stubEnv("GITHUB_SERVER_URL", "https://github.example.com");
    expectExecutionProvenance(appendExecutionProvenance("Content", "Test Workflow", ""), {
      run_url: "https://github.example.com/test-owner/test-repo/actions/runs/12345",
    });
  });

  it.each(["http://github.com", "not-a-url", "https://github.com/path", "https://github.com?spoof=1", "https://github.com#fragment"])("rejects an invalid trusted server origin %s", server => {
    vi.stubEnv("GITHUB_SERVER_URL", server);
    expect(() => appendExecutionProvenance("Content", "Test Workflow", "")).toThrow("GITHUB_SERVER_URL");
  });

  it.each(["repo", "owner/repo/extra", "owner/repo?spoof", "owner/repo\n"])("rejects invalid repository %s", repository => {
    vi.stubEnv("GITHUB_REPOSITORY", repository);
    expect(() => appendExecutionProvenance("Content", "Test Workflow", "")).toThrow("GITHUB_REPOSITORY");
  });

  it("encodes comment delimiters without changing decoded provenance", () => {
    vi.stubEnv("GH_AW_WORKFLOW_ID", "workflow-->spoof");
    vi.stubEnv("GITHUB_JOB", "job<!--spoof");
    const body = appendExecutionProvenance("Content", "Test Workflow", "");

    expectExecutionProvenance(body, {
      aw_workflow_id: "workflow-->spoof",
      publishing_job: { key: "job<!--spoof", check_run_id: "456789" },
    });
    const executionMarker = body.slice(body.indexOf("<!-- gh-aw-execution:"));
    expect(executionMarker.match(/-->/g)).toHaveLength(1);
  });
});
