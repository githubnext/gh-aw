import { describe, it, expect } from "vitest";

const { resolveInvocationContext } = await import("./invocation_context_helpers.cjs");

describe("invocation_context_helpers", () => {
  it("keeps native event context unchanged", () => {
    const resolved = resolveInvocationContext({
      eventName: "issue_comment",
      repo: { owner: "side-owner", repo: "side-repo" },
      payload: {
        issue: { number: 42 },
        repository: {
          owner: { login: "side-owner" },
          name: "side-repo",
        },
      },
    });

    expect(resolved.source).toBe("native");
    expect(resolved.eventName).toBe("issue_comment");
    expect(resolved.workflowRepo).toEqual({ owner: "side-owner", repo: "side-repo" });
    expect(resolved.eventRepo).toEqual({ owner: "side-owner", repo: "side-repo" });
    expect(resolved.eventPayload.issue.number).toBe(42);
  });

  it("unwraps repository_dispatch payload and repo", () => {
    const resolved = resolveInvocationContext({
      eventName: "repository_dispatch",
      repo: { owner: "side-owner", repo: "side-repo" },
      payload: {
        action: "issue_comment",
        client_payload: {
          issue: { number: 99 },
          repository: {
            owner: { login: "target-owner" },
            name: "target-repo",
          },
        },
      },
    });

    expect(resolved.source).toBe("repository_dispatch");
    expect(resolved.eventName).toBe("issue_comment");
    expect(resolved.workflowRepo).toEqual({ owner: "side-owner", repo: "side-repo" });
    expect(resolved.eventRepo).toEqual({ owner: "target-owner", repo: "target-repo" });
    expect(resolved.eventPayload.issue.number).toBe(99);
  });

  it("supports workflow_dispatch overrides from inputs", () => {
    const resolved = resolveInvocationContext({
      eventName: "workflow_dispatch",
      repo: { owner: "side-owner", repo: "side-repo" },
      payload: {
        inputs: {
          event_name: "issues",
          event_repo: "target-owner/target-repo",
          event_payload: JSON.stringify({
            issue: { number: 777 },
          }),
        },
      },
    });

    expect(resolved.source).toBe("workflow_dispatch");
    expect(resolved.eventName).toBe("issues");
    expect(resolved.workflowRepo).toEqual({ owner: "side-owner", repo: "side-repo" });
    expect(resolved.eventRepo).toEqual({ owner: "target-owner", repo: "target-repo" });
    expect(resolved.eventPayload.issue.number).toBe(777);
  });

  it.each(["target_repo", "targetRepo"])("rejects workflow_dispatch %s when not in allowlist", targetRepoKey => {
    const originalAllowedRepos = process.env.GH_AW_ALLOWED_REPOS;
    try {
      process.env.GH_AW_ALLOWED_REPOS = "allowed-owner/allowed-repo";

      expect(() =>
        resolveInvocationContext({
          eventName: "workflow_dispatch",
          repo: { owner: "side-owner", repo: "side-repo" },
          payload: {
            inputs: {
              [targetRepoKey]: "target-owner/target-repo",
            },
          },
        })
      ).toThrow(/ERR_VALIDATION: Repository 'target-owner\/target-repo' is not in the allowed-repos list/);
    } finally {
      if (originalAllowedRepos === undefined) {
        delete process.env.GH_AW_ALLOWED_REPOS;
      } else {
        process.env.GH_AW_ALLOWED_REPOS = originalAllowedRepos;
      }
    }
  });

  it("allows workflow_dispatch target_repo when it is in allowlist", () => {
    const originalAllowedRepos = process.env.GH_AW_ALLOWED_REPOS;
    try {
      process.env.GH_AW_ALLOWED_REPOS = "target-owner/target-repo";

      const resolved = resolveInvocationContext({
        eventName: "workflow_dispatch",
        repo: { owner: "side-owner", repo: "side-repo" },
        payload: {
          inputs: {
            target_repo: "target-owner/target-repo",
          },
        },
      });

      expect(resolved.eventRepo).toEqual({ owner: "target-owner", repo: "target-repo" });
    } finally {
      if (originalAllowedRepos === undefined) {
        delete process.env.GH_AW_ALLOWED_REPOS;
      } else {
        process.env.GH_AW_ALLOWED_REPOS = originalAllowedRepos;
      }
    }
  });

  it("allows workflow_dispatch without target_repo inputs", () => {
    const resolved = resolveInvocationContext({
      eventName: "workflow_dispatch",
      repo: { owner: "side-owner", repo: "side-repo" },
      payload: {
        inputs: {
          event_name: "issues",
        },
      },
    });

    expect(resolved.eventName).toBe("issues");
    expect(resolved.eventRepo).toEqual({ owner: "side-owner", repo: "side-repo" });
  });
});
