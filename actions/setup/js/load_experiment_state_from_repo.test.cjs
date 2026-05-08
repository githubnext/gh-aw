// @ts-check
import fs from "fs";
import os from "os";
import path from "path";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";

// Mock globals used by load_experiment_state_from_repo.cjs
const mockCore = {
  info: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
};

const mockGetOctokit = vi.fn();

global.core = mockCore;
global.getOctokit = mockGetOctokit;

const { fetchFileFromBranch, main, validateInputs } = await import("./load_experiment_state_from_repo.cjs");
const ENV_KEYS = ["GH_AW_EXPERIMENT_STATE_FILE", "GH_AW_EXPERIMENT_STATE_DIR", "GH_AW_EXPERIMENT_BRANCH", "GITHUB_REPOSITORY"];
const MAX_STATE_FILE_BYTES = 102400;

describe("load_experiment_state_from_repo", () => {
  /** @type {Record<string, string | undefined>} */
  let envBackup = {};

  beforeEach(() => {
    vi.clearAllMocks();
    envBackup = Object.fromEntries(ENV_KEYS.map(key => [key, process.env[key]]));
  });

  afterEach(() => {
    for (const key of ENV_KEYS) {
      if (envBackup[key] === undefined) {
        delete process.env[key];
      } else {
        process.env[key] = envBackup[key];
      }
    }
    delete global.github;
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

  describe("validateInputs", () => {
    const cases = [
      {
        name: "accepts valid inputs",
        args: ["experiments/my-workflow", "owner", "repo", "owner/repo"],
        expected: { valid: true },
      },
      {
        name: "rejects empty branch",
        args: ["", "owner", "repo", "owner/repo"],
        expected: { valid: false, error: "GH_AW_EXPERIMENT_BRANCH is not set" },
      },
      {
        name: "rejects branch names with invalid characters",
        args: ["experiments/my workflow", "owner", "repo", "owner/repo"],
        expected: { valid: false, error: "GH_AW_EXPERIMENT_BRANCH contains invalid characters" },
      },
      {
        name: "rejects branch names with path traversal patterns",
        args: ["experiments/../../etc/passwd", "owner", "repo", "owner/repo"],
        expected: { valid: false, error: "GH_AW_EXPERIMENT_BRANCH contains invalid characters" },
      },
      {
        name: "rejects missing owner",
        args: ["experiments/my-workflow", "", "repo", "/repo"],
        expected: { valid: false, error: "GITHUB_REPOSITORY is not set or invalid" },
      },
      {
        name: "rejects missing repo",
        args: ["experiments/my-workflow", "owner", "", "owner/"],
        expected: { valid: false, error: "GITHUB_REPOSITORY is not set or invalid" },
      },
      {
        name: "rejects repository with extra segments",
        args: ["experiments/my-workflow", "owner", "repo", "owner/repo/extra"],
        expected: { valid: false, error: "GITHUB_REPOSITORY is not set or invalid" },
      },
      {
        name: "rejects repository with whitespace",
        args: ["experiments/my-workflow", "ow ner", "repo", "ow ner/repo"],
        expected: { valid: false, error: "GITHUB_REPOSITORY is not set or invalid" },
      },
    ];

    for (const testCase of cases) {
      it(testCase.name, () => {
        expect(validateInputs(...testCase.args)).toEqual(testCase.expected);
      });
    }
  });

  describe("main", () => {
    it("skips fetch when branch name is invalid", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-state-"));
      try {
        const stateFile = path.join(tmpDir, "state.json");
        const getContent = vi.fn();

        process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
        process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
        process.env.GH_AW_EXPERIMENT_BRANCH = "experiments/my workflow";
        process.env.GITHUB_REPOSITORY = "owner/repo";

        global.github = {
          rest: {
            repos: {
              getContent,
            },
          },
        };

        await main();

        expect(getContent).not.toHaveBeenCalled();
        expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("contains invalid characters"));
        expect(fs.existsSync(stateFile)).toBe(false);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("skips fetch when GITHUB_REPOSITORY format is invalid", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-state-"));
      try {
        const stateFile = path.join(tmpDir, "state.json");
        const getContent = vi.fn();

        process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
        process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
        process.env.GH_AW_EXPERIMENT_BRANCH = "experiments/myworkflow";
        process.env.GITHUB_REPOSITORY = "owner/repo/extra";

        global.github = {
          rest: {
            repos: {
              getContent,
            },
          },
        };

        await main();

        expect(getContent).not.toHaveBeenCalled();
        expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("GITHUB_REPOSITORY is not set or invalid"));
        expect(fs.existsSync(stateFile)).toBe(false);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("skips oversized state files", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-state-"));
      try {
        const stateFile = path.join(tmpDir, "state.json");
        const oversizedContent = "x".repeat(MAX_STATE_FILE_BYTES + 1);
        const encoded = Buffer.from(oversizedContent, "utf8").toString("base64");

        process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
        process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
        process.env.GH_AW_EXPERIMENT_BRANCH = "experiments/myworkflow";
        process.env.GITHUB_REPOSITORY = "owner/repo";

        global.github = {
          rest: {
            repos: {
              getContent: vi.fn().mockResolvedValue({
                data: { type: "file", content: encoded },
              }),
            },
          },
        };

        await main();

        expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("exceeds max limit"));
        expect(fs.existsSync(stateFile)).toBe(false);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });

    it("accepts state file at max size boundary", async () => {
      const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), "gh-aw-state-"));
      try {
        const stateFile = path.join(tmpDir, "state.json");
        const prefix = '{"counts":{"a":"';
        const suffix = '"}}';
        const payloadLength = MAX_STATE_FILE_BYTES - prefix.length - suffix.length;
        const boundaryContent = `${prefix}${"x".repeat(payloadLength)}${suffix}`;
        const encoded = Buffer.from(boundaryContent, "utf8").toString("base64");

        process.env.GH_AW_EXPERIMENT_STATE_FILE = stateFile;
        process.env.GH_AW_EXPERIMENT_STATE_DIR = tmpDir;
        process.env.GH_AW_EXPERIMENT_BRANCH = "experiments/myworkflow";
        process.env.GITHUB_REPOSITORY = "owner/repo";

        global.github = {
          rest: {
            repos: {
              getContent: vi.fn().mockResolvedValue({
                data: { type: "file", content: encoded },
              }),
            },
          },
        };

        await main();

        expect(mockCore.warning).not.toHaveBeenCalledWith(expect.stringContaining("exceeds max limit"));
        expect(fs.readFileSync(stateFile, "utf8")).toBe(boundaryContent);
      } finally {
        fs.rmSync(tmpDir, { recursive: true, force: true });
      }
    });
  });
});
