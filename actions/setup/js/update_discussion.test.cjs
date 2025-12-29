import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

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
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  graphql: vi.fn(),
};

const mockContext = {
  eventName: "discussion",
  repo: { owner: "testowner", repo: "testrepo" },
  payload: { discussion: { number: 123 } },
};

global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("update_discussion.cjs", () => {
  let updateDiscussionScript;
  let tempFilePath;

  const setAgentOutput = data => {
    tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
    const content = typeof data === "string" ? data : JSON.stringify(data);
    fs.writeFileSync(tempFilePath, content);
    process.env.GH_AW_AGENT_OUTPUT = tempFilePath;
  };

  beforeEach(() => {
    vi.clearAllMocks();
    delete process.env.GH_AW_AGENT_OUTPUT;
    delete process.env.GH_AW_UPDATE_TITLE;
    delete process.env.GH_AW_UPDATE_BODY;
    delete process.env.GH_AW_UPDATE_LABELS;
    delete process.env.GH_AW_UPDATE_TARGET;
    process.env.GH_AW_UPDATE_TITLE = "false";
    process.env.GH_AW_UPDATE_BODY = "false";
    process.env.GH_AW_UPDATE_LABELS = "false";

    const scriptPath = path.join(__dirname, "update_discussion.cjs");
    updateDiscussionScript = fs.readFileSync(scriptPath, "utf8");
  });

  afterEach(() => {
    if (tempFilePath && fs.existsSync(tempFilePath)) {
      fs.unlinkSync(tempFilePath);
      tempFilePath = undefined;
    }
  });

  it("should skip when no agent output is provided", async () => {
    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);
    expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found");
    expect(mockGithub.graphql).not.toHaveBeenCalled();
  });

  it("should skip when agent output is empty", async () => {
    setAgentOutput("");
    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);
    expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
    expect(mockGithub.graphql).not.toHaveBeenCalled();
  });

  it("should skip when not in discussion context for triggering target", async () => {
    setAgentOutput({
      items: [{ type: "update_discussion", title: "Updated title" }],
    });
    process.env.GH_AW_UPDATE_TITLE = "true";
    global.context.eventName = "push";
    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);
    expect(mockCore.info).toHaveBeenCalledWith('Target is "triggering" but not running in discussion context, skipping discussion update');
    expect(mockGithub.graphql).not.toHaveBeenCalled();
  });

  it("should update discussion title successfully", async () => {
    setAgentOutput({
      items: [{ type: "update_discussion", title: "Updated discussion title" }],
    });
    process.env.GH_AW_UPDATE_TITLE = "true";
    global.context.eventName = "discussion";

    const mockDiscussion = {
      id: "D_kwDOABCD123",
      number: 123,
      title: "Old title",
      body: "Old body",
      url: "https://github.com/testowner/testrepo/discussions/123",
    };

    // Mock the query to get discussion ID
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: mockDiscussion,
      },
    });

    // Mock the update mutation
    mockGithub.graphql.mockResolvedValueOnce({
      updateDiscussion: {
        discussion: {
          ...mockDiscussion,
          title: "Updated discussion title",
        },
      },
    });

    // Mock the final query to get updated discussion
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: {
          ...mockDiscussion,
          title: "Updated discussion title",
        },
      },
    });

    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);

    expect(mockGithub.graphql).toHaveBeenCalledTimes(3);
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_number", 123);
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_url", mockDiscussion.url);
    expect(mockCore.summary.addRaw).toHaveBeenCalled();
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should update discussion body successfully", async () => {
    setAgentOutput({
      items: [{ type: "update_discussion", body: "New discussion body content" }],
    });
    process.env.GH_AW_UPDATE_BODY = "true";
    global.context.eventName = "discussion";

    const mockDiscussion = {
      id: "D_kwDOABCD123",
      number: 123,
      title: "Test Discussion",
      body: "Old body",
      url: "https://github.com/testowner/testrepo/discussions/123",
    };

    // Mock the query
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: mockDiscussion,
      },
    });

    // Mock the update
    mockGithub.graphql.mockResolvedValueOnce({
      updateDiscussion: {
        discussion: {
          ...mockDiscussion,
          body: "New discussion body content",
        },
      },
    });

    // Mock the final query
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: {
          ...mockDiscussion,
          body: "New discussion body content",
        },
      },
    });

    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);

    expect(mockGithub.graphql).toHaveBeenCalledTimes(3);
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_number", 123);
  });

  it("should update both title and body successfully", async () => {
    setAgentOutput({
      items: [
        {
          type: "update_discussion",
          title: "New title",
          body: "New body content",
        },
      ],
    });
    process.env.GH_AW_UPDATE_TITLE = "true";
    process.env.GH_AW_UPDATE_BODY = "true";
    global.context.eventName = "discussion";

    const mockDiscussion = {
      id: "D_kwDOABCD123",
      number: 123,
      title: "Old title",
      body: "Old body",
      url: "https://github.com/testowner/testrepo/discussions/123",
    };

    // Mock the query
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: mockDiscussion,
      },
    });

    // Mock the update
    mockGithub.graphql.mockResolvedValueOnce({
      updateDiscussion: {
        discussion: {
          ...mockDiscussion,
          title: "New title",
          body: "New body content",
        },
      },
    });

    // Mock the final query
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: {
          ...mockDiscussion,
          title: "New title",
          body: "New body content",
        },
      },
    });

    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);

    expect(mockGithub.graphql).toHaveBeenCalledTimes(3);
  });

  it('should handle explicit discussion number with target "*"', async () => {
    setAgentOutput({
      items: [
        {
          type: "update_discussion",
          discussion_number: 456,
          title: "Updated title",
        },
      ],
    });
    process.env.GH_AW_UPDATE_TITLE = "true";
    process.env.GH_AW_UPDATE_TARGET = "*";
    global.context.eventName = "push";

    const mockDiscussion = {
      id: "D_kwDOABCD456",
      number: 456,
      title: "Old title",
      body: "Body",
      url: "https://github.com/testowner/testrepo/discussions/456",
    };

    // Mock the query
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: mockDiscussion,
      },
    });

    // Mock the update
    mockGithub.graphql.mockResolvedValueOnce({
      updateDiscussion: {
        discussion: {
          ...mockDiscussion,
          title: "Updated title",
        },
      },
    });

    // Mock the final query
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: {
          ...mockDiscussion,
          title: "Updated title",
        },
      },
    });

    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);

    expect(mockGithub.graphql).toHaveBeenCalledTimes(3);
    // Should use the explicit discussion number 456
    expect(mockGithub.graphql).toHaveBeenNthCalledWith(
      1,
      expect.any(String),
      expect.objectContaining({
        number: 456,
      })
    );
  });

  it("should skip when no valid updates are provided", async () => {
    setAgentOutput({
      items: [{ type: "update_discussion", title: "New title" }],
    });
    process.env.GH_AW_UPDATE_TITLE = "false";
    process.env.GH_AW_UPDATE_BODY = "false";
    global.context.eventName = "discussion";

    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No valid updates to apply for this item");
    expect(mockGithub.graphql).not.toHaveBeenCalled();
  });

  it("should use custom footer message when configured", async () => {
    setAgentOutput({
      items: [{ type: "update_discussion", body: "New discussion body" }],
    });
    process.env.GH_AW_UPDATE_BODY = "true";
    process.env.GH_AW_WORKFLOW_NAME = "Custom Workflow";
    process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
      footer: "> Custom footer by [{workflow_name}]({run_url})",
    });
    global.context.eventName = "discussion";
    global.context.runId = 789;

    const mockDiscussion = {
      id: "D_kwDOABCD123",
      number: 123,
      title: "Test Discussion",
      body: "Old body",
      url: "https://github.com/testowner/testrepo/discussions/123",
    };

    // Mock the query
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: mockDiscussion,
      },
    });

    // Mock the update and capture the body parameter
    let capturedBody;
    mockGithub.graphql.mockImplementationOnce((query, variables) => {
      capturedBody = variables.body;
      return Promise.resolve({
        updateDiscussion: {
          discussion: {
            ...mockDiscussion,
            body: variables.body,
          },
        },
      });
    });

    // Mock the final query
    mockGithub.graphql.mockImplementationOnce(() => {
      return Promise.resolve({
        repository: {
          discussion: {
            ...mockDiscussion,
            body: capturedBody,
          },
        },
      });
    });

    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);

    expect(mockGithub.graphql).toHaveBeenCalledTimes(3);
    // Verify the custom footer was used
    expect(capturedBody).toContain("Custom footer by");
    expect(capturedBody).toContain("Custom Workflow");
  });

  it("should update discussion labels successfully", async () => {
    setAgentOutput({
      items: [{ type: "update_discussion", labels: ["bug", "enhancement"] }],
    });
    process.env.GH_AW_UPDATE_LABELS = "true";
    global.context.eventName = "discussion";

    const mockDiscussion = {
      id: "D_kwDOABCD123",
      number: 123,
      title: "Test Discussion",
      body: "Test body",
      url: "https://github.com/testowner/testrepo/discussions/123",
      labels: {
        nodes: [{ id: "L_kwDOABCD001", name: "old-label" }],
      },
    };

    const mockLabels = [
      { id: "L_kwDOABCD002", name: "bug" },
      { id: "L_kwDOABCD003", name: "enhancement" },
    ];

    // Mock the first query to get discussion with labels
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: mockDiscussion,
      },
    });

    // Mock the repository labels query
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        id: "R_kwDOABCD",
        labels: {
          nodes: mockLabels,
        },
      },
    });

    // Mock remove labels mutation
    mockGithub.graphql.mockResolvedValueOnce({
      removeLabelsFromLabelable: {
        clientMutationId: null,
      },
    });

    // Mock add labels mutation
    mockGithub.graphql.mockResolvedValueOnce({
      addLabelsToLabelable: {
        clientMutationId: null,
      },
    });

    // Mock the final query to get updated discussion
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: {
          ...mockDiscussion,
          labels: {
            nodes: mockLabels,
          },
        },
      },
    });

    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);

    expect(mockGithub.graphql).toHaveBeenCalledTimes(5);
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_number", 123);
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_url", mockDiscussion.url);
  });

  it("should update both title and labels successfully", async () => {
    setAgentOutput({
      items: [
        {
          type: "update_discussion",
          title: "New title",
          labels: ["question"],
        },
      ],
    });
    process.env.GH_AW_UPDATE_TITLE = "true";
    process.env.GH_AW_UPDATE_LABELS = "true";
    global.context.eventName = "discussion";

    const mockDiscussion = {
      id: "D_kwDOABCD123",
      number: 123,
      title: "Old title",
      body: "Old body",
      url: "https://github.com/testowner/testrepo/discussions/123",
      labels: {
        nodes: [],
      },
    };

    const mockLabels = [{ id: "L_kwDOABCD004", name: "question" }];

    // Mock the first query to get discussion
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: mockDiscussion,
      },
    });

    // Mock the title update mutation
    mockGithub.graphql.mockResolvedValueOnce({
      updateDiscussion: {
        discussion: {
          ...mockDiscussion,
          title: "New title",
        },
      },
    });

    // Mock the repository labels query
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        id: "R_kwDOABCD",
        labels: {
          nodes: mockLabels,
        },
      },
    });

    // Mock add labels mutation (no need to remove since there are no existing labels)
    mockGithub.graphql.mockResolvedValueOnce({
      addLabelsToLabelable: {
        clientMutationId: null,
      },
    });

    // Mock the final query to get updated discussion
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: {
          ...mockDiscussion,
          title: "New title",
          labels: {
            nodes: mockLabels,
          },
        },
      },
    });

    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);

    expect(mockGithub.graphql).toHaveBeenCalledTimes(5);
  });

  it("should handle repository with more than 100 labels using pagination", async () => {
    setAgentOutput({
      items: [{ type: "update_discussion", labels: ["label-150"] }],
    });
    process.env.GH_AW_UPDATE_LABELS = "true";
    global.context.eventName = "discussion";

    const mockDiscussion = {
      id: "D_kwDOABCD123",
      number: 123,
      title: "Test Discussion",
      body: "Test body",
      url: "https://github.com/testowner/testrepo/discussions/123",
      labels: {
        nodes: [],
      },
    };

    // Create 150 mock labels
    const firstPageLabels = Array.from({ length: 100 }, (_, i) => ({
      id: `L_kwDOABCD${String(i + 1).padStart(3, "0")}`,
      name: `label-${i + 1}`,
    }));

    const secondPageLabels = Array.from({ length: 50 }, (_, i) => ({
      id: `L_kwDOABCD${String(i + 101).padStart(3, "0")}`,
      name: `label-${i + 101}`,
    }));

    // Mock the first query to get discussion
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: mockDiscussion,
      },
    });

    // Mock the repository labels query - first page
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        id: "R_kwDOABCD",
        labels: {
          nodes: firstPageLabels,
          pageInfo: {
            hasNextPage: true,
            endCursor: "cursor-100",
          },
        },
      },
    });

    // Mock the repository labels query - second page
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        id: "R_kwDOABCD",
        labels: {
          nodes: secondPageLabels,
          pageInfo: {
            hasNextPage: false,
            endCursor: null,
          },
        },
      },
    });

    // Mock add labels mutation
    mockGithub.graphql.mockResolvedValueOnce({
      addLabelsToLabelable: {
        clientMutationId: null,
      },
    });

    // Mock the final query to get updated discussion
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        discussion: {
          ...mockDiscussion,
          labels: {
            nodes: [{ id: "L_kwDOABCD150", name: "label-150" }],
          },
        },
      },
    });

    await eval(`(async () => { ${updateDiscussionScript}; await main(); })()`);

    // Verify pagination occurred: 1 (get discussion) + 2 (label pages) + 1 (add labels) + 1 (final query) = 5
    expect(mockGithub.graphql).toHaveBeenCalledTimes(5);

    // Verify the second labels query included the cursor
    const secondLabelsCall = mockGithub.graphql.mock.calls[2];
    expect(secondLabelsCall[1]).toHaveProperty("cursor", "cursor-100");

    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_number", 123);
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_url", mockDiscussion.url);
  });
});
