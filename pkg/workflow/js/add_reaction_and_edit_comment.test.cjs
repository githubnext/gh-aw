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

  // Input/state functions
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
    issues: {
      createComment: vi.fn(),
    },
  },
};

const mockContext = {
  eventName: "issues",
  runId: 12345,
  repo: {
    owner: "testowner",
    repo: "testrepo",
  },
  payload: {
    issue: {
      number: 123,
    },
    repository: {
      html_url: "https://github.com/testowner/testrepo",
    },
  },
};

// Set up global variables
global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;

describe("add_reaction_and_edit_comment.cjs", () => {
  let reactionScript;

  beforeEach(() => {
    // Reset all mocks
    vi.clearAllMocks();

    // Reset environment variables
    delete process.env.GITHUB_AW_REACTION;
    delete process.env.GITHUB_AW_COMMAND;
    delete process.env.GITHUB_AW_WORKFLOW_NAME;

    // Reset context to default state
    global.context.eventName = "issues";
    global.context.payload = {
      issue: { number: 123 },
      repository: { html_url: "https://github.com/testowner/testrepo" },
    };

    // Read the script content
    const scriptPath = path.join(process.cwd(), "add_reaction_and_edit_comment.cjs");
    reactionScript = fs.readFileSync(scriptPath, "utf8");
  });

  describe("Issue reactions", () => {
    it("should add reaction to issue successfully", async () => {
      process.env.GITHUB_AW_REACTION = "eyes";
      global.context.eventName = "issues";
      global.context.payload.issue = { number: 123 };

      mockGithub.request.mockResolvedValue({
        data: { id: 456 },
      });

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      // Verify reaction was added
      expect(mockGithub.request).toHaveBeenCalledWith(
        "POST /repos/testowner/testrepo/issues/123/reactions",
        expect.objectContaining({
          content: "eyes",
        })
      );

      expect(mockCore.setOutput).toHaveBeenCalledWith("reaction-id", "456");
    });

    it("should reject invalid reaction type", async () => {
      process.env.GITHUB_AW_REACTION = "invalid";
      global.context.eventName = "issues";
      global.context.payload.issue = { number: 123 };

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Invalid reaction type: invalid"));
      expect(mockGithub.request).not.toHaveBeenCalled();
    });
  });

  describe("Pull request reactions", () => {
    it("should add reaction to pull request successfully", async () => {
      process.env.GITHUB_AW_REACTION = "heart";
      global.context.eventName = "pull_request";
      global.context.payload = {
        pull_request: { number: 456 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      mockGithub.request.mockResolvedValue({
        data: { id: 789 },
      });

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      // Verify reaction was added (PRs use issues endpoint)
      expect(mockGithub.request).toHaveBeenCalledWith(
        "POST /repos/testowner/testrepo/issues/456/reactions",
        expect.objectContaining({
          content: "heart",
        })
      );

      expect(mockCore.setOutput).toHaveBeenCalledWith("reaction-id", "789");
    });
  });

  describe("Discussion reactions", () => {
    it("should add reaction to discussion using GraphQL", async () => {
      process.env.GITHUB_AW_REACTION = "rocket";
      global.context.eventName = "discussion";
      global.context.payload = {
        discussion: { number: 10 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      // Mock GraphQL query to get discussion ID
      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: {
            discussion: {
              id: "D_kwDOABcD1M4AaBbC",
              url: "https://github.com/testowner/testrepo/discussions/10",
            },
          },
        })
        // Mock GraphQL mutation to add reaction
        .mockResolvedValueOnce({
          addReaction: {
            reaction: {
              id: "MDg6UmVhY3Rpb24xMjM0NTY3ODk=",
              content: "ROCKET",
            },
          },
        });

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      // Verify GraphQL query was called to get discussion ID
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("query"),
        expect.objectContaining({
          owner: "testowner",
          repo: "testrepo",
          num: 10,
        })
      );

      // Verify GraphQL mutation was called to add reaction
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("mutation"),
        expect.objectContaining({
          subjectId: "D_kwDOABcD1M4AaBbC",
          content: "ROCKET",
        })
      );

      expect(mockCore.setOutput).toHaveBeenCalledWith("reaction-id", "MDg6UmVhY3Rpb24xMjM0NTY3ODk=");
    });

    it("should map reaction types correctly for GraphQL", async () => {
      const reactionTests = [
        { input: "+1", expected: "THUMBS_UP" },
        { input: "-1", expected: "THUMBS_DOWN" },
        { input: "laugh", expected: "LAUGH" },
        { input: "confused", expected: "CONFUSED" },
        { input: "heart", expected: "HEART" },
        { input: "hooray", expected: "HOORAY" },
        { input: "rocket", expected: "ROCKET" },
        { input: "eyes", expected: "EYES" },
      ];

      for (const test of reactionTests) {
        vi.clearAllMocks();
        process.env.GITHUB_AW_REACTION = test.input;
        global.context.eventName = "discussion";
        global.context.payload = {
          discussion: { number: 10 },
          repository: { html_url: "https://github.com/testowner/testrepo" },
        };

        mockGithub.graphql
          .mockResolvedValueOnce({
            repository: {
              discussion: {
                id: "D_kwDOABcD1M4AaBbC",
                url: "https://github.com/testowner/testrepo/discussions/10",
              },
            },
          })
          .mockResolvedValueOnce({
            addReaction: {
              reaction: {
                id: "MDg6UmVhY3Rpb24xMjM0NTY3ODk=",
                content: test.expected,
              },
            },
          });

        await eval(`(async () => { ${reactionScript} })()`);

        expect(mockGithub.graphql).toHaveBeenCalledWith(
          expect.stringContaining("mutation"),
          expect.objectContaining({
            content: test.expected,
          })
        );
      }
    });
  });

  describe("Discussion comment reactions", () => {
    it("should add reaction to discussion comment using GraphQL", async () => {
      process.env.GITHUB_AW_REACTION = "heart";
      global.context.eventName = "discussion_comment";
      global.context.payload = {
        discussion: { number: 10 },
        comment: {
          id: 123,
          node_id: "DC_kwDOABcD1M4AaBbC",
          html_url: "https://github.com/testowner/testrepo/discussions/10#discussioncomment-123",
        },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      // Mock GraphQL mutation to add reaction
      mockGithub.graphql.mockResolvedValueOnce({
        addReaction: {
          reaction: {
            id: "MDg6UmVhY3Rpb24xMjM0NTY3ODk=",
            content: "HEART",
          },
        },
      });

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      // Verify GraphQL mutation was called with comment node ID
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("mutation"),
        expect.objectContaining({
          subjectId: "DC_kwDOABcD1M4AaBbC",
          content: "HEART",
        })
      );

      expect(mockCore.setOutput).toHaveBeenCalledWith("reaction-id", "MDg6UmVhY3Rpb24xMjM0NTY3ODk=");
    });

    it("should fail when discussion comment node_id is missing", async () => {
      process.env.GITHUB_AW_REACTION = "eyes";
      global.context.eventName = "discussion_comment";
      global.context.payload = {
        discussion: { number: 10 },
        comment: {
          id: 123,
          // node_id is missing
        },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Discussion comment node ID not found in event payload");
      expect(mockGithub.graphql).not.toHaveBeenCalled();
    });
  });

  describe("Comment creation (always creates new comments)", () => {
    it("should create new comment for issue event", async () => {
      process.env.GITHUB_AW_REACTION = "eyes";
      process.env.GITHUB_AW_WORKFLOW_NAME = "Test Workflow";
      global.context.eventName = "issues";
      global.context.payload = {
        issue: { number: 123 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      // Mock reaction call
      mockGithub.request
        .mockResolvedValueOnce({
          data: { id: 456 },
        })
        // Mock comment creation
        .mockResolvedValueOnce({
          data: { id: 789, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-789" },
        });

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      // Verify comment was created (not edited)
      expect(mockGithub.request).toHaveBeenCalledWith(
        "POST /repos/testowner/testrepo/issues/123/comments",
        expect.objectContaining({
          body: expect.stringContaining("Agentic [Test Workflow]"),
        })
      );

      // Verify outputs
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "789");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-url", "https://github.com/testowner/testrepo/issues/123#issuecomment-789");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-repo", "testowner/testrepo");
    });

    it("should create new comment for issue_comment event (not edit)", async () => {
      process.env.GITHUB_AW_REACTION = "eyes";
      process.env.GITHUB_AW_WORKFLOW_NAME = "Test Workflow";
      process.env.GITHUB_AW_COMMAND = "test-bot"; // Command workflow
      global.context.eventName = "issue_comment";
      global.context.payload = {
        issue: { number: 123 },
        comment: { id: 456 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      // Mock reaction call
      mockGithub.request
        .mockResolvedValueOnce({
          data: { id: 111 },
        })
        // Mock comment creation (not GET for edit)
        .mockResolvedValueOnce({
          data: { id: 789, html_url: "https://github.com/testowner/testrepo/issues/123#issuecomment-789" },
        });

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      // Verify new comment was created, NOT edited
      // Should be POST to comments endpoint, not GET then PATCH to specific comment
      expect(mockGithub.request).toHaveBeenCalledWith(
        "POST /repos/testowner/testrepo/issues/comments/456",
        expect.objectContaining({
          body: expect.stringContaining("Agentic [Test Workflow]"),
        })
      );

      // Verify GET (for editing) was NOT called
      expect(mockGithub.request).not.toHaveBeenCalledWith("GET /repos/testowner/testrepo/issues/comments/456", expect.anything());

      // Verify outputs
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "789");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-url", "https://github.com/testowner/testrepo/issues/123#issuecomment-789");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-repo", "testowner/testrepo");
    });

    it("should create new comment for pull_request_review_comment event (not edit)", async () => {
      process.env.GITHUB_AW_REACTION = "rocket";
      process.env.GITHUB_AW_WORKFLOW_NAME = "PR Review Bot";
      process.env.GITHUB_AW_COMMAND = "review-bot"; // Command workflow
      global.context.eventName = "pull_request_review_comment";
      global.context.payload = {
        pull_request: { number: 456 },
        comment: { id: 789 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      // Mock reaction call
      mockGithub.request
        .mockResolvedValueOnce({
          data: { id: 222 },
        })
        // Mock comment creation
        .mockResolvedValueOnce({
          data: { id: 999, html_url: "https://github.com/testowner/testrepo/pull/456#discussion_r999" },
        });

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      // Verify new comment was created
      expect(mockGithub.request).toHaveBeenCalledWith(
        "POST /repos/testowner/testrepo/pulls/comments/789",
        expect.objectContaining({
          body: expect.stringContaining("Agentic [PR Review Bot]"),
        })
      );

      // Verify GET (for editing) was NOT called
      expect(mockGithub.request).not.toHaveBeenCalledWith("GET /repos/testowner/testrepo/pulls/comments/789", expect.anything());

      // Verify outputs
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "999");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-url", "https://github.com/testowner/testrepo/pull/456#discussion_r999");
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-repo", "testowner/testrepo");
    });

    it("should create comment on discussion when shouldEditComment is true", async () => {
      process.env.GITHUB_AW_REACTION = "eyes";
      process.env.GITHUB_AW_WORKFLOW_NAME = "Test Workflow";
      global.context.eventName = "discussion";
      global.context.payload = {
        discussion: { number: 10 },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      // Mock GraphQL query to get discussion ID
      mockGithub.graphql
        .mockResolvedValueOnce({
          repository: {
            discussion: {
              id: "D_kwDOABcD1M4AaBbC",
              url: "https://github.com/testowner/testrepo/discussions/10",
            },
          },
        })
        // Mock GraphQL mutation to add reaction
        .mockResolvedValueOnce({
          addReaction: {
            reaction: {
              id: "MDg6UmVhY3Rpb24xMjM0NTY3ODk=",
              content: "EYES",
            },
          },
        })
        // Mock GraphQL query to get discussion ID again for comment
        .mockResolvedValueOnce({
          repository: {
            discussion: {
              id: "D_kwDOABcD1M4AaBbC",
            },
          },
        })
        // Mock GraphQL mutation to add comment
        .mockResolvedValueOnce({
          addDiscussionComment: {
            comment: {
              id: "DC_kwDOABcD1M4AaBbD",
              url: "https://github.com/testowner/testrepo/discussions/10#discussioncomment-456",
            },
          },
        });

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      // Verify comment was created
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("addDiscussionComment"),
        expect.objectContaining({
          dId: "D_kwDOABcD1M4AaBbC",
          body: expect.stringContaining("Agentic [Test Workflow]"),
        })
      );

      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "DC_kwDOABcD1M4AaBbD");
      expect(mockCore.setOutput).toHaveBeenCalledWith(
        "comment-url",
        "https://github.com/testowner/testrepo/discussions/10#discussioncomment-456"
      );
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-repo", "testowner/testrepo");
    });

    it("should create new comment for discussion_comment events", async () => {
      process.env.GITHUB_AW_REACTION = "eyes";
      process.env.GITHUB_AW_COMMAND = "test-bot"; // Command workflow
      process.env.GITHUB_AW_WORKFLOW_NAME = "Discussion Bot";
      global.context.eventName = "discussion_comment";
      global.context.payload = {
        discussion: { number: 10 },
        comment: {
          id: 123,
          node_id: "DC_kwDOABcD1M4AaBbC",
        },
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      // Mock GraphQL mutation to add reaction
      mockGithub.graphql
        .mockResolvedValueOnce({
          addReaction: {
            reaction: {
              id: "MDg6UmVhY3Rpb24xMjM0NTY3ODk=",
              content: "EYES",
            },
          },
        })
        // Mock GraphQL query to get discussion ID for comment creation
        .mockResolvedValueOnce({
          repository: {
            discussion: {
              id: "D_kwDOABcD1M4AaBbC",
            },
          },
        })
        // Mock GraphQL mutation to add comment
        .mockResolvedValueOnce({
          addDiscussionComment: {
            comment: {
              id: "DC_kwDOABcD1M4AaBbE",
              url: "https://github.com/testowner/testrepo/discussions/10#discussioncomment-789",
            },
          },
        });

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      // Verify comment was created
      expect(mockGithub.graphql).toHaveBeenCalledWith(
        expect.stringContaining("addDiscussionComment"),
        expect.objectContaining({
          dId: "D_kwDOABcD1M4AaBbC",
          body: expect.stringContaining("Agentic [Discussion Bot]"),
        })
      );

      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-id", "DC_kwDOABcD1M4AaBbE");
      expect(mockCore.setOutput).toHaveBeenCalledWith(
        "comment-url",
        "https://github.com/testowner/testrepo/discussions/10#discussioncomment-789"
      );
      expect(mockCore.setOutput).toHaveBeenCalledWith("comment-repo", "testowner/testrepo");
    });
  });

  describe("Error handling", () => {
    it("should handle missing discussion number", async () => {
      process.env.GITHUB_AW_REACTION = "eyes";
      global.context.eventName = "discussion";
      global.context.payload = {
        // discussion number is missing
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Discussion number not found in event payload");
    });

    it("should handle missing discussion or comment info for discussion_comment", async () => {
      process.env.GITHUB_AW_REACTION = "eyes";
      global.context.eventName = "discussion_comment";
      global.context.payload = {
        discussion: { number: 10 },
        // comment is missing
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Discussion or comment information not found in event payload");
    });

    it("should handle unsupported event types", async () => {
      process.env.GITHUB_AW_REACTION = "eyes";
      global.context.eventName = "push";
      global.context.payload = {
        repository: { html_url: "https://github.com/testowner/testrepo" },
      };

      // Execute the script
      await eval(`(async () => { ${reactionScript} })()`);

      expect(mockCore.setFailed).toHaveBeenCalledWith("Unsupported event type: push");
    });
  });
});
