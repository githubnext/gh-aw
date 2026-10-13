import { beforeEach, describe, it, expect, vi } from "vitest";
import { createRequire } from "module";
import { expectExecutionProvenance, useExecutionProvenanceFixture } from "./test_execution_provenance.js";

const require = createRequire(import.meta.url);
const { appendExecutionProvenance, resolvePublishingJobId } = require("./execution_provenance.cjs");

useExecutionProvenanceFixture();
beforeEach(() => vi.stubEnv("GITHUB_API_URL", "https://api.github.com"));

describe("publishing job resolution", () => {
  const job = {
    id: 123,
    name: "caller / safe_outputs (matrix)",
    run_id: 12345,
    run_attempt: 1,
    check_run_url: "https://api.github.com/repos/test-owner/test-repo/check-runs/456789",
  };
  const metadata = () => expectExecutionProvenance(appendExecutionProvenance("Content", "Workflow", ""));
  const client = jobs => ({
    rest: { actions: { listJobsForWorkflowRunAttempt: vi.fn() } },
    paginate: vi.fn().mockResolvedValue(jobs),
  });

  it("paginates the recorded attempt and distinguishes same-name reusable/matrix executions", async () => {
    const github = client([{ ...job, id: 122, check_run_url: job.check_run_url.replace("456789", "456788") }, job]);

    await expect(resolvePublishingJobId(github, metadata())).resolves.toBe("123");
    expect(github.paginate).toHaveBeenCalledWith(github.rest.actions.listJobsForWorkflowRunAttempt, {
      owner: "test-owner",
      repo: "test-repo",
      run_id: "12345",
      attempt_number: "1",
      per_page: 100,
    });
  });

  it.each([
    { name: "missing check run", jobs: [] },
    { name: "ambiguous check run", jobs: [job, { ...job, id: 124 }] },
    { name: "wrong run", jobs: [{ ...job, run_id: 54321 }] },
    { name: "wrong attempt", jobs: [{ ...job, run_attempt: 2 }] },
    { name: "wrong repository", jobs: [{ ...job, check_run_url: job.check_run_url.replace("test-repo", "destination") }] },
    { name: "wrong API host", jobs: [{ ...job, check_run_url: job.check_run_url.replace("api.github.com", "attacker.invalid") }] },
    { name: "invalid job ID", jobs: [{ ...job, id: 0 }] },
  ])("rejects $name instead of falling back to the logical job key", async ({ jobs }) => {
    await expect(resolvePublishingJobId(client(jobs), metadata())).rejects.toThrow("Cannot uniquely resolve");
  });

  it("supports enterprise API prefixes and Jobs responses without run_attempt", async () => {
    vi.stubEnv("GITHUB_API_URL", "https://github.example.com/api/v3/");
    const { run_attempt, ...legacyJob } = job;
    const github = client([{ ...legacyJob, check_run_url: job.check_run_url.replace("https://api.github.com", "https://github.example.com/api/v3") }]);
    await expect(resolvePublishingJobId(github, metadata())).resolves.toBe("123");
  });

  it.each([{ version: 2 }, { repository: "owner/repo/extra" }, { run_id: "0" }, { run_attempt: "" }, { publishing_job: { key: "safe_outputs" } }])("rejects invalid execution metadata before calling the API: %j", async overrides => {
    const github = client([job]);
    await expect(resolvePublishingJobId(github, { ...metadata(), ...overrides })).rejects.toThrow("Invalid");
    expect(github.paginate).not.toHaveBeenCalled();
  });
});
