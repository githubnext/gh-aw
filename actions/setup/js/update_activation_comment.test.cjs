import { describe, it, expect, beforeEach, vi } from "vitest";
import { readFileSync } from "fs";
import path from "path";
const createTestableFunction = scriptContent => {
  const beforeMainCall = scriptContent.match(/^([\s\S]*?)\s*module\.exports\s*=\s*{[\s\S]*?};?\s*$/);
  if (!beforeMainCall) throw new Error("Could not extract script content before module.exports");
  let scriptBody = beforeMainCall[1];
  // Mock the error_helpers and messages_core modules
  const mockRequire = module => {
    if (module === "./error_helpers.cjs") {
      return { getErrorMessage: error => (error instanceof Error ? error.message : String(error)) };
    }
    if (module === "./messages_core.cjs") {
      return {
        getMessages: () => {
          // Return messages config from environment
          const messagesEnv = process.env.GH_AW_SAFE_OUTPUT_MESSAGES;
          if (!messagesEnv) return null;
          try {
            return JSON.parse(messagesEnv);
          } catch (e) {
            return null;
          }
        },
      };
    }
    throw new Error(`Module ${module} not mocked in test`);
  };
  return new Function(`\n    const { github, core, context, process } = arguments[0];\n    const require = ${mockRequire.toString()};\n    \n    ${scriptBody}\n    \n    return { updateActivationComment };\n  `);
};
describe("update_activation_comment.cjs", () => {
  let createFunctionFromScript, mockDependencies;
  (beforeEach(() => {
    const scriptPath = path.join(process.cwd(), "update_activation_comment.cjs"),
      scriptContent = readFileSync(scriptPath, "utf8");
    ((createFunctionFromScript = createTestableFunction(scriptContent)),
      (mockDependencies = { github: { graphql: vi.fn(), request: vi.fn() }, core: { info: vi.fn(), warning: vi.fn(), setFailed: vi.fn() }, context: { repo: { owner: "testowner", repo: "testrepo" } }, process: { env: {} } }));
  }),
    it("should skip update when GH_AW_COMMENT_ID is not set", async () => {
      mockDependencies.process.env.GH_AW_COMMENT_ID = "";
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42),
        expect(mockDependencies.core.info).toHaveBeenCalledWith("No activation comment to update (GH_AW_COMMENT_ID not set)"),
        expect(mockDependencies.github.request).not.toHaveBeenCalled());
    }),
    it("should update issue comment with PR link", async () => {
      ((mockDependencies.process.env.GH_AW_COMMENT_ID = "123456"),
        (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"),
        mockDependencies.github.request.mockImplementation(async (method, params) =>
          method.startsWith("GET")
            ? { data: { body: "Agentic [workflow](https://github.com/testowner/testrepo/actions/runs/12345) triggered by this issue." } }
            : method.startsWith("PATCH")
              ? { data: { id: 123456, html_url: "https://github.com/testowner/testrepo/issues/1#issuecomment-123456" } }
              : { data: {} }
        ));
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42),
        expect(mockDependencies.github.request).toHaveBeenCalledWith("GET /repos/{owner}/{repo}/issues/comments/{comment_id}", expect.objectContaining({ owner: "testowner", repo: "testrepo", comment_id: 123456 })),
        expect(mockDependencies.github.request).toHaveBeenCalledWith(
          "PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}",
          expect.objectContaining({ owner: "testowner", repo: "testrepo", comment_id: 123456, body: expect.stringContaining("✅ Pull request created: [#42](https://github.com/testowner/testrepo/pull/42)") })
        ),
        expect(mockDependencies.core.info).toHaveBeenCalledWith("Successfully updated comment with pull request link"));
    }),
    it("should update discussion comment with PR link using GraphQL", async () => {
      ((mockDependencies.process.env.GH_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF"),
        (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"),
        mockDependencies.github.graphql.mockImplementation(async (query, params) =>
          query.includes("query")
            ? { node: { body: "Agentic [workflow](https://github.com/testowner/testrepo/actions/runs/12345) triggered by this discussion." } }
            : query.includes("mutation")
              ? { updateDiscussionComment: { comment: { id: "DC_kwDOABCDEF4ABCDEF", url: "https://github.com/testowner/testrepo/discussions/1#discussioncomment-123456" } } }
              : {}
        ));
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42),
        expect(mockDependencies.github.graphql).toHaveBeenCalledWith(expect.stringContaining("query($commentId: ID!)"), expect.objectContaining({ commentId: "DC_kwDOABCDEF4ABCDEF" })),
        expect(mockDependencies.github.graphql).toHaveBeenCalledWith(
          expect.stringContaining("mutation($commentId: ID!, $body: String!)"),
          expect.objectContaining({ commentId: "DC_kwDOABCDEF4ABCDEF", body: expect.stringContaining("✅ Pull request created: [#42](https://github.com/testowner/testrepo/pull/42)") })
        ),
        expect(mockDependencies.core.info).toHaveBeenCalledWith("Successfully updated discussion comment with pull request link"));
    }),
    it("should not fail workflow if comment update fails", async () => {
      ((mockDependencies.process.env.GH_AW_COMMENT_ID = "123456"), (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"), mockDependencies.github.request.mockRejectedValue(new Error("Comment update failed")));
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42),
        expect(mockDependencies.core.warning).toHaveBeenCalledWith("Failed to update activation comment: Comment update failed"),
        expect(mockDependencies.core.setFailed).not.toHaveBeenCalled());
    }),
    it("should use default repo from context if comment_repo not set", async () => {
      ((mockDependencies.process.env.GH_AW_COMMENT_ID = "123456"),
        mockDependencies.github.request.mockImplementation(async (method, params) =>
          method.startsWith("GET") ? { data: { body: "Original comment" } } : method.startsWith("PATCH") ? { data: { id: 123456, html_url: "https://github.com/testowner/testrepo/issues/1#issuecomment-123456" } } : { data: {} }
        ));
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42),
        expect(mockDependencies.github.request).toHaveBeenCalledWith("GET /repos/{owner}/{repo}/issues/comments/{comment_id}", expect.objectContaining({ owner: "testowner", repo: "testrepo" })));
    }),
    it("should handle invalid comment_repo format and fall back to context", async () => {
      ((mockDependencies.process.env.GH_AW_COMMENT_ID = "123456"),
        (mockDependencies.process.env.GH_AW_COMMENT_REPO = "invalid-format"),
        mockDependencies.github.request.mockImplementation(async (method, params) =>
          method.startsWith("GET") ? { data: { body: "Original comment" } } : method.startsWith("PATCH") ? { data: { id: 123456, html_url: "https://github.com/testowner/testrepo/issues/1#issuecomment-123456" } } : { data: {} }
        ));
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42),
        expect(mockDependencies.core.warning).toHaveBeenCalledWith('Invalid comment repo format: invalid-format, expected "owner/repo". Falling back to context.repo.'),
        expect(mockDependencies.github.request).toHaveBeenCalledWith("GET /repos/{owner}/{repo}/issues/comments/{comment_id}", expect.objectContaining({ owner: "testowner", repo: "testrepo" })));
    }),
    it("should handle deleted discussion comment (null body in GraphQL)", async () => {
      ((mockDependencies.process.env.GH_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF"), (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"), mockDependencies.github.graphql.mockResolvedValue({ node: { body: null } }));
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42),
        expect(mockDependencies.core.warning).toHaveBeenCalledWith("Unable to fetch current comment body, comment may have been deleted or is inaccessible"),
        expect(mockDependencies.github.graphql).toHaveBeenCalledTimes(1));
    }),
    it("should handle deleted discussion comment (null node in GraphQL)", async () => {
      ((mockDependencies.process.env.GH_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF"), (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"), mockDependencies.github.graphql.mockResolvedValue({ node: null }));
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42),
        expect(mockDependencies.core.warning).toHaveBeenCalledWith("Unable to fetch current comment body, comment may have been deleted or is inaccessible"),
        expect(mockDependencies.github.graphql).toHaveBeenCalledTimes(1));
    }),
    it("should handle deleted issue comment (null body in REST API)", async () => {
      ((mockDependencies.process.env.GH_AW_COMMENT_ID = "123456"), (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"), mockDependencies.github.request.mockResolvedValue({ data: { body: null } }));
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42),
        expect(mockDependencies.core.warning).toHaveBeenCalledWith("Unable to fetch current comment body, comment may have been deleted"),
        expect(mockDependencies.github.request).toHaveBeenCalledTimes(1));
    }),
    it("should handle deleted issue comment (undefined body in REST API)", async () => {
      ((mockDependencies.process.env.GH_AW_COMMENT_ID = "123456"), (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"), mockDependencies.github.request.mockResolvedValue({ data: {} }));
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/42", 42),
        expect(mockDependencies.core.warning).toHaveBeenCalledWith("Unable to fetch current comment body, comment may have been deleted"),
        expect(mockDependencies.github.request).toHaveBeenCalledTimes(1));
    }),
    it("should update issue comment with issue link when itemType is 'issue'", async () => {
      ((mockDependencies.process.env.GH_AW_COMMENT_ID = "123456"),
        (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"),
        mockDependencies.github.request.mockImplementation(async (method, params) =>
          method.startsWith("GET")
            ? { data: { body: "Agentic [workflow](https://github.com/testowner/testrepo/actions/runs/12345) triggered by this issue." } }
            : method.startsWith("PATCH")
              ? { data: { id: 123456, html_url: "https://github.com/testowner/testrepo/issues/1#issuecomment-123456" } }
              : { data: {} }
        ));
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/issues/99", 99, "issue"),
        expect(mockDependencies.github.request).toHaveBeenCalledWith(
          "PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}",
          expect.objectContaining({ owner: "testowner", repo: "testrepo", comment_id: 123456, body: expect.stringContaining("✅ Issue created: [#99](https://github.com/testowner/testrepo/issues/99)") })
        ),
        expect(mockDependencies.core.info).toHaveBeenCalledWith("Successfully updated comment with issue link"));
    }),
    it("should update discussion comment with issue link using GraphQL when itemType is 'issue'", async () => {
      ((mockDependencies.process.env.GH_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF"),
        (mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo"),
        mockDependencies.github.graphql.mockImplementation(async (query, params) =>
          query.includes("query")
            ? { node: { body: "Agentic [workflow](https://github.com/testowner/testrepo/actions/runs/12345) triggered by this discussion." } }
            : query.includes("mutation")
              ? { updateDiscussionComment: { comment: { id: "DC_kwDOABCDEF4ABCDEF", url: "https://github.com/testowner/testrepo/discussions/1#discussioncomment-123456" } } }
              : {}
        ));
      const { updateActivationComment } = createFunctionFromScript(mockDependencies);
      (await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/issues/99", 99, "issue"),
        expect(mockDependencies.github.graphql).toHaveBeenCalledWith(
          expect.stringContaining("mutation($commentId: ID!, $body: String!)"),
          expect.objectContaining({ commentId: "DC_kwDOABCDEF4ABCDEF", body: expect.stringContaining("✅ Issue created: [#99](https://github.com/testowner/testrepo/issues/99)") })
        ),
        expect(mockDependencies.core.info).toHaveBeenCalledWith("Successfully updated discussion comment with issue link"));
    }),
    describe("append-only-comments mode", () => {
      it("should create new issue comment instead of updating when append-only-comments is enabled", async () => {
        mockDependencies.process.env.GH_AW_COMMENT_ID = "123456";
        mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";
        mockDependencies.process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
          appendOnlyComments: true,
        });
        mockDependencies.context.eventName = "issues";
        mockDependencies.context.payload = { issue: { number: 42 } };

        mockDependencies.github.request.mockImplementation(async (method, params) => {
          if (method.startsWith("POST")) {
            return {
              data: {
                id: 789012,
                html_url: "https://github.com/testowner/testrepo/issues/42#issuecomment-789012",
              },
            };
          }
          return { data: {} };
        });

        const { updateActivationComment } = createFunctionFromScript(mockDependencies);
        await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/99", 99);

        // Should create a new comment, not update the existing one
        expect(mockDependencies.github.request).toHaveBeenCalledWith(
          "POST /repos/{owner}/{repo}/issues/{issue_number}/comments",
          expect.objectContaining({
            owner: "testowner",
            repo: "testrepo",
            issue_number: 42,
            body: expect.stringContaining("✅ Pull request created: [#99]"),
          })
        );
        expect(mockDependencies.core.info).toHaveBeenCalledWith("Append-only-comments enabled: creating a new comment");
        expect(mockDependencies.core.info).toHaveBeenCalledWith("Successfully created append-only comment with pull request link");

        // Should NOT call GET or PATCH for updating existing comment
        expect(mockDependencies.github.request).not.toHaveBeenCalledWith("GET /repos/{owner}/{repo}/issues/comments/{comment_id}", expect.any(Object));
        expect(mockDependencies.github.request).not.toHaveBeenCalledWith("PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}", expect.any(Object));
      });

      it("should create new discussion comment instead of updating when append-only-comments is enabled", async () => {
        mockDependencies.process.env.GH_AW_COMMENT_ID = "DC_kwDOABCDEF4ABCDEF";
        mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";
        mockDependencies.process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
          appendOnlyComments: true,
        });
        mockDependencies.context.eventName = "discussion";
        mockDependencies.context.payload = { discussion: { number: 10 } };

        mockDependencies.github.graphql.mockImplementation(async (query, params) => {
          if (query.includes("query") && query.includes("discussion(number")) {
            return {
              repository: {
                discussion: {
                  id: "D_kwDOABCDEF4ABCDEF",
                },
              },
            };
          }
          if (query.includes("mutation") && query.includes("addDiscussionComment")) {
            return {
              addDiscussionComment: {
                comment: {
                  id: "DC_kwDOABCDEF4NEW",
                  url: "https://github.com/testowner/testrepo/discussions/10#discussioncomment-new",
                },
              },
            };
          }
          return {};
        });

        const { updateActivationComment } = createFunctionFromScript(mockDependencies);
        await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/99", 99);

        // Should create a new discussion comment
        expect(mockDependencies.github.graphql).toHaveBeenCalledWith(
          expect.stringContaining("mutation"),
          expect.objectContaining({
            dId: "D_kwDOABCDEF4ABCDEF",
            body: expect.stringContaining("✅ Pull request created: [#99]"),
          })
        );
        expect(mockDependencies.core.info).toHaveBeenCalledWith("Successfully created append-only discussion comment with pull request link");

        // Should NOT call updateDiscussionComment mutation
        expect(mockDependencies.github.graphql).not.toHaveBeenCalledWith(expect.stringContaining("updateDiscussionComment"), expect.any(Object));
      });

      it("should handle missing issue number gracefully in append-only mode", async () => {
        mockDependencies.process.env.GH_AW_COMMENT_ID = "123456";
        mockDependencies.process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
          appendOnlyComments: true,
        });
        mockDependencies.context.eventName = "issues";
        mockDependencies.context.payload = {}; // No issue number

        const { updateActivationComment } = createFunctionFromScript(mockDependencies);
        await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/99", 99);

        expect(mockDependencies.core.warning).toHaveBeenCalledWith("Unable to determine issue/PR number for append-only comment; skipping");
        expect(mockDependencies.github.request).not.toHaveBeenCalled();
      });

      it("should update existing comment when append-only-comments is false", async () => {
        mockDependencies.process.env.GH_AW_COMMENT_ID = "123456";
        mockDependencies.process.env.GH_AW_COMMENT_REPO = "testowner/testrepo";
        mockDependencies.process.env.GH_AW_SAFE_OUTPUT_MESSAGES = JSON.stringify({
          appendOnlyComments: false,
        });

        mockDependencies.github.request.mockImplementation(async (method, params) => {
          if (method.startsWith("GET")) {
            return { data: { body: "Original comment" } };
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
        await updateActivationComment(mockDependencies.github, mockDependencies.context, mockDependencies.core, "https://github.com/testowner/testrepo/pull/99", 99);

        // Should update the existing comment (standard behavior)
        expect(mockDependencies.github.request).toHaveBeenCalledWith("GET /repos/{owner}/{repo}/issues/comments/{comment_id}", expect.objectContaining({ comment_id: 123456 }));
        expect(mockDependencies.github.request).toHaveBeenCalledWith("PATCH /repos/{owner}/{repo}/issues/comments/{comment_id}", expect.objectContaining({ comment_id: 123456 }));
      });
    }));
});
