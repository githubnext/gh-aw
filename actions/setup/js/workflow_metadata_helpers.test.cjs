// @ts-check

const { getWorkflowMetadata } = require("./workflow_metadata_helpers.cjs");

describe("getWorkflowMetadata", () => {
  let originalEnv;
  let originalContext;

  beforeEach(() => {
    // Save original environment
    originalEnv = { ...process.env };

    // Save and mock global context
    originalContext = global.context;
    global.context = {
      runId: 123456,
      payload: {
        repository: {
          html_url: "https://github.com/test-owner/test-repo",
        },
      },
    };
  });

  afterEach(() => {
    // Restore original environment
    process.env = originalEnv;

    // Restore original context
    global.context = originalContext;
  });

  it("should extract workflow metadata from environment and context", () => {
    // Set environment variables
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GH_AW_WORKFLOW_ID = "test-workflow-id";
    process.env.GITHUB_SERVER_URL = "https://github.com";

    const metadata = getWorkflowMetadata("test-owner", "test-repo");

    expect(metadata).toEqual({
      workflowName: "Test Workflow",
      workflowId: "test-workflow-id",
      runId: 123456,
      runUrl: "https://github.com/test-owner/test-repo/actions/runs/123456",
    });
  });

  it("should use defaults when environment variables are missing", () => {
    // Clear environment variables
    delete process.env.GH_AW_WORKFLOW_NAME;
    delete process.env.GH_AW_WORKFLOW_ID;
    delete process.env.GITHUB_SERVER_URL;

    const metadata = getWorkflowMetadata("test-owner", "test-repo");

    expect(metadata.workflowName).toBe("Workflow");
    expect(metadata.workflowId).toBe("");
    expect(metadata.runId).toBe(123456);
    expect(metadata.runUrl).toBe("https://github.com/test-owner/test-repo/actions/runs/123456");
  });

  it("should construct runUrl from githubServer when repository payload is missing", () => {
    // Mock context without repository payload
    global.context = {
      runId: 789012,
      payload: {},
    };

    process.env.GITHUB_SERVER_URL = "https://github.enterprise.com";

    const metadata = getWorkflowMetadata("enterprise-owner", "enterprise-repo");

    expect(metadata.runUrl).toBe("https://github.enterprise.com/enterprise-owner/enterprise-repo/actions/runs/789012");
  });

  it("should handle missing context gracefully", () => {
    // Mock context with missing runId
    global.context = {
      payload: {
        repository: {
          html_url: "https://github.com/test-owner/test-repo",
        },
      },
    };

    const metadata = getWorkflowMetadata("test-owner", "test-repo");

    expect(metadata.runId).toBe(0);
    expect(metadata.runUrl).toBe("https://github.com/test-owner/test-repo/actions/runs/0");
  });
});
