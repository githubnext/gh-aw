import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  setOutput: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn(),
  },
  warning: vi.fn(),
  error: vi.fn(),
};

const mockGithub = {
  request: vi.fn(),
};

const mockContext = {
  runId: 12345,
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

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("create_discussion.cjs", () => {
  let createDiscussionScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GITHUB_AW_AGENT_OUTPUT;
    delete process.env.GITHUB_AW_DISCUSSION_TITLE_PREFIX;
    delete process.env.GITHUB_AW_DISCUSSION_CATEGORY_ID;

    // Read the script content
    const scriptPath = path.join(
      process.cwd(),
      "pkg/workflow/js/create_discussion.cjs"
    );
    createDiscussionScript = fs.readFileSync(scriptPath, "utf8");
  });

  it("should handle missing GITHUB_AW_AGENT_OUTPUT environment variable", async () => {
    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "No GITHUB_AW_AGENT_OUTPUT environment variable found"
    );
    consoleSpy.mockRestore();
  });

  it("should handle empty agent output", async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = "   "; // Use spaces instead of empty string
    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith("Agent output content is empty");
    consoleSpy.mockRestore();
  });

  it("should handle invalid JSON in agent output", async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = "invalid json";
    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Check that it logs the content length first, then the error
    expect(consoleSpy).toHaveBeenCalledWith("Agent output content length:", 12);
    expect(consoleSpy).toHaveBeenCalledWith(
      "Error parsing agent output JSON:",
      expect.stringContaining("Unexpected token")
    );
    consoleSpy.mockRestore();
  });

  it("should handle missing create-discussion items", async () => {
    const validOutput = {
      items: [{ type: "create-issue", title: "Test Issue", body: "Test body" }],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    expect(consoleSpy).toHaveBeenCalledWith(
      "No create-discussion items found in agent output"
    );
    consoleSpy.mockRestore();
  });

  it("should create discussions successfully with basic configuration", async () => {
    // Mock the REST API responses
    mockGithub.request
      .mockResolvedValueOnce({
        // Discussion categories response
        data: [{ id: "DIC_test456", name: "General", slug: "general" }],
      })
      .mockResolvedValueOnce({
        // Create discussion response
        data: {
          id: "D_test789",
          number: 1,
          title: "Test Discussion",
          html_url: "https://github.com/testowner/testrepo/discussions/1",
        },
      });

    const validOutput = {
      items: [
        {
          type: "create-discussion",
          title: "Test Discussion",
          body: "Test discussion body",
        },
      ],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Verify REST API calls
    expect(mockGithub.request).toHaveBeenCalledTimes(2);

    // Verify discussion categories request
    expect(mockGithub.request).toHaveBeenNthCalledWith(
      1,
      "GET /repos/{owner}/{repo}/discussions/categories",
      { owner: "testowner", repo: "testrepo" }
    );

    // Verify create discussion request
    expect(mockGithub.request).toHaveBeenNthCalledWith(
      2,
      "POST /repos/{owner}/{repo}/discussions",
      {
        owner: "testowner",
        repo: "testrepo",
        category_id: "DIC_test456",
        title: "Test Discussion",
        body: expect.stringContaining("Test discussion body"),
      }
    );

    // Verify outputs were set
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_number", 1);
    expect(mockCore.setOutput).toHaveBeenCalledWith(
      "discussion_url",
      "https://github.com/testowner/testrepo/discussions/1"
    );

    // Verify summary was written
    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(
      expect.stringContaining("## GitHub Discussions")
    );
    expect(mockCore.summary.write).toHaveBeenCalled();

    consoleSpy.mockRestore();
  });

  it("should apply title prefix when configured", async () => {
    // Mock the REST API responses
    mockGithub.request
      .mockResolvedValueOnce({
        data: [{ id: "DIC_test456", name: "General", slug: "general" }],
      })
      .mockResolvedValueOnce({
        data: {
          id: "D_test789",
          number: 1,
          title: "[ai] Test Discussion",
          html_url: "https://github.com/testowner/testrepo/discussions/1",
        },
      });

    const validOutput = {
      items: [
        {
          type: "create-discussion",
          title: "Test Discussion",
          body: "Test discussion body",
        },
      ],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
    process.env.GITHUB_AW_DISCUSSION_TITLE_PREFIX = "[ai] ";

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Verify the title was prefixed
    expect(mockGithub.request).toHaveBeenNthCalledWith(
      2,
      "POST /repos/{owner}/{repo}/discussions",
      expect.objectContaining({
        title: "[ai] Test Discussion",
      })
    );

    consoleSpy.mockRestore();
  });

  it("should use specified category ID when configured", async () => {
    // Mock the REST API responses
    mockGithub.request
      .mockResolvedValueOnce({
        data: [
          { id: "DIC_test456", name: "General", slug: "general" },
          { id: "DIC_custom789", name: "Custom", slug: "custom" },
        ],
      })
      .mockResolvedValueOnce({
        data: {
          id: "D_test789",
          number: 1,
          title: "Test Discussion",
          html_url: "https://github.com/testowner/testrepo/discussions/1",
        },
      });

    const validOutput = {
      items: [
        {
          type: "create-discussion",
          title: "Test Discussion",
          body: "Test discussion body",
        },
      ],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
    process.env.GITHUB_AW_DISCUSSION_CATEGORY_ID = "DIC_custom789";

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Verify the specified category was used
    expect(mockGithub.request).toHaveBeenNthCalledWith(
      2,
      "POST /repos/{owner}/{repo}/discussions",
      expect.objectContaining({
        category_id: "DIC_custom789",
      })
    );

    consoleSpy.mockRestore();
  });

  it("should handle repositories without discussions enabled gracefully", async () => {
    // Mock the REST API to return 404 for discussion categories (simulating discussions not enabled)
    const discussionError = new Error("Not Found");
    discussionError.status = 404;
    mockGithub.request.mockRejectedValue(discussionError);

    const validOutput = {
      items: [
        {
          type: "create-discussion",
          title: "Test Discussion",
          body: "Test discussion body",
        },
      ],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);

    const consoleSpy = vi.spyOn(console, "log").mockImplementation(() => {});

    // Execute the script - should exit gracefully without throwing
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Should log appropriate warning message
    expect(consoleSpy).toHaveBeenCalledWith(
      "⚠ Cannot create discussions: Discussions are not enabled for this repository"
    );
    expect(consoleSpy).toHaveBeenCalledWith(
      "Consider enabling discussions in repository settings if you want to create discussions automatically"
    );

    // Should not attempt to create any discussions
    expect(mockGithub.request).toHaveBeenCalledTimes(1); // Only the categories call
    expect(mockGithub.request).not.toHaveBeenCalledWith(
      "POST /repos/{owner}/{repo}/discussions",
      expect.any(Object)
    );

    consoleSpy.mockRestore();
  });
});
