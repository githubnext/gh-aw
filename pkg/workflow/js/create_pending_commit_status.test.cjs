import { describe, it, expect, beforeEach, vi } from "vitest";

describe("create_pending_commit_status", () => {
  let mockCore;
  let mockGithub;
  let mockContext;

  beforeEach(() => {
    // Reset mocks before each test
    mockCore = {
      info: vi.fn(),
      setFailed: vi.fn(),
    };

    mockGithub = {
      rest: {
        repos: {
          createCommitStatus: vi.fn().mockResolvedValue({}),
        },
      },
    };

    mockContext = {
      sha: "abc123def456",
      runId: 123456,
      repo: {
        owner: "testowner",
        repo: "testrepo",
      },
      payload: {
        repository: {
          html_url: "https://github.com/testowner/testrepo",
        },
      },
    };

    // Set up global mocks
    global.core = mockCore;
    global.github = mockGithub;
    global.context = mockContext;

    // Reset environment variables
    delete process.env.GH_AW_COMMIT_STATUS_CONTEXT;
    delete process.env.GITHUB_SERVER_URL;
  });

  it("should create pending commit status with default context", async () => {
    await import("./create_pending_commit_status.cjs");

    expect(mockGithub.rest.repos.createCommitStatus).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      sha: "abc123def456",
      state: "pending",
      context: "agentic-workflow",
      description: "Agentic workflow is running",
      target_url: "https://github.com/testowner/testrepo/actions/runs/123456",
    });

    expect(mockCore.info).toHaveBeenCalledWith(
      "Creating pending commit status for commit: abc123def456"
    );
    expect(mockCore.info).toHaveBeenCalledWith(
      "Status context: agentic-workflow"
    );
    expect(mockCore.info).toHaveBeenCalledWith(
      "✓ Successfully created pending commit status"
    );
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });

  it("should skip when no commit SHA is available", async () => {
    // Set context.sha to empty string
    mockContext.sha = "";
    global.context = mockContext;

    await import("./create_pending_commit_status.cjs?nosha=" + Date.now());

    expect(mockCore.info).toHaveBeenCalledWith(
      "No commit SHA available in context - skipping commit status creation"
    );
    expect(mockGithub.rest.repos.createCommitStatus).not.toHaveBeenCalled();
    expect(mockCore.setFailed).not.toHaveBeenCalled();
  });
});
