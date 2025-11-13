import { describe, it, expect, beforeEach, vi } from "vitest";
import { readFileSync } from "fs";
import path from "path";

// Create testable function
const createTestableFunction = scriptContent => {
  const beforeMainCall = scriptContent.match(/^([\s\S]*?)\s*module\.exports\s*=\s*{[\s\S]*?};?\s*$/);
  if (!beforeMainCall) {
    throw new Error("Could not extract script content before module.exports");
  }

  let scriptBody = beforeMainCall[1];

  return new Function(`
    const { github, core, context, process } = arguments[0];
    
    ${scriptBody}
    
    return { updateActivationComment };
  `);
};

describe("update_activation_comment.cjs", () => {
  let createFunctionFromScript;
  let mockDependencies;

  beforeEach(() => {
    // Read the script content
    const scriptPath = path.join(process.cwd(), "update_activation_comment.cjs");
    const scriptContent = readFileSync(scriptPath, "utf8");

    // Create testable function
    createFunctionFromScript = createTestableFunction(scriptContent);

    // Set up mock dependencies
    mockDependencies = {
      github: {
        graphql: vi.fn(),
        request: vi.fn(),
      },
      core: {
        info: vi.fn(),
        warning: vi.fn(),
        setFailed: vi.fn(),
      },
      context: {
        repo: {
          owner: "testowner",
          repo: "testrepo",
        },
      },
      process: {
        env: {},
      },
    };
  });

  it("should skip update when GH_AW_COMMENT_ID is not set", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "";

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/pull/42",
      42
    );

    expect(mockDependencies.core.info).toHaveBeenCalledWith("No activation comment to update (GH_AW_COMMENT_ID not set)");
    expect(mockDependencies.github.request).not.toHaveBeenCalled();
  });

  it("should update issue comment with PR link", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "123456";
    mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";

    // Mock GET request to fetch current comment
    mockDependencies.github.request.mockImplementation(async (method, params) => {
      if (method.startsWith("GET")) {
        return {
          data: {
            body: "Agentic [workflow](https://github.com/testowner/testrepo/actions/runs/12345) triggered by this issue.",
          },
        };
      }
      // Mock PATCH request to update comment
      if (method.startsWith("PATCH")) {
        return {
          data: {
            id: 123456,
            html_url: "https://github.com/testowner/testrepo/issues/1#issuecomment-123456",
          },
        };
      }
      return { data: {} };
    });

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/pull/42",
      42
    );

    // Verify comment was fetched
    expect(mockDependencies.github.request).toHaveBeenCalledWith(
      "GET /repos/{owner}/{repo}/issues/comments/{comment_id}",
      expect.objectContaining({
        owner: "testowner",
        repo: "testrepo",
        comment_id: 123456,
      })
    );

    // Verify comment was updated with PR link
    expect(mockDependencies.github.request).toHaveBeenCalledWith(
      "PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}",
      expect.objectContaining({
        owner: "testowner",
        repo: "testrepo",
        comment_id: 123456,
        body: expect.stringContaining("✅ Pull request created: [#42](https://github.com/testowner/testrepo/pull/42)"),
      })
    );

    expect(mockDependencies.core.info).toHaveBeenCalledWith("Successfully updated comment with pull request link");
  });

  it("should update discussion comment with PR link using GraphQL", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF";
    mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";

    // Mock GraphQL for discussion comment
    mockDependencies.github.graphql.mockImplementation(async (query, params) => {
      if (query.includes("query")) {
        // Mock GET query
        return {
          node: {
            body: "Agentic [workflow](https://github.com/testowner/testrepo/actions/runs/12345) triggered by this discussion.",
          },
        };
      }
      if (query.includes("mutation")) {
        // Mock UPDATE mutation
        return {
          updateDiscussionComment: {
            comment: {
              id: "DC_kwDOABCDEF4ABCDEF",
              url: "https://github.com/testowner/testrepo/discussions/1#discussioncomment-123456",
            },
          },
        };
      }
      return {};
    });

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/pull/42",
      42
    );

    // Verify GraphQL was called to get current comment
    expect(mockDependencies.github.graphql).toHaveBeenCalledWith(
      expect.stringContaining("query($commentId: ID!)"),
      expect.objectContaining({
        commentId: "DC_kwDOABCDEF4ABCDEF",
      })
    );

    // Verify GraphQL was called to update comment
    expect(mockDependencies.github.graphql).toHaveBeenCalledWith(
      expect.stringContaining("mutation($commentId: ID!, $body: String!)"),
      expect.objectContaining({
        commentId: "DC_kwDOABCDEF4ABCDEF",
        body: expect.stringContaining("✅ Pull request created: [#42](https://github.com/testowner/testrepo/pull/42)"),
      })
    );

    expect(mockDependencies.core.info).toHaveBeenCalledWith("Successfully updated discussion comment with pull request link");
  });

  it("should not fail workflow if comment update fails", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "123456";
    mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";

    // Mock request to fail
    mockDependencies.github.request.mockRejectedValue(new Error("Comment update failed"));

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/pull/42",
      42
    );

    // Verify warning was logged but workflow didn't fail
    expect(mockDependencies.core.warning).toHaveBeenCalledWith("Failed to update activation comment: Comment update failed");
    expect(mockDependencies.core.setFailed).not.toHaveBeenCalled();
  });

  it("should use default repo from context if comment_repo not set", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "123456";
    // GH_AW_COMMENT_REPO not set

    mockDependencies.github.request.mockImplementation(async (method, params) => {
      if (method.startsWith("GET")) {
        return {
          data: {
            body: "Original comment",
          },
        };
      }
      if (method.startsWith("PATCH")) {
        return {
          data: {
            id: 123456,
            html_url: "https://github.com/testowner/testrepo/issues/1#issuecomment-123456",
          },
        };
      }
      return { data: {} };
    });

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/pull/42",
      42
    );

    // Verify request used context repo
    expect(mockDependencies.github.request).toHaveBeenCalledWith(
      "GET /repos/{owner}/{repo}/issues/comments/{comment_id}",
      expect.objectContaining({
        owner: "testowner",
        repo: "testrepo",
      })
    );
  });

  it("should handle invalid comment_repo format and fall back to context", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "123456";
    mockDependencies.process.env.GH_AW_COMMENT_REPO = "invalid-format";

    mockDependencies.github.request.mockImplementation(async (method, params) => {
      if (method.startsWith("GET")) {
        return {
          data: {
            body: "Original comment",
          },
        };
      }
      if (method.startsWith("PATCH")) {
        return {
          data: {
            id: 123456,
            html_url: "https://github.com/testowner/testrepo/issues/1#issuecomment-123456",
          },
        };
      }
      return { data: {} };
    });

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/pull/42",
      42
    );

    // Verify warning was logged
    expect(mockDependencies.core.warning).toHaveBeenCalledWith(
      'Invalid comment repo format: invalid-format, expected "owner/repo". Falling back to context.repo.'
    );

    // Verify request used context repo as fallback
    expect(mockDependencies.github.request).toHaveBeenCalledWith(
      "GET /repos/{owner}/{repo}/issues/comments/{comment_id}",
      expect.objectContaining({
        owner: "testowner",
        repo: "testrepo",
      })
    );
  });

  it("should handle deleted discussion comment (null body in GraphQL)", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF";
    mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";

    // Mock GraphQL to return null body (comment deleted)
    mockDependencies.github.graphql.mockResolvedValue({
      node: {
        body: null,
      },
    });

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/pull/42",
      42
    );

    // Verify warning was logged
    expect(mockDependencies.core.warning).toHaveBeenCalledWith(
      "Unable to fetch current comment body, comment may have been deleted or is inaccessible"
    );

    // Verify mutation was not attempted
    expect(mockDependencies.github.graphql).toHaveBeenCalledTimes(1);
  });

  it("should handle deleted discussion comment (null node in GraphQL)", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF";
    mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";

    // Mock GraphQL to return null node (comment deleted)
    mockDependencies.github.graphql.mockResolvedValue({
      node: null,
    });

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/pull/42",
      42
    );

    // Verify warning was logged
    expect(mockDependencies.core.warning).toHaveBeenCalledWith(
      "Unable to fetch current comment body, comment may have been deleted or is inaccessible"
    );

    // Verify mutation was not attempted
    expect(mockDependencies.github.graphql).toHaveBeenCalledTimes(1);
  });

  it("should handle deleted issue comment (null body in REST API)", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "123456";
    mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";

    // Mock REST API to return null body (comment deleted)
    mockDependencies.github.request.mockResolvedValue({
      data: {
        body: null,
      },
    });

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/pull/42",
      42
    );

    // Verify warning was logged
    expect(mockDependencies.core.warning).toHaveBeenCalledWith("Unable to fetch current comment body, comment may have been deleted");

    // Verify PATCH was not attempted
    expect(mockDependencies.github.request).toHaveBeenCalledTimes(1);
  });

  it("should handle deleted issue comment (undefined body in REST API)", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "123456";
    mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";

    // Mock REST API to return undefined body (comment deleted)
    mockDependencies.github.request.mockResolvedValue({
      data: {},
    });

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/pull/42",
      42
    );

    // Verify warning was logged
    expect(mockDependencies.core.warning).toHaveBeenCalledWith("Unable to fetch current comment body, comment may have been deleted");

    // Verify PATCH was not attempted
    expect(mockDependencies.github.request).toHaveBeenCalledTimes(1);
  });

  it("should update issue comment with issue link when itemType is 'issue'", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "123456";
    mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";

    // Mock GET request to fetch current comment
    mockDependencies.github.request.mockImplementation(async (method, params) => {
      if (method.startsWith("GET")) {
        return {
          data: {
            body: "Agentic [workflow](https://github.com/testowner/testrepo/actions/runs/12345) triggered by this issue.",
          },
        };
      }
      // Mock PATCH request to update comment
      if (method.startsWith("PATCH")) {
        return {
          data: {
            id: 123456,
            html_url: "https://github.com/testowner/testrepo/issues/1#issuecomment-123456",
          },
        };
      }
      return { data: {} };
    });

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/issues/99",
      99,
      "issue"
    );

    // Verify comment was updated with issue link
    expect(mockDependencies.github.request).toHaveBeenCalledWith(
      "PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}",
      expect.objectContaining({
        owner: "testowner",
        repo: "testrepo",
        comment_id: 123456,
        body: expect.stringContaining("✅ Issue created: [#99](https://github.com/testowner/testrepo/issues/99)"),
      })
    );

    expect(mockDependencies.core.info).toHaveBeenCalledWith("Successfully updated comment with issue link");
  });

  it("should update discussion comment with issue link using GraphQL when itemType is 'issue'", async () => {
    mockDependencies.process.env.GH_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF";
    mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";

    // Mock GraphQL for discussion comment
    mockDependencies.github.graphql.mockImplementation(async (query, params) => {
      if (query.includes("query")) {
        // Mock GET query
        return {
          node: {
            body: "Agentic [workflow](https://github.com/testowner/testrepo/actions/runs/12345) triggered by this discussion.",
          },
        };
      }
      if (query.includes("mutation")) {
        // Mock UPDATE mutation
        return {
          updateDiscussionComment: {
            comment: {
              id: "DC_kwDOABCDEF4ABCDEF",
              url: "https://github.com/testowner/testrepo/discussions/1#discussioncomment-123456",
            },
          },
        };
      }
      return {};
    });

    const { updateActivationComment } = createFunctionFromScript(mockDependencies);

    await updateActivationComment(
      mockDependencies.github,
      mockDependencies.context,
      mockDependencies.core,
      "https://github.com/testowner/testrepo/issues/99",
      99,
      "issue"
    );

    // Verify GraphQL was called to update comment
    expect(mockDependencies.github.graphql).toHaveBeenCalledWith(
      expect.stringContaining("mutation($commentId: ID!, $body: String!)"),
      expect.objectContaining({
        commentId: "DC_kwDOABCDEF4ABCDEF",
        body: expect.stringContaining("✅ Issue created: [#99](https://github.com/testowner/testrepo/issues/99)"),
      })
    );

    expect(mockDependencies.core.info).toHaveBeenCalledWith("Successfully updated discussion comment with issue link");
  });
});
