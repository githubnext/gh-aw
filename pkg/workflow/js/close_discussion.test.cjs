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
  eventName: "discussion",
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    discussion: {
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

describe("close_discussion", () => {
  let closeDiscussionScript;
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
    delete process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_LABELS;
    delete process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_TITLE_PREFIX;
    delete process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_CATEGORY;
    delete process.env.GH_AW_CLOSE_DISCUSSION_TARGET;
    delete process.env.GH_AW_WORKFLOW_NAME;
    delete process.env.GITHUB_SERVER_URL;

    // Reset context to default state
    global.context.eventName = "discussion";
    global.context.payload.discussion = { number: 42 };
    
    // Read the script content
    const scriptPath = path.join(process.cwd(), "close_discussion.cjs");
    closeDiscussionScript = fs.readFileSync(scriptPath, "utf8");
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
    await eval(`(async () => { ${closeDiscussionScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No close-discussion items found in agent output");
  });

  it("should handle missing agent output", async () => {
    // Don't set GH_AW_AGENT_OUTPUT

    // Execute the script
    await eval(`(async () => { ${closeDiscussionScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
  });

  it("should close discussion with comment in non-staged mode", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_discussion",
          body: "This discussion is resolved.",
          reason: "RESOLVED",
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";
    process.env.GITHUB_SERVER_URL = "https://github.com";

    // Mock getDiscussionDetails
    mockGithub.graphql
      .mockResolvedValueOnce({
        repository: {
          discussion: {
            id: "D_kwDOABCDEF01",
            title: "Test Discussion",
            category: { name: "General" },
            labels: { nodes: [] },
            url: "https://github.com/testowner/testrepo/discussions/42",
          },
        },
      })
      // Mock addDiscussionComment
      .mockResolvedValueOnce({
        addDiscussionComment: {
          comment: {
            id: "DC_kwDOABCDEF02",
            url: "https://github.com/testowner/testrepo/discussions/42#discussioncomment-123",
          },
        },
      })
      // Mock closeDiscussion
      .mockResolvedValueOnce({
        closeDiscussion: {
          discussion: {
            id: "D_kwDOABCDEF01",
            url: "https://github.com/testowner/testrepo/discussions/42",
          },
        },
      });

    await eval(`(async () => { ${closeDiscussionScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Found 1 close-discussion item(s)");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Processing close-discussion item 1/1"));
    expect(mockCore.info).toHaveBeenCalledWith("Adding comment to discussion #42");
    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Closing discussion #42 with reason: RESOLVED"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_number", 42);
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_url", expect.any(String));
    expect(mockCore.setOutput).toHaveBeenCalledWith("comment_url", expect.any(String));
  });

  it("should show preview in staged mode", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_discussion",
          body: "This discussion is resolved.",
          reason: "RESOLVED",
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);
    process.env.GH_AW_SAFE_OUTPUTS_STAGED = "true";

    await eval(`(async () => { ${closeDiscussionScript} })()`);

    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("🎭 Staged Mode: Close Discussions Preview"));
    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("**Target:** Current discussion"));
    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("**Reason:** RESOLVED"));
    expect(mockCore.summary.write).toHaveBeenCalled();
    expect(mockCore.info).toHaveBeenCalledWith("📝 Discussion close preview written to step summary");
  });

  it("should filter by required labels", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_discussion",
          body: "Closing this discussion.",
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);
    process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_LABELS = "resolved,completed";

    // Mock discussion without required labels
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: {
          id: "D_kwDOABCDEF01",
          title: "Test Discussion",
          category: { name: "General" },
          labels: { nodes: [{ name: "question" }] },
          url: "https://github.com/testowner/testrepo/discussions/42",
        },
      },
    });

    await eval(`(async () => { ${closeDiscussionScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Discussion #42 does not have required labels: resolved, completed");
    expect(mockCore.setOutput).not.toHaveBeenCalled();
  });

  it("should filter by title prefix", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_discussion",
          body: "Closing this discussion.",
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);
    process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_TITLE_PREFIX = "[task]";

    // Mock discussion without required title prefix
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: {
          id: "D_kwDOABCDEF01",
          title: "Test Discussion",
          category: { name: "General" },
          labels: { nodes: [] },
          url: "https://github.com/testowner/testrepo/discussions/42",
        },
      },
    });

    await eval(`(async () => { ${closeDiscussionScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Discussion #42 does not have required title prefix: [task]");
    expect(mockCore.setOutput).not.toHaveBeenCalled();
  });

  it("should filter by category", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_discussion",
          body: "Closing this discussion.",
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);
    process.env.GH_AW_CLOSE_DISCUSSION_REQUIRED_CATEGORY = "Announcements";

    // Mock discussion in different category
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: {
          id: "D_kwDOABCDEF01",
          title: "Test Discussion",
          category: { name: "General" },
          labels: { nodes: [] },
          url: "https://github.com/testowner/testrepo/discussions/42",
        },
      },
    });

    await eval(`(async () => { ${closeDiscussionScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Discussion #42 is not in required category: Announcements");
    expect(mockCore.setOutput).not.toHaveBeenCalled();
  });

  it("should handle explicit discussion_number", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_discussion",
          body: "Closing this discussion.",
          discussion_number: 99,
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);
    process.env.GH_AW_CLOSE_DISCUSSION_TARGET = "*";
    process.env.GH_AW_WORKFLOW_NAME = "Test Workflow";

    mockGithub.graphql
      .mockResolvedValueOnce({
        repository: {
          discussion: {
            id: "D_kwDOABCDEF01",
            title: "Test Discussion",
            category: { name: "General" },
            labels: { nodes: [] },
            url: "https://github.com/testowner/testrepo/discussions/99",
          },
        },
      })
      .mockResolvedValueOnce({
        addDiscussionComment: {
          comment: {
            id: "DC_kwDOABCDEF02",
            url: "https://github.com/testowner/testrepo/discussions/99#discussioncomment-123",
          },
        },
      })
      .mockResolvedValueOnce({
        closeDiscussion: {
          discussion: {
            id: "D_kwDOABCDEF01",
            url: "https://github.com/testowner/testrepo/discussions/99",
          },
        },
      });

    await eval(`(async () => { ${closeDiscussionScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Closing discussion #99"));
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_number", 99);
  });

  it("should skip if not in discussion context with triggering target", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_discussion",
          body: "Closing this discussion.",
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    // Change context to non-discussion
    mockContext.eventName = "issues";

    await eval(`(async () => { ${closeDiscussionScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith('Target is "triggering" but not running in discussion context, skipping discussion close');
    expect(mockCore.setOutput).not.toHaveBeenCalled();
  });

  it("should handle GraphQL errors gracefully", async () => {
    const validatedOutput = {
      items: [
        {
          type: "close_discussion",
          body: "This discussion is resolved.",
        },
      ],
      errors: [],
    };

    setAgentOutput(validatedOutput);

    // Mock GraphQL error
    mockGithub.graphql.mockRejectedValueOnce(new Error("GraphQL error: Discussion not found"));

    await expect(async () => {
      await eval(`(async () => { ${closeDiscussionScript} })()`);
    }).rejects.toThrow();

    expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to close discussion #42"));
  });
});
