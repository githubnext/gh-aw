import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),
  summary: {
    addHeading: vi.fn().mockReturnThis(),
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    repos: {
      createCommitStatus: vi.fn(),
    },
  },
  graphql: vi.fn(),
};

const mockContext = {
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  sha: "abc123def456",
  payload: {
    repository: {
      html_url: "https://github.com/testowner/testrepo",
    },
  },
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("create_commit_status.cjs", () => {
  let tempFilePath;

  // Helper function to set agent output via file
  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_COMMIT_STATUS_CONTEXT;
    delete process.env.GH_AW_WORKFLOW_NAME;

    // Reset context
    mockContext.sha = "abc123def456";
    mockContext.payload = {
      repository: {
        html_url: "https://github.com/testowner/testrepo",
      },
    };
  });

  afterEach(() => {
    // Clean up temp file
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
    }
  });

  it("should set empty outputs when no agent output file", async () => {
    // Don't set GH_AW_AGENT_OUTPUT
    await import("./create_commit_status.cjs");

    expect(mockCore.setOutput).toHaveBeenCalledWith("status_created", "");
    expect(mockCore.setOutput).toHaveBeenCalledWith("status_url", "");
  });

  it("should handle no create-commit-status items", async () => {
    setAgentOutput({
      items: [{ type: "create_issue", title: "Test", body: "Body" }],
      errors: [],
    });

    await import("./create_commit_status.cjs");

    expect(mockCore.info).toHaveBeenCalledWith("No create-commit-status items found in agent output");
  });

  it("should fail when state is missing", async () => {
    setAgentOutput({
      items: [
        {
          type: "create_commit_status",
          description: "Build passed",
        },
      ],
      errors: [],
    });

    await import("./create_commit_status.cjs");

    expect(mockCore.setFailed).toHaveBeenCalledWith("Commit status 'state' is required");
  });

  it("should fail when description is missing", async () => {
    setAgentOutput({
      items: [
        {
          type: "create_commit_status",
          state: "success",
        },
      ],
      errors: [],
    });

    await import("./create_commit_status.cjs");

    expect(mockCore.setFailed).toHaveBeenCalledWith("Commit status 'description' is required");
  });

  it("should fail when state is invalid", async () => {
    setAgentOutput({
      items: [
        {
          type: "create_commit_status",
          state: "invalid",
          description: "Build passed",
        },
      ],
      errors: [],
    });

    await import("./create_commit_status.cjs");

    expect(mockCore.setFailed).toHaveBeenCalledWith(
      expect.stringContaining("Invalid commit status state: invalid")
    );
  });

  it("should create commit status successfully with PR head SHA", async () => {
    mockContext.payload.pull_request = {
      head: { sha: "pr-sha-123" },
    };

    setAgentOutput({
      items: [
        {
          type: "create_commit_status",
          state: "success",
          description: "Build passed",
        },
      ],
      errors: [],
    });

    mockGithub.rest.repos.createCommitStatus.mockResolvedValue({
      data: { url: "https://api.github.com/status/1" },
    });

    await import("./create_commit_status.cjs");

    expect(mockGithub.rest.repos.createCommitStatus).toHaveBeenCalledWith({
      owner: "testowner",
      repo: "testrepo",
      sha: "pr-sha-123",
      state: "success",
      description: "Build passed",
      context: "default",
    });

    expect(mockCore.setOutput).toHaveBeenCalledWith("status_created", "true");
    expect(mockCore.setOutput).toHaveBeenCalledWith("status_url", "https://api.github.com/status/1");
  });

  it("should use custom context from environment", async () => {
    process.env.GH_AW_COMMIT_STATUS_CONTEXT = "ci/test";

    setAgentOutput({
      items: [
        {
          type: "create_commit_status",
          state: "pending",
          description: "Testing",
        },
      ],
      errors: [],
    });

    mockGithub.rest.repos.createCommitStatus.mockResolvedValue({
      data: { url: "https://api.github.com/status/2" },
    });

    await import("./create_commit_status.cjs");

    expect(mockGithub.rest.repos.createCommitStatus).toHaveBeenCalledWith(
      expect.objectContaining({
        context: "ci/test",
      })
    );
  });

  it("should use SHA from push event", async () => {
    mockContext.payload.after = "push-sha-456";

    setAgentOutput({
      items: [
        {
          type: "create_commit_status",
          state: "failure",
          description: "Build failed",
        },
      ],
      errors: [],
    });

    mockGithub.rest.repos.createCommitStatus.mockResolvedValue({
      data: { url: "https://api.github.com/status/3" },
    });

    await import("./create_commit_status.cjs");

    expect(mockGithub.rest.repos.createCommitStatus).toHaveBeenCalledWith(
      expect.objectContaining({
        sha: "push-sha-456",
      })
    );
  });

  it("should include optional target_url when provided", async () => {
    setAgentOutput({
      items: [
        {
          type: "create_commit_status",
          state: "success",
          description: "Build passed",
          target_url: "https://example.com/build/123",
        },
      ],
      errors: [],
    });

    mockGithub.rest.repos.createCommitStatus.mockResolvedValue({
      data: { url: "https://api.github.com/status/4" },
    });

    await import("./create_commit_status.cjs");

    expect(mockGithub.rest.repos.createCommitStatus).toHaveBeenCalledWith(
      expect.objectContaining({
        target_url: "https://example.com/build/123",
      })
    );
  });

  it("should use custom context from agent output", async () => {
    setAgentOutput({
      items: [
        {
          type: "create_commit_status",
          state: "error",
          description: "Deploy failed",
          context: "deploy/production",
        },
      ],
      errors: [],
    });

    mockGithub.rest.repos.createCommitStatus.mockResolvedValue({
      data: { url: "https://api.github.com/status/5" },
    });

    await import("./create_commit_status.cjs");

    expect(mockGithub.rest.repos.createCommitStatus).toHaveBeenCalledWith(
      expect.objectContaining({
        context: "deploy/production",
      })
    );
  });

  it("should validate target_url against allowed domains", async () => {
    process.env.GH_AW_COMMIT_STATUS_ALLOWED_DOMAINS = "example.com,trusted.org";

    setAgentOutput({
      items: [
        {
          type: "create_commit_status",
          state: "success",
          description: "Valid URL",
          target_url: "https://example.com/status",
        },
      ],
      errors: [],
    });

    mockGithub.rest.repos.createCommitStatus.mockResolvedValue({
      data: { url: "https://api.github.com/status/6" },
    });

    await import("./create_commit_status.cjs");

    expect(mockGithub.rest.repos.createCommitStatus).toHaveBeenCalledWith(
      expect.objectContaining({
        target_url: "https://example.com/status",
      })
    );
  });

  it("should reject target_url with disallowed domain", async () => {
    process.env.GH_AW_COMMIT_STATUS_ALLOWED_DOMAINS = "example.com";

    setAgentOutput({
      items: [
        {
          type: "create_commit_status",
          state: "success",
          description: "Invalid URL",
          target_url: "https://untrusted.com/status",
        },
      ],
      errors: [],
    });

    await import("./create_commit_status.cjs");

    expect(mockCore.setFailed).toHaveBeenCalledWith(
      expect.stringContaining('Target URL domain "untrusted.com" is not in the allowed domains list')
    );
  });

  it("should allow wildcard domain matching", async () => {
    process.env.GH_AW_COMMIT_STATUS_ALLOWED_DOMAINS = "*.example.com";

    setAgentOutput({
      items: [
        {
          type: "create_commit_status",
          state: "success",
          description: "Subdomain URL",
          target_url: "https://sub.example.com/status",
        },
      ],
      errors: [],
    });

    mockGithub.rest.repos.createCommitStatus.mockResolvedValue({
      data: { url: "https://api.github.com/status/7" },
    });

    await import("./create_commit_status.cjs");

    expect(mockGithub.rest.repos.createCommitStatus).toHaveBeenCalledWith(
      expect.objectContaining({
        target_url: "https://sub.example.com/status",
      })
    );
  });
});
