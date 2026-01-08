// @ts-check
const { describe, it, expect, beforeEach, vi } = require("vitest");
const { listProjectItemsWithUrls } = require("./list_project_items_with_urls.cjs");

// Mock the GitHub API
const mockGithub = {
  graphql: vi.fn(),
};

// Mock core
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  getInput: vi.fn(),
};

// Set up globals
global.github = mockGithub;
global.core = mockCore;

describe("listProjectItemsWithUrls", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("should list project items with URLs for org project", async () => {
    const projectUrl = "https://github.com/orgs/myorg/projects/123";

    // Mock project lookup
    mockGithub.graphql.mockResolvedValueOnce({
      organization: {
        projectV2: {
          id: "PVT_project123",
          title: "Test Project",
          url: projectUrl,
        },
      },
    });

    // Mock items query
    mockGithub.graphql.mockResolvedValueOnce({
      node: {
        items: {
          nodes: [
            {
              id: "PVTI_item1",
              type: "ISSUE",
              content: {
                __typename: "Issue",
                id: "I_issue1",
                number: 42,
                title: "Test Issue",
                url: "https://github.com/myorg/myrepo/issues/42",
                state: "OPEN",
                repository: {
                  name: "myrepo",
                  owner: {
                    login: "myorg",
                  },
                },
              },
              fieldValues: {
                nodes: [
                  {
                    name: "In Progress",
                    field: {
                      name: "Status",
                    },
                  },
                ],
              },
            },
            {
              id: "PVTI_item2",
              type: "PULL_REQUEST",
              content: {
                __typename: "PullRequest",
                id: "PR_pr1",
                number: 100,
                title: "Test PR",
                url: "https://github.com/myorg/myrepo/pull/100",
                state: "OPEN",
                repository: {
                  name: "myrepo",
                  owner: {
                    login: "myorg",
                  },
                },
              },
              fieldValues: {
                nodes: [],
              },
            },
          ],
          pageInfo: {
            hasNextPage: false,
            endCursor: null,
          },
        },
      },
    });

    const result = await listProjectItemsWithUrls({ project: projectUrl });

    expect(result).toHaveLength(2);
    
    // Check first item (Issue)
    expect(result[0]).toMatchObject({
      id: "PVTI_item1",
      type: "ISSUE",
      content: {
        type: "Issue",
        number: 42,
        title: "Test Issue",
        url: "https://github.com/myorg/myrepo/issues/42",
        state: "OPEN",
        repository: {
          owner: "myorg",
          name: "myrepo",
        },
      },
      fields: {
        Status: "In Progress",
      },
    });

    // Check second item (PR)
    expect(result[1]).toMatchObject({
      id: "PVTI_item2",
      type: "PULL_REQUEST",
      content: {
        type: "PullRequest",
        number: 100,
        title: "Test PR",
        url: "https://github.com/myorg/myrepo/pull/100",
        state: "OPEN",
      },
    });
  });

  it("should list project items with URLs for user project", async () => {
    const projectUrl = "https://github.com/users/johndoe/projects/456";

    // Mock project lookup
    mockGithub.graphql.mockResolvedValueOnce({
      user: {
        projectV2: {
          id: "PVT_user_project",
          title: "User Project",
          url: projectUrl,
        },
      },
    });

    // Mock items query
    mockGithub.graphql.mockResolvedValueOnce({
      node: {
        items: {
          nodes: [
            {
              id: "PVTI_item3",
              type: "DRAFT_ISSUE",
              content: {
                __typename: "DraftIssue",
                id: "DI_draft1",
                title: "Draft Issue",
              },
              fieldValues: {
                nodes: [],
              },
            },
          ],
          pageInfo: {
            hasNextPage: false,
            endCursor: null,
          },
        },
      },
    });

    const result = await listProjectItemsWithUrls({ project: projectUrl });

    expect(result).toHaveLength(1);
    expect(result[0]).toMatchObject({
      id: "PVTI_item3",
      type: "DRAFT_ISSUE",
      content: {
        type: "DraftIssue",
        title: "Draft Issue",
      },
    });
  });

  it("should handle pagination", async () => {
    const projectUrl = "https://github.com/orgs/myorg/projects/123";

    // Mock project lookup
    mockGithub.graphql.mockResolvedValueOnce({
      organization: {
        projectV2: {
          id: "PVT_project123",
          title: "Test Project",
          url: projectUrl,
        },
      },
    });

    // Mock first page
    mockGithub.graphql.mockResolvedValueOnce({
      node: {
        items: {
          nodes: [{ id: "PVTI_item1", type: "ISSUE", content: null, fieldValues: { nodes: [] } }],
          pageInfo: {
            hasNextPage: true,
            endCursor: "cursor1",
          },
        },
      },
    });

    // Mock second page
    mockGithub.graphql.mockResolvedValueOnce({
      node: {
        items: {
          nodes: [{ id: "PVTI_item2", type: "ISSUE", content: null, fieldValues: { nodes: [] } }],
          pageInfo: {
            hasNextPage: false,
            endCursor: null,
          },
        },
      },
    });

    const result = await listProjectItemsWithUrls({ project: projectUrl });

    expect(result).toHaveLength(2);
    expect(mockGithub.graphql).toHaveBeenCalledTimes(3); // 1 for project + 2 for items
  });

  it("should throw error for invalid project URL", async () => {
    await expect(
      listProjectItemsWithUrls({ project: "invalid-url" })
    ).rejects.toThrow("Invalid project URL");
  });

  it("should throw error when project not found", async () => {
    mockGithub.graphql.mockResolvedValueOnce({
      organization: {
        projectV2: null,
      },
    });

    await expect(
      listProjectItemsWithUrls({ project: "https://github.com/orgs/myorg/projects/999" })
    ).rejects.toThrow("Project #999 not found");
  });
});
