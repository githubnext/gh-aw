import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {},
  graphql: vi.fn(),
};

const mockContext = {
  eventName: "issue_comment",
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    issue: {
      number: 42,
    },
    repository: {
      html_url: "https://github.com/testowner/testrepo",
    },
  },
};

// Set up global mocks before importing the module
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("hide_comment", () => {
  let hideCommentScript;
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
    delete process.env.GH_AW_SAFE_OUTPUTS_STAGED;
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GITHUB_SERVER_URL;

    // Reset context to default state
    global.context.eventName = "issue_comment";
    global.context.payload.issue = { number: 42 };

    // Read the script content
    const scriptPath = path.join(process.cwd(), "hide_comment.cjs");
    hideCommentScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    // Clean up temp files
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }
  });

  it("should handle empty agent output", async () => {
    setAgentOutput({ items: [], errors: [] });

    // Execute the script
    await eval(`(async () => { ${hideCommentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No hide-comment items found in agent output");
  });

  it("should handle missing agent output", async () => {
    // Don't set GH_AW_AGENT_OUTPUT

    // Execute the script
    await eval(`(async () => { ${hideCommentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
  });

  it("should hide a comment successfully", async () => {
    const commentNodeId = "IC_kwDOABCD123456";

    setAgentOutput({
      items: [
        {
          type: "hide_comment",
          comment_id: commentNodeId,
        },
      ],
      errors: [],
    });

    // Mock GraphQL response for hide comment
    mockGithub.graphql.mockResolvedValueOnce({
      minimizeComment: {
        minimizedComment: {
          isMinimized: true,
        },
      },
    });

    // Execute the script
    await eval(`(async () => { ${hideCommentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Found 1 hide-comment item(s)");
    expect(mockCore.info).toHaveBeenCalledWith(`Hiding comment: ${commentNodeId}`);
    expect(mockCore.info).toHaveBeenCalledWith(`Successfully hidden comment: ${commentNodeId}`);
    expect(mockGithub.graphql).toHaveBeenCalledWith(
      expect.stringContaining("minimizeComment"),
      expect.objectContaining({ nodeId: commentNodeId })
    );
    expect(mockCore.setOutput).toHaveBeenCalledWith("comment_id", commentNodeId);
    expect(mockCore.setOutput).toHaveBeenCalledWith("is_hidden", "true");
  });

  it("should handle GraphQL errors", async () => {
    const commentNodeId = "IC_kwDOABCD123456";

    setAgentOutput({
      items: [
        {
          type: "hide_comment",
          comment_id: commentNodeId,
        },
      ],
      errors: [],
    });

    // Mock GraphQL error
    const errorMessage = "Comment not found";
    mockGithub.graphql.mockRejectedValueOnce(new Error(errorMessage));

    // Execute the script
    await eval(`(async () => { ${hideCommentScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining(errorMessage));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining(errorMessage));
  });

  it("should preview hiding in staged mode", async () => {
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

    const commentNodeId = "IC_kwDOABCD123456";

    setAgentOutput({
      items: [
        {
          type: "hide_comment",
          comment_id: commentNodeId,
        },
      ],
      errors: [],
    });

    // Execute the script
    await eval(`(async () => { ${hideCommentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Found 1 hide-comment item(s)");
    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Staged Mode: Hide Comments Preview"));
    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining(commentNodeId));
    expect(mockCore.summary.write).toHaveBeenCalled();
    expect(mockGithub.graphql).not.toHaveBeenCalled();
  });

  it("should handle multiple hide-comment items", async () => {
    const commentNodeId1 = "IC_kwDOABCD111111";
    const commentNodeId2 = "IC_kwDOABCD222222";

    setAgentOutput({
      items: [
        {
          type: "hide_comment",
          comment_id: commentNodeId1,
        },
        {
          type: "hide_comment",
          comment_id: commentNodeId2,
        },
      ],
      errors: [],
    });

    // Mock GraphQL responses
    mockGithub.graphql
      .mockResolvedValueOnce({
        minimizeComment: {
          minimizedComment: {
            isMinimized: true,
          },
        },
      })
      .mockResolvedValueOnce({
        minimizeComment: {
          minimizedComment: {
            isMinimized: true,
          },
        },
      });

    // Execute the script
    await eval(`(async () => { ${hideCommentScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Found 2 minimize-comment item(s)");
    expect(mockGithub.graphql).toHaveBeenCalledTimes(2);
    expect(mockCore.info).toHaveBeenCalledWith(`Successfully hidden comment: ${commentNodeId1}`);
    expect(mockCore.info).toHaveBeenCalledWith(`Successfully hidden comment: ${commentNodeId2}`);
  });

  it("should fail when comment_id is missing", async () => {
    setAgentOutput({
      items: [
        {
          type: "hide_comment",
          // Missing comment_id
        },
      ],
      errors: [],
    });

    // Execute the script
    await eval(`(async () => { ${hideCommentScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("comment_id is required"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("comment_id is required"));
  });

  it("should fail when hiding returns false", async () => {
    const commentNodeId = "IC_kwDOABCD123456";

    setAgentOutput({
      items: [
        {
          type: "hide_comment",
          comment_id: commentNodeId,
        },
      ],
      errors: [],
    });

    // Mock GraphQL response where minimize fails
    mockGithub.graphql.mockResolvedValueOnce({
      minimizeComment: {
        minimizedComment: {
          isMinimized: false,
        },
      },
    });

    // Execute the script
    await eval(`(async () => { ${hideCommentScript} })()`);

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to hide comment"));
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Failed to hide comment"));
  });
});
