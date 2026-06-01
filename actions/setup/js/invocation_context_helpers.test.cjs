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

  it("allows workflow_dispatch target_repo when handler allowlist includes it", () => {
    const originalAllowedRepos = process.env.GH_AW_ALLOWED_REPOS;
    const originalHandlerConfig = process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG;
    try {
      delete process.env.GH_AW_ALLOWED_REPOS;
      process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = JSON.stringify({
        create_pull_request: {
          allowed_repos: ["target-owner/target-repo"],
        },
      });

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
      if (originalHandlerConfig === undefined) {
        delete process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG;
      } else {
        process.env.GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG = originalHandlerConfig;
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

  it("derives workflow_dispatch event context from aw_context", () => {
    const awContext = {
      event_type: "issue_comment",
      item_type: "pull_request",
      item_number: "42",
      comment_id: "99",
      repo: "target-owner/target-repo",
    };
    const resolved = resolveInvocationContext({
      eventName: "workflow_dispatch",
      repo: { owner: "side-owner", repo: "side-repo" },
      payload: {
        inputs: {
          aw_context: JSON.stringify(awContext),
        },
      },
    });

    expect(resolved.source).toBe("workflow_dispatch");
    expect(resolved.eventName).toBe("issue_comment");
    expect(resolved.eventRepo).toEqual({ owner: "target-owner", repo: "target-repo" });
    expect(resolved.eventPayload.issue.number).toBe(42);
    expect(resolved.eventPayload.issue.pull_request).toEqual({});
    expect(resolved.eventPayload.comment.id).toBe(99);
  });

  it("derives discussion_comment node_id payload from aw_context", () => {
    const awContext = {
      event_type: "discussion_comment",
      item_type: "discussion",
      item_number: "8",
      comment_id: "15",
      comment_node_id: "DC_kwDOexample",
    };
    const resolved = resolveInvocationContext({
      eventName: "workflow_dispatch",
      repo: { owner: "side-owner", repo: "side-repo" },
      payload: {
        inputs: {
          aw_context: JSON.stringify(awContext),
        },
      },
    });

    expect(resolved.eventName).toBe("discussion_comment");
    expect(resolved.eventPayload.discussion.number).toBe(8);
    expect(resolved.eventPayload.comment.id).toBe(15);
    expect(resolved.eventPayload.comment.node_id).toBe("DC_kwDOexample");
  });
});
