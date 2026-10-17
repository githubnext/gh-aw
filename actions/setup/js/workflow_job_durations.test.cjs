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
    env: {
      GH_AW_JOB_RESULTS: JSON.stringify(needs),
      GH_AW_JOB_DISPLAY_NAMES: JSON.stringify(names),
      GH_AW_JOB_TIMING_ELIGIBLE: JSON.stringify(Object.fromEntries(Object.keys(needs).map(id => [id, true]))),
      GITHUB_RUN_ATTEMPT: "2",
    },
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

  it("never falls back to IDs for dynamic jobs with swapped resolved names", async () => {
    const options = fixture({ build: { result: "success" }, lint: { result: "success" }, unnamed: { result: "success" } }, { unnamed: "unnamed" });
    options.env.GH_AW_JOB_TIMING_ELIGIBLE = JSON.stringify({ build: false, lint: false, unnamed: true });
    expect(await collectJobDurations({ ...options, jobs: [completed("lint"), completed("build", { completed_at: "2026-10-16T10:00:03Z" }), completed("unnamed")] })).toEqual({ unnamed: 2000 });
    expect(options.warn).not.toHaveBeenCalled();
  });

  it.each(["Build", "Build (linux)", "Build / child"])("omits a grouped dependency even when a partial rerun returns only one child: %s", name => {
    const options = fixture({ grouped: { result: "failure" } }, { grouped: "Build" });
    options.env.GH_AW_JOB_TIMING_ELIGIBLE = JSON.stringify({ grouped: false });
    return collectJobDurations({ ...options, jobs: [completed(name)] }).then(result => {
      expect(result).toEqual({});
      expect(options.warn).not.toHaveBeenCalled();
    });
  });

  it.each(["Build (linux)", "Build / child"])("rejects a current-attempt child colliding with a separate job from a previous attempt: %s", name => {
    const options = fixture({ grouped: { result: "failure" }, separate: { result: "success" } }, { grouped: "Build", separate: name });
    options.env.GH_AW_JOB_TIMING_ELIGIBLE = JSON.stringify({ grouped: false, separate: true });
    return collectJobDurations({ ...options, jobs: [completed(name), completed(name, { run_attempt: 1, completed_at: "2026-10-16T10:00:08Z" })] }).then(result => {
      expect(result).toEqual({});
      expect(options.warn).toHaveBeenCalledExactlyOnceWith("Unified session: job duration unavailable for separate (no unique Actions job match)");
    });
  });

  it("requires explicit compiler timing eligibility rather than assuming unnamed jobs are singles", async () => {
    const options = fixture();
    delete options.env.GH_AW_JOB_TIMING_ELIGIBLE;
    expect(await collectJobDurations({ ...options, jobs: [completed("Agent")] })).toEqual({});
    expect(options.githubClient.rest.actions.listJobsForWorkflowRunAttempt).not.toHaveBeenCalled();
  });

  it.each(["null", "[]", "not-json"])("rejects malformed compiler timing eligibility: %s", async eligibility => {
    const options = fixture();
    options.env.GH_AW_JOB_TIMING_ELIGIBLE = eligibility;
    expect(await collectJobDurations(options)).toEqual({});
    expect(options.warn).toHaveBeenCalledExactlyOnceWith("Unified session: job durations unavailable (invalid job metadata)");
    expect(options.githubClient.rest.actions.listJobsForWorkflowRunAttempt).not.toHaveBeenCalled();
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

  it("warns and falls back to the API when cached durations are invalid", async () => {
    for (const cached of ['{"agent":-1}', '{"bad id":1}', "[1]", "null", "not json"]) {
      const options = fixture();
      options.env.GH_AW_JOB_DURATIONS = cached;
      expect(await collectJobDurations(options)).toEqual({ agent: 2000 });
      expect(options.warn).toHaveBeenCalledWith("Unified session: invalid cached job durations; reading Actions job timing");
    }
  });

  it("returns valid cached durations without an API request", async () => {
    const options = fixture();
    options.env.GH_AW_JOB_DURATIONS = '{"agent":5}';
    expect(await collectJobDurations(options)).toEqual({ agent: 5 });
    expect(options.githubClient.rest.actions.listJobsForWorkflowRunAttempt).not.toHaveBeenCalled();
  });

  it("rejects non-object job metadata", async () => {
    for (const key of ["GH_AW_JOB_RESULTS", "GH_AW_JOB_DISPLAY_NAMES", "GH_AW_JOB_TIMING_ELIGIBLE"]) {
      const options = fixture();
      options.env[key] = "[]";
      expect(await collectJobDurations(options)).toEqual({});
      expect(options.warn).toHaveBeenCalledWith("Unified session: job durations unavailable (invalid job metadata)");
    }
  });

  it("warns when the run context or attempt is missing", async () => {
    const options = fixture();
    options.env.GITHUB_RUN_ATTEMPT = "0";
    expect(await collectJobDurations(options)).toEqual({});
    expect(options.warn).toHaveBeenCalledWith("Unified session: job durations unavailable (missing Actions run context)");
  });

  it("skips jobs with invalid IDs or non-terminal results", async () => {
    const options = fixture({ "bad id": { result: "success" }, agent: { result: "skipped" } }, { agent: "Agent" });
    expect(await collectJobDurations(options)).toEqual({});
  });
});
