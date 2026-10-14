import { describe, expect, it, vi } from "vitest";
import { createRequire } from "module";

const require = createRequire(import.meta.url);
const { createOrUpdatePullRequest } = require("./create_or_update_pull_request.cjs");
const { TransportIntegrityError } = require("./transport_integrity.cjs");

describe("createOrUpdatePullRequest", () => {
  const options = {
    repoParts: { owner: "example-org", repo: "target-repo" },
    title: "Test pull request",
    body: "Test body",
    branchName: "example-org:feature",
    baseBranch: "main",
    draft: false,
  };

  it("includes head_repo for a same-organization fork", async () => {
    const githubClient = { rest: { pulls: { create: vi.fn().mockResolvedValue({ data: {} }) } } };

    await createOrUpdatePullRequest({ ...options, githubClient, headRepo: "example-org/automation-fork" });

    expect(githubClient.rest.pulls.create).toHaveBeenCalledWith(expect.objectContaining({ head: "example-org:feature", head_repo: "example-org/automation-fork" }));
  });

  it("omits head_repo for a pull request in the target repository", async () => {
    const githubClient = { rest: { pulls: { create: vi.fn().mockResolvedValue({ data: {} }) } } };

    await createOrUpdatePullRequest({ ...options, githubClient });
    expect(githubClient.rest.pulls.create).toHaveBeenCalledWith(expect.not.objectContaining({ head_repo: expect.anything() }));
  });

  it.each(["feature", "fix-network-timeout"])("rechecks transport integrity before a retried API write for %s", async branch => {
    vi.useFakeTimers();
    vi.stubGlobal("core", { info: vi.fn(), debug: vi.fn(), warning: vi.fn() });
    let tampered = false;
    const githubClient = {
      rest: {
        pulls: {
          create: vi.fn(async () => {
            tampered = true;
            throw Object.assign(new Error("Service unavailable"), { status: 503 });
          }),
        },
      },
    };
    const assertTransportUnchanged = vi.fn(() => {
      if (tampered) throw new TransportIntegrityError(`/tmp/gh-aw/aw-${branch}.patch`);
    });
    try {
      const result = createOrUpdatePullRequest({ ...options, githubClient, assertTransportUnchanged });
      const rejection = expect(result).rejects.toThrow("SECURITY:");
      await vi.runAllTimersAsync();
      await rejection;
      expect(assertTransportUnchanged).toHaveBeenCalledTimes(2);
      expect(githubClient.rest.pulls.create).toHaveBeenCalledTimes(1);
      expect(core.info.mock.calls.filter(([message]) => message.startsWith("Retry attempt"))).toHaveLength(1);
    } finally {
      vi.useRealTimers();
      vi.unstubAllGlobals();
    }
  });

  it.each(["network", "timeout", "rate-limit"])("does not retry a pre-write integrity failure containing %s", async branch => {
    vi.useFakeTimers();
    vi.stubGlobal("core", { info: vi.fn(), debug: vi.fn(), warning: vi.fn() });
    const githubClient = { rest: { pulls: { create: vi.fn() } } };
    const assertTransportUnchanged = vi.fn(() => {
      throw new TransportIntegrityError(`/tmp/gh-aw/aw-${branch}.patch`);
    });
    try {
      await expect(createOrUpdatePullRequest({ ...options, githubClient, assertTransportUnchanged })).rejects.toThrow("SECURITY:");
      expect(assertTransportUnchanged).toHaveBeenCalledTimes(1);
      expect(githubClient.rest.pulls.create).not.toHaveBeenCalled();
      expect(vi.getTimerCount()).toBe(0);
    } finally {
      vi.useRealTimers();
      vi.unstubAllGlobals();
    }
  });
});
