// @ts-check
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(undefined),
  },
};

const mockGetWorkflowRun = vi.fn();
const mockGithub = {
  rest: {
    actions: {
      getWorkflowRun: mockGetWorkflowRun,
    },
  },
};

const mockContext = {
  runId: 99999,
};

// Set up global mocks before importing the module
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

/**
 * Sets up a one-time mock response for getWorkflowRun with no referenced workflows.
 * Used for same-repo and same-org cross-repo tests where the API should not change the result.
 */
function mockNoReferencedWorkflowsOnce() {
  mockGetWorkflowRun.mockResolvedValueOnce({ data: { referenced_workflows: [] } });
}

describe("resolve_host_repo.cjs", () => {
  let main;

  beforeEach(async () => {
    vi.clearAllMocks();
    // Defensive reset of mock implementation as a safety measure.
    // All tests use *Once variants, but mockReset() ensures no state leaks
    // if a test adds a persistent mock or if a future test omits the Once variant.
    mockGetWorkflowRun.mockReset();
    mockCore.summary.addRaw.mockReturnThis();
    mockCore.summary.write.mockResolvedValue(undefined);

    const module = await import("./resolve_host_repo.cjs");
    main = module.main;
  });

  afterEach(() => {
    delete process.env.GITHUB_WORKFLOW_REF;
    delete process.env.GITHUB_REPOSITORY;
    delete process.env.GITHUB_REF;
    delete process.env.GITHUB_RUN_ID;
    // Reset context.runId to the default value to prevent test state leakage
    mockContext.runId = 99999;
  });

  it("should output the platform repo when invoked cross-repo", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/platform-repo");
  });

  it("should log a cross-repo detection message and write step summary", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Cross-repo invocation detected"));
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should output the current repo when same-repo invocation", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/platform-repo";
    mockNoReferencedWorkflowsOnce();

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/platform-repo");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Same-repo invocation"));
  });

  it("should not write step summary for same-repo invocations", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/platform-repo";
    mockNoReferencedWorkflowsOnce();

    await main();

    expect(mockCore.summary.write).not.toHaveBeenCalled();
  });

  it("should fall back to GITHUB_REPOSITORY when GITHUB_WORKFLOW_REF is empty", async () => {
    process.env.GITHUB_WORKFLOW_REF = "";
    process.env.GITHUB_REPOSITORY = "my-org/fallback-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/fallback-repo");
  });

  it("should fall back to GITHUB_REPOSITORY when GITHUB_WORKFLOW_REF has unexpected format", async () => {
    process.env.GITHUB_WORKFLOW_REF = "not-a-valid-ref";
    process.env.GITHUB_REPOSITORY = "my-org/fallback-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/fallback-repo");
  });

  it("should handle event-driven relay (issue_comment) that calls a cross-repo workflow", async () => {
    // This is the exact scenario from the bug report:
    // An issue_comment event in app-repo triggers a relay that calls the platform workflow.
    // GITHUB_WORKFLOW_REF reflects the platform workflow, GITHUB_REPOSITORY is the caller.
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/my-workflow.lock.yml@main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/platform-repo");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Cross-repo invocation detected"));
  });

  it("should fall back to empty string when GITHUB_REPOSITORY is also undefined", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    delete process.env.GITHUB_REPOSITORY;

    await main();

    // workflowRepo parsed from GITHUB_WORKFLOW_REF is "my-org/platform-repo"
    // currentRepo is "" since env var is deleted
    // targetRepo = workflowRepo || currentRepo = "my-org/platform-repo"
    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/platform-repo");
  });

  it("should log GITHUB_WORKFLOW_REF and GITHUB_REPOSITORY", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("GITHUB_WORKFLOW_REF:"));
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("GITHUB_REPOSITORY:"));
  });

  it("should output target_ref extracted from GITHUB_WORKFLOW_REF", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/feature-branch";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "refs/heads/feature-branch");
  });

  it("should output target_ref for a short branch ref (not refs/heads/...)", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "main");
  });

  it("should output target_ref for a feature branch in a caller-hosted relay", async () => {
    // This is the exact scenario from the bug report:
    // relay is pinned to @feature-branch, activation should check out feature-branch
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/platform-gateway.lock.yml@refs/heads/my-feature";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/platform-repo");
    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "refs/heads/my-feature");
  });

  it("should output empty target_ref when GITHUB_WORKFLOW_REF has no @ segment (no GITHUB_REF fallback)", async () => {
    // When GITHUB_WORKFLOW_REF has no '@', we cannot determine the callee ref.
    // We intentionally do NOT fall back to GITHUB_REF because in cross-repo scenarios
    // GITHUB_REF is the caller's ref, not the callee's. Empty string tells actions/checkout
    // to use the repository's default branch.
    process.env.GITHUB_WORKFLOW_REF = "not-a-valid-ref";
    process.env.GITHUB_REF = "refs/heads/fallback-branch";
    process.env.GITHUB_REPOSITORY = "my-org/fallback-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "");
  });

  it("should output empty target_ref when GITHUB_WORKFLOW_REF is empty", async () => {
    process.env.GITHUB_WORKFLOW_REF = "";
    delete process.env.GITHUB_REF;
    process.env.GITHUB_REPOSITORY = "my-org/fallback-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "");
  });

  it("should output target_ref for a tag ref", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/tags/v1.0.0";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "refs/tags/v1.0.0");
  });

  it("should output target_ref for a commit SHA", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@abc123def456";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "abc123def456");
  });

  it("should output target_repo_name when invoked cross-repo", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo_name", "platform-repo");
  });

  it("should output target_repo_name when same-repo invocation", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
    process.env.GITHUB_REPOSITORY = "my-org/platform-repo";
    mockNoReferencedWorkflowsOnce();

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo_name", "platform-repo");
  });

  it("should output target_repo_name without owner prefix when falling back to GITHUB_REPOSITORY", async () => {
    process.env.GITHUB_WORKFLOW_REF = "";
    process.env.GITHUB_REPOSITORY = "my-org/fallback-repo";

    await main();

    expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo_name", "fallback-repo");
  });

  it("should include target_ref in step summary for cross-repo invocations", async () => {
    process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/feature-branch";
    process.env.GITHUB_REPOSITORY = "my-org/app-repo";

    await main();

    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("refs/heads/feature-branch"));
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  describe("cross-org workflow_call scenarios", () => {
    it("should resolve callee repo via referenced_workflows API when GITHUB_WORKFLOW_REF matches GITHUB_REPOSITORY", async () => {
      // Cross-org workflow_call: GITHUB_WORKFLOW_REF points to the caller's repo (not the callee),
      // so workflowRepo === currentRepo. The referenced_workflows API returns the actual callee.
      process.env.GITHUB_WORKFLOW_REF = "caller-org/caller-repo/.github/workflows/relay.yml@refs/heads/main";
      process.env.GITHUB_REPOSITORY = "caller-org/caller-repo";
      process.env.GITHUB_RUN_ID = "12345";

      mockGetWorkflowRun.mockResolvedValueOnce({
        data: {
          referenced_workflows: [
            {
              path: "platform-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main",
              sha: "abc123def456",
              ref: "refs/heads/main",
            },
          ],
        },
      });

      await main();

      expect(mockGetWorkflowRun).toHaveBeenCalledWith({
        owner: "caller-org",
        repo: "caller-repo",
        run_id: 12345,
      });
      expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "platform-org/platform-repo");
      expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo_name", "platform-repo");
      // sha is preferred over ref
      expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "abc123def456");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Resolved callee repo from referenced_workflows"));
    });

    it("should use ref from referenced_workflows entry when sha is absent", async () => {
      process.env.GITHUB_WORKFLOW_REF = "caller-org/caller-repo/.github/workflows/relay.yml@refs/heads/main";
      process.env.GITHUB_REPOSITORY = "caller-org/caller-repo";
      process.env.GITHUB_RUN_ID = "12345";

      mockGetWorkflowRun.mockResolvedValueOnce({
        data: {
          referenced_workflows: [
            {
              path: "platform-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/feature",
              sha: undefined,
              ref: "refs/heads/feature",
            },
          ],
        },
      });

      await main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "platform-org/platform-repo");
      expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "refs/heads/feature");
    });

    it("should fall back to path-parsed ref when sha and ref are absent in referenced_workflows", async () => {
      process.env.GITHUB_WORKFLOW_REF = "caller-org/caller-repo/.github/workflows/relay.yml@refs/heads/main";
      process.env.GITHUB_REPOSITORY = "caller-org/caller-repo";
      process.env.GITHUB_RUN_ID = "12345";

      mockGetWorkflowRun.mockResolvedValueOnce({
        data: {
          referenced_workflows: [
            {
              path: "platform-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/stable",
              sha: undefined,
              ref: undefined,
            },
          ],
        },
      });

      await main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "platform-org/platform-repo");
      expect(mockCore.setOutput).toHaveBeenCalledWith("target_ref", "refs/heads/stable");
    });

    it("should log cross-repo detection and write step summary for cross-org callee", async () => {
      process.env.GITHUB_WORKFLOW_REF = "caller-org/caller-repo/.github/workflows/relay.yml@refs/heads/main";
      process.env.GITHUB_REPOSITORY = "caller-org/caller-repo";
      process.env.GITHUB_RUN_ID = "12345";

      mockGetWorkflowRun.mockResolvedValueOnce({
        data: {
          referenced_workflows: [
            {
              path: "platform-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main",
              sha: "abc123",
              ref: "refs/heads/main",
            },
          ],
        },
      });

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Cross-repo invocation detected"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("platform-org/platform-repo"));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should fall back to GITHUB_REPOSITORY when referenced_workflows has no cross-org entry", async () => {
      // workflowRepo === currentRepo but no cross-org entry (same-org same-repo, no callee)
      process.env.GITHUB_WORKFLOW_REF = "my-org/my-repo/.github/workflows/my-workflow.lock.yml@refs/heads/main";
      process.env.GITHUB_REPOSITORY = "my-org/my-repo";
      process.env.GITHUB_RUN_ID = "12345";

      mockGetWorkflowRun.mockResolvedValueOnce({ data: { referenced_workflows: [] } });

      await main();

      expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/my-repo");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("No cross-org callee found in referenced_workflows"));
    });

    it("should fall back to GITHUB_REPOSITORY when referenced_workflows has multiple cross-org entries (ambiguous)", async () => {
      // Cannot safely select one callee when multiple cross-repo workflows are referenced.
      process.env.GITHUB_WORKFLOW_REF = "caller-org/caller-repo/.github/workflows/relay.yml@refs/heads/main";
      process.env.GITHUB_REPOSITORY = "caller-org/caller-repo";
      process.env.GITHUB_RUN_ID = "12345";

      mockGetWorkflowRun.mockResolvedValueOnce({
        data: {
          referenced_workflows: [
            {
              path: "platform-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main",
              sha: "abc123",
              ref: "refs/heads/main",
            },
            {
              path: "other-org/other-repo/.github/workflows/other.lock.yml@refs/heads/main",
              sha: "def456",
              ref: "refs/heads/main",
            },
          ],
        },
      });

      await main();

      // Falls back to currentRepo since the result is ambiguous
      expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "caller-org/caller-repo");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Referenced workflows lookup is ambiguous"));
    });

    it("should fall back gracefully when referenced_workflows API call fails", async () => {
      process.env.GITHUB_WORKFLOW_REF = "caller-org/caller-repo/.github/workflows/relay.yml@refs/heads/main";
      process.env.GITHUB_REPOSITORY = "caller-org/caller-repo";
      process.env.GITHUB_RUN_ID = "12345";

      mockGetWorkflowRun.mockRejectedValueOnce(new Error("API unavailable"));

      await main();

      // Should fall back to the currentRepo (caller) — not ideal but safe degradation
      expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "caller-org/caller-repo");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("API unavailable"));
    });

    it("should fall back gracefully when GITHUB_RUN_ID is missing", async () => {
      process.env.GITHUB_WORKFLOW_REF = "caller-org/caller-repo/.github/workflows/relay.yml@refs/heads/main";
      process.env.GITHUB_REPOSITORY = "caller-org/caller-repo";
      delete process.env.GITHUB_RUN_ID;
      mockContext.runId = NaN;

      await main();

      expect(mockGetWorkflowRun).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "caller-org/caller-repo");
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Run ID is unavailable or invalid"));
    });

    it("should not call referenced_workflows API for normal cross-repo (same-org) invocations", async () => {
      // workflowRepo !== currentRepo → no API call needed
      process.env.GITHUB_WORKFLOW_REF = "my-org/platform-repo/.github/workflows/gateway.lock.yml@refs/heads/main";
      process.env.GITHUB_REPOSITORY = "my-org/app-repo";
      process.env.GITHUB_RUN_ID = "12345";

      await main();

      expect(mockGetWorkflowRun).not.toHaveBeenCalled();
      expect(mockCore.setOutput).toHaveBeenCalledWith("target_repo", "my-org/platform-repo");
    });
  });
});
