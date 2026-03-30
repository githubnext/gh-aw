import { describe, it, expect, beforeEach, vi } from "vitest";

const mockCore = {
  debug: vi.fn(),
  info: vi.fn(),
  notice: vi.fn(),
  warning: vi.fn(),
  error: vi.fn(),
  setFailed: vi.fn(),
  setOutput: vi.fn(),
  exportVariable: vi.fn(),
  summary: {
    addRaw: vi.fn().mockReturnThis(),
    write: vi.fn().mockResolvedValue(),
  },
};

const mockGithub = {
  rest: {
    repos: {
      getContent: vi.fn(),
    },
  },
};

const mockContext = {
  repo: {
    owner: "test-owner",
    repo: "test-repo",
  },
  sha: "abc123",
};

const mockExec = {
  exec: vi.fn(),
};

global.core = mockCore;
global.github = mockGithub;
global.context = mockContext;
global.exec = mockExec;

describe("check_workflow_timestamp_api.cjs", () => {
  let main;

  beforeEach(async () => {
    vi.clearAllMocks();
    delete process.env.GH_AW_WORKFLOW_FILE;

    // Dynamically import the module to get fresh instance
    const module = await import("./check_workflow_timestamp_api.cjs");
    main = module.main;
  });

  describe("when environment variables are missing", () => {
    it("should fail if GH_AW_WORKFLOW_FILE is not set", async () => {
      await main();
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("GH_AW_WORKFLOW_FILE not available"));
    });
  });

  describe("when lock file is up to date", () => {
    beforeEach(() => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";
    });

    it("should pass when hashes match", async () => {
      // Hash for frontmatter "engine: copilot"
      const validHash = "c2a79263dc72f28c76177afda9bf0935481b26da094407a50155a6e0244084e3";
      const lockFileContent = `# frontmatter-hash: ${validHash}
name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"`;

      const mdFileContent = `---
engine: copilot
---
# Test Workflow`;

      mockGithub.rest.repos.getContent
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(lockFileContent).toString("base64"),
          },
        })
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(mdFileContent).toString("base64"),
          },
        });

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("✅ Lock file is up to date (hashes match)"));
      expect(mockCore.setFailed).not.toHaveBeenCalled();
      expect(mockCore.summary.addRaw).not.toHaveBeenCalled();
    });
  });

  describe("when lock file is outdated (hashes differ)", () => {
    beforeEach(() => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";
    });

    it("should fail when hashes differ", async () => {
      const storedHash = "c2a79263dc72f28c76177afda9bf0935481b26da094407a50155a6e0244084e3";
      const lockFileContent = `# frontmatter-hash: ${storedHash}
name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"`;

      // Different frontmatter - will produce different hash
      const mdFileContent = `---
engine: claude
model: claude-sonnet-4
---
# Test Workflow`;

      mockGithub.rest.repos.getContent
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(lockFileContent).toString("base64"),
          },
        })
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(mdFileContent).toString("base64"),
          },
        });

      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("Lock file"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("is outdated"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("frontmatter has changed"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("gh aw compile"));
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should fail when lock file is newer than source but hashes differ", async () => {
      // Security: a tampered lock file committed after the source must still fail
      const storedHash = "c2a79263dc72f28c76177afda9bf0935481b26da094407a50155a6e0244084e3";
      const lockFileContent = `# frontmatter-hash: ${storedHash}
name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"`;

      // Different frontmatter - tampered source
      const mdFileContent = `---
engine: claude
model: claude-sonnet-4
---
# Test Workflow`;

      mockGithub.rest.repos.getContent
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(lockFileContent).toString("base64"),
          },
        })
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(mdFileContent).toString("base64"),
          },
        });

      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("is outdated"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("frontmatter has changed"));
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should fail when hash check cannot be performed (no hash in lock file)", async () => {
      const lockFileContent = `name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest`;

      mockGithub.rest.repos.getContent.mockResolvedValueOnce({
        data: {
          type: "file",
          encoding: "base64",
          content: Buffer.from(lockFileContent).toString("base64"),
        },
      });

      await main();

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not compare frontmatter hashes"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("integrity check failed"));
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should fail when lock file content cannot be fetched", async () => {
      mockGithub.rest.repos.getContent.mockResolvedValueOnce({ data: null });

      await main();

      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not compare frontmatter hashes"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("integrity check failed"));
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });

    it("should include file paths in failure message when hashes differ", async () => {
      process.env.GH_AW_WORKFLOW_FILE = "my-workflow.lock.yml";

      const storedHash = "c2a79263dc72f28c76177afda9bf0935481b26da094407a50155a6e0244084e3";
      const lockFileContent = `# frontmatter-hash: ${storedHash}
name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"`;

      // Different frontmatter
      const mdFileContent = `---
engine: claude
---
# Test Workflow`;

      mockGithub.rest.repos.getContent
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(lockFileContent).toString("base64"),
          },
        })
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(mdFileContent).toString("base64"),
          },
        });

      await main();

      const failMessage = mockCore.setFailed.mock.calls[0][0];
      expect(failMessage).toMatch(/my-workflow\.lock\.yml/);
      expect(failMessage).toMatch(/my-workflow\.md/);
      expect(failMessage).toMatch(/outdated/);
    });

    it("should add step summary with warning details when hashes differ", async () => {
      const storedHash = "c2a79263dc72f28c76177afda9bf0935481b26da094407a50155a6e0244084e3";
      const lockFileContent = `# frontmatter-hash: ${storedHash}
name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"`;

      // Different frontmatter - will produce different hash
      const mdFileContent = `---
engine: claude
---
# Test Workflow`;

      mockGithub.rest.repos.getContent
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(lockFileContent).toString("base64"),
          },
        })
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(mdFileContent).toString("base64"),
          },
        });

      await main();

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Workflow Lock File Warning"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("WARNING"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("gh aw compile"));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });
  });

  describe("error handling", () => {
    beforeEach(() => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";
    });

    it("should handle API errors gracefully by failing", async () => {
      mockGithub.rest.repos.getContent.mockRejectedValue(new Error("API error"));

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Unable to fetch lock file content"));
      expect(mockCore.warning).toHaveBeenCalledWith(expect.stringContaining("Could not compare frontmatter hashes"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("integrity check failed"));
    });
  });

  describe("hash comparison details", () => {
    beforeEach(() => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";
    });

    it("should log hash values during comparison", async () => {
      const storedHash = "c2a79263dc72f28c76177afda9bf0935481b26da094407a50155a6e0244084e3";
      const lockFileContent = `# frontmatter-hash: ${storedHash}
name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"`;

      // Same frontmatter - hashes will match
      const mdFileContent = `---
engine: copilot
---
# Test Workflow`;

      mockGithub.rest.repos.getContent
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(lockFileContent).toString("base64"),
          },
        })
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(mdFileContent).toString("base64"),
          },
        });

      await main();

      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Frontmatter hash comparison"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Lock file hash"));
      expect(mockCore.info).toHaveBeenCalledWith(expect.stringContaining("Recomputed hash"));
    });

    it("should include hash values in summary on mismatch", async () => {
      const storedHash = "c2a79263dc72f28c76177afda9bf0935481b26da094407a50155a6e0244084e3";
      const lockFileContent = `# frontmatter-hash: ${storedHash}
name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"`;

      // Different frontmatter
      const mdFileContent = `---
engine: claude
---
# Test Workflow`;

      mockGithub.rest.repos.getContent
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(lockFileContent).toString("base64"),
          },
        })
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(mdFileContent).toString("base64"),
          },
        });

      await main();

      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("frontmatter hash mismatch"));
      expect(mockCore.summary.addRaw).toHaveBeenCalledWith(expect.stringContaining("Stored hash"));
      expect(mockCore.summary.write).toHaveBeenCalled();
    });
  });

  describe("lock file newer than source file (security fix)", () => {
    beforeEach(() => {
      process.env.GH_AW_WORKFLOW_FILE = "test.lock.yml";
    });

    it("should fail when lock file is newer but hashes differ", async () => {
      // Security fix: tampered lock file with newer timestamp must be rejected
      const storedHash = "c2a79263dc72f28c76177afda9bf0935481b26da094407a50155a6e0244084e3";
      const lockFileContent = `# frontmatter-hash: ${storedHash}
name: Test Workflow
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "test"`;

      // Different frontmatter - will produce different hash
      const mdFileContent = `---
engine: claude
model: claude-sonnet-4
---
# Test Workflow`;

      mockGithub.rest.repos.getContent
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(lockFileContent).toString("base64"),
          },
        })
        .mockResolvedValueOnce({
          data: {
            type: "file",
            encoding: "base64",
            content: Buffer.from(mdFileContent).toString("base64"),
          },
        });

      await main();

      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("is outdated"));
      expect(mockCore.setFailed).toHaveBeenCalledWith(expect.stringContaining("frontmatter has changed"));
      expect(mockCore.summary.addRaw).toHaveBeenCalled();
      expect(mockCore.summary.write).toHaveBeenCalled();
    });
  });
});
