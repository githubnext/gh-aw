import { afterEach, describe, it, expect, vi } from "vitest";

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

  describe("handler target-repo fallback", () => {
    afterEach(() => {
      vi.unstubAllEnvs();
    });

    function resolveTargetRepo(targetRepo = "target-owner/target-repo") {
      return resolveInvocationContext({
        eventName: "workflow_dispatch",
        repo: { owner: "side-owner", repo: "side-repo" },
        payload: { inputs: { target_repo: targetRepo } },
      });
    }

    it.each(["target-owner/target-repo", "  target-owner/target-repo  "])("allows a handler target-repo of %j without allowed_repos", targetRepo => {
      vi.stubEnv("GH_AW_ALLOWED_REPOS", undefined);
      vi.stubEnv("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", JSON.stringify({ create_pull_request: { "target-repo": targetRepo } }));

      expect(resolveTargetRepo().eventRepo).toEqual({ owner: "target-owner", repo: "target-repo" });
    });

    it("combines handler targets and allowed_repos across handlers", () => {
      vi.stubEnv("GH_AW_ALLOWED_REPOS", "");
      vi.stubEnv(
        "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
        JSON.stringify({
          create_issue: { "target-repo": "issue-owner/issue-repo" },
          create_pull_request: { "target-repo": "target-owner/target-repo", allowed_repos: ["allowed-owner/allowed-repo"] },
        })
      );

      expect(resolveTargetRepo().eventRepo).toEqual({ owner: "target-owner", repo: "target-repo" });
      expect(resolveTargetRepo("issue-owner/issue-repo").eventRepo).toEqual({ owner: "issue-owner", repo: "issue-repo" });
      expect(resolveTargetRepo("allowed-owner/allowed-repo").eventRepo).toEqual({ owner: "allowed-owner", repo: "allowed-repo" });
      expect(resolveTargetRepo("side-owner/side-repo").eventRepo).toEqual({ owner: "side-owner", repo: "side-repo" });
      expect(() => resolveTargetRepo("other-owner/other-repo")).toThrow(/ERR_VALIDATION: Repository 'other-owner\/other-repo' is not in the allowed-repos list/);
    });

    it("does not extend an explicit global allowlist with handler targets", () => {
      vi.stubEnv("GH_AW_ALLOWED_REPOS", "allowed-owner/allowed-repo");
      vi.stubEnv("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", JSON.stringify({ create_pull_request: { "target-repo": "target-owner/target-repo" } }));

      expect(resolveTargetRepo("allowed-owner/allowed-repo").eventRepo).toEqual({ owner: "allowed-owner", repo: "allowed-repo" });
      expect(() => resolveTargetRepo()).toThrow(/ERR_VALIDATION: Repository 'target-owner\/target-repo' is not in the allowed-repos list/);
    });

    it.each(["*", "  *  ", "target-owner/*", "*/target-repo", "target-owner/target-*"])("refuses the entire request when handler target-repo %j contains a wildcard", targetRepo => {
      vi.stubEnv("GH_AW_ALLOWED_REPOS", undefined);
      vi.stubEnv(
        "GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG",
        JSON.stringify({
          create_issue: { "target-repo": "issue-owner/issue-repo", allowed_repos: ["allowed-owner/allowed-repo"] },
          add_comment: { "target-repo": targetRepo, allowed_repos: ["target-owner/*"] },
        })
      );

      for (const requestedRepo of ["target-owner/target-repo", "issue-owner/issue-repo", "allowed-owner/allowed-repo", "side-owner/side-repo"]) {
        expect(() => resolveTargetRepo(requestedRepo)).toThrow(/ERR_VALIDATION: Wildcard handler target-repo values are not permitted in workflow_dispatch requests/);
      }
    });

    it("refuses wildcard handler targets even with a global allowlist", () => {
      vi.stubEnv("GH_AW_ALLOWED_REPOS", "*");
      vi.stubEnv("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", JSON.stringify({ add_comment: { "target-repo": "*" } }));

      expect(() => resolveTargetRepo()).toThrow(/ERR_VALIDATION: Wildcard handler target-repo values are not permitted in workflow_dispatch requests/);
    });

    it.each(["global", "handler"])("preserves an explicit %s wildcard allowlist", allowlistSource => {
      vi.stubEnv("GH_AW_ALLOWED_REPOS", allowlistSource === "global" ? "target-owner/*" : undefined);
      vi.stubEnv("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", JSON.stringify({ add_comment: { ...(allowlistSource === "handler" ? { allowed_repos: ["target-owner/*"] } : {}) } }));

      expect(resolveTargetRepo().eventRepo).toEqual({ owner: "target-owner", repo: "target-repo" });
      expect(() => resolveTargetRepo("other-owner/other-repo")).toThrow(/ERR_VALIDATION: Repository 'other-owner\/other-repo' is not in the allowed-repos list/);
    });

    it.each([undefined, null, "", "   ", "bare-repo", "/repo", "owner/", "owner/repo/extra", 42, ["target-owner/target-repo"], { repo: "target-owner/target-repo" }])("ignores an invalid handler target-repo of %j", targetRepo => {
      vi.stubEnv("GH_AW_ALLOWED_REPOS", undefined);
      vi.stubEnv("GH_AW_SAFE_OUTPUTS_HANDLER_CONFIG", JSON.stringify({ create_pull_request: { "target-repo": targetRepo } }));

      expect(() => resolveTargetRepo()).toThrow(/ERR_VALIDATION: Repository 'target-owner\/target-repo' is not in the allowed-repos list/);
    });
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
