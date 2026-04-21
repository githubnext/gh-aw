import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import fs from "fs";
import path from "path";

const COMMENT_MEMORY_DIR = "/tmp/gh-aw/comment-memory";
const PROMPT_PATH = "/tmp/gh-aw/aw-prompts/prompt.txt";
const CONFIG_PATH = "/tmp/gh-aw/test-comment-memory-config.json";

describe("setup_comment_memory_files", () => {
  beforeEach(() => {
    fs.rmSync(COMMENT_MEMORY_DIR, { recursive: true, force: true });
    fs.rmSync(path.dirname(PROMPT_PATH), { recursive: true, force: true });
    fs.mkdirSync(path.dirname(PROMPT_PATH), { recursive: true });
    fs.writeFileSync(PROMPT_PATH, "base prompt\n");
    process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH = CONFIG_PATH;

    global.core = {
      info: vi.fn(),
      warning: vi.fn(),
      debug: vi.fn(),
    };

    global.context = {
      payload: { issue: { number: 42 } },
      repo: { owner: "octo", repo: "repo" },
    };
  });

  afterEach(() => {
    fs.rmSync(COMMENT_MEMORY_DIR, { recursive: true, force: true });
    fs.rmSync(path.dirname(PROMPT_PATH), { recursive: true, force: true });
    fs.rmSync(CONFIG_PATH, { force: true });
    delete process.env.GH_AW_SAFE_OUTPUTS_CONFIG_PATH;
  });

  it("extracts memory entries from managed comment body", async () => {
    const module = await import("./setup_comment_memory_files.cjs");
    const entries = module.extractCommentMemoryEntries('<gh-aw-comment-memory id="default">\nhello\n</gh-aw-comment-memory>');
    expect(entries).toEqual([{ memoryId: "default", content: "hello" }]);
  });

  it("writes comment memory files and injects prompt guidance", async () => {
    fs.writeFileSync(CONFIG_PATH, JSON.stringify({ "comment-memory": { target: "triggering" } }));
    global.github = {
      rest: {
        issues: {
          listComments: vi.fn().mockResolvedValue({
            data: [
              {
                body: '<gh-aw-comment-memory id="default">\nSaved memory\n</gh-aw-comment-memory>\nfooter',
              },
            ],
          }),
        },
      },
    };

    const module = await import("./setup_comment_memory_files.cjs");
    await module.main();

    const memoryFile = path.join(COMMENT_MEMORY_DIR, "default.md");
    expect(fs.existsSync(memoryFile)).toBe(true);
    expect(fs.readFileSync(memoryFile, "utf8")).toBe("Saved memory\n");

    const prompt = fs.readFileSync(PROMPT_PATH, "utf8");
    expect(prompt).toContain("/tmp/gh-aw/comment-memory");
    expect(prompt).toContain("/tmp/gh-aw/comment-memory/default.md");
  });

  it("continues scanning past initial pages without entries", async () => {
    fs.writeFileSync(CONFIG_PATH, JSON.stringify({ "comment-memory": { target: "triggering" } }));
    const listComments = vi.fn().mockImplementation(({ page }) => {
      if (page <= 5) {
        return Promise.resolve({
          data: Array.from({ length: 100 }, () => ({ body: "normal comment without memory marker" })),
        });
      }
      if (page === 6) {
        return Promise.resolve({ data: [{ body: '<gh-aw-comment-memory id="default">\nLate memory\n</gh-aw-comment-memory>' }] });
      }
      return Promise.resolve({ data: [] });
    });
    global.github = {
      rest: {
        issues: {
          listComments,
        },
      },
    };

    const module = await import("./setup_comment_memory_files.cjs");
    await module.main();

    const memoryFile = path.join(COMMENT_MEMORY_DIR, "default.md");
    expect(fs.existsSync(memoryFile)).toBe(true);
    expect(fs.readFileSync(memoryFile, "utf8")).toBe("Late memory\n");
    expect(listComments).toHaveBeenCalledTimes(6);
  });

  it("rejects cross-repo comment-memory setup when no allowlist is configured", async () => {
    fs.writeFileSync(CONFIG_PATH, JSON.stringify({ "comment-memory": { target: "triggering", "target-repo": "other-org/other-repo" } }));
    const listComments = vi.fn().mockResolvedValue({ data: [] });
    global.github = {
      rest: {
        issues: {
          listComments,
        },
      },
    };

    const module = await import("./setup_comment_memory_files.cjs");
    await module.main();

    expect(listComments).not.toHaveBeenCalled();
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("E004"));
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("No allowlist is configured"));
  });

  it("rejects cross-repo comment-memory setup when target repo is not in allowlist", async () => {
    fs.writeFileSync(
      CONFIG_PATH,
      JSON.stringify({
        "comment-memory": {
          target: "triggering",
          "target-repo": "other-org/other-repo",
          allowed_repos: ["other-org/different-repo"],
        },
      })
    );
    const listComments = vi.fn().mockResolvedValue({ data: [] });
    global.github = {
      rest: {
        issues: {
          listComments,
        },
      },
    };

    const module = await import("./setup_comment_memory_files.cjs");
    await module.main();

    expect(listComments).not.toHaveBeenCalled();
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("E004"));
    expect(global.core.warning).toHaveBeenCalledWith(expect.stringContaining("not in the allowed-repos list"));
  });

  it("allows cross-repo comment-memory setup when target repo is in allowlist", async () => {
    fs.writeFileSync(
      CONFIG_PATH,
      JSON.stringify({
        "comment-memory": {
          target: "triggering",
          "target-repo": "other-org/other-repo",
          allowed_repos: ["other-org/other-repo"],
        },
      })
    );
    const listComments = vi.fn().mockResolvedValue({
      data: [{ body: '<gh-aw-comment-memory id="default">\nCross repo memory\n</gh-aw-comment-memory>' }],
    });
    global.github = {
      rest: {
        issues: {
          listComments,
        },
      },
    };

    const module = await import("./setup_comment_memory_files.cjs");
    await module.main();

    expect(listComments).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "other-org",
        repo: "other-repo",
        issue_number: 42,
      })
    );
    const memoryFile = path.join(COMMENT_MEMORY_DIR, "default.md");
    expect(fs.existsSync(memoryFile)).toBe(true);
    expect(fs.readFileSync(memoryFile, "utf8")).toBe("Cross repo memory\n");
  });

  it("treats target-repo as same repo when slug differs only by case", async () => {
    fs.writeFileSync(
      CONFIG_PATH,
      JSON.stringify({
        "comment-memory": {
          target: "triggering",
          "target-repo": "Octo/Repo",
        },
      })
    );
    const listComments = vi.fn().mockResolvedValue({
      data: [{ body: '<gh-aw-comment-memory id="default">\nSame repo memory\n</gh-aw-comment-memory>' }],
    });
    global.github = {
      rest: {
        issues: {
          listComments,
        },
      },
    };

    const module = await import("./setup_comment_memory_files.cjs");
    await module.main();

    expect(listComments).toHaveBeenCalledWith(
      expect.objectContaining({
        owner: "Octo",
        repo: "Repo",
        issue_number: 42,
      })
    );
    expect(global.core.warning).not.toHaveBeenCalledWith(expect.stringContaining("E004"));
  });
});
