import { describe, it, expect, beforeEach, vi } from "vitest";
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
    isDebug: vi.fn().mockReturnValue(!1),
    getIDToken: vi.fn(),
    toPlatformPath: vi.fn(),
    toPosixPath: vi.fn(),
    toWin32Path: vi.fn(),
    summary: { addRaw: vi.fn().mockReturnThis(), write: vi.fn().mockResolvedValue() },
  },
  mockGithub = { request: vi.fn(), graphql: vi.fn(), rest: { repos: { getAllRepositoryDiscussionCategories: vi.fn(), createRepositoryDiscussion: vi.fn() } } },
  mockContext = { runId: 12345, repo: { owner: "testowner", repo: "testrepo" }, payload: { repository: { html_url: "https://github.com/testowner/testrepo" } } };
((global.core = mockCore),
  (global.github = mockGithub),
  (global.context = mockContext),
  describe("create_discussion.cjs", () => {
    let createDiscussionScript, tempFilePath;
    const setAgentOutput = data => {
      tempFilePath = path.join("/tmp", `test_agent_output_${Date.now()}_${Math.random().toString(36).slice(2)}.json`);
      const content = "string" == typeof data ? data : JSON.stringify(data);
      (fs.writeFileSync(tempFilePath, content), (process.env.GH_AW_AGENT_OUTPUT = tempFilePath));
    };
    (beforeEach(() => {
      (vi.clearAllMocks(),
        delete process.env.GH_AW_AGENT_OUTPUT,
        delete process.env.GH_AW_DISCUSSION_TITLE_PREFIX,
        delete process.env.GH_AW_DISCUSSION_CATEGORY,
        delete process.env.GH_AW_TARGET_REPO_SLUG,
        delete process.env.GH_AW_ALLOWED_REPOS);
      const scriptPath = path.join(process.cwd(), "create_discussion.cjs");
      ((createDiscussionScript = fs.readFileSync(scriptPath, "utf8")), (createDiscussionScript = createDiscussionScript.replace("export {};", "")));
    }),
      afterEach(() => {
        tempFilePath && require("fs").existsSync(tempFilePath) && (require("fs").unlinkSync(tempFilePath), (tempFilePath = void 0));
      }),
      it("should handle missing GH_AW_AGENT_OUTPUT environment variable", async () => {
        (await eval(`(async () => { ${createDiscussionScript}; await main(); })()`), expect(mockCore.info).toHaveBeenCalledWith("No GH_AW_AGENT_OUTPUT environment variable found"));
      }),
      it("should handle empty agent output", async () => {
        (setAgentOutput(""), await eval(`(async () => { ${createDiscussionScript}; await main(); })()`), expect(mockCore.info).toHaveBeenCalledWith("Agent output content is empty"));
      }),
      it("should handle invalid JSON in agent output", async () => {
        (setAgentOutput("invalid json"),
          await eval(`(async () => { ${createDiscussionScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Agent output content length: 12"),
          expect(mockCore.error).toHaveBeenCalledWith(expect.stringMatching(/Error parsing agent output JSON:.*Unexpected token/)));
      }),
      it("should handle missing create-discussion items", async () => {
        const validOutput = { items: [{ type: "create_issue", title: "Test Issue", body: "Test body" }] };
        (setAgentOutput(validOutput), await eval(`(async () => { ${createDiscussionScript}; await main(); })()`), expect(mockCore.warning).toHaveBeenCalledWith("No create-discussion items found in agent output"));
      }),
      it("should create discussions successfully with basic configuration", async () => {
        (mockGithub.graphql.mockResolvedValueOnce({ repository: { id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=", discussionCategories: { nodes: [{ id: "DIC_test456", name: "General", slug: "general" }] } } }),
          mockGithub.graphql.mockResolvedValueOnce({ createDiscussion: { discussion: { id: "D_test789", number: 1, title: "Test Discussion", url: "https://github.com/testowner/testrepo/discussions/1" } } }));
        const validOutput = { items: [{ type: "create_discussion", title: "Test Discussion", body: "Test discussion body" }] };
        (setAgentOutput(validOutput),
          await eval(`(async () => { ${createDiscussionScript}; await main(); })()`),
          expect(mockGithub.graphql).toHaveBeenCalledTimes(2),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("query($owner: String!, $repo: String!)"), { owner: "testowner", repo: "testrepo" }),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"), {
            repositoryId: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
            categoryId: "DIC_test456",
            title: "Test Discussion",
            body: expect.stringContaining("Test discussion body"),
          }),
          expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_number", 1),
          expect(mockCore.setOutput).toHaveBeenCalledWith("discussion_url", "https://github.com/testowner/testrepo/discussions/1"),
          expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("## GitHub Discussions")),
          expect(mockCore.summary.write).toHaveBeenCalled());
      }),
      it("should apply title prefix when configured", async () => {
        (mockGithub.graphql.mockResolvedValueOnce({ repository: { id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=", discussionCategories: { nodes: [{ id: "DIC_test456", name: "General", slug: "general" }] } } }),
          mockGithub.graphql.mockResolvedValueOnce({ createDiscussion: { discussion: { id: "D_test789", number: 1, title: "[ai] Test Discussion", url: "https://github.com/testowner/testrepo/discussions/1" } } }));
        const validOutput = { items: [{ type: "create_discussion", title: "Test Discussion", body: "Test discussion body" }] };
        (setAgentOutput(validOutput),
          (process.env.GH_AW_DISCUSSION_TITLE_PREFIX = "[ai] "),
          await eval(`(async () => { ${createDiscussionScript}; await main(); })()`),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"), expect.objectContaining({ title: "[ai] Test Discussion" })));
      }),
      it("should use specified category ID when configured", async () => {
        (mockGithub.graphql.mockResolvedValueOnce({
          repository: {
            id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
            discussionCategories: {
              nodes: [
                { id: "DIC_test456", name: "General", slug: "general" },
                { id: "DIC_custom789", name: "Custom", slug: "custom" },
              ],
            },
          },
        }),
          mockGithub.graphql.mockResolvedValueOnce({ createDiscussion: { discussion: { id: "D_test789", number: 1, title: "Test Discussion", url: "https://github.com/testowner/testrepo/discussions/1" } } }));
        const validOutput = { items: [{ type: "create_discussion", title: "Test Discussion", body: "Test discussion body" }] };
        (setAgentOutput(validOutput),
          (process.env.GH_AW_DISCUSSION_CATEGORY = "DIC_custom789"),
          await eval(`(async () => { ${createDiscussionScript}; await main(); })()`),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"), expect.objectContaining({ categoryId: "DIC_custom789" })));
      }),
      it("should handle repositories without discussions enabled gracefully", async () => {
        const discussionError = new Error("Could not resolve to a Repository");
        mockGithub.graphql.mockRejectedValue(discussionError);
        const validOutput = { items: [{ type: "create_discussion", title: "Test Discussion", body: "Test discussion body" }] };
        (setAgentOutput(validOutput),
          await eval(`(async () => { ${createDiscussionScript}; await main(); })()`),
          expect(mockCore.warning).toHaveBeenCalledWith("Skipping discussion: Discussions are not enabled for repository 'testowner/testrepo'"),
          expect(mockGithub.graphql).toHaveBeenCalledTimes(1));
      }),
      it("should match category by name when ID is not found", async () => {
        (mockGithub.graphql.mockResolvedValueOnce({
          repository: {
            id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
            discussionCategories: {
              nodes: [
                { id: "DIC_test456", name: "General", slug: "general" },
                { id: "DIC_custom789", name: "Custom", slug: "custom" },
              ],
            },
          },
        }),
          mockGithub.graphql.mockResolvedValueOnce({ createDiscussion: { discussion: { id: "D_test789", number: 1, title: "Test Discussion", url: "https://github.com/testowner/testrepo/discussions/1" } } }));
        const validOutput = { items: [{ type: "create_discussion", title: "Test Discussion", body: "Test discussion body" }] };
        (setAgentOutput(validOutput),
          (process.env.GH_AW_DISCUSSION_CATEGORY = "Custom"),
          await eval(`(async () => { ${createDiscussionScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Using category by name: Custom (DIC_custom789)"),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"), expect.objectContaining({ categoryId: "DIC_custom789" })));
      }),
      it("should match category by slug when ID and name are not found", async () => {
        (mockGithub.graphql.mockResolvedValueOnce({
          repository: {
            id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=",
            discussionCategories: {
              nodes: [
                { id: "DIC_test456", name: "General", slug: "general" },
                { id: "DIC_custom789", name: "Custom Category", slug: "custom-category" },
              ],
            },
          },
        }),
          mockGithub.graphql.mockResolvedValueOnce({ createDiscussion: { discussion: { id: "D_test789", number: 1, title: "Test Discussion", url: "https://github.com/testowner/testrepo/discussions/1" } } }));
        const validOutput = { items: [{ type: "create_discussion", title: "Test Discussion", body: "Test discussion body" }] };
        (setAgentOutput(validOutput),
          (process.env.GH_AW_DISCUSSION_CATEGORY = "custom-category"),
          await eval(`(async () => { ${createDiscussionScript}; await main(); })()`),
          expect(mockCore.info).toHaveBeenCalledWith("Using category by slug: Custom Category (DIC_custom789)"),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"), expect.objectContaining({ categoryId: "DIC_custom789" })));
      }),
      it("should warn and fall back to default when category is not found", async () => {
        (mockGithub.graphql.mockResolvedValueOnce({ repository: { id: "MDEwOlJlcG9zaXRvcnkxMjM0NTY3ODk=", discussionCategories: { nodes: [{ id: "DIC_test456", name: "General", slug: "general" }] } } }),
          mockGithub.graphql.mockResolvedValueOnce({ createDiscussion: { discussion: { id: "D_test789", number: 1, title: "Test Discussion", url: "https://github.com/testowner/testrepo/discussions/1" } } }));
        const validOutput = { items: [{ type: "create_discussion", title: "Test Discussion", body: "Test discussion body" }] };
        (setAgentOutput(validOutput),
          (process.env.GH_AW_DISCUSSION_CATEGORY = "NonExistent"),
          await eval(`(async () => { ${createDiscussionScript}; await main(); })()`),
          expect(mockCore.warning).toHaveBeenCalledWith('Category "NonExistent" not found by ID, name, or slug. Available categories: General'),
          expect(mockCore.info).toHaveBeenCalledWith("Falling back to default category: General (DIC_test456)"),
          expect(mockGithub.graphql).toHaveBeenCalledWith(expect.stringContaining("mutation($repositoryId: ID!, $categoryId: ID!, $title: String!, $body: String!)"), expect.objectContaining({ categoryId: "DIC_test456" })));
      }));
  }));
