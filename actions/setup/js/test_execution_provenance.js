import { afterEach, beforeEach, expect, vi } from "vitest";

export const executionEnv = {
  GITHUB_REPOSITORY_ID: "987654321",
  GITHUB_REPOSITORY: "test-owner/test-repo",
  GITHUB_RUN_ID: "12345",
  GITHUB_RUN_ATTEMPT: "1",
  GITHUB_JOB: "safe_outputs",
  GH_AW_JOB_CHECK_RUN_ID: "456789",
  GH_AW_WORKFLOW_ID: "test-workflow",
  GITHUB_SERVER_URL: "https://github.com",
};

export const requiredExecutionEnvKeys = Object.keys(executionEnv).filter(key => key !== "GITHUB_SERVER_URL");

export async function expectMissingRuntimeFailure(invoke, key) {
  vi.stubEnv(key, undefined);
  const result = await invoke().catch(error => ({ success: false, error: error.message }));
  expect(result).toEqual(expect.objectContaining({ success: false, error: expect.stringContaining(key) }));
}

export function useExecutionProvenanceFixture() {
  beforeEach(() => {
    for (const [key, value] of Object.entries(executionEnv)) {
      vi.stubEnv(key, value);
    }
  });
  afterEach(() => {
    vi.unstubAllEnvs();
  });
}

export function expectExecutionProvenance(body, overrides = {}) {
  const markers = [...body.matchAll(/<!-- gh-aw-execution:\s*(.*?)\s*-->/g)];
  expect(markers).toHaveLength(1);
  const payload = JSON.parse(markers[0][1]);
  expect(payload).toEqual({
    version: 1,
    repo_id: executionEnv.GITHUB_REPOSITORY_ID,
    repository: executionEnv.GITHUB_REPOSITORY,
    aw_workflow_id: executionEnv.GH_AW_WORKFLOW_ID,
    run_id: executionEnv.GITHUB_RUN_ID,
    run_attempt: executionEnv.GITHUB_RUN_ATTEMPT,
    run_url: "https://github.com/test-owner/test-repo/actions/runs/12345",
    publishing_job: {
      key: executionEnv.GITHUB_JOB,
      check_run_id: executionEnv.GH_AW_JOB_CHECK_RUN_ID,
    },
    ...overrides,
  });
  return payload;
}
