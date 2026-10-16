import { describe, expect, it, vi } from "vitest";
import { collectJobDurations } from "./workflow_job_durations.cjs";

const completed = (name, extra = {}) => ({
  name,
  status: "completed",
  conclusion: "success",
  started_at: "2026-10-16T10:00:00Z",
  completed_at: "2026-10-16T10:00:02Z",
  run_attempt: 2,
  ...extra,
});
function fixture(needs = { agent: { result: "success" } }, names = { agent: "Agent" }) {
  const list = vi.fn().mockResolvedValue({ data: { jobs: [completed("Agent")] } });
  return {
    githubClient: { rest: { actions: { listJobsForWorkflowRunAttempt: list } } },
    runContext: { repo: { owner: "owner", repo: "repo" }, runId: 123 },
    env: { GH_AW_JOB_RESULTS: JSON.stringify(needs), GH_AW_JOB_DISPLAY_NAMES: JSON.stringify(names), GITHUB_RUN_ATTEMPT: "2" },
    warn: vi.fn(),
  };
}

describe("workflow job durations", () => {
  it("reads completed-job runtime from the current attempt rather than queue time or other attempts", async () => {
    const options = fixture();
    options.githubClient.rest.actions.listJobsForWorkflowRunAttempt.mockResolvedValue({
      data: { jobs: [completed("Agent", { created_at: "2026-10-16T09:00:00Z" }), completed("Agent", { run_attempt: 1 })] },
    });
    expect(await collectJobDurations(options)).toEqual({ agent: 2000 });
    expect(options.githubClient.rest.actions.listJobsForWorkflowRunAttempt).toHaveBeenCalledExactlyOnceWith({ owner: "owner", repo: "repo", run_id: 123, attempt_number: 2, per_page: 100, page: 1 });
    expect(options.warn).not.toHaveBeenCalled();
  });

  it("reuses the conclusion outcome listing without another request", async () => {
    const options = fixture();
    expect(await collectJobDurations({ ...options, jobs: [completed("Agent")] })).toEqual({ agent: 2000 });
    expect(options.githubClient.rest.actions.listJobsForWorkflowRunAttempt).not.toHaveBeenCalled();
    options.env.GH_AW_JOB_DURATIONS = '{"agent":2000}';
    expect(await collectJobDurations(options)).toEqual({ agent: 2000 });
    expect(options.githubClient.rest.actions.listJobsForWorkflowRunAttempt).not.toHaveBeenCalled();
  });

  it.each(['{"agent":-1}', '{"agent":"PRIVATE"}', "[]", "null", "PRIVATE_JSON"])("warns and falls back when cached durations are invalid: %s", cached => {
    const options = fixture();
    options.env.GH_AW_JOB_DURATIONS = cached;
    return collectJobDurations(options).then(result => {
      expect(result).toEqual({ agent: 2000 });
      expect(options.warn).toHaveBeenCalledExactlyOnceWith("Unified session: invalid cached job durations; reading Actions job timing");
    });
  });

  it("paginates the current attempt's job listing", async () => {
    const options = fixture();
    options.githubClient.rest.actions.listJobsForWorkflowRunAttempt
      .mockResolvedValueOnce({ data: { jobs: Array.from({ length: 100 }, (_, index) => completed(`unrelated-${index}`)) } })
      .mockResolvedValueOnce({ data: { jobs: [completed("Agent")] } });
    expect(await collectJobDurations(options)).toEqual({ agent: 2000 });
    expect(options.githubClient.rest.actions.listJobsForWorkflowRunAttempt).toHaveBeenCalledTimes(2);
  });

  it.each(["success", "failure", "cancelled"])("retains measured zero duration for a %s dependency", async result => {
    const options = fixture({ agent: { result } });
    expect(await collectJobDurations({ ...options, jobs: [completed("Agent", { conclusion: result, completed_at: "2026-10-16T10:00:00Z" })] })).toEqual({ agent: 0 });
  });

  it("does not invent duration for skipped jobs", async () => {
    const options = fixture({ agent: { result: "skipped" } });
    expect(await collectJobDurations({ ...options, jobs: [completed("Agent")] })).toEqual({});
    expect(options.warn).not.toHaveBeenCalled();
  });

  it.each([{ completed_at: null }, { started_at: null }, { started_at: "invalid" }, { completed_at: "2026-10-16T09:59:59Z" }, { status: "in_progress" }, { conclusion: "skipped" }])(
    "omits invalid or incomplete job timing: %j",
    async extra => {
      const options = fixture();
      expect(await collectJobDurations({ ...options, jobs: [completed("Agent", extra)] })).toEqual({});
      expect(options.warn).toHaveBeenCalledOnce();
    }
  );

  it.each([[], [completed("Agent"), completed("Agent")], [completed("Agent (linux)"), completed("Agent (windows)")], [completed("caller / Agent")]].map(jobs => ({ jobs })))("omits unmatched or grouped jobs: %j", async ({ jobs }) => {
    const options = fixture();
    expect(await collectJobDurations({ ...options, jobs })).toEqual({});
    expect(options.warn).toHaveBeenCalledOnce();
  });

  it("rejects duplicate display names even if the API returns one matching job", async () => {
    const options = fixture({ agent: { result: "success" } }, { agent: "Agent", custom: "Agent" });
    expect(await collectJobDurations({ ...options, jobs: [completed("Agent")] })).toEqual({});
    expect(options.warn).toHaveBeenCalledOnce();
  });

  it("warns on API failures without logging request details or fabricating durations", async () => {
    const options = fixture();
    options.githubClient.rest.actions.listJobsForWorkflowRunAttempt.mockRejectedValue(new Error("PRIVATE_TOKEN"));
    expect(await collectJobDurations(options)).toEqual({});
    expect(options.warn).toHaveBeenCalledExactlyOnceWith("Unified session: job durations unavailable (Actions API request failed)");
  });

  it("performs no lookup without dependency metadata", async () => {
    const options = fixture();
    delete options.env.GH_AW_JOB_RESULTS;
    expect(await collectJobDurations(options)).toEqual({});
    expect(options.githubClient.rest.actions.listJobsForWorkflowRunAttempt).not.toHaveBeenCalled();
  });

  it.each(["not-json", "null", "[]"])("warns without calling the API for invalid metadata: %s", metadata => {
    const options = fixture();
    options.env.GH_AW_JOB_RESULTS = metadata;
    return collectJobDurations(options).then(result => {
      expect(result).toEqual({});
      expect(options.warn).toHaveBeenCalledOnce();
      expect(options.githubClient.rest.actions.listJobsForWorkflowRunAttempt).not.toHaveBeenCalled();
    });
  });
});
