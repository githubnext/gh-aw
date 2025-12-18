import { describe, it, expect, beforeEach, vi } from "vitest";
const mockCore = { debug: vi.fn(), info: vi.fn(), warning: vi.fn(), error: vi.fn(), setFailed: vi.fn(), setOutput: vi.fn() },
  mockGithub = { graphql: vi.fn() };
((global.core = mockCore),
  (global.github = mockGithub),
  describe("close_older_discussions.cjs", () => {
    (beforeEach(() => {
      (vi.clearAllMocks(), vi.resetModules(), delete process.env.GH_AW_SAFE_OUTPUT_MESSAGES, delete process.env.GH_AW_WORKFLOW_NAME, delete process.env.GITHUB_SERVER_URL);
    }),
      describe("searchOlderDiscussions", () => {
        (it("should return empty array when no discussions found", async () => {
          const { searchOlderDiscussions } = await import("./close_older_discussions.cjs");
          mockGithub.graphql.mockResolvedValueOnce({ search: { nodes: [] } });
          const result = await searchOlderDiscussions(mockGithub, "testowner", "testrepo", "[weekly-report] ", [], "DIC_test123", 1);
          expect(result).toEqual([]);
        }),
          it("should filter out the newly created discussion", async () => {
            const { searchOlderDiscussions } = await import("./close_older_discussions.cjs");
            mockGithub.graphql.mockResolvedValueOnce({
              search: {
                nodes: [
                  { id: "D_old1", number: 5, title: "[weekly-report] Old Report 1", url: "https://github.com/testowner/testrepo/discussions/5", category: { id: "DIC_test123" }, labels: { nodes: [] }, closed: !1 },
                  { id: "D_new", number: 10, title: "[weekly-report] New Report", url: "https://github.com/testowner/testrepo/discussions/10", category: { id: "DIC_test123" }, labels: { nodes: [] }, closed: !1 },
                ],
              },
            });
            const result = await searchOlderDiscussions(mockGithub, "testowner", "testrepo", "[weekly-report] ", [], "DIC_test123", 10);
            (expect(result).toHaveLength(1), expect(result[0].number).toBe(5));
          }),
          it("should filter out already closed discussions", async () => {
            const { searchOlderDiscussions } = await import("./close_older_discussions.cjs");
            mockGithub.graphql.mockResolvedValueOnce({
              search: {
                nodes: [
                  { id: "D_open", number: 5, title: "[weekly-report] Open Report", url: "https://github.com/testowner/testrepo/discussions/5", category: { id: "DIC_test123" }, labels: { nodes: [] }, closed: !1 },
                  { id: "D_closed", number: 6, title: "[weekly-report] Closed Report", url: "https://github.com/testowner/testrepo/discussions/6", category: { id: "DIC_test123" }, labels: { nodes: [] }, closed: !0 },
                ],
              },
            });
            const result = await searchOlderDiscussions(mockGithub, "testowner", "testrepo", "[weekly-report] ", [], "DIC_test123", 10);
            (expect(result).toHaveLength(1), expect(result[0].number).toBe(5));
          }),
          it("should filter by title prefix", async () => {
            const { searchOlderDiscussions } = await import("./close_older_discussions.cjs");
            mockGithub.graphql.mockResolvedValueOnce({
              search: {
                nodes: [
                  { id: "D_match", number: 5, title: "[weekly-report] Matching Report", url: "https://github.com/testowner/testrepo/discussions/5", category: { id: "DIC_test123" }, labels: { nodes: [] }, closed: !1 },
                  { id: "D_nomatch", number: 6, title: "[daily-report] Non-matching Report", url: "https://github.com/testowner/testrepo/discussions/6", category: { id: "DIC_test123" }, labels: { nodes: [] }, closed: !1 },
                ],
              },
            });
            const result = await searchOlderDiscussions(mockGithub, "testowner", "testrepo", "[weekly-report] ", [], "DIC_test123", 10);
            (expect(result).toHaveLength(1), expect(result[0].number).toBe(5));
          }),
          it("should filter by labels", async () => {
            const { searchOlderDiscussions } = await import("./close_older_discussions.cjs");
            mockGithub.graphql.mockResolvedValueOnce({
              search: {
                nodes: [
                  {
                    id: "D_haslabels",
                    number: 5,
                    title: "Report with labels",
                    url: "https://github.com/testowner/testrepo/discussions/5",
                    category: { id: "DIC_test123" },
                    labels: { nodes: [{ name: "weekly-report" }, { name: "automation" }] },
                    closed: !1,
                  },
                  {
                    id: "D_nolabels",
                    number: 6,
                    title: "Report without required labels",
                    url: "https://github.com/testowner/testrepo/discussions/6",
                    category: { id: "DIC_test123" },
                    labels: { nodes: [{ name: "other-label" }] },
                    closed: !1,
                  },
                ],
              },
            });
            const result = await searchOlderDiscussions(mockGithub, "testowner", "testrepo", "", ["weekly-report", "automation"], "DIC_test123", 10);
            (expect(result).toHaveLength(1), expect(result[0].number).toBe(5));
          }),
          it("should filter by category when specified", async () => {
            const { searchOlderDiscussions } = await import("./close_older_discussions.cjs");
            mockGithub.graphql.mockResolvedValueOnce({
              search: {
                nodes: [
                  { id: "D_rightcat", number: 5, title: "[weekly-report] Report in right category", url: "https://github.com/testowner/testrepo/discussions/5", category: { id: "DIC_reports" }, labels: { nodes: [] }, closed: !1 },
                  { id: "D_wrongcat", number: 6, title: "[weekly-report] Report in wrong category", url: "https://github.com/testowner/testrepo/discussions/6", category: { id: "DIC_general" }, labels: { nodes: [] }, closed: !1 },
                ],
              },
            });
            const result = await searchOlderDiscussions(mockGithub, "testowner", "testrepo", "[weekly-report] ", [], "DIC_reports", 10);
            (expect(result).toHaveLength(1), expect(result[0].number).toBe(5));
          }),
          it("should include all categories when categoryId is undefined", async () => {
            const { searchOlderDiscussions } = await import("./close_older_discussions.cjs");
            mockGithub.graphql.mockResolvedValueOnce({
              search: {
                nodes: [
                  { id: "D_cat1", number: 5, title: "[weekly-report] Report 1", url: "https://github.com/testowner/testrepo/discussions/5", category: { id: "DIC_reports" }, labels: { nodes: [] }, closed: !1 },
                  { id: "D_cat2", number: 6, title: "[weekly-report] Report 2", url: "https://github.com/testowner/testrepo/discussions/6", category: { id: "DIC_general" }, labels: { nodes: [] }, closed: !1 },
                ],
              },
            });
            const result = await searchOlderDiscussions(mockGithub, "testowner", "testrepo", "[weekly-report] ", [], void 0, 10);
            expect(result).toHaveLength(2);
          }),
          it("should match by both title prefix and labels when both specified", async () => {
            const { searchOlderDiscussions } = await import("./close_older_discussions.cjs");
            mockGithub.graphql.mockResolvedValueOnce({
              search: {
                nodes: [
                  { id: "D_both", number: 5, title: "[weekly-report] With labels", url: "https://github.com/testowner/testrepo/discussions/5", category: { id: "DIC_test123" }, labels: { nodes: [{ name: "automation" }] }, closed: !1 },
                  { id: "D_titleonly", number: 6, title: "[weekly-report] No labels", url: "https://github.com/testowner/testrepo/discussions/6", category: { id: "DIC_test123" }, labels: { nodes: [] }, closed: !1 },
                  { id: "D_labelonly", number: 7, title: "Different title with labels", url: "https://github.com/testowner/testrepo/discussions/7", category: { id: "DIC_test123" }, labels: { nodes: [{ name: "automation" }] }, closed: !1 },
                ],
              },
            });
            const result = await searchOlderDiscussions(mockGithub, "testowner", "testrepo", "[weekly-report] ", ["automation"], "DIC_test123", 10);
            (expect(result).toHaveLength(1), expect(result[0].number).toBe(5));
          }));
      }),
      describe("closeOlderDiscussions", () => {
        (it("should close older discussions and add comments", async () => {
          const { closeOlderDiscussions } = await import("./close_older_discussions.cjs");
          (mockGithub.graphql.mockResolvedValueOnce({
            search: { nodes: [{ id: "D_old1", number: 5, title: "[weekly-report] Old Report", url: "https://github.com/testowner/testrepo/discussions/5", category: { id: "DIC_test123" }, labels: { nodes: [] }, closed: !1 }] },
          }),
            mockGithub.graphql.mockResolvedValueOnce({ addDiscussionComment: { comment: { id: "DC_comment1", url: "https://github.com/testowner/testrepo/discussions/5#comment-1" } } }),
            mockGithub.graphql.mockResolvedValueOnce({ closeDiscussion: { discussion: { id: "D_old1", url: "https://github.com/testowner/testrepo/discussions/5" } } }));
          const result = await closeOlderDiscussions(
            mockGithub,
            "testowner",
            "testrepo",
            "[weekly-report] ",
            [],
            "DIC_test123",
            { number: 10, url: "https://github.com/testowner/testrepo/discussions/10" },
            "Test Workflow",
            "https://github.com/testowner/testrepo/actions/runs/123"
          );
          (expect(result).toHaveLength(1), expect(result[0].number).toBe(5), expect(mockGithub.graphql).toHaveBeenCalledTimes(3));
        }),
          it("should limit closed discussions to MAX_CLOSE_COUNT (10)", async () => {
            const { closeOlderDiscussions, MAX_CLOSE_COUNT } = await import("./close_older_discussions.cjs"),
              discussions = Array.from({ length: 15 }, (_, i) => ({
                id: `D_old${i}`,
                number: i + 1,
                title: `[weekly-report] Old Report ${i + 1}`,
                url: `https://github.com/testowner/testrepo/discussions/${i + 1}`,
                category: { id: "DIC_test123" },
                labels: { nodes: [] },
                closed: !1,
              }));
            mockGithub.graphql.mockResolvedValueOnce({ search: { nodes: discussions } });
            for (let i = 0; i < MAX_CLOSE_COUNT; i++)
              (mockGithub.graphql.mockResolvedValueOnce({ addDiscussionComment: { comment: { id: `DC_comment${i}`, url: `https://github.com/testowner/testrepo/discussions/${i + 1}#comment-1` } } }),
                mockGithub.graphql.mockResolvedValueOnce({ closeDiscussion: { discussion: { id: `D_old${i}`, url: `https://github.com/testowner/testrepo/discussions/${i + 1}` } } }));
            const result = await closeOlderDiscussions(
              mockGithub,
              "testowner",
              "testrepo",
              "[weekly-report] ",
              [],
              "DIC_test123",
              { number: 100, url: "https://github.com/testowner/testrepo/discussions/100" },
              "Test Workflow",
              "https://github.com/testowner/testrepo/actions/runs/123"
            );
            (expect(result).toHaveLength(MAX_CLOSE_COUNT), expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining(`Found 15 older discussions, but only closing the first ${MAX_CLOSE_COUNT}`)));
          }),
          it("should return empty array when no older discussions found", async () => {
            const { closeOlderDiscussions } = await import("./close_older_discussions.cjs");
            mockGithub.graphql.mockResolvedValueOnce({ search: { nodes: [] } });
            const result = await closeOlderDiscussions(
              mockGithub,
              "testowner",
              "testrepo",
              "[weekly-report] ",
              [],
              "DIC_test123",
              { number: 10, url: "https://github.com/testowner/testrepo/discussions/10" },
              "Test Workflow",
              "https://github.com/testowner/testrepo/actions/runs/123"
            );
            (expect(result).toEqual([]), expect(mockCore.info).toHaveBeenCalledWith("No older discussions found to close"));
          }),
          it("should continue closing other discussions if one fails", async () => {
            const { closeOlderDiscussions } = await import("./close_older_discussions.cjs");
            (mockGithub.graphql.mockResolvedValueOnce({
              search: {
                nodes: [
                  { id: "D_fail", number: 5, title: "[weekly-report] Will Fail", url: "https://github.com/testowner/testrepo/discussions/5", category: { id: "DIC_test123" }, labels: { nodes: [] }, closed: !1 },
                  { id: "D_success", number: 6, title: "[weekly-report] Will Succeed", url: "https://github.com/testowner/testrepo/discussions/6", category: { id: "DIC_test123" }, labels: { nodes: [] }, closed: !1 },
                ],
              },
            }),
              mockGithub.graphql.mockRejectedValueOnce(new Error("GraphQL error")),
              mockGithub.graphql.mockResolvedValueOnce({ addDiscussionComment: { comment: { id: "DC_comment2", url: "https://github.com/testowner/testrepo/discussions/6#comment-1" } } }),
              mockGithub.graphql.mockResolvedValueOnce({ closeDiscussion: { discussion: { id: "D_success", url: "https://github.com/testowner/testrepo/discussions/6" } } }));
            const result = await closeOlderDiscussions(
              mockGithub,
              "testowner",
              "testrepo",
              "[weekly-report] ",
              [],
              "DIC_test123",
              { number: 10, url: "https://github.com/testowner/testrepo/discussions/10" },
              "Test Workflow",
              "https://github.com/testowner/testrepo/actions/runs/123"
            );
            (expect(result).toHaveLength(1), expect(result[0].number).toBe(6), expect(mockCore.error).toHaveBeenCalledWith(expect.stringContaining("Failed to close discussion #5")));
          }),
          it("should close discussions by labels only", async () => {
            const { closeOlderDiscussions } = await import("./close_older_discussions.cjs");
            (mockGithub.graphql.mockResolvedValueOnce({
              search: {
                nodes: [
                  {
                    id: "D_labeled",
                    number: 5,
                    title: "Some Report with Labels",
                    url: "https://github.com/testowner/testrepo/discussions/5",
                    category: { id: "DIC_test123" },
                    labels: { nodes: [{ name: "weekly-report" }, { name: "automation" }] },
                    closed: !1,
                  },
                ],
              },
            }),
              mockGithub.graphql.mockResolvedValueOnce({ addDiscussionComment: { comment: { id: "DC_comment1", url: "https://github.com/testowner/testrepo/discussions/5#comment-1" } } }),
              mockGithub.graphql.mockResolvedValueOnce({ closeDiscussion: { discussion: { id: "D_labeled", url: "https://github.com/testowner/testrepo/discussions/5" } } }));
            const result = await closeOlderDiscussions(
              mockGithub,
              "testowner",
              "testrepo",
              "",
              ["weekly-report", "automation"],
              "DIC_test123",
              { number: 10, url: "https://github.com/testowner/testrepo/discussions/10" },
              "Test Workflow",
              "https://github.com/testowner/testrepo/actions/runs/123"
            );
            (expect(result).toHaveLength(1), expect(result[0].number).toBe(5), expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("labels: [weekly-report, automation]")));
          }));
      }),
      describe("MAX_CLOSE_COUNT", () => {
        it("should be set to 10", async () => {
          const { MAX_CLOSE_COUNT } = await import("./close_older_discussions.cjs");
          expect(MAX_CLOSE_COUNT).toBe(10);
        });
      }),
      describe("GRAPHQL_DELAY_MS", () => {
        it("should be set to 500ms", async () => {
          const { GRAPHQL_DELAY_MS } = await import("./close_older_discussions.cjs");
          expect(GRAPHQL_DELAY_MS).toBe(500);
        });
      }));
  }));
