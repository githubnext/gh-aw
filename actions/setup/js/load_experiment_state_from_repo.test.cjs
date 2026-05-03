// @ts-check
import { describe, it, expect, beforeEach, vi } from "vitest";

// Mock globals used by load_experiment_state_from_repo.cjs
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

const mockGetOctokit = vi.fn();

global.core = mockCore;
global.getOctokit = mockGetOctokit;

const { fetchFileFromBranch } = await import("./load_experiment_state_from_repo.cjs");

describe("load_experiment_state_from_repo", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe("fetchFileFromBranch", () => {
    it("returns file content when branch and file exist", async () => {
      const stateContent = JSON.stringify({ counts: { my_exp: { A: 3, B: 3 } } });
      const encoded = Buffer.from(stateContent, "utf8").toString("base64");
      const mockOctokit = {
        rest: {
          repos: {
            getContent: vi.fn().mockResolvedValue({
              data: { type: "file", content: encoded + "\n" },
            }),
          },
        },
      };

      const result = await fetchFileFromBranch(mockOctokit, "owner", "repo", "experiments/myworkflow", "state.json");

      expect(result).toBe(stateContent);
      expect(mockOctokit.rest.repos.getContent).toHaveBeenCalledWith({
        owner: "owner",
        repo: "repo",
        path: "state.json",
        ref: "experiments/myworkflow",
      });
    });

    it("returns null when the branch does not exist (404)", async () => {
      const err = new Error("Not Found");
      // @ts-ignore
      err.status = 404;
      const mockOctokit = {
        rest: {
          repos: {
            getContent: vi.fn().mockRejectedValue(err),
          },
        },
      };

      const result = await fetchFileFromBranch(mockOctokit, "owner", "repo", "experiments/new-workflow", "state.json");

      expect(result).toBeNull();
    });

    it("returns null when the file does not exist (404)", async () => {
      const err = new Error("Not Found");
      // @ts-ignore
      err.status = 404;
      const mockOctokit = {
        rest: {
          repos: {
            getContent: vi.fn().mockRejectedValue(err),
          },
        },
      };

      const result = await fetchFileFromBranch(mockOctokit, "owner", "repo", "experiments/myworkflow", "state.json");

      expect(result).toBeNull();
    });

    it("rethrows non-404 errors", async () => {
      const err = new Error("Server Error");
      // @ts-ignore
      err.status = 500;
      const mockOctokit = {
        rest: {
          repos: {
            getContent: vi.fn().mockRejectedValue(err),
          },
        },
      };

      await expect(fetchFileFromBranch(mockOctokit, "owner", "repo", "experiments/myworkflow", "state.json")).rejects.toThrow("Server Error");
    });

    it("returns null when the API returns a directory", async () => {
      const mockOctokit = {
        rest: {
          repos: {
            getContent: vi.fn().mockResolvedValue({
              data: [{ type: "file", name: "state.json" }],
            }),
          },
        },
      };

      const result = await fetchFileFromBranch(mockOctokit, "owner", "repo", "experiments/myworkflow", "state.json");

      expect(result).toBeNull();
    });
  });
});
