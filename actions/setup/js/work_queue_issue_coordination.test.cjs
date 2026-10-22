import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { withProjectionLocks } = require("./work_queue_issue_coordination.cjs");

describe("projection lock acquisition diagnostics", () => {
  let stderr;
  const output = () => stderr.mock.calls.map(call => call[0]).join("");

  beforeEach(() => {
    vi.stubEnv("DEBUG", "");
    vi.stubEnv("ACTIONS_RUNNER_DEBUG", "");
    vi.stubEnv("RUNNER_DEBUG", "");
    stderr = vi.spyOn(process.stderr, "write").mockReturnValue(true);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllEnvs();
  });

  it("reports incomplete acquisition before rejecting an unscoped projection without inventing a target count", async () => {
    const githubClient = { graphql: vi.fn().mockResolvedValue({ l0: null }) };
    const operation = vi.fn();

    await expect(withProjectionLocks({ githubClient, repositoryId: "private-repository", head: "a".repeat(40), keys: ["work:private-work"] }, operation)).rejects.toMatchObject({ code: "projection_coordination_pending" });

    expect(operation).not.toHaveBeenCalled();
    expect(githubClient.graphql).toHaveBeenCalledTimes(1);
    expect(githubClient.graphql.mock.calls[0][1].request).toEqual({ retries: 0, timeout: 30000 });
    expect(output()).toContain("projection.locks.acquire.pending requested=1 acknowledged=0");
    expect(output()).not.toContain("pending_targets");
    expect(output()).not.toContain("private");
  });

  it("retains the known pending target count for scoped projections", async () => {
    const githubClient = { graphql: vi.fn().mockResolvedValue({ l0: null }) };
    const operation = vi.fn();

    const result = await withProjectionLocks({ githubClient, repositoryId: "private-repository", head: "a".repeat(40), keys: ["work:private-work"], scopes: [{ work_id: "private-work", keys: ["work:private-work"] }] }, operation);

    expect(result.pending).toHaveLength(1);
    expect(operation).not.toHaveBeenCalled();
    expect(githubClient.graphql).toHaveBeenCalledTimes(1);
    expect(output()).toContain("projection.locks.acquire.pending requested=1 acknowledged=0 pending_targets=1");
    expect(output()).not.toContain("private");
  });
});
