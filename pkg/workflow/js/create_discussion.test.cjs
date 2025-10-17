import { describe, it, expect, beforeEach, vi } from "vitest";
import fs from "fs";
import path from "path";

// Mock the global objects that GitHub Actions provides
const mockCore = {
  // Core logging functions
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),

  // Core workflow functions
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  setSecret: vi.fn(),

  // Input/state functions (less commonly used but included for completeness)
  getInput: vi.fn(),
  getBooleanInput: vi.fn(),
  getMultilineInput: vi.fn(),
  getState: vi.fn(),
  saveState: vi.fn(),

  // Group functions
  startGroup: vi.fn(),
  endGroup: vi.fn(),
  group: vi.fn(),

  // Other utility functions
  addPath: vi.fn(),
  setCommandEcho: vi.fn(),
  isDebug: vi.fn().mockReturnValue(false),
  getIDToken: vi.fn(),
  toPlatformPath: vi.fn(),
  toPosixPath: vi.fn(),
  toWin32Path: vi.fn(),

  // Summary object with chainable methods
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  request: vi.fn(),
  graphql: vi.fn(),
  rest: {
    repos: {
      getAllRepositoryDiscussionCategories: vi.fn(),
      createRepositoryDiscussion: vi.fn(),
    },
  },
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
    delete process.env.GITHUB_AW_DISCUSSION_CATEGORY;

    // Read the script content
    const scriptPath = path.join(process.cwd(), "create_discussion.cjs");
    createDiscussionScript = fs.readFileSync(scriptPath, "utf8");
    createDiscussionScript = createDiscussionScript.replace("export {};", "");
  });

  it("should handle missing GITHUB_AW_AGENT_OUTPUT environment variable", async () => {
    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("No GITHUB_AW_AGENT_OUTPUT environment variable found");
  });

  it("should handle empty agent output", async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = "   "; // Use spaces instead of empty string
    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty");
  });

  it("should handle invalid JSON in agent output", async () => {
    process.env.GITHUB_AW_AGENT_OUTPUT = "invalid json";
    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Check that it logs the content length first, then the error
    expect(mockCore.info).toHaveBeenCalledWith("Agent output content length: 12");
    expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringMatching(/Error parsing agent output JSON:.*Unexpected token/));
  });

  it("should handle missing create-discussion items", async () => {
    const validOutput = {
      items: [{ type: "create_issue", title: "Test Issue", body: "Test body" }],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    expect(mockCore.warning).toHaveBeenCalledWith("No create-discussion items found in agent output");
  });

  it("should create discussions successfully with basic configuration", async () => {
    // Mock the GraphQL API responses
    mockGithub.graphql.mockResolvedValueOnce({
      // Repository query response with categories
      repository: {
        id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
        discussionCategories: {
          nodes: [{ id: "DIC_test456", name: "General", slug: "general" }],
        },
      },
    });

    mockGithub.graphql.mockResolvedValueOnce({
      // Create discussion mutation response
      createDiscussion: {
        discussion: {
          id: "D_test789",
          number: 1,
          title: "Test Discussion",
          url: "https://github.com/testowner/testrepo/discussions/1",
        },
      },
    });

    const validOutput = {
      items: [
        {
          type: "create_discussion",
          title: "Test Discussion",
          body: "Test discussion body",
        },
      ],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Verify GraphQL API calls
    expect(mockGithub.graphql).toHaveBeenCalledTimes(2);

    // Verify repository query with categories
    expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("query($owner: String!, $repo: String!)"), {
      owner: "testowner",
      repo: "testrepo",
    });

    // Verify create discussion mutation
    expect(mockGithub.graphql).toHaveBeenCalledWith(
      expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"),
      {
        repositoryId: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
        categoryId: "DIC_test456",
        title: "Test Discussion",
        body: expect.stringContaining("Test discussion body"),
      }
    );

    // Verify outputs were set
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_number", 1);
    expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_url", "https://github.com/testowner/testrepo/discussions/1");

    // Verify summary was written
    expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("## GitHub Discussions"));
    expect(mockCore.summary.write).toHaveBeenCalled();
  });

  it("should apply title prefix when configured", async () => {
    // Mock the GraphQL API responses
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
        discussionCategories: {
          nodes: [{ id: "DIC_test456", name: "General", slug: "general" }],
        },
      },
    });

    mockGithub.graphql.mockResolvedValueOnce({
      createDiscussion: {
        discussion: {
          id: "D_test789",
          number: 1,
          title: "[ai] Test Discussion",
          url: "https://github.com/testowner/testrepo/discussions/1",
        },
      },
    });

    const validOutput = {
      items: [
        {
          type: "create_discussion",
          title: "Test Discussion",
          body: "Test discussion body",
        },
      ],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
    process.env.GITHUB_AW_DISCUSSION_TITLE_PREFIX = "[ai] ";

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Verify the title was prefixed in the GraphQL mutation
    expect(mockGithub.graphql).toHaveBeenCalledWith(
      expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"),
      expect.objectContaining({
        title: "[ai] Test Discussion",
      })
    );
  });

  it("should use specified category ID when configured", async () => {
    // Mock the GraphQL API responses
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
        discussionCategories: {
          nodes: [
            { id: "DIC_test456", name: "General", slug: "general" },
            { id: "DIC_custom789", name: "Custom", slug: "custom" },
          ],
        },
      },
    });

    mockGithub.graphql.mockResolvedValueOnce({
      createDiscussion: {
        discussion: {
          id: "D_test789",
          number: 1,
          title: "Test Discussion",
          url: "https://github.com/testowner/testrepo/discussions/1",
        },
      },
    });

    const validOutput = {
      items: [
        {
          type: "create_discussion",
          title: "Test Discussion",
          body: "Test discussion body",
        },
      ],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
    process.env.GITHUB_AW_DISCUSSION_CATEGORY = "DIC_custom789";

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Verify the specified category was used in the GraphQL mutation
    expect(mockGithub.graphql).toHaveBeenCalledWith(
      expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"),
      expect.objectContaining({
        categoryId: "DIC_custom789",
      })
    );
  });

  it("should handle repositories without discussions enabled gracefully", async () => {
    // Mock the GraphQL API to return error for discussion categories (simulating discussions not enabled)
    const discussionError = new Error("Could not resolve to a Repository");
    mockGithub.graphql.mockRejectedValue(discussionError);

    const validOutput = {
      items: [
        {
          type: "create_discussion",
          title: "Test Discussion",
          body: "Test discussion body",
        },
      ],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);

    // Execute the script - should exit gracefully without throwing
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Should log appropriate warning message
    expect(mockCore.info).toHaveBeenCalledWith("⚠ Cannot create discussions: Discussions are not enabled for this repository");
    expect(mockCore.info).toHaveBeenCalledWith(
      "Consider enabling discussions in repository settings if you want to create discussions automatically"
    );

    // Should only attempt the GraphQL query once and not attempt to create discussions
    expect(mockGithub.graphql).toHaveBeenCalledTimes(1);
  });

  it("should match category by name when ID is not found", async () => {
    // Mock the GraphQL API responses
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
        discussionCategories: {
          nodes: [
            { id: "DIC_test456", name: "General", slug: "general" },
            { id: "DIC_custom789", name: "Custom", slug: "custom" },
          ],
        },
      },
    });

    mockGithub.graphql.mockResolvedValueOnce({
      createDiscussion: {
        discussion: {
          id: "D_test789",
          number: 1,
          title: "Test Discussion",
          url: "https://github.com/testowner/testrepo/discussions/1",
        },
      },
    });

    const validOutput = {
      items: [
        {
          type: "create_discussion",
          title: "Test Discussion",
          body: "Test discussion body",
        },
      ],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
    process.env.GITHUB_AW_DISCUSSION_CATEGORY = "Custom"; // Use category name instead of ID

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Verify the category was matched by name and its ID was used
    expect(mockCore.info).toHaveBeenCalledWith("Using category by name: Custom (DIC_custom789)");
    expect(mockGithub.graphql).toHaveBeenCalledWith(
      expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"),
      expect.objectContaining({
        categoryId: "DIC_custom789",
      })
    );
  });

  it("should match category by slug when ID and name are not found", async () => {
    // Mock the GraphQL API responses
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
        discussionCategories: {
          nodes: [
            { id: "DIC_test456", name: "General", slug: "general" },
            { id: "DIC_custom789", name: "Custom Category", slug: "custom-category" },
          ],
        },
      },
    });

    mockGithub.graphql.mockResolvedValueOnce({
      createDiscussion: {
        discussion: {
          id: "D_test789",
          number: 1,
          title: "Test Discussion",
          url: "https://github.com/testowner/testrepo/discussions/1",
        },
      },
    });

    const validOutput = {
      items: [
        {
          type: "create_discussion",
          title: "Test Discussion",
          body: "Test discussion body",
        },
      ],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
    process.env.GITHUB_AW_DISCUSSION_CATEGORY = "custom-category"; // Use category slug instead of ID or name

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Verify the category was matched by slug and its ID was used
    expect(mockCore.info).toHaveBeenCalledWith("Using category by slug: Custom Category (DIC_custom789)");
    expect(mockGithub.graphql).toHaveBeenCalledWith(
      expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"),
      expect.objectContaining({
        categoryId: "DIC_custom789",
      })
    );
  });

  it("should warn and fall back to default when category is not found", async () => {
    // Mock the GraphQL API responses
    mockGithub.graphql.mockResolvedValueOnce({
      repository: {
        id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
        discussionCategories: {
          nodes: [{ id: "DIC_test456", name: "General", slug: "general" }],
        },
      },
    });

    mockGithub.graphql.mockResolvedValueOnce({
      createDiscussion: {
        discussion: {
          id: "D_test789",
          number: 1,
          title: "Test Discussion",
          url: "https://github.com/testowner/testrepo/discussions/1",
        },
      },
    });

    const validOutput = {
      items: [
        {
          type: "create_discussion",
          title: "Test Discussion",
          body: "Test discussion body",
        },
      ],
    };
    process.env.GITHUB_AW_AGENT_OUTPUT = JSON.stringify(validOutput);
    process.env.GITHUB_AW_DISCUSSION_CATEGORY = "NonExistent"; // Category that doesn't exist

    // Execute the script
    await eval(`(async () => { ${createDiscussionScript} })()`);

    // Verify warning was logged and fallback was used
    expect(mockCore.warning).toHaveBeenCalledWith('Category "NonExistent" not found by ID, name, or slug. Available categories: General');
    expect(mockCore.info).toHaveBeenCalledWith("Falling back to default category: General (DIC_test456)");
    expect(mockGithub.graphql).toHaveBeenCalledWith(
      expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"),
      expect.objectContaining({
        categoryId: "DIC_test456",
      })
    );
  });
});
