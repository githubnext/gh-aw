// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";
import { main } from "./dispatch_workflow.cjs";

// Mock dependencies
global.core = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

global.context = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  ref: "refs/heads/main",
  payload: {
    repository: {
      default_branch: "main",
    },
  },
};

global.github = {
  rest: {
    actions: {
      createWorkflowDispatch: vi.fn().mockResolvedValue({ data: { workflow_run_id: 123456 } }),
    },
    repos: {
      get: vi.fn().mockResolvedValue({
        data: {
          default_branch: "main",
        },
      }),
    },
  },
};

describe("dispatch_workflow handler factory", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    process.env.GITHUB_REF = "refs/heads/main";
    delete process.env.GITHUB_HEAD_REF; // Clean up PR environment variable
    // Reset shared context to a known baseline so tests are order-independent
    global.context.ref = "refs/heads/main";
    global.context.payload = { repository: { default_branch: "main" } };
  });

  it("should create a handler function", async () => {
    const handler = await main({});
    expect(typeof handler).toBe("function");
  });

  it("should dispatch workflows with valid configuration", async () => {
    const config = {
      workflows: ["test-workflow"],
      workflow_files: {
        "test-workflow": ".lock.yml",
      },
      aw_context_workflows: ["test-workflow"],
      max: 5,
    };
    const handler = await main(config);

    const message = {
      type: "dispatch_workflow",
      workflow_name: "test-workflow",
      inputs: {
        param1: "value1",
        param2: 42,
      },
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(result.workflow_name).toBe("test-workflow");
    expect(result.run_id).toBe(123456);
    // Should use the extension from config
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.lock.yml",
      ref: expect.any(String),
      inputs: {
        param1: "value1",
        param2: "42",
        aw_context: expect.any(String),
      },
      return_run_details: true,
    });
  });

  it("should inject aw_context with correct fields", async () => {
    process.env.GITHUB_WORKFLOW_REF = "test-owner/test-repo/.github/workflows/dispatcher.yml@refs/heads/main";
    process.env.GITHUB_RUN_ID = "99999";
    process.env.GITHUB_RUN_ATTEMPT = "2";
    global.context.runId = 99999;
    global.context.actor = "octocat";
    global.context.eventName = "issues";

    const config = {
      workflows: ["test-workflow"],
      workflow_files: { "test-workflow": ".lock.yml" },
      aw_context_workflows: ["test-workflow"],
      max: 5,
    };
    const handler = await main(config);

    const result = await handler({ type: "dispatch_workflow", workflow_name: "test-workflow", inputs: {} }, {});

    expect(result.success).toBe(true);
    const callArgs = github.rest.actions.createWorkflowDispatch.mock.calls[0][0];
    const awContextRaw = callArgs.inputs["aw_context"];
    expect(awContextRaw).toBeDefined();

    const awContext = JSON.parse(awContextRaw);
    expect(awContext).toHaveProperty("repo");
    expect(awContext).toHaveProperty("run_id");
    expect(awContext).toHaveProperty("workflow_id");
    expect(awContext).toHaveProperty("workflow_call_id");
    expect(awContext).toHaveProperty("time");
    expect(awContext).toHaveProperty("actor");
    expect(awContext).toHaveProperty("event_type");
    // Validate time is a valid ISO 8601 timestamp
    expect(() => new Date(awContext.time)).not.toThrow();
    expect(new Date(awContext.time).toISOString()).toBe(awContext.time);
    // repo should match mocked context
    expect(awContext.repo).toBe("test-owner/test-repo");
    // workflow_id uses GITHUB_WORKFLOW_REF (full workflow file path)
    expect(awContext.workflow_id).toBe("test-owner/test-repo/.github/workflows/dispatcher.yml@refs/heads/main");
    // workflow_call_id combines run_id and run_attempt for uniqueness
    expect(awContext.workflow_call_id).toBe("99999-2");
  });

  it("should reject workflows not in allowed list", async () => {
    const config = {
      workflows: ["allowed-workflow"],
      max: 5,
    };
    const handler = await main(config);

    const message = {
      type: "dispatch_workflow",
      workflow_name: "unauthorized-workflow",
      inputs: {},
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not in the allowed workflows list");
    expect(github.rest.actions.createWorkflowDispatch).not.toHaveBeenCalled();
  });

  it("should enforce max count", async () => {
    const config = {
      workflows: ["workflow1", "workflow2"],
      workflow_files: {
        workflow1: ".lock.yml",
        workflow2: ".yml",
      },
      max: 1,
    };
    const handler = await main(config);

    // First message should succeed
    const message1 = {
      type: "dispatch_workflow",
      workflow_name: "workflow1",
      inputs: {},
    };
    const result1 = await handler(message1, {});
    expect(result1.success).toBe(true);

    // Second message should be rejected due to max count
    const message2 = {
      type: "dispatch_workflow",
      workflow_name: "workflow2",
      inputs: {},
    };
    const result2 = await handler(message2, {});
    expect(result2.success).toBe(false);
    expect(result2.error).toContain("Max count");
  });

  it("should handle empty workflow name", async () => {
    const handler = await main({});

    const message = {
      type: "dispatch_workflow",
      workflow_name: "",
      inputs: {},
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("empty");
    expect(github.rest.actions.createWorkflowDispatch).not.toHaveBeenCalled();
  });

  it("should handle dispatch errors", async () => {
    const handler = await main({
      workflows: ["missing-workflow"],
      workflow_files: {}, // No extension for missing-workflow
    });

    const message = {
      type: "dispatch_workflow",
      workflow_name: "missing-workflow",
      inputs: {},
    };

    const result = await handler(message, {});

    expect(result.success).toBe(false);
    expect(result.error).toContain("not found in configuration");
  });

  it("should convert input values to strings", async () => {
    const config = {
      workflows: ["test-workflow"],
      workflow_files: {
        "test-workflow": ".lock.yml",
      },
      aw_context_workflows: ["test-workflow"],
    };
    const handler = await main(config);

    const message = {
      type: "dispatch_workflow",
      workflow_name: "test-workflow",
      inputs: {
        string: "hello",
        number: 42,
        boolean: true,
        object: { key: "value" },
        null: null,
        undefined: undefined,
      },
    };

    await handler(message, {});

    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
      expect.objectContaining({
        inputs: expect.objectContaining({
          string: "hello",
          number: "42",
          boolean: "true",
          object: '{"key":"value"}',
          null: "",
          undefined: "",
          aw_context: expect.any(String),
        }),
      })
    );
  });

  it("should handle workflows with no inputs", async () => {
    const config = {
      workflows: ["no-inputs-workflow"],
      workflow_files: {
        "no-inputs-workflow": ".lock.yml",
      },
      aw_context_workflows: ["no-inputs-workflow"],
    };
    const handler = await main(config);

    // Test with inputs property missing entirely
    const message = {
      type: "dispatch_workflow",
      workflow_name: "no-inputs-workflow",
    };

    const result = await handler(message, {});

    expect(result.success).toBe(true);
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "no-inputs-workflow.lock.yml",
      ref: expect.any(String),
      inputs: expect.objectContaining({
        aw_context: expect.any(String),
      }),
      return_run_details: true,
    });
  });

  it("should delay 5 seconds between dispatches", async () => {
    const config = {
      workflows: ["workflow1", "workflow2"],
      workflow_files: {
        workflow1: ".lock.yml",
        workflow2: ".yml",
      },
      max: 5,
    };
    const handler = await main(config);

    const message1 = {
      type: "dispatch_workflow",
      workflow_name: "workflow1",
      inputs: {},
    };

    const message2 = {
      type: "dispatch_workflow",
      workflow_name: "workflow2",
      inputs: {},
    };

    // Dispatch first workflow
    const startTime = Date.now();
    await handler(message1, {});
    const firstDispatchTime = Date.now();

    // Dispatch second workflow (should be delayed)
    await handler(message2, {});
    const secondDispatchTime = Date.now();

    // Verify first dispatch had no delay
    expect(firstDispatchTime - startTime).toBeLessThan(1000);

    // Verify second dispatch was delayed by approximately 5 seconds
    // Use a slightly lower threshold (4995ms) to account for timing jitter
    expect(secondDispatchTime - firstDispatchTime).toBeGreaterThanOrEqual(4995);
    expect(secondDispatchTime - firstDispatchTime).toBeLessThan(6000);
  });

  it("should use PR branch ref when GITHUB_HEAD_REF is set", async () => {
    // Simulate PR context where GITHUB_REF is the merge ref
    process.env.GITHUB_REF = "refs/pull/123/merge";
    process.env.GITHUB_HEAD_REF = "feature-branch";

    const config = {
      workflows: ["test-workflow"],
      workflow_files: {
        "test-workflow": ".lock.yml",
      },
      aw_context_workflows: ["test-workflow"],
    };
    const handler = await main(config);

    const message = {
      type: "dispatch_workflow",
      workflow_name: "test-workflow",
      inputs: {},
    };

    await handler(message, {});

    // Should use the PR branch ref, not the merge ref
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.lock.yml",
      ref: "refs/heads/feature-branch",
      inputs: expect.objectContaining({ aw_context: expect.any(String) }),
      return_run_details: true,
    });
  });

  it("should use GITHUB_REF when not in PR context", async () => {
    process.env.GITHUB_REF = "refs/heads/main";
    delete process.env.GITHUB_HEAD_REF;

    const config = {
      workflows: ["test-workflow"],
      workflow_files: {
        "test-workflow": ".lock.yml",
      },
      aw_context_workflows: ["test-workflow"],
    };
    const handler = await main(config);

    const message = {
      type: "dispatch_workflow",
      workflow_name: "test-workflow",
      inputs: {},
    };

    await handler(message, {});

    // Should use GITHUB_REF directly
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.lock.yml",
      ref: "refs/heads/main",
      inputs: expect.objectContaining({ aw_context: expect.any(String) }),
      return_run_details: true,
    });
  });

  it("should handle PR context with slashes in branch names", async () => {
    process.env.GITHUB_REF = "refs/pull/456/merge";
    process.env.GITHUB_HEAD_REF = "feature/add-new-feature";

    const config = {
      workflows: ["test-workflow"],
      workflow_files: {
        "test-workflow": ".lock.yml",
      },
      aw_context_workflows: ["test-workflow"],
    };
    const handler = await main(config);

    const message = {
      type: "dispatch_workflow",
      workflow_name: "test-workflow",
      inputs: {},
    };

    await handler(message, {});

    // Should correctly handle branch names with slashes
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.lock.yml",
      ref: "refs/heads/feature/add-new-feature",
      inputs: expect.objectContaining({ aw_context: expect.any(String) }),
      return_run_details: true,
    });
  });

  it("should use repository default branch when no GITHUB_REF is set", async () => {
    delete process.env.GITHUB_REF;
    delete process.env.GITHUB_HEAD_REF;
    global.context.ref = undefined;
    global.context.payload.repository.default_branch = "develop";

    const config = {
      workflows: ["test-workflow"],
      workflow_files: {
        "test-workflow": ".lock.yml",
      },
      aw_context_workflows: ["test-workflow"],
    };
    const handler = await main(config);

    const message = {
      type: "dispatch_workflow",
      workflow_name: "test-workflow",
      inputs: {},
    };

    await handler(message, {});

    // Should use the repository's default branch from context
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.lock.yml",
      ref: "refs/heads/develop",
      inputs: expect.objectContaining({ aw_context: expect.any(String) }),
      return_run_details: true,
    });
  });

  it("should fall back to API when context payload is missing", async () => {
    delete process.env.GITHUB_REF;
    delete process.env.GITHUB_HEAD_REF;
    global.context.ref = undefined;
    global.context.payload = {};

    github.rest.repos.get.mockResolvedValueOnce({
      data: {
        default_branch: "staging",
      },
    });

    const config = {
      workflows: ["test-workflow"],
      workflow_files: {
        "test-workflow": ".lock.yml",
      },
      aw_context_workflows: ["test-workflow"],
    };
    const handler = await main(config);

    const message = {
      type: "dispatch_workflow",
      workflow_name: "test-workflow",
      inputs: {},
    };

    await handler(message, {});

    // Should fetch default branch from API
    expect(github.rest.repos.get).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
    });

    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith({
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.lock.yml",
      ref: "refs/heads/staging",
      inputs: expect.objectContaining({ aw_context: expect.any(String) }),
      return_run_details: true,
    });
  });

  it("should return run_id when API returns workflow_run_id", async () => {
    github.rest.actions.createWorkflowDispatch.mockResolvedValueOnce({
      data: { workflow_run_id: 987654 },
    });

    const config = {
      workflows: ["test-workflow"],
      workflow_files: { "test-workflow": ".lock.yml" },
    };
    const handler = await main(config);

    const result = await handler({ type: "dispatch_workflow", workflow_name: "test-workflow", inputs: {} }, {});

    expect(result.success).toBe(true);
    expect(result.run_id).toBe(987654);
    expect(core.info).toHaveBeenCalledWith(expect.stringContaining("run ID: 987654"));
  });

  it("should succeed without run_id when API returns no workflow_run_id", async () => {
    github.rest.actions.createWorkflowDispatch.mockResolvedValueOnce({ data: {} });

    const config = {
      workflows: ["test-workflow"],
      workflow_files: { "test-workflow": ".lock.yml" },
    };
    const handler = await main(config);

    const result = await handler({ type: "dispatch_workflow", workflow_name: "test-workflow", inputs: {} }, {});

    expect(result.success).toBe(true);
    expect(result.run_id).toBeUndefined();
  });

  it("should retry without return_run_details when API rejects with 422 mentioning it, and still succeed", async () => {
    const error = new Error("Unprocessable Entity");
    // @ts-ignore
    error.status = 422;
    // @ts-ignore
    error.response = { data: { message: "Unknown field 'return_run_details'" } };

    github.rest.actions.createWorkflowDispatch.mockRejectedValueOnce(error).mockResolvedValueOnce({ data: {} });

    const config = {
      workflows: ["test-workflow"],
      workflow_files: { "test-workflow": ".lock.yml" },
      aw_context_workflows: ["test-workflow"],
    };
    const handler = await main(config);

    const result = await handler({ type: "dispatch_workflow", workflow_name: "test-workflow", inputs: {} }, {});

    expect(result.success).toBe(true);
    expect(result.run_id).toBeUndefined();

    // First call should include return_run_details: true
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenNthCalledWith(1, {
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.lock.yml",
      ref: "refs/heads/main",
      inputs: expect.objectContaining({ aw_context: expect.any(String) }),
      return_run_details: true,
    });

    // Second call should retry without return_run_details
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenNthCalledWith(2, {
      owner: "test-owner",
      repo: "test-repo",
      workflow_id: "test-workflow.lock.yml",
      ref: "refs/heads/main",
      inputs: expect.objectContaining({ aw_context: expect.any(String) }),
    });

    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledTimes(2);
  });

  it("should not retry when API rejects with 422 for an unrelated reason", async () => {
    const error = new Error("Unprocessable Entity");
    // @ts-ignore
    error.status = 422;
    // @ts-ignore
    error.response = { data: { message: "Workflow does not exist" } };

    github.rest.actions.createWorkflowDispatch.mockRejectedValueOnce(error);

    const config = {
      workflows: ["test-workflow"],
      workflow_files: { "test-workflow": ".lock.yml" },
    };
    const handler = await main(config);

    const result = await handler({ type: "dispatch_workflow", workflow_name: "test-workflow", inputs: {} }, {});

    expect(result.success).toBe(false);
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledTimes(1);
  });

  it("dispatches to target-repo when configured", async () => {
    process.env.GITHUB_REF = "refs/heads/main";

    const config = {
      "target-repo": "platform-org/platform-repo",
      allowed_repos: ["platform-org/platform-repo"],
      workflows: ["platform-worker"],
      workflow_files: { "platform-worker": ".lock.yml" },
    };
    const handler = await main(config);

    const result = await handler({ type: "dispatch_workflow", workflow_name: "platform-worker", inputs: {} }, {});

    expect(result.success).toBe(true);
    // Must dispatch to the configured target-repo, NOT context.repo
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "platform-org",
        repo: "platform-repo",
        workflow_id: "platform-worker.lock.yml",
      })
    );
  });

  it("default-branch lookup uses target-repo when configured", async () => {
    const originalRef = global.context.ref;
    const originalPayload = global.context.payload;

    try {
      delete process.env.GITHUB_REF;
      delete process.env.GITHUB_HEAD_REF;
      global.context.ref = undefined;
      // context.payload has a default_branch for the caller repo – must be ignored for cross-repo dispatch
      global.context.payload = { repository: { default_branch: "caller-main" } };

      github.rest.repos.get.mockResolvedValueOnce({
        data: { default_branch: "platform-main" },
      });

      const config = {
        "target-repo": "platform-org/platform-repo",
        allowed_repos: ["platform-org/platform-repo"],
        workflows: ["platform-worker"],
        workflow_files: { "platform-worker": ".lock.yml" },
      };
      const handler = await main(config);

      const result = await handler({ type: "dispatch_workflow", workflow_name: "platform-worker", inputs: {} }, {});

      expect(result.success).toBe(true);
      // Default-branch API lookup must target the configured target-repo
      expect(github.rest.repos.get).toHaveBeenCalledWith({
        owner: "platform-org",
        repo: "platform-repo",
      });
      // Dispatch must use the target repo's default branch
      expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
        expect.objectContaining({
          owner: "platform-org",
          repo: "platform-repo",
          ref: "refs/heads/platform-main",
        })
      );
    } finally {
      global.context.ref = originalRef;
      global.context.payload = originalPayload;
    }
  });

  it("falls back to context.repo when no target-repo is configured", async () => {
    process.env.GITHUB_REF = "refs/heads/main";

    const config = {
      workflows: ["test-workflow"],
      workflow_files: { "test-workflow": ".lock.yml" },
    };
    const handler = await main(config);

    const result = await handler({ type: "dispatch_workflow", workflow_name: "test-workflow", inputs: {} }, {});

    expect(result.success).toBe(true);
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "test-owner",
        repo: "test-repo",
      })
    );
  });

  it("falls back to context.repo and warns when target-repo is an invalid slug", async () => {
    process.env.GITHUB_REF = "refs/heads/main";

    const config = {
      "target-repo": "not-a-valid-slug",
      workflows: ["test-workflow"],
      workflow_files: { "test-workflow": ".lock.yml" },
    };
    const handler = await main(config);

    const result = await handler({ type: "dispatch_workflow", workflow_name: "test-workflow", inputs: {} }, {});

    expect(result.success).toBe(true);
    // Must emit a warning about the invalid slug including the bad value and the fallback
    expect(core.warning).toHaveBeenCalledWith(expect.stringMatching(/Invalid 'target-repo' configuration value 'not-a-valid-slug'.*falling back.*test-owner\/test-repo/));
    // Must fall back to context.repo
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "test-owner",
        repo: "test-repo",
      })
    );
  });

  it("should use configured target-ref when dispatching cross-repo", async () => {
    // Caller is on refs/heads/main, target workflow should run on feature-branch
    process.env.GITHUB_REF = "refs/heads/main";
    delete process.env.GITHUB_HEAD_REF;

    const config = {
      "target-repo": "other-org/other-repo",
      allowed_repos: ["other-org/other-repo"],
      "target-ref": "refs/heads/feature-branch",
      workflows: ["target-workflow"],
      workflow_files: { "target-workflow": ".lock.yml" },
    };
    const handler = await main(config);

    await handler({ type: "dispatch_workflow", workflow_name: "target-workflow", inputs: {} }, {});

    // Should dispatch to the configured target ref, NOT the caller's main
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "other-org",
        repo: "other-repo",
        ref: "refs/heads/feature-branch",
      })
    );
  });

  it("should use caller GITHUB_REF when dispatching to same repo without target-ref", async () => {
    process.env.GITHUB_REF = "refs/heads/feature-branch";
    delete process.env.GITHUB_HEAD_REF;

    const config = {
      workflows: ["local-workflow"],
      workflow_files: { "local-workflow": ".lock.yml" },
    };
    const handler = await main(config);

    await handler({ type: "dispatch_workflow", workflow_name: "local-workflow", inputs: {} }, {});

    // Same-repo dispatch should still use the caller's GITHUB_REF
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "test-owner",
        repo: "test-repo",
        ref: "refs/heads/feature-branch",
      })
    );
  });

  it("should prefer configured target-ref over GITHUB_HEAD_REF for cross-repo dispatch", async () => {
    process.env.GITHUB_REF = "refs/pull/42/merge";
    process.env.GITHUB_HEAD_REF = "pr-branch";

    const config = {
      "target-repo": "other-org/other-repo",
      allowed_repos: ["other-org/other-repo"],
      "target-ref": "refs/heads/feature-branch",
      workflows: ["target-workflow"],
      workflow_files: { "target-workflow": ".lock.yml" },
    };
    const handler = await main(config);

    await handler({ type: "dispatch_workflow", workflow_name: "target-workflow", inputs: {} }, {});

    // Cross-repo should use configured target-ref, not the PR branch
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "other-org",
        repo: "other-repo",
        ref: "refs/heads/feature-branch",
      })
    );
  });

  it("throws E004 when cross-repo dispatch is attempted without any allowlist configured", async () => {
    process.env.GITHUB_REF = "refs/heads/main";

    const config = {
      "target-repo": "other-org/other-repo",
      workflows: ["target-workflow"],
      workflow_files: { "target-workflow": ".lock.yml" },
    };

    await expect(main(config)).rejects.toThrow(/E004.*No allowlist is configured/);
    expect(github.rest.actions.createWorkflowDispatch).not.toHaveBeenCalled();
  });

  it("throws E004 when target repo is not present in the allowlist", async () => {
    process.env.GITHUB_REF = "refs/heads/main";

    const config = {
      "target-repo": "other-org/other-repo",
      allowed_repos: ["other-org/different-repo"],
      workflows: ["target-workflow"],
      workflow_files: { "target-workflow": ".lock.yml" },
    };

    await expect(main(config)).rejects.toThrow(/E004.*not in the allowed-repos list/);
    expect(github.rest.actions.createWorkflowDispatch).not.toHaveBeenCalled();
  });

  it("allows cross-repo dispatch when target repo is present in the allowlist", async () => {
    process.env.GITHUB_REF = "refs/heads/main";

    const config = {
      "target-repo": "allowed-org/allowed-repo",
      allowed_repos: ["allowed-org/allowed-repo", "other-org/other-repo"],
      workflows: ["target-workflow"],
      workflow_files: { "target-workflow": ".lock.yml" },
    };

    const handler = await main(config);
    const result = await handler({ type: "dispatch_workflow", workflow_name: "target-workflow", inputs: {} }, {});

    expect(result.success).toBe(true);
    expect(core.info).toHaveBeenCalledWith("Cross-repo allowlist check passed for allowed-org/allowed-repo");
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "allowed-org",
        repo: "allowed-repo",
      })
    );
  });

  it("does not apply allowlist check for same-repo dispatch", async () => {
    process.env.GITHUB_REF = "refs/heads/main";

    // No allowed-repos configured, but dispatching to the same repo (context.repo)
    const config = {
      workflows: ["local-workflow"],
      workflow_files: { "local-workflow": ".lock.yml" },
    };

    const handler = await main(config);
    const result = await handler({ type: "dispatch_workflow", workflow_name: "local-workflow", inputs: {} }, {});

    expect(result.success).toBe(true);
    expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "test-owner",
        repo: "test-repo",
      })
    );
  });

  describe("temporary ID resolution in inputs", () => {
    const baseConfig = {
      workflows: ["create-pr"],
      workflow_files: { "create-pr": ".lock.yml" },
    };

    it("should resolve a pure temporary ID input value to the issue number", async () => {
      const handler = await main(baseConfig);

      const resolvedTemporaryIds = {
        aw_slosync: { repo: "test-owner/test-repo", number: 6499 },
      };

      const message = {
        type: "dispatch_workflow",
        workflow_name: "create-pr",
        inputs: {
          issue_number: "#aw_slosync",
        },
      };

      const result = await handler(message, resolvedTemporaryIds);

      expect(result.success).toBe(true);
      expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
        expect.objectContaining({
          inputs: expect.objectContaining({
            issue_number: "6499",
          }),
        })
      );
    });

    it("should resolve temporary ID without hash prefix", async () => {
      const handler = await main(baseConfig);

      const resolvedTemporaryIds = {
        aw_myissue: { repo: "test-owner/test-repo", number: 42 },
      };

      const message = {
        type: "dispatch_workflow",
        workflow_name: "create-pr",
        inputs: {
          item_number: "aw_myissue",
        },
      };

      const result = await handler(message, resolvedTemporaryIds);

      expect(result.success).toBe(true);
      expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
        expect.objectContaining({
          inputs: expect.objectContaining({
            item_number: "42",
          }),
        })
      );
    });

    it("should replace embedded temporary ID references in text input values", async () => {
      const handler = await main(baseConfig);

      const resolvedTemporaryIds = {
        aw_track: { repo: "test-owner/test-repo", number: 100 },
      };

      const message = {
        type: "dispatch_workflow",
        workflow_name: "create-pr",
        inputs: {
          description: "Tracking issue is #aw_track",
        },
      };

      const result = await handler(message, resolvedTemporaryIds);

      expect(result.success).toBe(true);
      const callArgs = github.rest.actions.createWorkflowDispatch.mock.calls[0][0];
      // Embedded references are replaced with cross-reference format (e.g. #100 or owner/repo#100)
      expect(callArgs.inputs.description).toContain("100");
      expect(callArgs.inputs.description).not.toContain("#aw_track");
    });

    it("should warn and keep original value when temporary ID is unresolved", async () => {
      const handler = await main(baseConfig);

      const message = {
        type: "dispatch_workflow",
        workflow_name: "create-pr",
        inputs: {
          issue_number: "#aw_missing",
        },
      };

      const result = await handler(message, {});

      expect(result.success).toBe(true);
      expect(core.warning).toHaveBeenCalledWith(expect.stringContaining("aw_missing"));
      // Original value kept when unresolved
      expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
        expect.objectContaining({
          inputs: expect.objectContaining({
            issue_number: "#aw_missing",
          }),
        })
      );
    });

    it("should leave non-temporary-ID values unchanged", async () => {
      const handler = await main(baseConfig);

      const resolvedTemporaryIds = {
        aw_slosync: { repo: "test-owner/test-repo", number: 6499 },
      };

      const message = {
        type: "dispatch_workflow",
        workflow_name: "create-pr",
        inputs: {
          issue_number: "123",
          label: "bug",
          count: 5,
        },
      };

      const result = await handler(message, resolvedTemporaryIds);

      expect(result.success).toBe(true);
      expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
        expect.objectContaining({
          inputs: expect.objectContaining({
            issue_number: "123",
            label: "bug",
            count: "5",
          }),
        })
      );
    });

    it("should handle multiple temporary ID inputs in a single dispatch", async () => {
      const handler = await main(baseConfig);

      const resolvedTemporaryIds = {
        aw_issue1: { repo: "test-owner/test-repo", number: 10 },
        aw_issue2: { repo: "test-owner/test-repo", number: 20 },
      };

      const message = {
        type: "dispatch_workflow",
        workflow_name: "create-pr",
        inputs: {
          parent_issue: "#aw_issue1",
          child_issue: "#aw_issue2",
        },
      };

      const result = await handler(message, resolvedTemporaryIds);

      expect(result.success).toBe(true);
      expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
        expect.objectContaining({
          inputs: expect.objectContaining({
            parent_issue: "10",
            child_issue: "20",
          }),
        })
      );
    });

    it("should use owner/repo#number format when pure temporary ID resolves to a different repo", async () => {
      // The dispatch target is the context repo (test-owner/test-repo).
      // The temp ID was created in a different repo. The resolved value should carry
      // full cross-repo reference so the dispatched workflow knows which repo the
      // issue belongs to, rather than silently passing a bare number that would be
      // misinterpreted as an issue in the dispatch-target repo.
      const handler = await main(baseConfig);

      const resolvedTemporaryIds = {
        aw_crossrepo: { repo: "other-org/other-repo", number: 999 },
      };

      const message = {
        type: "dispatch_workflow",
        workflow_name: "create-pr",
        inputs: {
          tracking_issue: "#aw_crossrepo",
        },
      };

      const result = await handler(message, resolvedTemporaryIds);

      expect(result.success).toBe(true);
      expect(github.rest.actions.createWorkflowDispatch).toHaveBeenCalledWith(
        expect.objectContaining({
          inputs: expect.objectContaining({
            tracking_issue: "other-org/other-repo#999",
          }),
        })
      );
    });
  });
});
